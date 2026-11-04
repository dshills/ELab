package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dshills/elab/workbench"
)

const (
	backupFormat      = "elab-library"
	maxBackupArchive  = 64 << 20
	maxBackupExpanded = 128 << 20
	maxBackupRecords  = 256
	maxBackupManifest = 1 << 20
)

type backupRecord struct {
	Kind      string         `json:"kind"`
	SourceID  string         `json:"source_id"`
	File      string         `json:"file"`
	Size      int64          `json:"size"`
	SHA256    string         `json:"sha256"`
	Entry     *projectEntry  `json:"entry,omitempty"`
	Workspace *workspaceInfo `json:"workspace,omitempty"`
	Dirty     bool           `json:"dirty,omitempty"`
}
type backupManifest struct {
	Format    string         `json:"format"`
	Version   int            `json:"version"`
	CreatedAt string         `json:"created_at"`
	Records   []backupRecord `json:"records"`
}
type backupPayload struct {
	record backupRecord
	data   []byte
}

func digest(data []byte) string   { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func validTime(value string) bool { _, err := time.Parse(time.RFC3339Nano, value); return err == nil }
func validateBackupRecord(record backupRecord, index int) error {
	if !managedID.MatchString(record.SourceID) || record.File != fmt.Sprintf("records/%04d.elab", index) || record.Size < 1 || record.Size > workbench.MaxProjectBytes || len(record.SHA256) != 64 {
		return fmt.Errorf("invalid backup record %d", index+1)
	}
	if _, err := hex.DecodeString(record.SHA256); err != nil {
		return fmt.Errorf("invalid document checksum")
	}
	if record.Kind == "workspaces" {
		if record.Workspace == nil || record.Entry != nil || !validTime(record.Workspace.AutosavedAt) || !utf8.ValidString(record.Workspace.Name) || len(record.Workspace.Name) > 1024 {
			return fmt.Errorf("invalid workspace metadata")
		}
	} else if record.Kind == "projects" || record.Kind == "history" {
		e := record.Entry
		if e == nil || record.Workspace != nil || e.ID != record.SourceID || e.SHA256 != record.SHA256 || e.File != "" || e.Checkpoint || record.Dirty || !validTime(e.UpdatedAt) || !utf8.ValidString(e.Name) || len(e.Name) > 1024 {
			return fmt.Errorf("invalid project metadata")
		}
	} else {
		return fmt.Errorf("unknown backup record kind")
	}
	return nil
}

// Called with the library store and all active workspace file locks held.
func collectLibrary(root string) ([]backupPayload, error) {
	payloads := []backupPayload{}
	seen := map[string]string{}
	var total int64
	for _, kind := range []string{"projects", "history", "workspaces"} {
		locations, err := libraryLocations(root, kind)
		if err != nil {
			return nil, err
		}
		// Earlier web builds kept projects inside workspace directories. Include
		// dormant legacy stores too, without exporting scratch/control files.
		if kind != "workspaces" {
			workspaceLocations, e := libraryLocations(root, "workspaces")
			if e != nil {
				return nil, e
			}
			for _, base := range workspaceLocations {
				workspaces, e := os.ReadDir(base)
				if e != nil {
					return nil, e
				}
				for _, ws := range workspaces {
					if !managedID.MatchString(ws.Name()) {
						continue
					}
					if !ws.IsDir() {
						return nil, fmt.Errorf("invalid workspace directory")
					}
					legacy, e := libraryLocations(filepath.Join(base, ws.Name()), kind)
					if e != nil {
						return nil, e
					}
					locations = append(locations, legacy...)
				}
			}
		}
		for _, location := range locations {
			dirs, err := os.ReadDir(location)
			if err != nil {
				return nil, err
			}
			for _, dir := range dirs {
				if !managedID.MatchString(dir.Name()) {
					continue
				}
				if !dir.IsDir() {
					return nil, fmt.Errorf("%s contains a linked or invalid entry", kind)
				}
				record := backupRecord{Kind: kind, SourceID: dir.Name(), File: fmt.Sprintf("records/%04d.elab", len(payloads))}
				var data []byte
				if kind == "workspaces" {
					checkpoint, e := readCheckpoint(filepath.Join(location, dir.Name(), "workspace.json"))
					if e != nil {
						return nil, fmt.Errorf("workspace checkpoint unavailable: %w", e)
					}
					info := checkpoint.Workspace
					info.Error = ""
					record.Workspace = &info
					record.Dirty = checkpoint.Dirty
					data = checkpoint.Document
				} else {
					raw, e := readBounded(filepath.Join(location, dir.Name(), "entry.json"), 8192)
					if e != nil {
						return nil, fmt.Errorf("%s metadata unavailable: %w", kind, e)
					}
					var entry projectEntry
					if workbench.DecodeStrict(raw, &entry) != nil || entry.ID != dir.Name() || (entry.File != "" && !managedID.MatchString(entry.File)) {
						return nil, fmt.Errorf("invalid %s metadata", kind)
					}
					filename := "project.elab"
					if entry.File != "" {
						filename = entry.File + ".elab"
					}
					data, err = readBounded(filepath.Join(location, dir.Name(), filename), workbench.MaxProjectBytes)
					if err != nil {
						return nil, err
					}
					entry.File = ""
					record.Entry = &entry
				}
				record.Size = int64(len(data))
				record.SHA256 = digest(data)
				if kind == "workspaces" {
					record.SourceID = digest([]byte("workspace:" + record.SourceID + ":" + record.SHA256))[:48]
				}
				identity := record.Kind + "/" + record.SourceID
				if previous, exists := seen[identity]; exists {
					if previous == record.SHA256 {
						continue
					}
					// An older workspace copy may predate a shared-library overwrite. Keep
					// both versions under distinct portable identities, without migrating disk.
					record.SourceID = digest([]byte(identity + ":" + record.SHA256))[:48]
					if record.Entry != nil {
						record.Entry.ID = record.SourceID
					}
					identity = record.Kind + "/" + record.SourceID
					if seen[identity] == record.SHA256 {
						continue
					}
				}
				if len(payloads) >= maxBackupRecords {
					return nil, fmt.Errorf("library exceeds the %d-record backup limit", maxBackupRecords)
				}
				seen[identity] = record.SHA256
				if err := validateBackupRecord(record, len(payloads)); err != nil {
					return nil, err
				}
				if err := workbench.ValidateProject(data); err != nil {
					return nil, fmt.Errorf("invalid %s document: %w", kind, err)
				}
				total += record.Size
				if total > maxBackupExpanded {
					return nil, fmt.Errorf("library exceeds the 128 MiB expanded backup limit")
				}
				payloads = append(payloads, backupPayload{record, data})
			}
		}
	}
	return payloads, nil
}

type boundedBackupWriter struct {
	writer    io.Writer
	remaining int
}

func (w *boundedBackupWriter) Write(p []byte) (int, error) {
	if len(p) > w.remaining {
		return 0, fmt.Errorf("backup exceeds the 64 MiB archive limit")
	}
	n, err := w.writer.Write(p)
	w.remaining -= n
	return n, err
}
func encodeLibraryBackup(payloads []backupPayload) ([]byte, error) {
	manifest := backupManifest{Format: backupFormat, Version: 1, CreatedAt: stamp(), Records: []backupRecord{}}
	for _, p := range payloads {
		manifest.Records = append(manifest.Records, p.record)
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBackupManifest {
		return nil, fmt.Errorf("backup manifest exceeds its limit")
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&boundedBackupWriter{&out, maxBackupArchive})
	add := func(name string, data []byte) error {
		f, e := zw.Create(name)
		if e != nil {
			return e
		}
		_, e = f.Write(data)
		return e
	}
	if err = add("manifest.json", raw); err != nil {
		return nil, err
	}
	for _, p := range payloads {
		if err = add(p.record.File, p.data); err != nil {
			return nil, err
		}
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// A single bounded transfer at a time limits aggregate ZIP/validation memory.
func (s *webServer) admitLibraryTransfer(w http.ResponseWriter) bool {
	select {
	case s.libraryIO <- struct{}{}:
		return true
	default:
		webError(w, 409, "Another library backup or restore is in progress. Try again after it finishes.")
		return false
	}
}
func (s *webServer) libraryBackup(w http.ResponseWriter, r *http.Request, ws *workspace) {
	if r.Method != http.MethodPost {
		webError(w, 405, "POST required")
		return
	}
	if !s.admitLibraryTransfer(w) {
		return
	}
	defer func() { <-s.libraryIO }()
	rev, gen, err := guards(r)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	// Lock order matches workspace creation: registry, library, workspace files.
	s.mu.Lock()
	s.storeMu.Lock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	sessions := make([]*workspace, 0, len(ids))
	for _, id := range ids {
		item := s.sessions[id]
		item.files.Lock()
		sessions = append(sessions, item)
	}
	s.mu.Unlock()
	payloads, err := func() ([]backupPayload, error) {
		defer s.storeMu.Unlock()
		defer func() {
			for i := len(sessions) - 1; i >= 0; i-- {
				sessions[i].files.Unlock()
			}
		}()
		state := ws.service.Snapshot()
		if state.CommandRevision != rev || state.Document.Generation != gen {
			return nil, fmt.Errorf("workspace changed; refresh before backing up")
		}
		for _, item := range sessions {
			if err := item.persist(); err != nil {
				return nil, fmt.Errorf("backup stopped because an active workspace could not be checkpointed: %w", err)
			}
		}
		if err := r.Context().Err(); err != nil {
			return nil, err
		}
		return collectLibrary(s.root)
	}()
	if err != nil {
		webError(w, 409, err.Error())
		return
	}
	data, err := encodeLibraryBackup(payloads)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="ELab-library-`+time.Now().UTC().Format("20060102T150405Z")+`.zip"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-ELab-Backup-SHA256", digest(data))
	_, _ = w.Write(data)
}
func readZipFile(f *zip.File, limit int64) ([]byte, error) {
	if !f.Mode().IsRegular() || f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("invalid or oversized backup file")
	}
	reader, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("backup file exceeds its bound")
	}
	return data, nil
}
func decodeLibraryBackup(data []byte) ([]backupPayload, error) {
	if len(data) > maxBackupArchive {
		return nil, fmt.Errorf("backup exceeds 64 MiB")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("choose an ELab library ZIP backup")
	}
	if len(zr.File) > maxBackupRecords+1 {
		return nil, fmt.Errorf("backup has too many files")
	}
	files := map[string]*zip.File{}
	var expanded uint64
	for _, f := range zr.File {
		if files[f.Name] != nil || !f.Mode().IsRegular() {
			return nil, fmt.Errorf("duplicate or non-regular backup file")
		}
		expanded += f.UncompressedSize64
		if f.UncompressedSize64 > workbench.MaxProjectBytes || expanded > maxBackupExpanded+maxBackupManifest {
			return nil, fmt.Errorf("backup exceeds expanded limits")
		}
		files[f.Name] = f
	}
	mf := files["manifest.json"]
	if mf == nil {
		return nil, fmt.Errorf("backup manifest is missing")
	}
	raw, err := readZipFile(mf, maxBackupManifest)
	if err != nil {
		return nil, err
	}
	var manifest backupManifest
	if workbench.DecodeStrict(raw, &manifest) != nil || manifest.Format != backupFormat || manifest.Version != 1 || !validTime(manifest.CreatedAt) || manifest.Records == nil || len(manifest.Records) > maxBackupRecords || len(files) != len(manifest.Records)+1 {
		return nil, fmt.Errorf("unsupported or invalid library manifest")
	}
	payloads := []backupPayload{}
	seen := map[string]bool{}
	var total int64
	for i, record := range manifest.Records {
		if err := validateBackupRecord(record, i); err != nil {
			return nil, err
		}
		identity := record.Kind + "/" + record.SourceID
		if seen[identity] {
			return nil, fmt.Errorf("duplicate library identity")
		}
		seen[identity] = true
		f := files[record.File]
		if f == nil || f.UncompressedSize64 != uint64(record.Size) {
			return nil, fmt.Errorf("backup document is missing or has the wrong size")
		}
		total += record.Size
		if total > maxBackupExpanded {
			return nil, fmt.Errorf("backup exceeds expanded limits")
		}
		document, err := readZipFile(f, workbench.MaxProjectBytes)
		if err != nil {
			return nil, err
		}
		if digest(document) != record.SHA256 {
			return nil, fmt.Errorf("backup document checksum mismatch")
		}
		if err = workbench.ValidateProject(document); err != nil {
			return nil, fmt.Errorf("invalid backed-up project: %w", err)
		}
		payloads = append(payloads, backupPayload{record, document})
	}
	return payloads, nil
}

func stageLibraryRestore(root string, payloads []backupPayload) (string, map[string]int, error) {
	stage, err := os.MkdirTemp(root, ".restore-")
	if err != nil {
		return "", nil, err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(stage)
		}
	}()
	counts := map[string]int{"projects": 0, "history": 0, "workspaces": 0}
	for _, payload := range payloads {
		record := payload.record
		id := token()
		dir := filepath.Join(stage, record.Kind, id)
		if record.Kind == "workspaces" {
			info := *record.Workspace
			info.ProjectID = ""
			info.SavedSHA = ""
			info.SavedAt = ""
			info.Error = ""
			info.Recovered = true
			info.Name = strings.TrimSpace(info.Name)
			if info.Name == "" {
				info.Name = "Untitled circuit"
			}
			info.Name += " (restored workspace)"
			if err = writeJSON(filepath.Join(dir, "workspace.json"), workspaceCheckpoint{Version: 1, Workspace: info, Dirty: true, Document: payload.data}); err != nil {
				return "", nil, err
			}
		} else {
			entry := *record.Entry
			entry.ID = id
			entry.File = ""
			entry.Retained = record.Kind == "history"
			if err = atomicWrite(filepath.Join(dir, "project.elab"), payload.data); err != nil {
				return "", nil, err
			}
			if err = writeJSON(filepath.Join(dir, "entry.json"), entry); err != nil {
				return "", nil, err
			}
		}
		counts[record.Kind]++
	}
	success = true
	return stage, counts, nil
}
func (s *webServer) libraryRestore(w http.ResponseWriter, r *http.Request, ws *workspace) {
	if r.Method != http.MethodPost {
		webError(w, 405, "POST required")
		return
	}
	if !s.admitLibraryTransfer(w) {
		return
	}
	defer func() { <-s.libraryIO }()
	rev, gen, err := guards(r)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBackupArchive))
	if err != nil {
		webError(w, 400, "Backup exceeds 64 MiB or could not be read.")
		return
	}
	payloads, err := decodeLibraryBackup(raw)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	if len(payloads) == 0 {
		webError(w, 400, "This backup has no projects or workspaces to restore.")
		return
	}
	if r.URL.Query().Get("preview") == "true" {
		counts := map[string]int{"projects": 0, "history": 0, "workspaces": 0, "trash": 0}
		for _, p := range payloads {
			counts[p.record.Kind]++
			if p.record.Entry != nil && p.record.Entry.Trashed {
				counts["trash"]++
			}
		}
		jsonReply(w, 200, map[string]any{"contents": counts})
		return
	}
	stage, counts, err := stageLibraryRestore(s.root, payloads)
	if err != nil {
		webError(w, 500, "Could not stage restore: "+err.Error())
		return
	}
	defer os.RemoveAll(stage)
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	ws.files.Lock()
	defer ws.files.Unlock()
	state := ws.service.Snapshot()
	if state.CommandRevision != rev || state.Document.Generation != gen {
		webError(w, 409, "The workspace changed; review before restoring copies.")
		return
	}
	if err = r.Context().Err(); err != nil {
		return
	}
	batches := filepath.Join(s.root, "restores")
	if err = os.MkdirAll(batches, 0700); err == nil {
		_, err = libraryLocations(s.root, "projects")
	}
	if err == nil {
		err = os.Rename(stage, filepath.Join(batches, token()))
	}
	if err != nil {
		webError(w, 500, "Restore was not published: "+err.Error())
		return
	}
	if dir, e := os.Open(batches); e == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	jsonReply(w, 200, map[string]any{"restored": counts, "message": "Restored copies added. Existing projects and the open circuit were preserved."})
}
