package main

import (
	"context"
	"testing"

	"github.com/dshills/vellum/ui"
)

func TestSimulationButtonFocusContinuesAcrossReplacement(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	s.app.Do(func() { s.requestKeyboardFocus("simulation.pause") })
	sequence := s.keyboardFocus.Sequence
	for _, step := range []struct {
		command, target string
		paused          bool
	}{{"simulation.pause", "simulation.resume", true}, {"simulation.resume", "simulation.pause", false}} {
		uiAction(t, s, step.command, nil)
		s.app.Do(func() {
			root := s.view()
			if s.service.Snapshot().Paused != step.paused || root.Focus.ID != step.target || root.Focus.Sequence <= sequence {
				t.Fatalf("%s lost focus when its button was replaced: %+v", step.command, root.Focus)
			}
			if _, ok := ui.FocusTarget(root, step.target); !ok {
				t.Fatal("focus continuation targets an unavailable control")
			}
			if fresh := s.view(); fresh.Focus != root.Focus {
				t.Fatal("redraw reissues focus and can steal it from the next control")
			}
			sequence = root.Focus.Sequence
		})
	}
	// An unavailable command must not move focus or toggle the physical clock.
	s.app.Do(func() {
		before := s.keyboardFocus
		s.fileBusy = true
		if err := s.commands.Invoke(context.Background(), "simulation.pause"); err == nil || s.keyboardFocus != before || s.service.Snapshot().Paused {
			t.Fatal("refused toggle changed focus or clock")
		}
		s.fileBusy = false
	})
}
