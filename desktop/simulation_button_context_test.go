package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRenderedSimulationButtonRefusesAgentDocumentReplacement(t *testing.T) {
	for _, id := range []string{"simulation.pause", "simulation.resume", "simulation.restart"} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "example.rc", nil)
			if id != "simulation.pause" {
				uiAction(t, s, "simulation.pause", nil)
			}
			var captured ui.Node
			s.app.Do(func() {
				var ok bool
				captured, ok = findNode(s.view(), id)
				if !ok || captured.Activate == nil {
					t.Fatal("simulation control missing")
				}
			})
			// The agent replaces the document and advances its paused circuit.
			// The retained old button must not restart/resume this new experiment.
			agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "simulation.restart", struct{}{})
			agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
			before := s.service.Snapshot()
			focus := s.keyboardFocus
			s.app.Do(func() { captured.Activate() })
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.keyboardFocus != focus {
				t.Fatal("old rendered simulation control acted on the replacement document")
			}
			// A newly rendered control must still be usable with the new context.
			fresh := id
			if id == "simulation.pause" {
				fresh = "simulation.resume"
			}
			uiAction(t, s, fresh, nil)
			if s.errorMessage != "" {
				t.Fatalf("fresh control refused: %s", s.errorMessage)
			}
		})
	}
}

func TestRenderedSimulationButtonChecksDocumentWithCurrentCommandRevision(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	generation := s.service.Snapshot().Document.Generation
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
	before := s.service.Snapshot()
	s.app.Do(func() {
		// Isolate publication-only document replacement from command conflicts.
		previous := s.actionContext
		s.actionContext = &actionContext{before.CommandRevision, generation}
		captured := s.button("simulation.restart", "Restart")
		s.actionContext = previous
		captured.Activate()
	})
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("matching command revision bypassed the rendered document guard")
	}
}

func TestRenderedPauseAcceptsLiveSamplesWithoutInterveningCommands(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	var captured ui.Node
	s.app.Do(func() { captured, _ = findNode(s.view(), "simulation.pause") })
	before := s.service.Snapshot()
	deadline := time.Now().Add(3 * time.Second)
	for s.service.Snapshot().Transient.Sample.Time <= before.Transient.Sample.Time {
		if time.Now().After(deadline) {
			t.Fatal("real RC clock did not advance")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if s.service.Snapshot().CommandRevision != before.CommandRevision {
		t.Fatal("fixture changed a command instead of publishing live samples")
	}
	s.app.Do(func() { captured.Activate() })
	state := s.service.Snapshot()
	if s.errorMessage != "" || !state.Paused || state.Transient.Stale || state.Transient.Sample.Time <= before.Transient.Sample.Time {
		t.Fatal("live solver publication incorrectly invalidated rendered Pause")
	}
}
