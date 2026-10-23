package workbench

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestInspectorNavigationPreservesWorkAndRemainsSessionOnly(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := settled(t, s)
	if before.InspectorSection != "component" {
		t.Fatal("unexpected initial inspector")
	}
	for _, section := range []string{"simulation", "document", "component"} {
		call(t, s, "workspace.inspector.set", InspectorParams{Section: section}, section)
		after := s.Snapshot()
		if after.InspectorSection != section || after.Document != before.Document || after.ElectricalRevision != before.ElectricalRevision || after.RunID != before.RunID || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Scope, before.Scope) || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Wires, before.Wires) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) {
			t.Fatal("inspector navigation changed work")
		}
	}
	for _, bad := range []map[string]any{{}, {"section": nil}, {"section": "missing"}, {"section": "document", "extra": true}} {
		before := s.Snapshot()
		refuse(t, s, "workspace.inspector.set", bad, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("rejected navigation changed state")
		}
	}
	call(t, s, "workspace.inspector.set", InspectorParams{Section: "document"}, "document-again")
	path := filepath.Join(t.TempDir(), "project.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	call(t, s, "project.load", LoadParams{Path: path}, "reload")
	if s.Snapshot().InspectorSection != "document" {
		t.Fatal("document replacement lost inspector selection")
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "fresh")
	if fresh.Snapshot().InspectorSection != "component" {
		t.Fatal("inspector presentation leaked into project")
	}
}
