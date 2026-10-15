package main

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeSharedPulseClockControlsAndStaleField(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.npn-pulse", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "VIN"})
	uiAction(t, s, "pulse-frequency", &ui.ChangeEvent{Text: "20 Hz"})
	uiAction(t, s, "pulse-duty", &ui.ChangeEvent{Text: "25 %"})
	uiAction(t, s, "pulse-period-time", &ui.ChangeEvent{Text: "100 ms"})
	state := s.service.Snapshot()
	if state.TransientSettings.HighSeconds != .025 || math.Abs(state.TransientSettings.LowSeconds-.075) > 1e-16 || !state.PulseTimings[0].Shared {
		t.Fatal("native shared clock does not match global settings")
	}
	s.app.Do(func() {
		note, found := findNode(s.view(), "legacy-pulse-model")
		if !found || !strings.Contains(note.Text, "every ideal-square") {
			t.Fatal("shared effect is not visible")
		}
	})
	uiAction(t, s, "simulation.restart", nil)
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .025})
	if s.service.Snapshot().NPN.State != "cutoff" {
		t.Fatal("native quarter-cycle blink edge missed")
	}
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .075})
	if s.service.Snapshot().NPN.State != "saturation" {
		t.Fatal("native next-cycle blink edge missed")
	}
	var old ui.Node
	s.app.Do(func() { old, _ = findAction(s.view(), "pulse-period-time") })
	agentAction(t, s.service, "simulation.pulse.timing.set", workbench.IdealPulseTimingParams{ID: "VIN", Frequency: "25 Hz", Duty: "40 %"})
	before := s.service.Snapshot()
	s.app.Do(func() { old.Change(ui.ChangeEvent{Text: "100 ms"}) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old shared field overwrote newer agent timing")
	}
}
