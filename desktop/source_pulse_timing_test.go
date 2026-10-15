package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativePulseTimingFieldsShareExactCommandsAndPreserveInvalidState(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := smokeConstructWaveform(s, "finite-pulse-source"); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "pulse-frequency", &ui.ChangeEvent{Text: "20 Hz"})
	uiAction(t, s, "pulse-duty", &ui.ChangeEvent{Text: "25 %"})
	before := s.service.Snapshot()
	clock := before.PulseTimings[0]
	if clock.Frequency != 20 || clock.Duty != 25 || clock.Period != .05 || before.UndoName != "Change pulse timing" {
		t.Fatal("native frequency/duty does not match shared timing", clock)
	}
	uiAction(t, s, "pulse-frequency", &ui.ChangeEvent{Text: clock.FrequencyText})
	uiAction(t, s, "pulse-duty", &ui.ChangeEvent{Text: clock.DutyText})
	after := s.service.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.Components, after.Components) {
		t.Fatal("displayed clock reentry changed native source/history")
	}
	s.app.Do(func() {
		node, _ := findAction(s.view(), "pulse-frequency")
		node.Change(ui.ChangeEvent{Text: "1 kHz"})
	})
	if s.errorMessage == "" || !reflect.DeepEqual(after, s.service.Snapshot()) {
		t.Fatal("invalid frequency shortened ramps or changed state")
	}
	uiAction(t, s, "pulse-period-time", &ui.ChangeEvent{Text: "100 ms"})
	state := s.service.Snapshot()
	if got := state.PulseTimings[0]; got.Frequency != 10 || got.Duty != 25 {
		t.Fatal("period field disagrees with agent timing", got)
	}
	uiAction(t, s, "simulation.restart", nil)
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .005})
	state = s.service.Snapshot()
	want := 5 * (1 - 10*(math.Exp(-.004/.01)-math.Exp(-.005/.01)))
	if state.Transient.Sample.Input != 5 || math.Abs(state.Mixed.Capacitors[0].Voltage-want) > .003 {
		t.Fatal("native timing does not drive the same RC ramp experiment")
	}
}

func TestNativePulseTimingFieldsRefuseInterveningAgentEdit(t *testing.T) {
	for _, field := range []struct{ id, text string }{{"pulse-frequency", "20 Hz"}, {"pulse-period-time", "100 ms"}, {"pulse-duty", "25 %"}} {
		t.Run(field.id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			if err := smokeConstructWaveform(s, "finite-pulse-source"); err != nil {
				t.Fatal(err)
			}
			uiAction(t, s, "simulation.pause", nil)
			var old ui.Node
			s.app.Do(func() {
				var found bool
				old, found = findAction(s.view(), field.id)
				if !found {
					t.Fatal("missing clock field", field.id)
				}
			})
			id := s.service.Snapshot().PulseTimings[0].ID
			agentAction(t, s.service, "source.pulse.timing.set", workbench.PulseTimingParams{ID: id, Frequency: "25 Hz", Duty: "30 %"})
			before := s.service.Snapshot()
			s.app.Do(func() { old.Change(ui.ChangeEvent{Text: field.text}) })
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("old clock field overwrote a newer agent edit", field.id)
			}
		})
	}
}
