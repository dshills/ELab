package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeCurrentExampleResetIsProtectedAndShared(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	original := s.service.Snapshot()
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "20k"})
	uiAction(t, s, "apply-value", nil)
	uiAction(t, s, "example.current.reset", nil)
	if s.pendingCommand != "example.current.reset" || s.service.Snapshot().Example.Info.ID != "rc" || !s.service.Snapshot().Example.Modified {
		t.Fatal("native reset bypasses unsaved protection or selects LED")
	}
	uiAction(t, s, "cancel-file-action", nil)
	if !s.service.Snapshot().Example.Modified {
		t.Fatal("cancelled reset changes edited circuit")
	}
	uiAction(t, s, "example.current.reset", nil)
	uiAction(t, s, "discard-continue", nil)
	state := s.service.Snapshot()
	if state.Example.Info.ID != "rc" || state.Example.Modified || !state.Paused || !reflect.DeepEqual(state.Components, original.Components) || state.Transient.Sample.Time != 0 {
		t.Fatal("native reset differs from protected current-template command")
	}
	s.app.Do(func() {
		root := s.view()
		for _, id := range []string{"example-question", "example-expected", "example-try", "example-limitation"} {
			node, ok := findNode(root, id)
			if !ok || node.Text == "" {
				t.Fatal("missing guided example field", id)
			}
		}
	})
}

func TestNativeStaleExampleResetButtonRefusesReplacement(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	var captured ui.Node
	s.app.Do(func() {
		node, ok := findNode(s.view(), "example.current.reset")
		if !ok {
			t.Fatal("reset missing")
		}
		captured = node
	})
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "npn", DiscardChanges: true})
	before := s.service.Snapshot()
	s.app.Do(func() { captured.Activate() })
	if s.errorMessage == "" || s.pendingCommand != "" || s.service.Snapshot().Example.Info.ID != "npn" || !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) {
		t.Fatal("old reset button changes a replacement example")
	}
}
