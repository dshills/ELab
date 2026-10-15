package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeNarrowWindowReportsNoRowsAndDisablesCSV(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: 1e-12})
	agentAction(t, s.service, "scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1"}})
	uiAction(t, s, "window-start", &ui.ChangeEvent{Text: "0.4 ps"})
	uiAction(t, s, "window-end", &ui.ChangeEvent{Text: "0.6 ps"})
	state := s.service.Snapshot()
	if state.Scope.WindowValid || len(state.Scope.Traces[0].Points) != 0 || len(state.Scope.Statistics) != 0 {
		t.Fatal("native fields borrowed outside rows")
	}
	s.app.Do(func() {
		root := s.view()
		export, ok := findNode(root, "export-csv")
		if !ok || !export.Disabled {
			t.Fatal("empty interval permits CSV")
		}
		warning, ok := findNode(root, "window-warning")
		if !ok || warning.Text == "" {
			t.Fatal("no empty-window explanation")
		}
	})
	uiAction(t, s, "window-end", &ui.ChangeEvent{Text: "1 ps"})
	uiAction(t, s, "window-start", &ui.ChangeEvent{Text: "0.75 ps"})
	if sc := s.service.Snapshot().Scope; !sc.WindowValid || len(sc.Traces[0].Points) != 1 {
		t.Fatal("native one-row interval", sc)
	}
	duration := 1e-12
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	uiAction(t, s, "recorded-window-start", &ui.ChangeEvent{Text: "0.4 ps"})
	uiAction(t, s, "recorded-window-end", &ui.ChangeEvent{Text: "0.6 ps"})
	r := s.service.Snapshot().Recording
	if r.Scope.WindowValid || r.WindowSamples != 0 || len(r.Readings) != 0 || r.LastSampleTime != 1e-12 {
		t.Fatal("recorded window fabricated data or lost captured endpoint")
	}
	s.app.Do(func() {
		root := s.view()
		node, ok := findNode(root, "recorded-no-interval-samples")
		if !ok || node.Text == "" {
			t.Fatal("no recorded empty-window explanation")
		}
	})
	uiAction(t, s, "recorded-whole-window", nil)
	r = s.service.Snapshot().Recording
	if !r.Scope.WindowValid || r.WindowSamples != 2 || r.Scope.Window.End != r.LastSampleTime {
		t.Fatal("whole-recording control did not restore saved rows")
	}
}
