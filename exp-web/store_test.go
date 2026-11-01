package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func (f *webFixture) project(action, id, name string, discard bool) *httpResult {
	state := f.state()
	q := url.Values{"revision": {fmt.Sprint(state.CommandRevision)}, "generation": {fmt.Sprint(state.Document.Generation)}}
	body, _ := json.Marshal(map[string]any{"action": action, "id": id, "name": name, "discard_changes": discard})
	r := f.request("POST", "/api/projects?"+q.Encode(), body, "")
	return &httpResult{code: r.Code, body: r.Body.Bytes()}
}

type httpResult struct {
	code int
	body []byte
}

func (f *webFixture) projects() ([]projectEntry, []projectEntry, workspaceInfo) {
	f.t.Helper()
	r := f.request("GET", "/api/projects", nil, "")
	var result struct {
		Projects  []projectEntry
		History   []projectEntry
		Workspace workspaceInfo
	}
	if json.Unmarshal(r.Body.Bytes(), &result) != nil {
		f.t.Fatal(r.Body.String())
	}
	return result.Projects, result.History, result.Workspace
}
func TestWorkspacePersistsAcknowledgedEditsWithoutSaving(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "22k"})
	_, _, info := f.projects()
	if info.AutosavedAt == "" {
		t.Fatal("autosave not reported")
	}
	if !f.state().Document.Dirty {
		t.Fatal("autosave marked editor saved")
	}
	// Simulate losing the service without the web server's graceful-close save.
	f.server.mu.Lock()
	old := f.server.sessions[f.cookie.Value]
	delete(f.server.sessions, f.cookie.Value)
	f.server.mu.Unlock()
	old.service.Close()
	next := newWebServer(f.server.root)
	t.Cleanup(next.close)
	g := &webFixture{server: next, handler: next.handler(), cookie: f.cookie, t: t}
	restored := g.state()
	if !restored.Document.Dirty {
		t.Fatal("unsaved recovery was marked clean")
	}
	found := false
	for _, p := range restored.Components {
		if p.ID == "R1" && p.Value == 22000 {
			found = true
		}
	}
	if !found {
		t.Fatal("acknowledged edit was lost")
	}
	r := g.request("GET", "/api/state", nil, "")
	var payload map[string]any
	_ = json.Unmarshal(r.Body.Bytes(), &payload)
	if payload["instance_id"] == f.server.instance {
		t.Fatal("restart identity not changed")
	}
	if len(g.cookie.Value) != 48 || g.cookie.Name != "elab-workspace" {
		t.Fatal("invalid session cookie")
	}
}
func TestNamedProjectsHistoryAndTrash(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	first := f.project("save", "", "First project", false)
	if first.code != 200 {
		t.Fatal(string(first.body))
	}
	saved, _, meta := f.projects()
	if len(saved) != 1 || meta.ProjectID == "" || f.state().Document.Dirty {
		t.Fatal("save did not establish a named clean project")
	}
	id := saved[0].ID
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "33k"})
	if got := f.project("open", id, "", false); got.code != 409 {
		t.Fatal("dirty open was allowed")
	}
	if got := f.project("save", id, "First project", false); got.code != 200 {
		t.Fatal(string(got.body))
	}
	_, history, _ := f.projects()
	if len(history) < 2 {
		t.Fatal("previous save/replaced circuit not recoverable")
	}
	f.command("project.clear", workbench.ReplaceParams{DiscardChanges: true})
	if got := f.project("open", id, "", true); got.code != 200 {
		t.Fatal(string(got.body))
	}
	if len(f.state().Components) != 4 {
		t.Fatal("named project did not reopen")
	}
	for _, action := range []string{"trash", "untrash"} {
		if got := f.project(action, id, "", false); got.code != 200 {
			t.Fatal(string(got.body))
		}
		saved, _, _ = f.projects()
		if saved[0].Trashed != (action == "trash") {
			t.Fatal("trash not reversible")
		}
	}
	if got := f.project("recover", history[0].ID, "", true); got.code != 200 {
		t.Fatal(string(got.body))
	}
	_, _, meta = f.projects()
	if meta.ProjectID != "" || !meta.Recovered || !f.state().Document.Dirty {
		t.Fatal("recovery did not create an unsaved independent copy")
	}
}
func TestProjectBoundariesAndCorruptCheckpoint(t *testing.T) {
	f := fixture(t)
	if got := f.project("open", "../../private", "", true); got.code != 400 {
		t.Fatal("path traversal allowed")
	}
	if got := f.project("save", "", " ", false); got.code != 400 {
		t.Fatal("empty name allowed")
	}
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "12k"})
	ws := f.server.sessions[f.cookie.Value]
	path := filepath.Join(ws.directory, "workspace.json")
	original, _ := os.ReadFile(path)
	if len(original) == 0 {
		t.Fatal("missing checkpoint")
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	f.server.mu.Lock()
	delete(f.server.sessions, f.cookie.Value)
	f.server.mu.Unlock()
	ws.service.Close()
	g := &webFixture{server: f.server, handler: f.handler, cookie: f.cookie, t: t}
	r := g.request("GET", "/api/state", nil, "")
	if r.Code != 503 || !bytes.Contains(r.Body.Bytes(), []byte("preserved")) {
		t.Fatal("corrupt checkpoint was silently replaced", r.Body.String())
	}
	after, _ := os.ReadFile(path)
	if string(after) != "broken" {
		t.Fatal("corrupt recovery evidence destroyed")
	}
}
func TestStoredProjectHashAndGuardProtection(t *testing.T) {
	f := fixture(t)
	if got := f.project("save", "", "Hash test", false); got.code != 200 {
		t.Fatal(string(got.body))
	}
	saved, _, _ := f.projects()
	entry := saved[0]
	ws := f.server.sessions[f.cookie.Value]
	path := entryPath(ws.library, "projects", entry)
	data, _ := os.ReadFile(path)
	data = append(data, ' ')
	_ = os.WriteFile(path, data, 0600)
	if got := f.project("open", entry.ID, "", true); got.code != 409 {
		t.Fatal("modified file accepted")
	}
	old := f.state()
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "15k"})
	body := []byte(`{"action":"save","name":"Stale"}`)
	r := f.request("POST", fmt.Sprintf("/api/projects?revision=%d&generation=%d", old.CommandRevision, old.Document.Generation), body, "")
	if r.Code != http.StatusConflict {
		t.Fatal("stale save allowed")
	}
}
func TestPortableTransfersAndSchemas(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("simulation.verify", map[string]any{"duration_s": .02, "sample_step_s": .001, "probe_ids": []string{"v:C1"}})
	// simulation.wait does not wait verification; use the completed operation helper.
	for i := 0; i < 200; i++ {
		state := f.state()
		if len(state.Operations) > 0 && state.Operations[0].Status == "completed" {
			break
		}
		f.command("simulation.wait", struct{}{})
	}
	state := f.state()
	f.command("operation.select", workbench.IDParams{ID: state.LastOperation})
	f.command("recording.keep.set", map[string]any{"id": state.LastOperation, "keep": true})
	result := f.request("POST", "/api/download?"+f.query("recording-json"), nil, "")
	if result.Code != 200 {
		t.Fatal(result.Body.String())
	}
	imported := f.request("POST", "/api/import-result?kind=recording&"+f.query(""), result.Body.Bytes(), "")
	if imported.Code != 200 || bytes.Contains(imported.Body.Bytes(), []byte(`"error":`)) {
		t.Fatal(imported.Body.String())
	}
	catalog := f.request("GET", "/api/catalog", nil, "")
	if !strings.Contains(catalog.Body.String(), `"schemas"`) || !strings.Contains(catalog.Body.String(), `"model_defaults"`) {
		t.Fatal("native form metadata missing")
	}
	if !permitted("audio.render") || permitted("audio.play") {
		t.Fatal("browser audio would launch native playback")
	}
}

func TestSharedLibraryRejectsConcurrentLostUpdates(t *testing.T) {
	f := fixture(t)
	if r := f.project("save", "", "Shared circuit", false); r.code != 200 {
		t.Fatal(string(r.body))
	}
	saved, _, _ := f.projects()
	id := saved[0].ID
	other := &webFixture{server: f.server, handler: f.handler, t: t}
	_ = other.state()
	listed, _, _ := other.projects()
	if len(listed) != 1 || listed[0].ID != id {
		t.Fatal("new browser cannot find saved projects")
	}
	if r := other.project("open", id, "", true); r.code != 200 {
		t.Fatal(string(r.body))
	}
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "22k"})
	other.command("component.set", workbench.ValueParams{ID: "R1", Value: "33k"})
	results := make(chan *httpResult, 2)
	go func() { results <- f.project("save", id, "Shared circuit", false) }()
	go func() { results <- other.project("save", id, "Shared circuit", false) }()
	a, b := <-results, <-results
	if !((a.code == 200 && b.code == 409) || (a.code == 409 && b.code == 200)) {
		t.Fatalf("both workspaces overwrote the same baseline: %d %d", a.code, b.code)
	}
	saved, history, _ := f.projects()
	if len(history) < 2 {
		t.Fatal("previous save missing from history")
	}
	versions, err := os.ReadDir(filepath.Join(f.server.root, "projects", id))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, v := range versions {
		if strings.HasSuffix(v.Name(), ".elab") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("unbounded orphan revisions: %d", count)
	}
	if r := f.project("open", saved[0].ID, "", true); r.code != 200 {
		t.Fatal(string(r.body))
	}
}

func TestStorageMetadataChangesArePublishedWhilePaused(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	ws := f.server.sessions[f.cookie.Value]
	state := f.state()
	checkpoint := ws.info.AutosavedAt
	ws.files.Lock()
	ws.info.Error = "Autosave failed: test disk unavailable"
	ws.files.Unlock()
	query := url.Values{"after": {fmt.Sprint(state.Revision)}, "instance": {f.server.instance}, "checkpoint": {checkpoint}, "storage_error": {""}}
	r := f.request("GET", "/api/state?"+query.Encode(), nil, "")
	if r.Code != 200 || !bytes.Contains(r.Body.Bytes(), []byte("test disk unavailable")) {
		t.Fatal("paused workspace hid storage failure", r.Code, r.Body.String())
	}
	query.Set("storage_error", ws.info.Error)
	query.Set("instance", "previous-server")
	r = f.request("GET", "/api/state?"+query.Encode(), nil, "")
	if r.Code != 200 {
		t.Fatal("equal revision from another server suppressed recovery")
	}
}

func TestStoreExcludesSecondWriterAndCanReopen(t *testing.T) {
	root := t.TempDir()
	release, err := lockStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if unlock, err := lockStore(root); err == nil {
		unlock()
		t.Fatal("second writer acquired workspace")
	}
	release()
	release, err = lockStore(root)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestResultImportRejectsTrailingPayload(t *testing.T) {
	f := fixture(t)
	result := f.request("POST", "/api/import-result?kind=recording&"+f.query(""), []byte(`{} {}`), "")
	if result.Code != 400 {
		t.Fatal("multiple JSON objects accepted")
	}
}

func TestLostCookieRecoveryCopiesAndGuardsAutosavedWorkspace(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "47k"})
	original := f.server.sessions[f.cookie.Value]
	before, err := os.ReadFile(filepath.Join(original.directory, "workspace.json"))
	if err != nil {
		t.Fatal(err)
	}
	other := &webFixture{server: f.server, handler: f.handler, t: t}
	_ = other.state()
	_, history, _ := other.projects()
	if len(history) != 1 || !history[0].Checkpoint {
		t.Fatal("unsaved workspace hidden after losing cookie", history)
	}
	entry := history[0]
	body, _ := json.Marshal(map[string]any{"action": "recover", "id": entry.ID, "checkpoint": true, "sha256": "old", "discard_changes": true})
	response := other.request("POST", "/api/projects?"+other.query(""), body, "")
	if response.Code != 409 {
		t.Fatal("stale recovery digest accepted")
	}
	body, _ = json.Marshal(map[string]any{"action": "recover", "id": entry.ID, "checkpoint": true, "sha256": entry.SHA256, "discard_changes": true})
	response = other.request("POST", "/api/projects?"+other.query(""), body, "")
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	recovered := other.state()
	found := false
	for _, p := range recovered.Components {
		if p.ID == "R1" && p.Value == 47000 {
			found = true
		}
	}
	if !found || !recovered.Document.Dirty {
		t.Fatal("recovery did not restore unsaved graph")
	}
	after, _ := os.ReadFile(filepath.Join(original.directory, "workspace.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("recovering a copy altered the original workspace")
	}
}

func TestAutosaveFailureKeepsEditorAndPortableCopyAvailable(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	ws := f.server.sessions[f.cookie.Value]
	path := filepath.Join(ws.directory, "workspace.json")
	if err := os.Rename(path, path+".previous"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "56k"})
	response := f.request("GET", "/api/state", nil, "")
	if !bytes.Contains(response.Body.Bytes(), []byte("Autosave failed")) {
		t.Fatal("storage error not visible")
	}
	copy := f.request("POST", "/api/download?"+f.query("project"), nil, "")
	if copy.Code != 200 || !bytes.Contains(copy.Body.Bytes(), []byte("56000")) {
		t.Fatal("download could not rescue edited data")
	}
	if !f.state().Document.Dirty {
		t.Fatal("failed autosave cleared dirty state")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	ws.files.Lock()
	err := ws.persist()
	ws.files.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := readCheckpoint(path)
	if err != nil || !bytes.Contains(checkpoint.Document, []byte("56000")) || checkpoint.Workspace.Error != "" {
		t.Fatal("retry failed to checkpoint the latest editor")
	}
}

func TestExamplePreviewPreservesNamedProjectAndHistory(t *testing.T) {
	f := fixture(t)
	if result := f.project("save", "", "Named original", false); result.code != 200 {
		t.Fatal(string(result.body))
	}
	_, beforeHistory, before := f.projects()
	f.command("example.preview", workbench.IDParams{ID: "discrete-hbridge"})
	_, afterHistory, after := f.projects()
	if before.ProjectID != after.ProjectID || before.Name != after.Name || before.SavedSHA != after.SavedSHA || len(beforeHistory) != len(afterHistory) {
		t.Fatal("read-only example preview changed the project library")
	}
}
