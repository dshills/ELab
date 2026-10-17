package workbench

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

func TestRecordingDampingMatchesCapturedIntegrationMethod(t *testing.T) {
	for _, example := range []string{"flasher", "pmos", "opamp-follower", "rlc"} {
		t.Run(example, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: example}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			settled(t, s)
			path, _ := exportedRecording(t, s, .02, nil)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var d VerificationDocument
			if err := json.Unmarshal(raw, &d); err != nil {
				t.Fatal(err)
			}
			if d.Metadata.Mixed == nil || d.Metadata.Mixed.NumericalDamping != 0 {
				t.Fatal("fresh capture contains a backward-Euler damping readout")
			}
			if _, err := decodeRecording(raw); err != nil {
				t.Fatal("fresh capture refused", err)
			}
			c, _, diagnostic := d.Metadata.Graph.Compile()
			if diagnostic != nil {
				t.Fatal(diagnostic)
			}
			for _, value := range []float64{1e-6, -1e-6, math.NaN(), math.Inf(1)} {
				d.Metadata.Mixed.NumericalDamping = value
				if err := validateRecordedResults(d.Metadata, c); err == nil || !strings.Contains(err.Error(), "damping") {
					t.Fatal("invalid method-specific damping accepted", value, err)
				}
				if finite(value) {
					changed, err := json.Marshal(d)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := decodeRecording(changed); err == nil || !strings.Contains(err.Error(), "damping") {
						t.Fatal("invalid serialized damping accepted", value, err)
					}
				}
			}
		})
	}
}
