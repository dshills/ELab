package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDetailedDiscoveryReadOnlyOwnedAndBackwardCompatible(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	for _, cap := range Catalog() {
		data, _ := json.Marshal(CapabilityQuery{Command: cap.ID})
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: cap.ID, Command: "capabilities.list", Params: data}, "agent")
		if r.Error != nil {
			t.Fatal(cap.ID, r.Error)
		}
		list := r.Result.(map[string]any)["capabilities"].([]Capability)
		if len(list) != 1 || list[0].ID != cap.ID || list[0].Mutation != cap.Mutation || !reflect.DeepEqual(list[0].OutputSchema, cap.OutputSchema) {
			t.Fatal("filtered query lost command/output identity", cap.ID)
		}
		var input map[string]any
		_ = json.Unmarshal(list[0].Input, &input)
		if _, ref := input["$ref"]; ref || input["type"] != "object" {
			t.Fatal("root input was not expanded", cap.ID)
		}
		list[0].Input[0] = '!'
		fresh, err := detailedCapabilities(cap.ID)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("caller corrupted later discovery")
		}
	}
	for _, bad := range []struct{ raw, code string }{
		{`{"command":"unknown"}`, "not-found"},
		{`{"command":1}`, "invalid-params"},
		{`{"command":null}`, "invalid-params"},
		{`{"command":"source.sine.set","extra":true}`, "invalid-params"},
		{`{"command":"x","command":"source.sine.set"}`, "invalid-params"},
		{`{"command":"` + strings.Repeat("x", 129) + `"}`, "invalid-params"},
	} {
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: "bad", Command: "capabilities.list", Params: json.RawMessage(bad.raw)}, "agent")
		if r.Error == nil || r.Error.Code != bad.code {
			t.Fatal("bad filtered discovery admitted", bad, r.Error)
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("discovery mutated state")
	}
	data, _ := json.Marshal(Catalog())
	if len(data) > 65536 {
		t.Fatal("ordinary discovery exceeds existing limit", len(data))
	}
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "full", Command: "capabilities.list", Params: json.RawMessage(`{}`)}, "agent")
	if r.Error != nil || len(r.Result.(map[string]any)["capabilities"].([]Capability)) != len(Catalog()) {
		t.Fatal("ordinary enumeration changed")
	}
}

func TestSineDetailedNumericBoundsMatchValidator(t *testing.T) {
	caps, _ := detailedCapabilities("source.sine.set")
	var schema map[string]any
	_ = json.Unmarshal(caps[0].Input, &schema)
	for _, tc := range []struct {
		field string
		set   func(*SineWaveform, float64)
	}{
		{"amplitude_v", func(w *SineWaveform, v float64) { w.Offset = 0; w.Amplitude = v }},
		{"offset_v", func(w *SineWaveform, v float64) { w.Amplitude = 0; w.Offset = v }},
		{"frequency_hz", func(w *SineWaveform, v float64) { w.Frequency = v }},
		{"phase_degrees", func(w *SineWaveform, v float64) { w.Phase = v }},
	} {
		p := inputProperty(schema, "waveform."+tc.field)
		min, max := p["minimum"].(float64), p["maximum"].(float64)
		for _, v := range []float64{min, max} {
			w := DefaultSineWaveform()
			tc.set(&w, v)
			if err := w.Validate(); err != nil {
				t.Fatal("published boundary rejected", tc.field, v, err)
			}
		}
		for _, v := range []float64{min - .001, max + .001} {
			w := DefaultSineWaveform()
			tc.set(&w, v)
			if w.Validate() == nil {
				t.Fatal("outside boundary admitted", tc.field, v)
			}
		}
	}
}

func TestPulseDetailedBoundsAndRelationsMatchValidator(t *testing.T) {
	caps, _ := detailedCapabilities("source.pulse.set")
	var schema map[string]any
	_ = json.Unmarshal(caps[0].Input, &schema)
	for _, tc := range []struct {
		field string
		set   func(*PulseWaveform, float64)
	}{
		{"high_v", func(w *PulseWaveform, v float64) { w.LowVoltage = -12; w.HighVoltage = v }},
		{"low_v", func(w *PulseWaveform, v float64) { w.HighVoltage = 12; w.LowVoltage = v }},
		{"high_s", func(w *PulseWaveform, v float64) { w.RiseSeconds = 1e-6; w.HighSeconds = v }},
		{"low_s", func(w *PulseWaveform, v float64) { w.FallSeconds = 1e-6; w.LowSeconds = v }},
		{"rise_s", func(w *PulseWaveform, v float64) { w.HighSeconds = 10; w.RiseSeconds = v }},
		{"fall_s", func(w *PulseWaveform, v float64) { w.LowSeconds = 10; w.FallSeconds = v }},
	} {
		p := inputProperty(schema, "waveform."+tc.field)
		min, max := p["minimum"].(float64), p["maximum"].(float64)
		for _, v := range []float64{min, max} {
			w := DefaultPulseWaveform()
			tc.set(&w, v)
			if err := w.Validate(); err != nil {
				t.Fatal("published pulse boundary rejected", tc.field, v, err)
			}
		}
		for _, v := range []float64{min - 1e-7, max + 1e-7} {
			w := DefaultPulseWaveform()
			tc.set(&w, v)
			if w.Validate() == nil {
				t.Fatal("outside pulse bound admitted", tc.field, v)
			}
		}
	}
	for _, set := range []func(*PulseWaveform){
		func(w *PulseWaveform) { w.LowVoltage = w.HighVoltage + 1 },
		func(w *PulseWaveform) { w.RiseSeconds = w.HighSeconds + .001 },
		func(w *PulseWaveform) { w.FallSeconds = w.LowSeconds + .001 },
		func(w *PulseWaveform) {
			w.HighSeconds = .0004
			w.LowSeconds = .0004
			w.RiseSeconds = .0001
			w.FallSeconds = .0001
		},
	} {
		w := DefaultPulseWaveform()
		set(&w)
		if w.Validate() == nil {
			t.Fatal("published relation does not constrain runtime")
		}
	}
}

func TestDetailedElectricalExamplesAreExecutable(t *testing.T) {
	for _, id := range []string{"source.sine.set", "source.pulse.set", "simulation.transient.set", "potentiometer.wiper.set", "example.load"} {
		t.Run(id, func(t *testing.T) {
			caps, err := detailedCapabilities(id)
			if err != nil {
				t.Fatal(err)
			}
			var schema map[string]any
			_ = json.Unmarshal(caps[0].Input, &schema)
			if id == "source.pulse.set" && (len(schema["x-normalized-si-relations"].([]any)) != 4 || inputProperty(schema, "waveform.rise_s")["$ref"] != TimeInputSchemaID) {
				t.Fatal("pulse relationships/time grammar omitted")
			}
			for _, example := range schema["examples"].([]any) {
				s := New()
				defer s.Close()
				switch id {
				case "source.sine.set":
					call(t, s, "component.place", PlaceParams{Kind: "sine-source", ID: "V1"}, "part")
				case "source.pulse.set":
					call(t, s, "component.place", PlaceParams{Kind: "finite-pulse-source", ID: "V1"}, "part")
				case "potentiometer.wiper.set":
					call(t, s, "component.place", PlaceParams{Kind: "potentiometer", ID: "P1"}, "part")
				}
				call(t, s, id, example, "example")
			}
		})
	}
}
