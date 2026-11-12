package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestSharedReplayEmbedsCapturedProjectAndEscapesNote(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.wait()
	id := captureForControls(t, f, .02, []string{"v:C1"})
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "22k"})
	f.wait()
	before := f.state()
	path := "/api/schematic-replay/share?id=" + id + "&" + f.query("")
	note := "</script><script>alert(1)</script>"
	params := replayShareParams{Position: replayPosition{Index: 1}, Note: note, IncludeProject: true}
	request := func(p replayShareParams) (int, string) {
		t.Helper()
		body, _ := json.Marshal(p)
		response := f.request("POST", path, body, "")
		if response.Code == http.StatusOK && !strings.HasPrefix(response.Header().Get("Content-Disposition"), "attachment;") {
			t.Fatal(response.Header())
		}
		return response.Code, response.Body.String()
	}
	code, page := request(params)
	if code != 200 || strings.Contains(page, note) || !strings.Contains(page, "&lt;/script&gt;") {
		t.Fatalf("unexpected replay response: %d", code)
	}
	_, script, ok := strings.Cut(page, "<script>")
	if !ok {
		t.Fatal("missing embedded viewer")
	}
	script, _, _ = strings.Cut(script, "</script>")
	hash := sha256.Sum256([]byte(script))
	if !strings.Contains(html.UnescapeString(page), "'sha256-"+base64.StdEncoding.EncodeToString(hash[:])+"'") {
		t.Fatal("script digest did not match exported script")
	}
	_, encoded, ok := strings.Cut(script, "const share = ")
	encoded, _, _ = strings.Cut(encoded, ";\n")
	var payload replaySharePayload
	if !ok || json.Unmarshal([]byte(encoded), &payload) != nil || payload.Position != params.Position || len(payload.Capture.Rows) < 2 {
		t.Fatal("capture or requested starting sample was lost")
	}
	if err := workbench.ValidateProject(payload.Project); err != nil {
		t.Fatal(err)
	}
	var project workbench.Document
	if err := json.Unmarshal(payload.Project, &project); err != nil {
		t.Fatal(err)
	}
	part, _ := project.Graph.Component("R1")
	if part.Value != 10000 || project.Paused == nil || !*project.Paused {
		t.Fatal("editable copy did not use the captured circuit", part.Value)
	}
	params.IncludeProject = false
	if code, html := request(params); code != 200 || strings.Contains(html, "Download editable .elab project") || strings.Contains(html, `"project":`) {
		t.Fatal("editable copy was included without opt-in", code)
	}
	params.Position.Index = -1
	if code, _ := request(params); code != 400 {
		t.Fatal("invalid sample was accepted", code)
	}
	after := f.state()
	if before.CommandRevision != after.CommandRevision || before.Document != after.Document {
		t.Fatal("sharing edited the project")
	}
}
