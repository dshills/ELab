package workbench

import (
	"math"
	"reflect"
	"testing"
)

// Independent load-line reference for the pinned generic pair: use the
// textbook piecewise equation and a scalar bisection, not the MNA/Jacobian.
func cmosReference(input float64) (output, pCurrent, nCurrent float64) {
	channel := func(gate, drain float64) float64 {
		overdrive := gate - 1.5
		if overdrive <= 0 {
			return 0
		}
		base := .05 * overdrive * overdrive
		if drain < overdrive {
			base = .1 * (overdrive*drain - drain*drain/2)
		}
		return base * (1 + .02*drain)
	}
	lo, hi := 0., 5.
	for range 70 {
		out := (lo + hi) / 2
		if channel(5-input, 5-out) > channel(input, out)+out/10000 {
			lo = out
		} else {
			hi = out
		}
	}
	output = (lo + hi) / 2
	return output, -channel(5-input, 5-output), channel(input, output)
}

func TestCMOSExampleLoadLineAndSupplyConservation(t *testing.T) {
	for _, input := range []float64{0, 1.5, 2.5, 3.5, 5} {
		g := CMOSExampleGraph()
		for i := range g.Components {
			if g.Components[i].ID == "VIN" {
				g.Components[i], _ = MakeComponent("VIN", "dc-source", g.Components[i].Position, FormatValue(input, "V"))
			}
		}
		state, _ := solveFloatingDC(t, g)
		wantV, wantP, wantN := cmosReference(input)
		r := state.result
		mp, mn := r.MOSFETs[0], r.MOSFETs[1]
		// The scalar reference omits the pinned body's <=1 nA leakage.
		if math.Abs(mn.VDS-wantV) > 1e-5 || math.Abs(mp.DrainCurrent-wantP) > 1e-8 || math.Abs(mn.DrainCurrent-wantN) > 1e-8 || mp.GateCurrent != 0 || mn.GateCurrent != 0 || r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 {
			t.Fatal("CMOS load-line/current/conservation", input, r, wantV, wantP, wantN)
		}
		if input == 2.5 && (mp.DrainCurrent > -.05 || mn.DrainCurrent < .05) {
			t.Fatal("intermediate gate drive must expose simultaneous conduction")
		}
	}
}

func TestCMOSGuidedPulseCaptureBothEdgeSidesAndNoLiveMutation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "cmos"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	initial := settled(t, s)
	probes := []string{"v:VIN.positive", "v:MN", "i:MP.drain", "i:MN.drain"}
	if initial.Selected != "MN" || initial.Example.Modified || !reflect.DeepEqual(initial.Scope.Selected, probes) || initial.Mixed.MOSFETs[1].VDS > 1e-6 {
		t.Fatal("inverter did not open with useful default inspection")
	}
	duration, sampleStep := .51, .005
	_, captured := verify(t, s, VerifyParams{Duration: &duration, Step: &sampleStep, ProbeIDs: probes}, "capture")
	if captured.Metadata.PMOSModel == nil || captured.Metadata.NMOSModel == nil || !reflect.DeepEqual(initial.Mixed, s.Snapshot().Mixed) || !reflect.DeepEqual(initial.Transient, s.Snapshot().Transient) {
		t.Fatal("Verify lost either model or changed accepted live physics")
	}
	edges := map[float64]map[float64]bool{.2: {}, .5: {}}
	page := call(t, s, "operation.samples", VerificationPageParams{ID: captured.Operation.ID, Limit: 200}, "samples").Result.(VerificationPage)
	if page.NextOffset != page.Total {
		t.Fatal("incomplete inverter capture")
	}
	for _, row := range page.Rows {
		wantV, wantP, wantN := cmosReference(row.Values[0])
		if math.Abs(row.Values[1]-wantV) > 1e-5 || math.Abs(row.Values[2]-wantP) > 1e-8 || math.Abs(row.Values[3]-wantN) > 1e-8 {
			t.Fatal("captured inverter row differs from independent reference", row)
		}
		for at, sides := range edges {
			if math.Abs(row.Time-at) < 1e-12 {
				sides[row.Values[0]] = true
			}
		}
	}
	for at, sides := range edges {
		if !sides[0] || !sides[5] {
			t.Fatal("capture omitted one ideal-edge side", at, sides)
		}
	}
	for _, step := range []float64{.2, .3} {
		call(t, s, "simulation.step", StepParams{Duration: step}, FormatValue(step, "s"))
		state := s.Snapshot()
		want, _, _ := cmosReference(state.Transient.Sample.Input)
		if math.Abs(state.Mixed.MOSFETs[1].VDS-want) > 1e-5 {
			t.Fatal("live inverter did not switch at accepted edge", state.Mixed)
		}
	}
}
