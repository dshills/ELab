package workbench

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
)

func sweepForTest(t *testing.T, g Graph, cfg SweepSettings) *SweepResult {
	t.Helper()
	plan, err := newSweepPlan(g, cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := computeDCSweep(context.Background(), plan, func(int) {})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range result.Points {
		if p.KCLResidual > 1e-9 || math.Abs(p.PowerResidual) > 1e-8 {
			t.Fatalf("DC conservation: %+v", p)
		}
	}
	return result
}
func TestDCSweepLEDAndNPNMatchExistingIndependentKernels(t *testing.T) {
	for _, g := range []Graph{ExampleGraph(), NPNExampleGraph()} {
		component, probe := "V1", "i:D1"
		if len(g.Components) > 5 {
			component = "VB"
			probe = "i:Q1.collector"
		}
		// The template source identity is derived from the graph, rather than an
		// assumed transistor input alias.
		if g.Components[0].Kind == "npn" {
			t.Fatal("unexpected fixture")
		}
		compiled, _, d := g.Compile()
		if d != nil {
			t.Fatal(d)
		}
		if compiled.Analysis == NPNAnalysis {
			component = compiled.Input.ID
			probe = "i:" + compiled.Transistor.ID + ".collector"
		}
		result := sweepForTest(t, g, SweepSettings{ComponentID: component, Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 11, ProbeIDs: []string{probe}})
		for _, point := range result.Points {
			input := g.Clone()
			for i := range input.Components {
				if input.Components[i].ID == component {
					input.Components[i].Value = point.Input
				}
			}
			c, _, d := input.Compile()
			if d != nil {
				t.Fatal(d)
			}
			var want float64
			if c.Analysis == NPNAnalysis {
				r, err := SolveNPN(context.Background(), c)
				if err != nil {
					t.Fatal(err)
				}
				want = r.CollectorCurrent
			} else {
				r, err := SolveLED(context.Background(), c.Circuit)
				if err != nil {
					t.Fatal(err)
				}
				want = r.Current
			}
			if math.Abs(point.Values[0]-want) > 2e-9 {
				t.Fatalf("%s at %g: %.12g != %.12g", component, point.Input, point.Values[0], want)
			}
		}
	}
}
func TestDCSweepLoadedWiperAndDescendingResistance(t *testing.T) {
	g := potGraph(t, "resistor")
	result := sweepForTest(t, g, SweepSettings{ComponentID: "P1", Parameter: "wiper", Start: "0", Stop: "1", Points: 5, ProbeIDs: []string{"v:P1.wiper"}})
	for _, p := range result.Points {
		upper, lower := 1+9998*(1-p.Input), 1+9998*p.Input
		parallel := 1 / (1/lower + 1./10000)
		want := 5 * parallel / (upper + parallel)
		if math.Abs(p.Values[0]-want) > 1e-10 {
			t.Fatalf("wiper %g: %g != %g", p.Input, p.Values[0], want)
		}
	}
	result = sweepForTest(t, g, SweepSettings{ComponentID: "R1", Parameter: "resistance", Start: "10kΩ", Stop: "100 ohm", Points: 7, ProbeIDs: []string{"v:P1.wiper"}})
	if result.Points[0].Input != 10000 || result.Points[6].Input != 100 {
		t.Fatal("endpoints drifted")
	}
	for _, p := range result.Points {
		parallel := 1 / (1./5000 + 1/p.Input)
		want := 5 * parallel / (5000 + parallel)
		if math.Abs(p.Values[0]-want) > 1e-10 {
			t.Fatal("loaded divider mismatch")
		}
	}
}
func TestDCOperatingPointOpensCapacitorsAndUsesWaveformBias(t *testing.T) {
	g := potGraph(t, "capacitor")
	for i := range g.Components {
		if g.Components[i].Kind == "capacitor" {
			v := -9.
			g.Components[i].InitialVoltage = &v
		}
	}
	before := g.Clone()
	r, _, err := solveDCOperatingPoint(context.Background(), g, dcSourceLevels(g))
	if err != nil || len(r.Capacitors) != 1 || r.Capacitors[0].Current != 0 || math.Abs(r.Capacitors[0].Voltage-2.5) > 1e-12 {
		t.Fatal("capacitor startup was used as DC", r, err)
	}
	if !reflect.DeepEqual(g, before) {
		t.Fatal("DC calculation mutated input graph")
	}
	g = RectifierExampleGraph()
	for i := range g.Components {
		if g.Components[i].Kind == "sine-source" {
			g.Components[i].Sine.Offset = -1
			g.Components[i].Sine.Phase = 90
		}
	}
	levels := dcSourceLevels(g)
	if len(levels) != 1 || levels[0].DCLevel != -1 {
		t.Fatal("sine bias used amplitude/phase")
	}
	r, _, err = solveDCOperatingPoint(context.Background(), g, levels)
	if err != nil || math.Abs(r.Sources[0].Voltage+1) > 1e-12 || r.Capacitors[0].Current != 0 {
		t.Fatal("sine DC bias failed", r, err)
	}
	// A legacy square contributes its zero low bias, not its configured high value.
	g = RCExampleGraph()
	levels = dcSourceLevels(g)
	if levels[0].DCLevel != 0 {
		t.Fatal("ideal pulse DC baseline")
	}
	r, _, err = solveDCOperatingPoint(context.Background(), g, levels)
	if err != nil || r.Capacitors[0].Voltage != 0 || r.Capacitors[0].Current != 0 {
		t.Fatal("RC DC baseline", r, err)
	}
}
func TestDCSweepCancelReturnsNoPartialResultAndOwnsInput(t *testing.T) {
	g := ExampleGraph()
	settings := SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0", Stop: "5", Points: 11, ProbeIDs: []string{"i:D1"}}
	plan, err := newSweepPlan(g, settings)
	if err != nil {
		t.Fatal(err)
	}
	settings.ProbeIDs[0] = "removed"
	g.Components[0].Value = 12
	if plan.probes[0].ID != "i:D1" || plan.graph.Components[0].Value == 12 {
		t.Fatal("plan aliases input")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := computeDCSweep(ctx, plan, func(done int) {
		if done == 3 {
			cancel()
		}
	})
	if result != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("canceled sweep exposed partial points", result, err)
	}
	for _, s := range []SweepSettings{
		{ComponentID: "V1", Parameter: "voltage", Start: "NaN", Stop: "5", Points: 3},
		{ComponentID: "V1", Parameter: "voltage", Start: "-13V", Stop: "5", Points: 3},
		{ComponentID: "V1", Parameter: "voltage", Start: "0", Stop: "5", Points: 202},
		{ComponentID: "V1", Parameter: "voltage", Start: "5", Stop: "5", Points: 3},
		{ComponentID: "R1", Parameter: "resistance", Start: "1", Stop: "1k", Points: 3},
	} {
		if _, err := newSweepPlan(ExampleGraph(), s); err == nil {
			t.Fatalf("accepted invalid sweep: %+v", s)
		}
	}
}
