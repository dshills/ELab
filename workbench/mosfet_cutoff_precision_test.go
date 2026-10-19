package workbench

import (
	"math"
	"reflect"
	"testing"
)

// At cutoff the channel is zero and the reverse body diode supplies 1 nA
// to the 5 V /1k/red-LED loop. Its reverse exponential correction is below
// 1e-40 A at this operating point, so this independent scalar load line
// bounds the declared model's drain voltage far below the 1 mV test limit.
func mosfetCutoffDrainReference(t *testing.T) float64 {
	t.Helper()
	vt := 1.380649e-23 * 300.15 / 1.602176634e-19
	current := 1e-9
	voltage := 5 - 1000*current - (2*vt*math.Log1p(current/1e-20) + current)
	if correction := current * math.Exp((-voltage+current)/(1.7*vt)); correction > 1e-40 {
		t.Fatalf("reverse body approximation exceeds the reference bound: %g A", correction)
	}
	return voltage
}

func TestMOSFETCutoffDrainVoltagePrecision(t *testing.T) {
	for _, kind := range []string{"nmos", "pmos"} {
		t.Run(kind, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: kind}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			settled(t, s)
			probes := []string{"v:M1", "i:M1.drain", "i:D1", "v:VIN.positive"}
			call(t, s, "scope.probes.set", ProbeParams{IDs: probes}, "probes")
			// Retain both sides of the 200 ms and 500 ms pulse transitions,
			// covering cold and warm cutoff points and ordinary held samples.
			call(t, s, "simulation.step", StepParams{Duration: .2}, "fall")
			call(t, s, "simulation.step", StepParams{Duration: .3}, "rise")
			state := s.Snapshot()
			var voltage, source []ScopePoint
			for _, trace := range state.Scope.Traces {
				if trace.Probe.ID == "v:M1" {
					voltage = trace.Points
				}
				if trace.Probe.ID == "v:VIN.positive" {
					source = trace.Points
				}
			}
			if len(voltage) == 0 || len(voltage) != len(source) {
				t.Fatal("missing aligned drain/source observations")
			}
			sign, offSource := 1., 0.
			if kind == "pmos" {
				sign, offSource = -1, 5
			}
			want, count, maximum := sign*mosfetCutoffDrainReference(t), 0, 0.
			for i, point := range voltage {
				if point.Time != source[i].Time {
					t.Fatal("drain/source times differ")
				}
				if math.Abs(source[i].Value-offSource) > 1e-9 {
					continue
				}
				count++
				maximum = math.Max(maximum, math.Abs(point.Value-want))
			}
			if count < 2 || maximum > .001 {
				t.Fatalf("%d cutoff samples: maximum drain error %.12g V; limit 0.001 V", count, maximum)
			}
			t.Logf("%d cutoff samples, max drain error %.12g V", count, maximum)
			before := s.Snapshot()
			duration := .5
			_, capture := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: probes}, "capture")
			if !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) || !reflect.DeepEqual(before.Transient, s.Snapshot().Transient) {
				t.Fatal("Verify changed live physics")
			}
			rows := s.operations[capture.Operation.ID].record.rows
			capturedOff, capturedMaximum := 0, 0.
			for _, row := range rows {
				if len(row.Values) != len(probes) {
					t.Fatal("capture omitted a probe")
				}
				if math.Abs(row.Values[3]-offSource) > 1e-9 {
					continue
				}
				capturedOff++
				capturedMaximum = math.Max(capturedMaximum, math.Abs(row.Values[0]-want))
				if math.Abs(row.Values[1]-sign*1e-9) > 1e-12 || math.Abs(row.Values[2]-1e-9) > 1e-12 {
					t.Fatal("cutoff currents disagree with independent leakage reference")
				}
			}
			if capturedOff < 2 || capturedMaximum > .001 {
				t.Fatalf("%d captured cutoff samples: maximum drain error %.12g V", capturedOff, capturedMaximum)
			}
			t.Logf("%d captured cutoff samples, max drain error %.12g V", capturedOff, capturedMaximum)

		})
	}
}
