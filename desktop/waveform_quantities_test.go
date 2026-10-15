package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeWaveformQuantityPrecisionNoOpAndStaleFields(t *testing.T) {
	for _, tc := range []struct {
		kind, id, text string
		want           float64
	}{
		{"sine-source", "sine-amplitude", "1234.56789012345 mV", 1.23456789012345},
		{"sine-source", "sine-offset", "-123.456789012345 mV", -.123456789012345},
		{"sine-source", "sine-frequency", ".123456789012345 kHz", 123.456789012345},
		{"sine-source", "sine-phase", "-123.456789012345 °", -123.456789012345},
		{"finite-pulse-source", "pulse-high-v", "4123.456789012345 mV", 4.123456789012345},
		{"finite-pulse-source", "pulse-low-v", "-2345.6789012345 mV", -2.3456789012345},
	} {
		t.Run(tc.id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "Wave", Kind: tc.kind})
			uiAction(t, s, tc.id, &ui.ChangeEvent{Text: tc.text})
			state := s.service.Snapshot()
			part := state.Components[0]
			var got float64
			switch tc.id {
			case "sine-amplitude":
				got = part.Sine.Amplitude
			case "sine-offset":
				got = part.Sine.Offset
			case "sine-frequency":
				got = part.Sine.Frequency
			case "sine-phase":
				got = part.Sine.Phase
			case "pulse-high-v":
				got = part.Pulse.HighVoltage
			case "pulse-low-v":
				got = part.Pulse.LowVoltage
			}
			if got != tc.want {
				t.Fatalf("native commit lost decimal precision: %.17g want %.17g", got, tc.want)
			}
			var field ui.Node
			s.app.Do(func() { field, _ = findNode(s.view(), tc.id) })
			if field.Kind != ui.TextField {
				t.Fatal("waveform quantity is not text")
			}
			uiAction(t, s, tc.id, &ui.ChangeEvent{Text: field.Text})
			same := s.service.Snapshot()
			if !reflect.DeepEqual(state.Components, same.Components) || state.Document != same.Document || !reflect.DeepEqual(state.UndoEntry, same.UndoEntry) || state.RunID != same.RunID {
				t.Fatal("displayed exact text manufactured an edit")
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				invalid, _ := findNode(s.view(), tc.id)
				invalid.Change(ui.ChangeEvent{Text: "1e-999 V"})
			})
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("invalid committed text changed source")
			}
			s.app.Do(func() { field, _ = findNode(s.view(), tc.id) })
			if tc.kind == "sine-source" {
				w := *part.Sine
				w.Amplitude = 3
				agentAction(t, s.service, "source.sine.set", workbench.SineParams{ID: "Wave", Waveform: w})
			} else {
				w := *part.Pulse
				w.HighVoltage = 6
				agentAction(t, s.service, "source.pulse.set", workbench.PulseParams{ID: "Wave", Waveform: w})
			}
			before = s.service.Snapshot()
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: tc.text}) })
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("stale waveform text replaced agent settings")
			}
		})
	}
}
