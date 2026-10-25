package workbench

import (
	"math"
	"testing"
)

func TestDynamicOpAmpOutputStagePassivityAndRailClampJacobian(t *testing.T) {
	for _, pole := range []float64{-10, -4, 0, 4, 20} {
		for _, out := range []float64{-8, -3, 0, 2, 8} {
			v := opAmpVoltages{.01, 0, out, 2, -3}
			e := opAmpCurrentsControlled(v, pole, true)
			power, sum := 0., 0.
			for k, i := range e.current {
				sum += i
				power += (v[k] - v[4]) * i
			}
			if power < -1e-12 || math.Abs(sum) > 1e-12 || e.target < v[4] || e.target > v[3] {
				t.Fatal("nonpassive/clipped rail stage", pole, out, e.target, power, sum)
			}
			for col := 0; col < 6; col++ {
				a, b := v, v
				pa, pb := pole, pole
				h := 1e-7
				if col < 5 {
					a[col] += h
					b[col] -= h
				} else {
					pa += h
					pb -= h
				}
				ea, eb := opAmpCurrentsControlled(a, pa, true), opAmpCurrentsControlled(b, pb, true)
				for row := range e.current {
					want := (ea.current[row] - eb.current[row]) / (2 * h)
					got := e.poleDerivative[row]
					if col < 5 {
						got = e.jacobian[row][col]
					}
					if math.Abs(want-got) > 1e-8 {
						t.Fatal("clamped derivative", pole, out, row, col, want, got)
					}
				}
			}
			translated := v
			for k := range translated {
				translated[k] += 1.25
			}
			shifted := opAmpCurrentsControlled(translated, pole+1.25, true)
			for k, i := range e.current {
				if math.Abs(i-shifted.current[k]) > 1e-12 {
					t.Fatal("local ground changed output law")
				}
			}
		}
	}
}
func TestDynamicOpAmpSupplyDroopPreservesChargeButClampsOutput(t *testing.T) {
	g := OpAmpDynamicsExampleGraph(false)
	for k, p := range g.Components {
		switch p.ID {
		case "VIN":
			v, _ := MakeComponent(p.ID, "dc-source", p.Position, "1")
			v.Rotation = p.Rotation
			g.Components[k] = v
		case "VP":
			v, _ := MakeComponent(p.ID, "finite-pulse-source", p.Position, "5")
			v.Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: .2, HighSeconds: .02, LowSeconds: .01, RiseSeconds: .0001, FallSeconds: .0001}
			g.Components[k] = v
		}
	}
	c, status, d := g.Compile()
	if status != "" || d != nil {
		t.Fatal(status, d)
	}
	k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	held := false
	for j := 1; j <= 1800; j++ {
		at := float64(j) * .00002
		if err := k.advance(t.Context(), at); err != nil {
			t.Fatal(j, err)
		}
		r := k.mixed.result
		vp := r.Nodes[c.TerminalNet["U1.vplus"]]
		vm := r.Nodes[c.TerminalNet["U1.vminus"]]
		output := r.OpAmps[0].OutputVoltage
		if output > vp+1e-8 || output < vm-1e-8 || r.OpAmps[0].Power < -1e-9 || math.Abs(r.PowerResidual) > 1e-6 {
			t.Fatal("supply droop supplied fictitious output energy", at, vp, vm, output, r.PowerResidual)
		}
		if at > .021 && at < .029 {
			held = held || r.Dynamics[0].Voltage+vm > vp+.5
		}
	}
	if !held || math.Abs(k.mixed.result.OpAmps[0].OutputVoltage-1) > .01 {
		t.Fatal("charge did not persist/recover", held, k.mixed.result.OpAmps)
	}
}
