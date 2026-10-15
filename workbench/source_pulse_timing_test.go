package workbench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func pausedFinitePulse(t *testing.T) *Service {
	t.Helper()
	s := newGraphService(SolveLED, waveformRCGraph(t, "finite-pulse-source"))
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	return s
}

func TestPulseTimingEditPreservesRampsLevelsAndExactNoOp(t *testing.T) {
	s := pausedFinitePulse(t)
	original, _ := s.graph.Component("V1")
	before := s.Snapshot()
	call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: "20 Hz", Duty: "25 %"}, "clock")
	after := s.Snapshot()
	part, _ := s.graph.Component("V1")
	if part.Pulse.HighSeconds != .0125 || math.Abs(part.Pulse.LowSeconds-.0375) > 1e-16 || part.Pulse.HighSeconds+part.Pulse.LowSeconds != .05 || part.Pulse.RiseSeconds != original.Pulse.RiseSeconds || part.Pulse.FallSeconds != original.Pulse.FallSeconds || part.Value != original.Value || after.ElectricalRevision != before.ElectricalRevision+1 || !after.Paused {
		t.Fatal("timing edit changed retained configuration or missed electrical semantics", part.Pulse)
	}
	clock := after.PulseTimings[0]
	call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: clock.FrequencyText, Duty: clock.DutyText}, "same-clock")
	stable := s.Snapshot()
	if stable.Document != after.Document || !reflect.DeepEqual(stable.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(stable.Transient, after.Transient) || stable.RunID != after.RunID || !reflect.DeepEqual(stable.Components, after.Components) {
		t.Fatal("reentering displayed timing manufactured a physical/history edit")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if got, _ := s.graph.Component("V1"); !reflect.DeepEqual(got.Pulse, original.Pulse) {
		t.Fatal("one undo did not restore both phases")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	if got, _ := s.graph.Component("V1"); !reflect.DeepEqual(got.Pulse, part.Pulse) {
		t.Fatal("redo did not restore timing")
	}
}

func TestPulseTimingDisplayedTextPreservesNontrivialPhaseBits(t *testing.T) {
	s := pausedFinitePulse(t)
	for i, phases := range [][2]float64{{.0030000000000000005, .007}, {.0125, .037500000000000006}, {.02345678901234567, .07123456789012345}} {
		wave := DefaultPulseWaveform()
		wave.HighSeconds, wave.LowSeconds = phases[0], phases[1]
		call(t, s, "source.pulse.set", PulseParams{ID: "V1", Waveform: wave}, "phases"+number(float64(i)))
		before := s.Snapshot()
		clock := before.PulseTimings[0]
		call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: clock.FrequencyText, Duty: clock.DutyText}, "redisplay"+number(float64(i)))
		after := s.Snapshot()
		if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || before.Document != after.Document || before.RunID != after.RunID {
			t.Fatal("displayed timing changed binary phase definitions", phases)
		}
	}
}

func TestPulseTimingRefusalsArePureAndBoundsRemainAuthoritative(t *testing.T) {
	s := pausedFinitePulse(t)
	for _, bad := range []any{
		map[string]any{"id": "V1", "duty_percent": "50"},
		map[string]any{"id": "V1", "duty_percent": "50", "frequency_hz": "10", "period_s": .1},
		map[string]any{"id": "V1", "duty_percent": 50, "frequency_hz": "10"},
		map[string]any{"id": "V1", "duty_percent": "50", "period_s": nil},
		PulseTimingParams{ID: "V1", Frequency: "1 kHz", Duty: "50"}, // Retained 1 ms ramps cannot fit.
		PulseTimingParams{ID: "V1", Frequency: "10", Duty: "0"},
		PulseTimingParams{ID: "V1", Frequency: "10", Duty: "100"},
		PulseTimingParams{ID: "V1", Frequency: "NaN", Duty: "50"},
		PulseTimingParams{ID: "V1", Frequency: "10 %", Duty: "50"},
		PulseTimingParams{ID: "V1", Frequency: "10", Duty: "50 Hz"},
		PulseTimingParams{ID: "V1", Frequency: "1e-999 Hz", Duty: "50"},
		PulseTimingParams{ID: "V1", Frequency: "1000.0000000000001 Hz", Duty: "50"},
		PulseTimingParams{ID: "V1", Period: math.Nextafter(.001, 0), Duty: "50"},
		PulseTimingParams{ID: "V1", Period: math.Nextafter(20, math.Inf(1)), Duty: "50"},
	} {
		before := s.Snapshot()
		refuse(t, s, "source.pulse.timing.set", bad, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid timing changed state", bad)
		}
	}
	refuse(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "missing", Frequency: "10", Duty: "50"}, "not-found")
	refuse(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "R1", Frequency: "10", Duty: "50"}, "invalid-params")
	for i, p := range []PulseTimingParams{{ID: "V1", Frequency: "50 mHz", Duty: "50"}, {ID: "V1", Frequency: "10", Duty: "50"}} {
		call(t, s, "source.pulse.timing.set", p, "boundary"+number(float64(i)))
	}
	ramps := *s.graph.Components[0].Pulse
	ramps.RiseSeconds, ramps.FallSeconds = 1e-6, 1e-6
	call(t, s, "source.pulse.set", PulseParams{ID: "V1", Waveform: ramps}, "short-ramps")
	call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: "1 kHz", Duty: "0.1 %"}, "minimum-phase")
	if got := s.Snapshot().Components[0].Pulse; got.HighSeconds != 1e-6 || got.HighSeconds+got.LowSeconds < .001 {
		t.Fatal("minimum phase/period endpoint lost", got)
	}
}

func TestPulseTimingUnitReplayBatchAndPortability(t *testing.T) {
	s := pausedFinitePulse(t)
	first := call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: "20 Hz", Duty: "25 %"}, "units")
	retry := call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: ".02 kHz", Duty: "2.5e1"}, "units")
	if !retry.Duplicate || !reflect.DeepEqual(first.Result, retry.Result) {
		t.Fatal("equivalent quantity units did not replay original intent")
	}
	copy := first.Result.(Snapshot)
	copy.PulseTimings[0].Duty = 999
	own := call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: "20", Duty: "25"}, "units").Result.(Snapshot)
	if own.PulseTimings[0].Duty == 999 || s.Snapshot().PulseTimings[0].Duty == 999 {
		t.Fatal("configured timing snapshot aliases caller/cache")
	}
	before := s.Snapshot()
	batch := BatchParams{Name: "Pulse clock", Edits: []BatchEdit{edit("source.pulse.timing.set", map[string]any{"id": "V1", "period_s": "100 ms", "duty_percent": "50 %"}), edit("component.set", ValueParams{ID: "R1", Value: "bad"})}}
	refuse(t, s, "edit.batch", batch, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("failed clock batch committed partially")
	}
	batch.Edits = batch.Edits[:1]
	call(t, s, "edit.batch", batch, "batch")
	path := filepath.Join(t.TempDir(), "clock.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil || document["pulse_timings"] != nil {
		t.Fatal("derived timing leaked into saved format", err)
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	if !reflect.DeepEqual(fresh.Snapshot().PulseTimings, s.Snapshot().PulseTimings) {
		t.Fatal("source phase timing did not round-trip")
	}
}

func TestPulseTimingDrivesExistingRCModelAndIncompleteSources(t *testing.T) {
	s := pausedFinitePulse(t)
	call(t, s, "source.pulse.timing.set", PulseTimingParams{ID: "V1", Frequency: "20 Hz", Duty: "25 %"}, "clock")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .005}, "advance")
	state := s.Snapshot()
	// Independent first-order RC solution for a 1 ms linear rise then held 5 V.
	want := 5 * (1 - 10*(math.Exp(-.004/.01)-math.Exp(-.005/.01)))
	if state.Transient.Sample.Input != 5 || math.Abs(state.Mixed.Capacitors[0].Voltage-want) > .003 || state.Mixed.KCLResidual > 1e-9 {
		t.Fatal("clock edit did not drive the retained ramp/RC model", state.Mixed.Capacitors[0], want)
	}
	incomplete := New()
	defer incomplete.Close()
	call(t, incomplete, "component.place", PlaceParams{ID: "Clock", Kind: "finite-pulse-source", Position: Point{}}, "source")
	call(t, incomplete, "source.pulse.timing.set", map[string]any{"id": "Clock", "period_s": "200 ms", "duty_percent": "75 %"}, "clock")
	clock := incomplete.Snapshot().PulseTimings[0]
	if clock.ID != "Clock" || clock.Frequency != 5 || math.Abs(clock.Duty-75) > 1e-12 {
		t.Fatal("incomplete source cannot configure its own timing", clock)
	}
}

func TestPulseTimingDiscoveryKeepsQuantityInputsAndNumericOutputs(t *testing.T) {
	schema := sweepDetailSchema(t, "source.pulse.timing.set")
	if len(schema["oneOf"].([]any)) != 2 || inputProperty(schema, "period_s")["$ref"] != TimeInputSchemaID || inputProperty(schema, "frequency_hz")["type"] != "string" || inputProperty(schema, "duty_percent")["type"] != "string" {
		t.Fatal("timing quantity/alternative schema does not match admission")
	}
	var snapshot map[string]any
	if err := json.Unmarshal(SchemaRegistry()[SnapshotSchemaID], &snapshot); err != nil {
		t.Fatal(err)
	}
	item := inputProperty(snapshot, "pulse_timings")["items"].(map[string]any)
	for _, name := range []string{"frequency_hz", "period_s", "duty_percent"} {
		if inputProperty(item, name)["type"] != "number" {
			t.Fatal("quantity grammar leaked into numeric output", name)
		}
	}
}
