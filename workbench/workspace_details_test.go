package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWorkspaceDetailsPreserveExperimentAndFileBoundary(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	for _, section := range workspaceDetailSections() {
		call(t, s, "workspace.details.set", WorkspaceDetailsParams{section, true}, "show-"+section)
		after := s.Snapshot()
		if !after.WorkspaceDetails.Expanded(section) {
			t.Fatal("details not shared", section)
		}
		if after.Document != before.Document || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Scope, before.Scope) || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) {
			t.Fatal("disclosure changed experiment", section)
		}
	}
	p := filepath.Join(t.TempDir(), "details.elab")
	call(t, s, "project.save", SaveParams{Path: p}, "save")
	data, _ := os.ReadFile(p)
	if strings.Contains(string(data), "workspace_details") {
		t.Fatal("session choice persisted")
	}
	call(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "replace")
	if !s.Snapshot().WorkspaceDetails.Component {
		t.Fatal("document replacement lost session choice")
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: p}, "load")
	if fresh.Snapshot().WorkspaceDetails != (WorkspaceDetails{}) {
		t.Fatal("details leaked through file")
	}
}

func TestWorkspaceDetailsRefuseAtomicAndRespectGuards(t *testing.T) {
	s := New()
	defer s.Close()
	for _, p := range []any{map[string]any{}, map[string]any{"section": "component"}, map[string]any{"section": nil, "visible": true}, map[string]any{"section": "component", "visible": nil}, map[string]any{"section": "component", "visible": true, "extra": 1}, WorkspaceDetailsParams{"unknown", true}} {
		before := s.Snapshot()
		refuse(t, s, "workspace.details.set", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid request mutated state")
		}
	}
	before := s.Snapshot()
	raw, _ := json.Marshal(WorkspaceDetailsParams{"component", true})
	r := Request{Version: 1, ID: "details", Key: "details", Command: "workspace.details.set", Params: raw, ExpectedCommandRevision: &before.CommandRevision, ExpectedDocumentGeneration: &before.Document.Generation}
	if got := s.Dispatch(context.Background(), r, "agent"); got.Error != nil {
		t.Fatal(got.Error)
	}
	call(t, s, "workspace.details.set", WorkspaceDetailsParams{"component", false}, "close")
	current := s.Snapshot()
	got := s.Dispatch(context.Background(), r, "agent")
	if got.Error != nil || !got.Duplicate || !reflect.DeepEqual(current, s.Snapshot()) {
		t.Fatal("duplicate reapplied visibility")
	}
	r.Key = "stale"
	if got = s.Dispatch(context.Background(), r, "agent"); got.Error == nil || got.Error.Code != "conflict" {
		t.Fatal("stale details accepted")
	}
}
