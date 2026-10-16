package workbench

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// Build explicit wires and a real ground symbol from the internal fixture's
// intended nets. Unlike the foundation tests, these exercise Graph.Compile's
// actual connectivity derivation, reference checks and analysis selection.
func (f *opAmpNetworkFixture) wiredGraph(t *testing.T) Graph {
	t.Helper()
	g := f.graph.Clone()
	ground, err := MakeComponent("GND", "ground", Point{}, "")
	if err != nil {
		t.Fatal(err)
	}
	g.Components = append(g.Components, ground)
	for k := range g.Components {
		g.Components[k].Position = Point{X: float64(k) * 150}
	}
	groups := map[string][]string{"0": {"GND.reference"}}
	for _, part := range f.graph.Components {
		for _, pin := range part.Terminals {
			ref := part.ID + "." + pin
			net, ok := f.pins[ref]
			if !ok {
				t.Fatal("fixture omitted terminal", ref)
			}
			groups[net] = append(groups[net], ref)
		}
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		members := groups[name]
		sort.Strings(members)
		for _, other := range members[1:] {
			g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", len(g.Wires)+1), From: members[0], To: other, Connected: true})
		}
	}
	return g
}

func (f *opAmpNetworkFixture) remove(id string) {
	for k, part := range f.graph.Components {
		if part.ID == id {
			f.graph.Components = append(f.graph.Components[:k], f.graph.Components[k+1:]...)
			return
		}
	}
}

func TestOpAmpCompilerFeedbackNetworksAgainstFrozenReferences(t *testing.T) {
	for _, reference := range opAmpReferences(t).Amplifiers {
		g := opAmpAmplifierFixture(t, reference.Kind, reference.Vin, reference.High, reference.Low, reference.Load).wiredGraph(t)
		c, status, diagnostic := g.Compile()
		if diagnostic != nil || status != "" || c.Analysis != NetworkDCAnalysis || c.Output != c.TerminalNet["U1.out"] {
			t.Fatal("op-amp graph failed compilation or output/analysis selection", reference, status, diagnostic)
		}
		k, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: DefaultTransientSettings(), mode: "auto"})
		if err != nil {
			t.Fatal("compiled amplifier kernel", err)
		}
		checkOpAmpNetworkBalance(t, k.mixed)
		if math.Abs(k.mixed.result.Nodes[c.Output]-reference.Output) > 1e-7 {
			t.Fatal("compiled amplifier differs from frozen reference", reference, k.mixed.result.Nodes[c.Output])
		}
	}
}

func TestOpAmpCompilerRefusesUnreferencedInputGroups(t *testing.T) {
	for _, tied := range []bool{false, true} {
		f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
		f.pins["U1.noninverting"] = "floating-plus"
		f.pins["U1.inverting"] = "floating-minus"
		if tied {
			f.pins["U1.inverting"] = "floating-plus"
		} else {
			f.add(t, "RFLOAT", "resistor", "1k", "floating-plus", "floating-minus")
		}
		f.remove("V1")
		_, status, d := f.wiredGraph(t).Compile()
		if status != "incomplete" || d == nil || d.Code != "floating-island" {
			t.Fatal("control/supply derivatives invented an input common-mode reference", tied, status, d)
		}
	}
}

func TestOpAmpCompilerDoesNotUseMOSGateAsReference(t *testing.T) {
	for _, kind := range []string{"nmos", "pmos"} {
		for _, supply := range []bool{false, true} {
			f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
			if supply {
				f.remove("VP")
				f.pins["U1.vplus"] = "floating"
			} else {
				f.remove("V1")
				f.pins["U1.noninverting"] = "floating"
				f.pins["U1.inverting"] = "floating"
			}
			f.add(t, "M1", kind, "", "floating", "output", "0")
			_, status, d := f.wiredGraph(t).Compile()
			if status != "incomplete" || d == nil || d.Code != "floating-island" {
				t.Fatal("ideal MOS gate created an op-amp reference", kind, supply, status, d)
			}
		}
	}
}

func TestOpAmpCompilerEqualRailsAndUnloadedOutput(t *testing.T) {
	f := newOpAmpNetworkFixture()
	f.add(t, "V1", "dc-source", "5", "rail", "0")
	f.add(t, "U1", "opamp", "", "0", "0", "output", "rail", "rail")
	c, status, d := f.wiredGraph(t).Compile()
	if status != "" || d != nil {
		t.Fatal("equal power rails or unloaded output incorrectly refused", status, d)
	}
	s, err := newMixedNetwork(c).solve(nil, nil, 0, &Transient{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	checkOpAmpNetworkBalance(t, s)
	if s.result.OpAmps[0].State != "unpowered" || math.Abs(s.result.Nodes[c.Output]-5) > 1e-9 {
		t.Fatal("unloaded equal-rail output did not follow the declared bleed", s.result)
	}
}

func TestOpAmpCompilerCapacityAndOrdinarySourceConstraints(t *testing.T) {
	f := opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
	f.add(t, "U2", "opamp", "", "output", "second", "second", "positive", "negative")
	if _, status, d := f.wiredGraph(t).Compile(); status != "" || d != nil {
		t.Fatal("two-device graph refused", status, d)
	}
	f.add(t, "U3", "opamp", "", "second", "third", "third", "positive", "negative")
	if _, status, d := f.wiredGraph(t).Compile(); status != "unsupported" || d == nil || d.Code != "mixed-limit" {
		t.Fatal("unbounded op-amp expansion", status, d)
	}
	f = opAmpAmplifierFixture(t, "follower", 1, 5, -5, 1000)
	f.add(t, "C1", "capacitor", "1uF", "input", "0")
	if _, status, d := f.wiredGraph(t).Compile(); status != "unsupported" || d == nil || d.Code != "ideal-source-capacitor" {
		t.Fatal("new part bypassed existing ideal jump refusal", status, d)
	}
}
