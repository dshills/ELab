package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestCompactInspectorPreservesControlsAndSharedDetails(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "C1"})
	before := s.service.Snapshot()
	root := s.view()
	for _, id := range []string{"value-entry", "apply-value", "rotate-part", "delete-object"} {
		if _, ok := ui.FocusTarget(root, id); !ok {
			t.Fatal("primary action hidden", id)
		}
	}
	if _, ok := ui.FocusTarget(root, "cap-initial"); ok {
		t.Fatal("advanced startup mounted initially")
	}
	uiAction(t, s, "workspace-details-component", &ui.ChangeEvent{Checked: true})
	after := s.service.Snapshot()
	if !after.WorkspaceDetails.Component || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("details not shared or changed circuit")
	}
	for _, id := range []string{"cap-initial", "cap-initial-default", "duplicate-part"} {
		if _, ok := findNode(s.view(), id); !ok {
			t.Fatal("existing detail lost", id)
		}
	}
	held, _ := findAction(s.view(), "workspace-details-component")
	agentAction(t, s.service, "workspace.details.set", workbench.WorkspaceDetailsParams{Section: "selection", Visible: true})
	current := s.service.Snapshot()
	s.app.Do(func() { held.Change(ui.ChangeEvent{Checked: false}) })
	if !reflect.DeepEqual(current, s.service.Snapshot()) {
		t.Fatal("held disclosure adopted newer agent context")
	}
}
