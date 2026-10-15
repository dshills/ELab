package workbench

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestWaveformQuantityUnitsAndRefusals(t *testing.T) {
	for _, tc := range []struct {
		text, unit string
		want       float64
	}{
		{"1234.56789012345 mV", "V", 1.23456789012345}, {"-750000 µV", "V", -.75}, {"750000 μV", "V", .75}, {"1e6 uV", "V", 1}, {"+1 V", "V", 1}, {".02 kHz", "Hz", 20}, {"20000 mHz", "Hz", 20}, {"30 °", "deg", 30}, {"-3e1deg", "deg", -30}, {"0e-999 V", "V", 0},
	} {
		got, err := parseWaveformQuantity(tc.text, tc.unit)
		if err != nil || got != tc.want {
			t.Fatalf("%s: %.17g, %v", tc.text, got, err)
		}
	}
	for _, tc := range []struct{ text, unit string }{
		{"1 Hz", "V"}, {"1 V", "Hz"}, {"30 rad", "deg"}, {"1 mV", "deg"}, {"NaN", "V"}, {"Inf", "Hz"}, {"1e-999 V", "V"}, {"0x1p0 V", "V"}, {"1,2 V", "V"}, {"1e309 V", "V"}, {"1 MV", "V"}, {"1v", "V"},
	} {
		if _, err := parseWaveformQuantity(tc.text, tc.unit); err == nil {
			t.Fatal("accepted incompatible quantity", tc)
		}
	}
}

func sineQuantityParams() map[string]any {
	return map[string]any{"id": "Wave", "waveform": map[string]any{"amplitude_v": "2000 mV", "offset_v": "1000000 uV", "frequency_hz": ".02 kHz", "phase_degrees": "30 °"}}
}

func TestWaveformQuantityReplayNoOpAndBatch(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.place", PlaceParams{ID: "Wave", Kind: "sine-source"}, "place")
	first := call(t, s, "source.sine.set", sineQuantityParams(), "quantity")
	numeric := SineParams{ID: "Wave", Waveform: SineWaveform{Amplitude: 2, Offset: 1, Frequency: 20, Phase: 30}}
	retry := call(t, s, "source.sine.set", numeric, "quantity")
	if !retry.Duplicate || !reflect.DeepEqual(first.Result, retry.Result) {
		t.Fatal("numeric/quantity retries do not share intent")
	}
	before := s.Snapshot()
	call(t, s, "source.sine.set", numeric, "no-op")
	after := s.Snapshot()
	if before.Document != after.Document || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) {
		t.Fatal("re-entering exact waveform changed document/physics")
	}
	batch := BatchParams{Name: "Unit waveform edit", Edits: []BatchEdit{edit("source.sine.set", map[string]any{"id": "Wave", "waveform": map[string]any{"amplitude_v": "1234.56789012345 mV", "offset_v": 1, "frequency_hz": "20 Hz", "phase_degrees": "-45 deg"}})}}
	call(t, s, "edit.batch", batch, "batch")
	if got := s.Snapshot().Components[0].Sine; got.Amplitude != 1.23456789012345 || got.Phase != -45 {
		t.Fatal("batch rounded quantity", got)
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if !reflect.DeepEqual(s.Snapshot().Components, before.Components) {
		t.Fatal("quantity batch undo did not restore source")
	}
	batch.Edits = append(batch.Edits, edit("source.sine.set", map[string]any{"id": "Wave", "waveform": map[string]any{"amplitude_v": "11 V", "offset_v": "2 V", "frequency_hz": "20 Hz", "phase_degrees": 0}}))
	before = s.Snapshot()
	refuse(t, s, "edit.batch", batch, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("coupled quantity refusal partially committed")
	}
	for _, v := range []any{"1 Hz", "1e-999 V", nil, true, 13} {
		p := sineQuantityParams()
		p["waveform"].(map[string]any)["amplitude_v"] = v
		refuse(t, s, "source.sine.set", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid quantity altered state", v)
		}
	}
}

func TestPulseQuantityMixedUnitsAndStrictSavedWaveforms(t *testing.T) {
	s := pausedFinitePulse(t)
	p := map[string]any{"id": "V1", "waveform": map[string]any{"high_v": "2500 mV", "low_v": "-500000 µV", "high_s": "50 ms", "low_s": "50 ms", "rise_s": "1 ms", "fall_s": "1 ms"}}
	call(t, s, "source.pulse.set", p, "units")
	got := s.Snapshot().Components[0].Pulse
	if got.HighVoltage != 2.5 || got.LowVoltage != -.5 || got.HighSeconds != .05 {
		t.Fatal("mixed time/voltage quantity changed waveform", got)
	}
	before := s.Snapshot()
	p["waveform"].(map[string]any)["low_v"] = "3000 mV"
	refuse(t, s, "source.pulse.set", p, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("ordered voltage refusal altered source")
	}
	s.mu.Lock()
	d := s.documentLocked()
	s.mu.Unlock()
	raw, _ := json.Marshal(d)
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	graph := doc["graph"].(map[string]any)
	for _, part := range graph["components"].([]any) {
		if part.(map[string]any)["id"] == "V1" {
			part.(map[string]any)["pulse"].(map[string]any)["high_v"] = "2.5 V"
		}
	}
	raw, _ = json.Marshal(doc)
	if _, err := decodeDocument(raw); err == nil {
		t.Fatal("command quantity convenience relaxed stored waveforms")
	}
	for _, id := range []string{"source.sine.set", "source.pulse.set"} {
		schema := sweepDetailSchema(t, id)
		field := "waveform.high_v"
		if id == "source.sine.set" {
			field = "waveform.frequency_hz"
		}
		p := inputProperty(schema, field)
		if p["$ref"] != waveformQuantitySchemaID(map[string]string{"source.sine.set": "Hz", "source.pulse.set": "V"}[id]) || p["x-normalized-si-range"] == nil {
			t.Fatal("quantity inputs lack typed/unit bounds", id)
		}
	}
}
