package workbench

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSharedPulseClockDrivesBlinkPhasesAndNoOpHistory(t *testing.T) {
	s := pausedNPNPulse(t)
	defer s.Close()
	before := s.Snapshot()
	call(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "VIN", Frequency: "20 Hz", Duty: "25 %"}, "clock")
	state := s.Snapshot()
	if state.TransientSettings.HighSeconds != .0125 || state.TransientSettings.HighSeconds+state.TransientSettings.LowSeconds != .05 || state.TransientSettings.StepSeconds != before.TransientSettings.StepSeconds || state.TransientSettings.PlaybackSpeed != before.TransientSettings.PlaybackSpeed || !reflect.DeepEqual(before.Components, state.Components) || state.UndoName != "Change shared pulse timing" {
		t.Fatal("shared clock changed unrelated data or missed history", state.TransientSettings)
	}
	clock := state.PulseTimings[0]
	if !clock.Shared || clock.SourceKind != "pulse-source" || clock.ID != "VIN" {
		t.Fatal("shared clock scope not discoverable", clock)
	}
	call(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "VIN", Frequency: clock.FrequencyText, Duty: clock.DutyText}, "same")
	after := s.Snapshot()
	if state.Document != after.Document || !reflect.DeepEqual(state.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(state.Transient, after.Transient) || state.RunID != after.RunID {
		t.Fatal("shared displayed timing reentry changed history/physics")
	}
	call(t, s, "simulation.restart", struct{}{}, "restart-shared-clock")
	if s.Snapshot().NPN.State != "saturation" {
		t.Fatal("blink did not start high")
	}
	call(t, s, "simulation.step", StepParams{Duration: .0125}, "fall")
	if got := s.Snapshot(); got.NPN.State != "cutoff" || got.Transient.Sample.Input != 0 {
		t.Fatal("25 percent falling edge not applied")
	}
	call(t, s, "simulation.step", StepParams{Duration: .0375}, "rise")
	if got := s.Snapshot(); got.NPN.State != "saturation" || got.Transient.Sample.Input != 5 {
		t.Fatal("20 Hz next cycle did not start")
	}
}

func TestSharedClockNoOpStillChecksCurrentSampling(t *testing.T) {
	s := New()
	defer s.Close()
	cfg := DefaultTransientSettings()
	cfg.StepSeconds, cfg.HighSeconds, cfg.LowSeconds = .01, .001, .001
	call(t, s, "simulation.transient.set", cfg, "configure-before-source")
	call(t, s, "component.place", PlaceParams{ID: "Clock", Kind: "pulse-source", Position: Point{}}, "source")
	before := s.Snapshot()
	clock := before.PulseTimings[0]
	refuse(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "Clock", Frequency: clock.FrequencyText, Duty: clock.DutyText}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("invalid current clock bypassed sampling check")
	}
	call(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "Clock", Frequency: "20 Hz", Duty: "50 %"}, "repair")
	if s.Snapshot().TransientSettings.HighSeconds != .025 {
		t.Fatal("shared clock could not repair inherited sampling mismatch")
	}
}

func TestSharedClockAffectsAllSquaresButNoFiniteDefinition(t *testing.T) {
	s := pausedFinitePulse(t)
	for i, id := range []string{"SquareA", "SquareB"} {
		call(t, s, "component.place", PlaceParams{ID: id, Kind: "pulse-source", Value: number(float64(3 + 6*i)), Position: Point{X: float64(400 + 200*i), Y: 400}}, id)
	}
	before := s.Snapshot()
	call(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "SquareB", Period: .2, Duty: "60 %"}, "shared")
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) {
		t.Fatal("global timing rewrote source definitions")
	}
	for _, clock := range after.PulseTimings {
		if clock.Shared {
			if clock.Frequency != 5 || math.Abs(clock.Duty-60) > 1e-12 {
				t.Fatal("square source did not share clock", clock)
			}
		} else if clock.ID != "V1" || clock.Frequency != 10 || clock.Duty != 50 {
			t.Fatal("finite waveform changed with shared clock", clock)
		}
	}
}

func TestSharedPulseClockSamplingBoundsAndBatchRollback(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	refuse(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "V1", Frequency: "1 kHz", Duty: "50 %"}, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("sampling refusal changed settings")
	}
	bad := BatchParams{Name: "Rejected shared clock", Edits: []BatchEdit{edit("simulation.pulse.timing.set", IdealPulseTimingParams{ID: "V1", Frequency: "20", Duty: "25"}), edit("component.set", ValueParams{ID: "R1", Value: "bad"})}}
	refuse(t, s, "edit.batch", bad, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("failed settings batch committed partially")
	}
	bad.Edits = bad.Edits[:1]
	call(t, s, "edit.batch", bad, "batch")
	if s.Snapshot().TransientSettings.HighSeconds != .0125 || s.Snapshot().UndoName != bad.Name {
		t.Fatal("batch dropped staged shared settings")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if s.Snapshot().TransientSettings != before.TransientSettings {
		t.Fatal("one undo did not restore shared settings")
	}
	cfg := s.Snapshot().TransientSettings
	cfg.StepSeconds, cfg.PlaybackSpeed = 1e-6, .001
	call(t, s, "simulation.transient.set", cfg, "fine-sampling")
	call(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "V1", Frequency: "500 kHz", Duty: "50 %"}, "maximum")
	state := s.Snapshot()
	if state.TransientSettings.HighSeconds != 1e-6 || state.TransientSettings.LowSeconds != 1e-6 || state.TransientSettings.StepSeconds != 1e-6 || state.TransientSettings.PlaybackSpeed != .001 {
		t.Fatal("maximum shared frequency changed explicit sampling", state.TransientSettings)
	}
	path := filepath.Join(t.TempDir(), "shared.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	if fresh.Snapshot().TransientSettings != state.TransientSettings || !reflect.DeepEqual(fresh.Snapshot().PulseTimings, state.PulseTimings) {
		t.Fatal("shared clock did not restore from settings")
	}
}

func TestSharedClockDiscoveryAndQuantityReplay(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	schema := sweepDetailSchema(t, "simulation.pulse.timing.set")
	if schema["x-command-contract"].(map[string]any)["scope"] != "ideal-square-shared-clock" || inputProperty(schema, "period_s")["$ref"] != TimeInputSchemaID {
		t.Fatal("missing shared clock contract")
	}
	first := call(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "V1", Frequency: "20 Hz", Duty: "25 %"}, "units")
	second := call(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "V1", Frequency: ".02 kHz", Duty: "2.5e1"}, "units")
	if !second.Duplicate || !reflect.DeepEqual(first.Result, second.Result) {
		t.Fatal("shared unit aliases did not replay")
	}
	for _, params := range []any{IdealPulseTimingParams{ID: "R1", Frequency: "20", Duty: "25"}, map[string]any{"id": "V1", "frequency_hz": "20", "period_s": .05, "duty_percent": "25"}, map[string]any{"id": "V1", "frequency_hz": 20, "duty_percent": "25"}} {
		refuse(t, s, "simulation.pulse.timing.set", params, "invalid-params")
	}
	refuse(t, s, "simulation.pulse.timing.set", IdealPulseTimingParams{ID: "missing", Frequency: "20", Duty: "25"}, "not-found")
}
