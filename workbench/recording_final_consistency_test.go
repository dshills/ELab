package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestRecordedLEDAndNPNFinalResultsMatchCapturedChannelsAndIDs(t *testing.T) {
	for _, example := range []string{"example.reset", "example.npn", "example.npn-pulse"} {
		t.Run(example, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, example, ReplaceParams{}, "example")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			state := settled(t, s)
			probes := []string{"i:D1"}
			mutations := []string{"led-current"}
			if example != "example.reset" {
				id := ""
				for _, component := range state.Components {
					if component.Kind == "npn" {
						id = component.ID
					}
				}
				probes = []string{"i:" + id + ".base", "i:" + id + ".collector", "i:" + id + ".emitter"}
				mutations = []string{"base-current", "collector-current", "emitter-current", "base-resistor-id", "load-resistor-id", "led-id"}
			}
			duration := 0.
			if IsTransientAnalysis(state.Analysis) {
				duration = .02
			}
			path, _ := exportedRecording(t, s, duration, probes)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			record, err := decodeRecording(original)
			if err != nil {
				t.Fatal("fresh capture refused", err)
			}
			if example != "example.reset" && record.metadata.NPN == nil {
				t.Fatal("NPN fixture did not capture its expected final result", record.metadata.Analysis)
			}
			reencoded, err := encodeVerificationExport(record, "json")
			if err != nil || !bytes.Equal(original, reencoded) {
				t.Fatal("valid capture changed on decode/export", err)
			}
			for _, mutation := range mutations {
				t.Run(mutation, func(t *testing.T) {
					var d VerificationDocument
					if err := json.Unmarshal(original, &d); err != nil {
						t.Fatal(err)
					}
					switch mutation {
					case "led-current":
						d.Metadata.LED.Current += .001
					case "base-current":
						d.Metadata.NPN.BaseCurrent += .001
					case "collector-current":
						d.Metadata.NPN.CollectorCurrent += .001
					case "emitter-current":
						d.Metadata.NPN.EmitterCurrent += .001
					case "base-resistor-id":
						d.Metadata.NPN.BaseResistorID = "Other"
					case "load-resistor-id":
						d.Metadata.NPN.LoadResistorID = "Other"
					case "led-id":
						d.Metadata.NPN.LEDID = "Other"
					}
					bad, err := json.Marshal(d)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := decodeRecording(bad); err == nil {
						t.Fatal("changed final result was accepted with untouched probe rows/statistics")
					}
					s.mu.Lock()
					project := s.documentLocked()
					s.mu.Unlock()
					project.Recordings.Entries[0].Data = d
					embedded, err := json.Marshal(project)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := decodeDocument(embedded); err == nil {
						t.Fatal("embedded recording bypassed final-result consistency validation")
					}
				})
			}
		})
	}
}

func TestRecordedNPNFinalResultRetainsPulseEdgeSides(t *testing.T) {
	for _, mode := range []string{"auto", "high", "low"} {
		for _, duration := range []float64{.05, .1} {
			t.Run(fmt.Sprintf("%s/%g", mode, duration), func(t *testing.T) {
				s := newGraphService(SolveLED, NPNPulseExampleGraph())
				defer s.Close()
				_, view := verify(t, s, VerifyParams{Duration: &duration, InputMode: mode,
					ProbeIDs: []string{"i:Q1.base", "i:Q1.collector", "i:Q1.emitter"}}, "edge-capture")
				s.mu.Lock()
				rows := s.operations[view.Operation.ID].record.rows
				s.mu.Unlock()
				document := VerificationDocument{Metadata: *view.Metadata, Rows: rows}
				original, err := json.MarshalIndent(document, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				original = append(original, '\n')
				record, err := decodeRecording(original)
				if err != nil {
					t.Fatal("valid pulse edge or override refused", err)
				}
				reencoded, err := encodeVerificationExport(record, "json")
				if err != nil || !bytes.Equal(original, reencoded) {
					t.Fatal("pulse edge/override samples changed", err)
				}
			})
		}
	}
}
