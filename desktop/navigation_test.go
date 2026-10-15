package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestObjectNavigationUsesSharedCommands(t *testing.T) {
	s, close := testScreen(t)
	defer close()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
	uiAction(t, s, "selection-next", nil)
	first := s.service.Snapshot()
	if first.Selected != first.Components[0].ID || first.View.Center != first.Components[0].Position {
		t.Fatal("next control", first.Selected, first.View)
	}
	uiAction(t, s, "selection-previous", nil)
	last := s.service.Snapshot()
	if last.Selected != last.Wires[len(last.Wires)-1].ID {
		t.Fatal("previous control", last.Selected)
	}
	agentAction(t, s.service, "view.pan", map[string]float64{"dx": 100, "dy": 100})
	uiAction(t, s, "selection-center", nil)
	if s.service.Snapshot().View != last.View {
		t.Fatal("center control")
	}
	captured := s.service.Snapshot()
	s.app.Do(func() {
		s.drawing.navigateObject(navigationDirection("alt+home"), captured.CommandRevision, captured.Document.Generation)
	})
	if s.service.Snapshot().Selected != first.Selected {
		t.Fatal("first shortcut")
	}
	before := s.service.Snapshot()
	s.app.Do(func() { s.drawing.navigateObject("last", captured.CommandRevision, captured.Document.Generation) })
	if s.service.Snapshot().CommandRevision != before.CommandRevision {
		t.Fatal("stale shortcut admitted")
	}
	s.app.Do(func() {
		s.errorMessage = ""
		s.fileBusy = true
		s.drawing.navigateObject("last", before.CommandRevision, before.Document.Generation)
		s.fileBusy = false
	})
	if s.service.Snapshot().CommandRevision != before.CommandRevision {
		t.Fatal("busy shortcut admitted")
	}
}

func TestNavigationChordsDoNotConsumeTextArrows(t *testing.T) {
	for chord, want := range map[string]string{"alt+leftarrow": "previous", "alt+rightarrow": "next", "alt+home": "first", "alt+end": "last", "leftarrow": "", "rightarrow": "", "home": "", "end": ""} {
		if got := navigationDirection(chord); got != want {
			t.Fatal(chord, got, want)
		}
	}
}
