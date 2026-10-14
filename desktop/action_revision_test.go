package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRenderedApplyRefusesInterveningAgentValueEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "3k"})
	var captured ui.Node
	s.app.Do(func() {
		var ok bool
		captured, ok = findNode(s.view(), "apply-value")
		if !ok {
			t.Fatal("apply control missing")
		}
	})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	s.app.Do(func() { captured.Activate() })
	after := s.service.Snapshot()
	if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
		t.Fatal("stale rendered Apply overwrote agent work")
	}
	// A fresh draft/action can still apply the intended value normally.
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "3k"})
	uiAction(t, s, "apply-value", nil)
	state := s.service.Snapshot()
	if state.Circuit.Resistance != 3000 || state.UndoEntry.Actor != "user" {
		t.Fatal("fresh Apply lost normal shared command behavior")
	}
}

func TestRenderedDeleteRefusesInterveningAgentSelection(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	var captured ui.Node
	s.app.Do(func() {
		var ok bool
		captured, ok = findNode(s.view(), "delete-object")
		if !ok {
			t.Fatal("delete control missing")
		}
	})
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "D1"})
	before := s.service.Snapshot()
	s.app.Do(func() { captured.Activate() })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("stale rendered Delete acted on an obsolete selection")
	}
}
