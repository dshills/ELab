package workbench

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func inductorNPNCapGraph(t *testing.T) Graph {
	t.Helper()
	g := FlybackExampleGraph()
	for k, part := range g.Components {
		switch part.ID {
		case "M1":
			g.Components[k], _ = MakeComponent("Q1", "npn", part.Position, "")
		case "RG":
			g.Components[k], _ = SetValue(part, "10k Ω")
		}
	}
	references := strings.NewReplacer("M1.gate", "Q1.base", "M1.drain", "Q1.collector", "M1.source", "Q1.emitter")
	for k := range g.Wires {
		g.Wires[k].From = references.Replace(g.Wires[k].From)
		g.Wires[k].To = references.Replace(g.Wires[k].To)
	}
	cap, _ := MakeComponent("CB", "capacitor", Point{X: 700, Y: 200}, "1 µF")
	g.Components = append(g.Components, cap)
	g.Wires = append(g.Wires, Wire{ID: "WCB1", From: "CB.1", To: "Q1.base", Connected: true}, Wire{ID: "WCB2", From: "CB.2", To: "GND1.reference", Connected: true})
	return g
}

func TestInductorCoupledNPNGateCapacitorStepRefinement(t *testing.T) {
	if productionNumericalReference(t) {
		return
	}
	for _, resistance := range []float64{100, 120} {
		t.Run(fmt.Sprint(resistance), func(t *testing.T) {
			g := inductorNPNCapGraph(t)
			for k := range g.Components {
				if g.Components[k].ID == "RL" {
					g.Components[k].Value = resistance
				}
			}
			checkInductorNPNRefinement(t, g)
		})
	}
}

func checkInductorNPNRefinement(t *testing.T, g Graph) {
	t.Helper()
	c, _, diagnostic := g.Compile()
	if diagnostic != nil {
		t.Fatal(diagnostic)
	}
	kernels := make([]*simulationKernel, 3)
	for k := range kernels {
		var err error
		kernels[k], err = newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
		if err != nil {
			t.Fatal("mixed RLC/NPN initialization", err)
		}
	}
	for sample := 1; sample <= 200; sample++ {
		for k, kernel := range kernels {
			substeps := 1 << k
			for sub := 1; sub <= substeps; sub++ {
				end := (float64(sample-1) + float64(sub)/float64(substeps)) * .001
				if err := kernel.advance(t.Context(), end); err != nil {
					t.Fatal("mixed RLC/NPN refinement", k, sample, sub, err)
				}
			}
			r := kernel.mixed.result
			if r.KCLResidual > 1e-9 || math.Abs(r.PowerResidual) > 1e-8 || !finite(r.Inductors[0].Energy+r.Capacitors[0].Energy) {
				t.Fatal("mixed storage conservation", k, sample, r)
			}
		}
		ref := kernels[2].mixed.result
		for k := 0; k < 2; k++ {
			r := kernels[k].mixed.result
			if math.Abs(r.Inductors[0].Current-ref.Inductors[0].Current) > 2e-5 || math.Abs(r.Capacitors[0].Voltage-ref.Capacitors[0].Voltage) > .002 || math.Abs(r.Transistors[0].VCE-ref.Transistors[0].VCE) > .002 {
				t.Fatal("mixed output does not agree with half/quarter-step runs", k, sample, r, ref)
			}
		}
	}
}

func TestInductorNPNLoadedSwitchingWorkEnvelope(t *testing.T) {
	for _, resistance := range []float64{100, 120} {
		t.Run(fmt.Sprint(resistance), func(t *testing.T) {
			g := inductorNPNCapGraph(t)
			for i := range g.Components {
				if g.Components[i].ID == "RL" {
					g.Components[i].Value = resistance
				}
			}
			state, trace := initializeInductorTest(t, g)
			peakTime, peakAttempts, peakNewton := time.Duration(0), 0, 0
			retries := 0
			for sample := 1; sample <= 200; sample++ {
				budget := newIntegrationBudget()
				// Keep the production attempt bound under race instrumentation.
				// Wall deadlines belong to ordinary public performance runs;
				// instrumented execution is not workstation latency evidence.
				budget.elapsed = func() time.Duration { return 0 }
				next, nextTrace, err := advanceMixedBudget(state, trace, float64(sample)*.001, budget)
				if err != nil {
					t.Fatalf("load %.0f Ω, sample %d, attempts %d, Newton %d: %v", resistance, sample, budget.attempts, budget.newtonIterations, err)
				}
				state, trace = next, nextTrace
				peakTime = max(peakTime, time.Since(budget.started))
				peakAttempts, peakNewton = max(peakAttempts, budget.attempts), max(peakNewton, budget.newtonIterations)
				retries += budget.roundoffRetries
			}
			if resistance == 120 && retries == 0 {
				t.Fatal("fixture did not exercise the checked roundoff retry")
			}
			t.Logf("200 ms, 1 ms outputs: peak %s, %d attempts, %d Newton iterations; %d checked roundoff retries", peakTime, peakAttempts, peakNewton, retries)
		})
	}
}

// Independent reduced reference in the forward-active / diode-clamped
// regions. These pin the published v1 constants, not production device calls.
// Reverse B-C transport is negligible in the chosen early/late intervals.
const npnCoilVT = 1.380649e-23 * 300.15 / 1.602176634e-19

func npnCoilForward(base float64) float64 { return 1e-14 * math.Expm1(base/npnCoilVT) }
func npnCoilBaseRate(base, input float64) float64 {
	return ((input-base)/10000 - base/100000 - npnCoilForward(base)/100 + 1e-14) / 1e-6
}

func npnCoilEarlyReference(end float64) (base, current, collector float64) {
	for at := 0.; at < end; {
		h := math.Min(1e-7, end-at)
		a := npnCoilBaseRate(base, 5)
		b := npnCoilBaseRate(base+h*a/2, 5)
		c := npnCoilBaseRate(base+h*b/2, 5)
		d := npnCoilBaseRate(base+h*c, 5)
		base += h * (a + 2*b + 2*c + d) / 6
		at += h
	}
	current = npnCoilForward(base) + 2e-14 - 1e-9
	derivative := 1e-14 * math.Exp(base/npnCoilVT) / npnCoilVT * npnCoilBaseRate(base, 5)
	collector = 5 - 100*current - derivative // L=1 H.
	return
}

func npnCoilSteadyReference() (base, current float64) {
	lo, hi := 0., 1.
	for k := 0; k < 80; k++ {
		mid := (lo + hi) / 2
		if npnCoilBaseRate(mid, 5) > 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	base = (lo + hi) / 2
	return base, npnCoilForward(base) + 2e-14 - 1e-9
}

func npnCoilOffReference(end float64) (base, current, collector float64) {
	base, current = npnCoilSteadyReference()
	// Skip the first 1 ps after the ideal edge. Even the full ±30 V
	// envelope can change 1 H current by at most 30 pA in that interval.
	at := 1e-12
	base += at * npnCoilBaseRate(base, 0)
	rate := func(v, i float64) (float64, float64) {
		diode := i - npnCoilForward(v) - 2e-14
		drop := 1.7*npnCoilVT*math.Log1p(diode/1e-9) + diode
		return npnCoilBaseRate(v, 0), -100*i - drop
	}
	for at < end {
		h := 1e-7
		if at < 1e-9 {
			h = 1e-10
		} else if at < 1e-6 {
			h = 1e-8
		}
		h = math.Min(h, end-at)
		av, ai := rate(base, current)
		bv, bi := rate(base+h*av/2, current+h*ai/2)
		cv, ci := rate(base+h*bv/2, current+h*bi/2)
		dv, di := rate(base+h*cv, current+h*ci)
		base += h * (av + 2*bv + 2*cv + dv) / 6
		current += h * (ai + 2*bi + 2*ci + di) / 6
		at += h
	}
	diode := current - npnCoilForward(base) - 2e-14
	collector = 5 + 1.7*npnCoilVT*math.Log1p(diode/1e-9) + diode
	return
}

func TestInductorNPNIndependentActiveAndFlybackReference(t *testing.T) {
	if productionNumericalReference(t) {
		return
	}
	g := inductorNPNCapGraph(t)
	c, _, _ := g.Compile()
	kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	advance := func(end float64) {
		t.Helper()
		for kernel.mixed.result.Time < end {
			stop := math.Min(kernel.mixed.result.Time+.001, end)
			if err := kernel.advance(t.Context(), stop); err != nil {
				t.Fatal(end, err)
			}
		}
	}
	check := func(base, current, collector float64) {
		t.Helper()
		r := kernel.mixed.result
		if !finite(base) || !finite(current) || !finite(collector) || math.Abs(r.Capacitors[0].Voltage-base) > 2e-5 || math.Abs(r.Inductors[0].Current-current) > 5e-7 || math.Abs(r.Transistors[0].VCE-collector) > .001 {
			t.Fatal("independent reduced NPN/RLC reference", r.Time, r.Capacitors[0], r.Inductors[0], r.Transistors[0], base, current, collector)
		}
	}
	for _, at := range []float64{.001, .0011, .0012, .001273, .0013} {
		advance(at)
		check(npnCoilEarlyReference(at))
	}
	advance(.049)
	base, current := npnCoilSteadyReference()
	check(base, current, 5-100*current)
	advance(.05)
	check(base, current, 5-100*current)
	for _, off := range []float64{1e-6, 1e-5, .0001, .001, .005} {
		advance(.05 + off)
		check(npnCoilOffReference(off))
	}
}
