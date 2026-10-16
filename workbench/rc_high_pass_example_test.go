package workbench

import (
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

// Independent linear RC solution: each source edge adds ΔV exp(-(t-edge)/RC)
// to output. Capacitor voltage is continuous, including both saved edge sides.
func highPassReference(at, input float64) (output, capacitor float64) {
	rightOutput, rightInput := 0., 0.
	for edge := 0; float64(edge)*.05 <= at+1e-12; edge++ {
		delta := 5.
		if edge%2 == 1 {
			delta = -5
		}
		rightInput += delta
		rightOutput += delta * math.Exp(-math.Max(0, at-float64(edge)*.05)/.01)
	}
	capacitor = rightInput - rightOutput
	return input - capacitor, capacitor
}

func TestRCHighPassExamplePulseResponseAndEdgeContinuity(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-high-pass"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	initial := settled(t, s)
	if initial.Selected != "R1" || initial.Example.Modified || !reflect.DeepEqual(initial.Scope.Selected, []string{"v:V1.positive", "v:R1", "v:C1"}) {
		t.Fatal("high-pass example is not ready to inspect output", initial.Scope.Selected)
	}
	for sample := 0; sample <= 120; sample++ {
		if sample > 0 {
			call(t, s, "simulation.step", StepParams{Duration: .001}, fmt.Sprintf("step-%d", sample))
		}
		state := s.Snapshot()
		wantOutput, wantCap := highPassReference(state.Transient.Sample.Time, state.Transient.Sample.Input)
		if math.Abs(state.Mixed.Resistors[0].Voltage-wantOutput) > math.Max(.001, .001*math.Abs(wantOutput)) || math.Abs(state.Mixed.Capacitors[0].Voltage-wantCap) > math.Max(.001, .001*math.Abs(wantCap)) || math.Abs(state.Mixed.Resistors[0].Current-wantOutput/10000) > math.Max(.001, .001*math.Abs(wantOutput))/10000 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
			t.Fatal("high-pass exponential/current/conservation", sample, state.Mixed, wantOutput, wantCap)
		}
	}
	state := s.Snapshot()
	for _, edge := range []float64{.05, .1} {
		for _, probeID := range []string{"v:R1", "v:C1"} {
			found := false
			for _, trace := range state.Scope.Traces {
				if trace.Probe.ID != probeID {
					continue
				}
				for i, point := range trace.Points {
					if i == 0 || math.Abs(point.Time-edge) > 1e-12 || point.Time != trace.Points[i-1].Time {
						continue
					}
					jump := point.Value - trace.Points[i-1].Value
					want := 0.
					if probeID == "v:R1" {
						want = 5
						if edge == .05 {
							want = -5
						}
					}
					// A sample-grid endpoint may repeat the post-edge value.
					if probeID == "v:R1" && math.Abs(jump) < 1e-10 {
						continue
					}
					if math.Abs(jump-want) > 1e-10 {
						t.Fatal("edge loses output jump or capacitor continuity", edge, probeID, jump)
					}
					found = true
				}
			}
			if !found {
				t.Fatal("missing saved edge sides", edge, probeID)
			}
		}
	}
	duration := .12
	_, capture := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: initial.Scope.Selected}, "verify")
	page := call(t, s, "operation.samples", VerificationPageParams{ID: capture.Operation.ID, Limit: 200}, "samples").Result.(VerificationPage)
	if page.NextOffset != page.Total {
		t.Fatal("incomplete high-pass capture page")
	}
	for _, row := range page.Rows {
		output, capacitor := highPassReference(row.Time, row.Values[0])
		if math.Abs(row.Values[1]-output) > math.Max(.001, .001*math.Abs(output)) || math.Abs(row.Values[2]-capacitor) > math.Max(.001, .001*math.Abs(capacitor)) {
			t.Fatal("Verify differs from analytic edge response", row, output, capacitor)
		}
	}
	if !reflect.DeepEqual(state.Mixed, s.Snapshot().Mixed) || !reflect.DeepEqual(state.Transient, s.Snapshot().Transient) {
		t.Fatal("Verify changes the live experiment")
	}
}

func TestRCHighPassProbeDefaultsResetAndSavedOverrides(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-high-pass"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"i:C1"}}, "custom-probe")
	path := filepath.Join(t.TempDir(), "high-pass.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	loaded := settled(t, fresh)
	if loaded.Example.Info.ID != "rc-high-pass" || loaded.Example.Modified || loaded.Transient.Sample.Time != 0 || !reflect.DeepEqual(loaded.Scope.Selected, []string{"i:C1"}) {
		t.Fatal("saved high-pass loses origin, startup or selected probes")
	}
	call(t, fresh, "example.current.reset", ExampleResetParams{}, "reset")
	reset := settled(t, fresh)
	if !reflect.DeepEqual(reset.Scope.Selected, []string{"v:V1.positive", "v:R1", "v:C1"}) || !reset.Paused {
		t.Fatal("reset does not restore example probes and pause preference", reset.Scope.Selected)
	}
	call(t, fresh, "example.load", ExampleLoadParams{ID: "rc", DiscardChanges: true}, "low-pass")
	lowPass := settled(t, fresh)
	if lowPass.Analysis != "RC transient" || !reflect.DeepEqual(lowPass.Scope.Selected, []string{"v:C1.1", "v:V1.positive"}) {
		t.Fatal("new default probes leak into existing low-pass example", lowPass.Scope.Selected)
	}
}
