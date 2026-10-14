package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeTraceSourceFocusMatchesAgentAndChangesMesh(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.flasher", nil)
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "scope.probes.set", workbench.ProbeParams{IDs: []string{"i:D1", "v:C1"}})
	before := s.service.Snapshot()
	mesh, err := s.mesh(*s.drawing.View)
	if err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "scope-trace-focus", &ui.ChangeEvent{SelectedID: "i:D1"})
	focused := s.service.Snapshot()
	changed, err := s.mesh(*s.drawing.View)
	if err != nil {
		t.Fatal(err)
	}
	if focused.TraceFocus == nil || focused.TraceFocus.Objects[0] != "D1" || focused.RunID != before.RunID || !reflect.DeepEqual(focused.Transient, before.Transient) || reflect.DeepEqual(mesh.Vertices(), changed.Vertices()) {
		t.Fatal("native focus missing or changes physics")
	}
	agentAction(t, s.service, "scope.trace.select", workbench.IDParams{})
	if s.service.Snapshot().TraceFocus != nil {
		t.Fatal("shared clear failed")
	}
}
