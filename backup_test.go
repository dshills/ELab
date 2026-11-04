package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

func backupOf(t *testing.T, f *webFixture) []byte {
	t.Helper()
	r := f.request("POST", "/api/library/backup?"+f.query(""), nil, "")
	if r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	if r.Header().Get("X-ELab-Backup-SHA256") != digest(r.Body.Bytes()) {
		t.Fatal("archive checksum missing")
	}
	return append([]byte(nil), r.Body.Bytes()...)
}
func restoreBackup(f *webFixture, data []byte) *httpResult {
	r := f.request("POST", "/api/library/restore?"+f.query(""), data, "")
	return &httpResult{r.Code, r.Body.Bytes()}
}
func libraryPayload(t *testing.T, root string) []backupPayload {
	t.Helper()
	data, err := collectLibrary(root)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestLibraryBackupRestorePreservesNativeBytesTrashHistoryAndUnsavedWork(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("simulation.verify", map[string]any{"duration_s": .02, "sample_step_s": .001, "probe_ids": []string{"v:C1"}})
	deadline := time.Now().Add(10 * time.Second)
	for {
		state := f.state()
		if len(state.Operations) > 0 && state.Operations[0].Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("capture did not finish")
		}
		time.Sleep(5 * time.Millisecond)
	}
	state := f.state()
	f.command("operation.select", workbench.IDParams{ID: state.LastOperation})
	f.command("recording.keep.set", map[string]any{"id": state.LastOperation, "keep": true})
	if r := f.project("save", "", "Kept recording", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	saved, _, _ := f.projects()
	originalID := saved[0].ID
	if r := f.project("save", "", "Trash copy", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	_, _, info := f.projects()
	if r := f.project("trash", info.ProjectID, "", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "22k"})
	// Neither control credentials nor arbitrary scratch files belong to a backup.
	ws := f.server.sessions[f.cookie.Value]
	for _, path := range []string{filepath.Join(ws.directory, workspaceAgentFile), filepath.Join(f.server.root, ".host.json"), filepath.Join(ws.directory, "scratch.elab")} {
		if err := atomicWrite(path, []byte("secret-agent-credential")); err != nil {
			t.Fatal(err)
		}
	}
	archive := backupOf(t, f)
	payloads, err := decodeLibraryBackup(archive)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	hasKept := false
	for _, p := range payloads {
		counts[p.record.Kind]++
		if p.record.Kind == "workspaces" && p.record.SourceID == f.cookie.Value {
			t.Fatal("workspace cookie exported as a source identity")
		}
		if bytes.Contains(p.data, []byte("secret-agent-credential")) {
			t.Fatal("credential leaked")
		}
		hasKept = hasKept || bytes.Contains(p.data, []byte(`"recordings":`))
	}
	if counts["projects"] != 2 || counts["history"] < 2 || counts["workspaces"] != 1 || !hasKept {
		t.Fatal("incomplete backup", counts)
	}
	zr, _ := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	for _, file := range zr.File {
		if file.Name != "manifest.json" && !strings.HasPrefix(file.Name, "records/") {
			t.Fatal("unexpected archived file", file.Name)
		}
	}
	target := fixture(t)
	target.command("component.set", workbench.ValueParams{ID: "C1", Value: "4.7uF"})
	if r := target.project("save", "", "Existing project", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	before := target.state()
	existing, _, _ := target.projects()
	existingPath := entryPath(target.server.root, "projects", existing[0])
	existingData, _ := os.ReadFile(existingPath)
	preview := target.request("POST", "/api/library/restore?"+target.query("")+"&preview=true", archive, "")
	if preview.Code != 200 {
		t.Fatal(preview.Body.String())
	}
	unchanged, _, _ := target.projects()
	if len(unchanged) != 1 {
		t.Fatal("preview published copies")
	}
	if r := restoreBackup(target, archive); r.code != 200 {
		t.Fatal(string(r.body))
	}
	after := target.state()
	if after.CommandRevision != before.CommandRevision || after.Document.Generation != before.Document.Generation {
		t.Fatal("restore replaced open workspace")
	}
	if data, _ := os.ReadFile(existingPath); !bytes.Equal(data, existingData) {
		t.Fatal("existing project changed")
	}
	imported, history, _ := target.projects()
	if len(imported) != 3 {
		t.Fatal("saved projects missing", imported)
	}
	var restoredID string
	trash := 0
	for _, e := range imported {
		if e.ID == originalID {
			t.Fatal("source project identity reused")
		}
		if e.Trashed {
			trash++
		}
		if e.Name == "Kept recording" {
			restoredID = e.ID
		}
	}
	if trash != 1 || restoredID == "" {
		t.Fatal("trash or project metadata lost")
	}
	checkpoints := 0
	for _, e := range history {
		if e.Checkpoint {
			checkpoints++
		}
	}
	if checkpoints != 1 {
		t.Fatal("restored unsaved work not discoverable", history)
	}
	if r := target.project("open", restoredID, "", true); r.code != 200 {
		t.Fatal(string(r.body))
	}
	restored := target.state()
	if len(restored.Operations) != 1 || restored.Operations[0].Status != "completed" {
		t.Fatal("retained recording did not reopen", restored.Operations)
	}
	// Compare exact payload bytes, not a reserialization that might lose results/models.
	all := libraryPayload(t, target.server.root)
	for _, source := range payloads {
		found := false
		for _, copy := range all {
			if copy.record.Kind == source.record.Kind && bytes.Equal(copy.data, source.data) {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("native document bytes lost", source.record.Kind, source.record.SHA256)
		}
	}
	if err := filepath.WalkDir(target.server.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == workspaceAgentFile {
			return fmt.Errorf("agent credential restored at %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Restored entries remain ordinary editable projects, including overwrite/trash.
	target.command("component.set", workbench.ValueParams{ID: "R1", Value: "33k"})
	if r := target.project("save", restoredID, "Kept recording", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	if r := target.project("trash", restoredID, "", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	if r := target.project("untrash", restoredID, "", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	// A second backup includes imported batches and never their authorization files.
	if _, err := decodeLibraryBackup(backupOf(t, target)); err != nil {
		t.Fatal(err)
	}
}
func rewrittenBackup(t *testing.T, data []byte, change func(*backupManifest, map[string][]byte)) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		raw, e := readZipFile(f, maxBackupExpanded)
		if e != nil {
			t.Fatal(e)
		}
		files[f.Name] = raw
	}
	var manifest backupManifest
	if err = json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	change(&manifest, files)
	files["manifest.json"], _ = json.Marshal(manifest)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for name, raw := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write(raw)
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func TestLibraryBackupRefusesMalformedArchivesWithoutPublication(t *testing.T) {
	source := fixture(t)
	good := backupOf(t, source)
	cases := map[string][]byte{"not-zip": []byte(`{}`)}
	mutations := map[string]func(*backupManifest, map[string][]byte){
		"version":      func(m *backupManifest, _ map[string][]byte) { m.Version = 99 },
		"traversal":    func(_ *backupManifest, f map[string][]byte) { f["../escape"] = []byte("bad") },
		"unknown-file": func(_ *backupManifest, f map[string][]byte) { f["agent.endpoint.json"] = []byte("bad") },
		"checksum": func(m *backupManifest, f map[string][]byte) {
			p := m.Records[0]
			f[p.File] = bytes.Repeat([]byte("x"), int(p.Size))
		},
		"wrong-size": func(m *backupManifest, _ map[string][]byte) { m.Records[0].Size++ },
		"bad-document": func(m *backupManifest, f map[string][]byte) {
			p := &m.Records[0]
			f[p.File] = []byte(`{}`)
			p.Size = 2
			p.SHA256 = digest(f[p.File])
		},
		"path-reference": func(m *backupManifest, _ map[string][]byte) { m.Records[0].File = "../../outside.elab" },
		"duplicate-identity": func(m *backupManifest, f map[string][]byte) {
			p := m.Records[0]
			p.File = "records/0001.elab"
			m.Records = append(m.Records, p)
			f[p.File] = f["records/0000.elab"]
		},
		"too-many": func(m *backupManifest, _ map[string][]byte) {
			for len(m.Records) <= maxBackupRecords {
				m.Records = append(m.Records, m.Records[0])
			}
		},
	}
	for name, change := range mutations {
		cases[name] = rewrittenBackup(t, good, change)
	}
	// Reject symlinks and oversized raw ZIP declarations before decompression.
	for name, mode := range map[string]os.FileMode{"symlink": os.ModeSymlink | 0777, "directory": os.ModeDir | 0700} {
		var out bytes.Buffer
		z := zip.NewWriter(&out)
		h := &zip.FileHeader{Name: "manifest.json"}
		h.SetMode(mode)
		w, _ := z.CreateHeader(h)
		_, _ = w.Write([]byte("bad"))
		_ = z.Close()
		cases[name] = out.Bytes()
	}
	{
		var out bytes.Buffer
		z := zip.NewWriter(&out)
		w, _ := z.CreateRaw(&zip.FileHeader{Name: "manifest.json", Method: zip.Store, UncompressedSize64: maxBackupExpanded + 1})
		_, _ = w.Write([]byte("x"))
		_ = z.Close()
		cases["zip-bomb-header"] = out.Bytes()
	}
	target := fixture(t)
	before := target.state()
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			r := restoreBackup(target, data)
			if r.code != 400 {
				t.Fatal(r.code, string(r.body))
			}
			if dirs, _ := os.ReadDir(filepath.Join(target.server.root, "restores")); len(dirs) != 0 {
				t.Fatal("invalid restore published data")
			}
			after := target.state()
			if before.CommandRevision != after.CommandRevision {
				t.Fatal("invalid archive changed document")
			}
		})
	}
}
func TestLibraryBackupGuardsFailureAtomicityAndOtherWorkspaceCheckpoint(t *testing.T) {
	f := fixture(t)
	other := &webFixture{server: f.server, handler: f.handler, t: t}
	other.state()
	other.command("component.set", workbench.ValueParams{ID: "R1", Value: "44k"})
	archive := backupOf(t, f)
	payloads, err := decodeLibraryBackup(archive)
	if err != nil {
		t.Fatal(err)
	}
	if len(payloads) != 2 {
		t.Fatal("other workspace omitted", len(payloads))
	}
	ws := f.server.sessions[f.cookie.Value]
	checkpoint := filepath.Join(ws.directory, "workspace.json")
	old, _ := os.ReadFile(checkpoint)
	if err = os.Remove(checkpoint); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(checkpoint, 0700); err != nil {
		t.Fatal(err)
	}
	ws.persisted = ""
	r := f.request("POST", "/api/library/backup?"+f.query(""), nil, "")
	if r.Code != 409 || !strings.Contains(r.Body.String(), "checkpoint") {
		t.Fatal("failed autosave produced a backup", r.Body.String())
	}
	_ = os.Remove(checkpoint)
	_ = atomicWrite(checkpoint, old)
	stale := f.query("")
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "33k"})
	r = f.request("POST", "/api/library/restore?"+stale, archive, "")
	if r.Code != 409 {
		t.Fatal("stale restore admitted", r.Body.String())
	}
	if err = atomicWrite(filepath.Join(f.server.root, "restores"), []byte("owned unrelated file")); err != nil {
		t.Fatal(err)
	}
	r = f.request("POST", "/api/library/restore?"+f.query(""), archive, "")
	if r.Code != 500 {
		t.Fatal("failed publication reported success", r.Body.String())
	}
	leftovers, _ := filepath.Glob(filepath.Join(f.server.root, ".restore-*"))
	if len(leftovers) != 0 {
		t.Fatal("unpublished staging not cleaned up", leftovers)
	}
	data, _ := os.ReadFile(filepath.Join(f.server.root, "restores"))
	if string(data) != "owned unrelated file" {
		t.Fatal("unrelated file replaced")
	}
}
func TestRestoredHistorySurvivesAutomaticRetentionAndRestart(t *testing.T) {
	f := fixture(t)
	if r := f.project("save", "", "Before", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	data := backupOf(t, f)
	if r := restoreBackup(f, data); r.code != 200 {
		t.Fatal(string(r.body))
	}
	ws := f.server.sessions[f.cookie.Value]
	entries, err := ws.entries("history")
	if err != nil {
		t.Fatal(err)
	}
	var retained projectEntry
	for _, e := range entries {
		if e.Retained {
			retained = e
		}
	}
	if retained.ID == "" {
		t.Fatal("restored history not marked retained")
	}
	capture, err := ws.service.CaptureProject()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 52; i++ {
		if err = ws.archiveData(capture.Data, fmt.Sprintf("automatic-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ = ws.entries("history")
	found := false
	for _, e := range entries {
		found = found || e.ID == retained.ID
	}
	if !found || len(entries) != 51 {
		t.Fatal("restored history was rotated", len(entries))
	}
	// A new host indexes the atomically published batch without a separate cache.
	fresh := newWebServer(f.server.root)
	defer fresh.close()
	next := &webFixture{server: fresh, handler: fresh.handler(), t: t}
	next.state()
	projects, history, _ := next.projects()
	if len(projects) != 2 || len(history) < 51 {
		t.Fatal("restored batch missing after host restart", projects, len(history))
	}
	if !reflect.DeepEqual(capture.Data, mustRead(t, entryPath(f.server.root, "history", retained))) {
		t.Fatal("retained bytes changed")
	}
}
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestLibraryRestoreRejectsCrossOrigin(t *testing.T) {
	f := fixture(t)
	r := f.request(http.MethodPost, "/api/library/restore?"+f.query(""), backupOf(t, f), "https://evil.example")
	if r.Code != 403 {
		t.Fatal("cross-origin restore allowed", r.Code)
	}
}

func TestLibraryBackupIncludesDormantLegacyStoresAndOlderVersions(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	if r := f.project("save", "", "Shared version", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	projects, _, _ := f.projects()
	entry := projects[0]
	oldData := mustRead(t, entryPath(f.server.root, "projects", entry))
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "22k"})
	if r := f.project("save", entry.ID, "Shared version", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	// Simulate an inactive pre-shared-library workspace with an older saved copy.
	legacy := filepath.Join(f.server.root, "workspaces", token())
	current := f.server.sessions[f.cookie.Value]
	if err := atomicWrite(filepath.Join(legacy, "workspace.json"), mustRead(t, filepath.Join(current.directory, "workspace.json"))); err != nil {
		t.Fatal(err)
	}
	entry.File = ""
	entry.SHA256 = digest(oldData)
	if err := atomicWrite(filepath.Join(legacy, "projects", entry.ID, "project.elab"), oldData); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(legacy, "projects", entry.ID, "entry.json"), entry); err != nil {
		t.Fatal(err)
	}
	payloads, err := decodeLibraryBackup(backupOf(t, f))
	if err != nil {
		t.Fatal(err)
	}
	saved := 0
	foundOld := false
	for _, p := range payloads {
		if p.record.Kind == "projects" {
			saved++
			foundOld = foundOld || bytes.Equal(p.data, oldData)
		}
	}
	if saved != 2 || !foundOld {
		t.Fatal("dormant legacy version omitted", saved)
	}
	if !bytes.Equal(oldData, mustRead(t, filepath.Join(legacy, "projects", entry.ID, "project.elab"))) {
		t.Fatal("backup migrated or modified legacy source")
	}
}

func TestLibraryTransferAdmissionBoundsConcurrentMemory(t *testing.T) {
	f := fixture(t)
	archive := backupOf(t, f)
	f.server.libraryIO <- struct{}{}
	for _, path := range []string{"backup", "restore"} {
		r := f.request("POST", "/api/library/"+path+"?"+f.query(""), archive, "")
		if r.Code != 409 || !strings.Contains(r.Body.String(), "in progress") {
			t.Fatal("overlapping transfer was admitted", r.Body.String())
		}
	}
	<-f.server.libraryIO
	if r := restoreBackup(f, archive); r.code != 200 {
		t.Fatal("transfer slot did not recover", string(r.body))
	}
}

func TestLibraryBackupRestoresKeptACResultsWithoutRerunning(t *testing.T) {
	f := fixture(t)
	f.command("example.load", workbench.ExampleLoadParams{ID: "ac-rc", DiscardChanges: true})
	f.command("simulation.pause", struct{}{})
	f.wait()
	f.command("sweep.start", workbench.SweepStartParams{Settings: &workbench.SweepSettings{ComponentID: "V1", Parameter: "frequency", Start: "10 Hz", Stop: "10 kHz", Points: 9, Spacing: "logarithmic", ProbeIDs: []string{"v:C1"}}})
	deadline := time.Now().Add(10 * time.Second)
	for f.state().DCSweep.Result == nil {
		if time.Now().After(deadline) {
			t.Fatal("AC sweep did not complete")
		}
		time.Sleep(5 * time.Millisecond)
	}
	sweep := f.state().DCSweep
	f.command("sweep.keep.set", workbench.RecordingKeepParams{ID: sweep.Selected, Keep: true})
	if r := f.project("save", "", "Kept AC", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	archive := backupOf(t, f)
	target := fixture(t)
	if r := restoreBackup(target, archive); r.code != 200 {
		t.Fatal(string(r.body))
	}
	projects, _, _ := target.projects()
	if len(projects) != 1 {
		t.Fatal(projects)
	}
	if r := target.project("open", projects[0].ID, "", true); r.code != 200 {
		t.Fatal(string(r.body))
	}
	restored := target.state().DCSweep
	if restored == nil || restored.Result == nil || !reflect.DeepEqual(restored.Result.Points, sweep.Result.Points) || !reflect.DeepEqual(restored.Result.AC, sweep.Result.AC) {
		t.Fatal("kept AC result changed during backup/restore")
	}
}
func TestLibraryBackupStopsOnCorruptPublishedProject(t *testing.T) {
	f := fixture(t)
	if r := f.project("save", "", "Corrupt fixture", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	projects, _, _ := f.projects()
	path := entryPath(f.server.root, "projects", projects[0])
	changed := append(mustRead(t, path), ' ')
	if err := atomicWrite(path, changed); err != nil {
		t.Fatal(err)
	}
	r := f.request("POST", "/api/library/backup?"+f.query(""), nil, "")
	if r.Code != 409 || r.Header().Get("Content-Type") == "application/zip" {
		t.Fatal("corrupt library produced a partial backup", r.Body.String())
	}
	if !bytes.Equal(changed, mustRead(t, path)) {
		t.Fatal("failed backup modified corrupt source")
	}
}
