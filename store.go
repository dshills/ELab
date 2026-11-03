package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dshills/elab/workbench"
)

var managedID = regexp.MustCompile(`^[a-f0-9]{48}$`)

type workspaceInfo struct {
	Name        string `json:"name"`
	SavedSHA    string `json:"saved_sha256,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	Recovered   bool   `json:"recovered"`
	SavedAt     string `json:"saved_at,omitempty"`
	AutosavedAt string `json:"autosaved_at,omitempty"`
	Error       string `json:"error,omitempty"`
}
type workspaceCheckpoint struct {
	Version   int             `json:"version"`
	Workspace workspaceInfo   `json:"workspace"`
	Dirty     bool            `json:"dirty"`
	Document  json.RawMessage `json:"document"`
}
type projectEntry struct {
	Checkpoint bool   `json:"checkpoint,omitempty"`
	File       string `json:"file,omitempty"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	UpdatedAt  string `json:"updated_at"`
	SHA256     string `json:"sha256"`
	Trashed    bool   `json:"trashed,omitempty"`
}

func stamp() string { return time.Now().UTC().Format(time.RFC3339Nano) }
func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".elab-web-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return err
	}
	if dir, e := os.Open(filepath.Dir(path)); e == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}
func writeJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}
func readBounded(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("expected an owned regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		return nil, fmt.Errorf("expected a regular project file")
	}
	if st.Size() > limit {
		return nil, fmt.Errorf("project exceeds size limit")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("project exceeds size limit")
	}
	return data, err
}
func (ws *workspace) persist() error {
	capture, err := ws.service.CaptureProject()
	if err != nil {
		ws.info.Error = err.Error()
		return err
	}
	sum := sha256.Sum256(capture.Data)
	signature := fmt.Sprintf("%x:%t:%s:%s:%t:%s:%s", sum, capture.Dirty, ws.info.Name, ws.info.ProjectID, ws.info.Recovered, ws.info.SavedAt, ws.info.SavedSHA)
	if signature == ws.persisted && ws.info.Error == "" {
		return nil
	}
	next := ws.info
	next.AutosavedAt = stamp()
	next.Error = ""
	checkpoint := workspaceCheckpoint{Version: 1, Workspace: next, Dirty: capture.Dirty || next.Recovered, Document: capture.Data}
	if err = writeJSON(filepath.Join(ws.directory, "workspace.json"), checkpoint); err != nil {
		ws.info.Error = "Autosave failed: " + err.Error()
		return err
	}
	ws.info = next
	ws.persisted = signature
	return nil
}
func (ws *workspace) restore(ctx context.Context) error {
	data, err := readBounded(filepath.Join(ws.directory, "workspace.json"), workbench.MaxProjectBytes+65536)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var checkpoint workspaceCheckpoint
	if err = json.Unmarshal(data, &checkpoint); err != nil || checkpoint.Version != 1 || len(checkpoint.Document) == 0 {
		return fmt.Errorf("workspace checkpoint is invalid; preserved for recovery")
	}
	path := filepath.Join(ws.directory, "restore.elab")
	if err = atomicWrite(path, checkpoint.Document); err != nil {
		return err
	}
	state := ws.service.Snapshot()
	response := fileCommand(ctx, ws, "project.load", workbench.LoadParams{Path: path, DiscardChanges: true}, state.CommandRevision, state.Document.Generation)
	if response.Error != nil {
		return fmt.Errorf("restore: %s", response.Error.Message)
	}
	ws.info = checkpoint.Workspace
	ws.info.Recovered = checkpoint.Dirty
	ws.restored = true
	return nil
}
func (ws *workspace) archive() error {
	capture, err := ws.service.CaptureProject()
	if err != nil {
		return err
	}
	return ws.archiveData(capture.Data, ws.title())
}
func (ws *workspace) archiveData(data []byte, name string) error {
	sum := sha256.Sum256(data)
	entry := projectEntry{ID: token(), Name: name, UpdatedAt: stamp(), SHA256: hex.EncodeToString(sum[:])}
	dir := filepath.Join(ws.library, "history", entry.ID)
	if err := atomicWrite(filepath.Join(dir, "project.elab"), data); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "entry.json"), entry); err != nil {
		return err
	}
	// Retention affects only owned automatic history, never saved projects.
	entries, _ := ws.entries("history")
	if len(entries) > 50 {
		for _, old := range entries[50:] {
			_ = os.RemoveAll(filepath.Join(ws.library, "history", old.ID))
		}
	}
	return nil
}
func (ws *workspace) title() string {
	if ws.info.Name != "" {
		return ws.info.Name
	}
	state := ws.service.Snapshot()
	if state.Example != nil {
		return state.Example.Info.Title
	}
	return "Untitled circuit"
}
func (ws *workspace) entries(kind string) ([]projectEntry, error) {
	dirs, err := os.ReadDir(filepath.Join(ws.library, kind))
	if os.IsNotExist(err) {
		return []projectEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries := []projectEntry{}
	for _, d := range dirs {
		if !d.IsDir() || !managedID.MatchString(d.Name()) {
			continue
		}
		data, e := readBounded(filepath.Join(ws.library, kind, d.Name(), "entry.json"), 8192)
		if e != nil {
			continue
		}
		var entry projectEntry
		if json.Unmarshal(data, &entry) == nil && entry.ID == d.Name() {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].UpdatedAt > entries[j].UpdatedAt })
	return entries, nil
}
func (ws *workspace) presentation(state workbench.Snapshot) map[string]any {
	state.Document.Dirty = state.Document.Dirty || ws.info.Recovered
	return map[string]any{"state": state, "scene": makeScene(state), "workspace": ws.info}
}
func decodeJSON(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("one JSON object required")
	}
	return nil
}
func (s *webServer) projects(w http.ResponseWriter, r *http.Request, ws *workspace) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()
	ws.files.Lock()
	defer ws.files.Unlock()
	if r.Method == "GET" {
		saved, err := ws.entries("projects")
		if err != nil {
			webError(w, 500, err.Error())
			return
		}
		history, err := ws.entries("history")
		if err != nil {
			webError(w, 500, err.Error())
			return
		}
		checkpoints, err := ws.otherCheckpoints()
		if err != nil {
			webError(w, 500, err.Error())
			return
		}
		history = append(history, checkpoints...)
		sort.Slice(history, func(i, j int) bool { return history[i].UpdatedAt > history[j].UpdatedAt })
		jsonReply(w, 200, map[string]any{"projects": saved, "history": history, "workspace": ws.info})
		return
	}
	if r.Method != "POST" {
		webError(w, 405, "GET or POST required")
		return
	}
	rev, gen, err := guards(r)
	if err != nil {
		webError(w, 400, err.Error())
		return
	}
	var input struct {
		Checkpoint bool   `json:"checkpoint"`
		SHA256     string `json:"sha256"`
		Action     string `json:"action"`
		ID         string `json:"id"`
		Name       string `json:"name"`
		Discard    bool   `json:"discard_changes"`
	}
	if decodeJSON(w, r, &input) != nil {
		webError(w, 400, "Invalid project request")
		return
	}
	if input.ID != "" && !managedID.MatchString(input.ID) {
		webError(w, 400, "Choose a listed project")
		return
	}
	state := ws.service.Snapshot()
	if state.CommandRevision != rev || state.Document.Generation != gen {
		webError(w, 409, "The workspace changed; refresh before continuing.")
		return
	}
	switch input.Action {
	case "save":
		name := strings.TrimSpace(input.Name)
		if name == "" || utf8.RuneCountInString(name) > 120 {
			webError(w, 400, "Choose a project name of 1–120 characters.")
			return
		}
		id := input.ID
		if id == "" {
			id = token()
		} else if _, e := readBounded(filepath.Join(ws.library, "projects", id, "entry.json"), 8192); e != nil {
			webError(w, 404, "Project not found")
			return
		}
		if input.ID != "" {
			raw, e := readBounded(filepath.Join(ws.library, "projects", id, "entry.json"), 8192)
			if e != nil {
				webError(w, 500, e.Error())
				return
			}
			var previous projectEntry
			if json.Unmarshal(raw, &previous) != nil || previous.ID != id || (previous.File != "" && !managedID.MatchString(previous.File)) {
				webError(w, 400, "Invalid project metadata")
				return
			}
			if ws.info.ProjectID != id || ws.info.SavedSHA != previous.SHA256 || ws.info.SavedAt != previous.UpdatedAt {
				webError(w, 409, "Project changed in another workspace. Open its current version or save a copy.")
				return
			}
			data, e := readBounded(entryPath(ws.library, "projects", previous), workbench.MaxProjectBytes)
			if e != nil {
				webError(w, 500, e.Error())
				return
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != previous.SHA256 {
				webError(w, 409, "Stored project changed; save a copy to preserve both versions.")
				return
			}
			if err = ws.archiveData(data, previous.Name); err != nil {
				webError(w, 500, err.Error())
				return
			}
		} else if err = ws.archive(); err != nil {
			webError(w, 500, err.Error())
			return
		}
		dir := filepath.Join(ws.library, "projects", id)
		if err = os.MkdirAll(dir, 0700); err != nil {
			webError(w, 500, err.Error())
			return
		}
		version := token()
		path := filepath.Join(dir, version+".elab")
		response := fileCommand(r.Context(), ws, "project.save", workbench.SaveParams{Path: path, Overwrite: false}, rev, gen)
		if response.Error != nil {
			jsonReply(w, 409, response)
			return
		}
		ws.info.Recovered = true
		data, e := readBounded(path, workbench.MaxProjectBytes)
		if e != nil {
			_ = ws.persist()
			webError(w, 500, e.Error())
			return
		}
		sum := sha256.Sum256(data)
		entry := projectEntry{ID: id, Name: name, UpdatedAt: stamp(), SHA256: hex.EncodeToString(sum[:]), File: version}
		if err = writeJSON(filepath.Join(dir, "entry.json"), entry); err != nil {
			// The native savepoint advanced, but publication failed. Preserve the
			// editor as an unsaved recovery copy and keep the old entry intact.
			ws.info.Recovered = true
			_ = ws.persist()
			ws.info.Error = "Project save failed: " + err.Error()
			webError(w, 500, ws.info.Error)
			return
		}
		// Previous published data was archived before changing the entry pointer.
		// Remove only obsolete owned revisions; never touch another project.
		revisions, _ := os.ReadDir(dir)
		for _, revision := range revisions {
			base := strings.TrimSuffix(revision.Name(), ".elab")
			if revision.Name() != version+".elab" && managedID.MatchString(base) && strings.HasSuffix(revision.Name(), ".elab") {
				_ = os.Remove(filepath.Join(dir, revision.Name()))
			}
		}
		ws.info.Name = name
		ws.info.ProjectID = id
		ws.info.Recovered = false
		ws.info.SavedAt = entry.UpdatedAt
		ws.info.SavedSHA = entry.SHA256
	case "open", "recover":
		if (state.Document.Dirty || ws.info.Recovered) && !input.Discard {
			webError(w, 409, "Save or confirm replacing your current circuit first.")
			return
		}
		kind := "projects"
		if input.Action == "recover" {
			kind = "history"
		}

		var entry projectEntry
		var data []byte
		var path string
		if input.Checkpoint && input.Action == "recover" {
			checkpoint, e := readCheckpoint(filepath.Join(ws.library, "workspaces", input.ID, "workspace.json"))
			if e != nil {
				webError(w, 404, "Autosaved workspace is unavailable; its files were preserved.")
				return
			}
			data = checkpoint.Document
			sum := sha256.Sum256(data)
			digest := hex.EncodeToString(sum[:])
			if input.SHA256 == "" || digest != input.SHA256 {
				webError(w, 409, "This autosaved workspace changed. Refresh recovery history before restoring.")
				return
			}
			entry = projectEntry{ID: input.ID, Name: checkpoint.Workspace.Name, SHA256: digest}
			if entry.Name == "" {
				entry.Name = "Autosaved circuit"
			}
			path = filepath.Join(ws.directory, "recovery-import.elab")
			if err = atomicWrite(path, data); err != nil {
				webError(w, 500, err.Error())
				return
			}
			defer os.Remove(path)
		} else {
			meta, e := readBounded(filepath.Join(ws.library, kind, input.ID, "entry.json"), 8192)
			if e != nil {
				webError(w, 404, "Project not found")
				return
			}
			if json.Unmarshal(meta, &entry) != nil || entry.ID != input.ID || (entry.File != "" && !managedID.MatchString(entry.File)) {
				webError(w, 400, "Invalid project metadata")
				return
			}
			path = entryPath(ws.library, kind, entry)
			data, e = readBounded(path, workbench.MaxProjectBytes)
			if e != nil {
				webError(w, 500, e.Error())
				return
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != entry.SHA256 {
				webError(w, 409, "The stored project changed; its previous metadata was preserved.")
				return
			}
		}
		if input.Action == "recover" && !input.Checkpoint {
			// Archiving can rotate the oldest history entry. Load the already
			// validated owned bytes so selecting that entry is still safe.
			path = filepath.Join(ws.directory, "recovery-import.elab")
			if err = atomicWrite(path, data); err != nil {
				webError(w, 500, err.Error())
				return
			}
			defer os.Remove(path)
		}

		if err = ws.archive(); err != nil {
			webError(w, 500, err.Error())
			return
		}
		response := fileCommand(r.Context(), ws, "project.load", workbench.LoadParams{Path: path, DiscardChanges: input.Discard}, rev, gen)
		if response.Error != nil {
			jsonReply(w, 409, response)
			return
		}
		ws.info.Name = entry.Name
		ws.info.ProjectID = input.ID
		ws.info.Recovered = false
		ws.info.SavedAt = entry.UpdatedAt
		ws.info.SavedSHA = entry.SHA256
		if input.Action == "recover" {
			ws.info.ProjectID = ""
			ws.info.Recovered = true
			ws.info.Name = entry.Name + " (recovered)"
		}
	case "trash", "untrash":
		dir := filepath.Join(ws.library, "projects", input.ID)
		data, e := readBounded(filepath.Join(dir, "entry.json"), 8192)
		if e != nil {
			webError(w, 404, "Project not found")
			return
		}
		var entry projectEntry
		if json.Unmarshal(data, &entry) != nil {
			webError(w, 400, "Invalid project metadata")
			return
		}
		entry.Trashed = input.Action == "trash"
		if err = writeJSON(filepath.Join(dir, "entry.json"), entry); err != nil {
			webError(w, 500, err.Error())
			return
		}
	default:
		webError(w, 400, "Choose save, open, recover, trash, or untrash")
		return
	}
	_ = ws.persist()
	jsonReply(w, 200, ws.presentation(ws.service.Snapshot()))
}

func (s *webServer) autosave(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		s.mu.Lock()
		sessions := make([]*workspace, 0, len(s.sessions))
		for _, ws := range s.sessions {
			sessions = append(sessions, ws)
		}
		s.mu.Unlock()
		for _, ws := range sessions {
			ws.files.Lock()
			_ = ws.persist()
			ws.files.Unlock()
		}
	}
}

func entryPath(directory, kind string, entry projectEntry) string {
	file := "project.elab"
	if entry.File != "" && managedID.MatchString(entry.File) {
		file = entry.File + ".elab"
	}
	return filepath.Join(directory, kind, entry.ID, file)
}

func (ws *workspace) migrateLibrary() error {
	marker := filepath.Join(ws.directory, "shared-library-v1.json")
	if _, err := os.Stat(marker); err == nil {
		return nil
	}
	for _, kind := range []string{"projects", "history"} {
		dirs, err := os.ReadDir(filepath.Join(ws.directory, kind))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		for _, dir := range dirs {
			if !dir.IsDir() || !managedID.MatchString(dir.Name()) {
				continue
			}
			raw, err := readBounded(filepath.Join(ws.directory, kind, dir.Name(), "entry.json"), 8192)
			if err != nil {
				return err
			}
			var entry projectEntry
			if json.Unmarshal(raw, &entry) != nil || entry.ID != dir.Name() || (entry.File != "" && !managedID.MatchString(entry.File)) {
				return fmt.Errorf("invalid legacy project metadata; original preserved")
			}
			target := filepath.Join(ws.library, kind, entry.ID, "entry.json")
			if _, err := os.Stat(target); err == nil {
				continue
			}
			data, err := readBounded(entryPath(ws.directory, kind, entry), workbench.MaxProjectBytes)
			if err != nil {
				return err
			}
			if err = atomicWrite(entryPath(ws.library, kind, entry), data); err != nil {
				return err
			}
			if err = atomicWrite(target, raw); err != nil {
				return err
			}
		}
	}
	return writeJSON(marker, map[string]int{"version": 1})
}

// Other browser workspaces remain discoverable after a lost/cleared cookie.
// Restoring copies bytes and never alters the source checkpoint.
func readCheckpoint(path string) (workspaceCheckpoint, error) {
	var checkpoint workspaceCheckpoint
	data, err := readBounded(path, workbench.MaxProjectBytes+65536)
	if err != nil {
		return checkpoint, err
	}
	if err = json.Unmarshal(data, &checkpoint); err != nil {
		return checkpoint, err
	}
	if checkpoint.Version != 1 || len(checkpoint.Document) == 0 {
		return checkpoint, fmt.Errorf("invalid checkpoint")
	}
	return checkpoint, nil
}
func (ws *workspace) otherCheckpoints() ([]projectEntry, error) {
	entries := []projectEntry{}
	dirs, err := os.ReadDir(filepath.Join(ws.library, "workspaces"))
	if os.IsNotExist(err) {
		return entries, nil
	}
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		if !dir.IsDir() || !managedID.MatchString(dir.Name()) || dir.Name() == filepath.Base(ws.directory) {
			continue
		}
		checkpoint, err := readCheckpoint(filepath.Join(ws.library, "workspaces", dir.Name(), "workspace.json"))
		if err != nil || !checkpoint.Dirty {
			continue
		}
		sum := sha256.Sum256(checkpoint.Document)
		name := checkpoint.Workspace.Name
		if name == "" {
			name = "Untitled circuit"
		}
		entries = append(entries, projectEntry{Checkpoint: true, ID: dir.Name(), Name: name + " · autosaved workspace", UpdatedAt: checkpoint.Workspace.AutosavedAt, SHA256: hex.EncodeToString(sum[:])})
	}
	return entries, nil
}
