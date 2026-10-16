package workbench

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func highRailLEDGraph(volts float64) Graph {
	g := linearStackGraph(volts)
	led, _ := MakeComponent("D1", "led", Point{X: 400}, "")
	g.Components = append(g.Components, led)
	g.Wires[1].From = "D1.cathode"
	g.Wires = append(g.Wires, Wire{"W7", "R1.2", "D1.anode", true})
	return g
}

func TestHighRailLEDColdStartMatchesIndependentLoadLine(t *testing.T) {
	for _, volts := range []float64{24, 30} {
		g := highRailLEDGraph(volts)
		state, _ := solveFloatingDC(t, g)
		// Independent inverse Shockley load line, using the published generic
		// red LED constants without calling its current/drop/Jacobian helpers.
		lo, hi := 0., volts/1000
		for k := 0; k < 100; k++ {
			i := (lo + hi) / 2
			drop := 2*1.380649e-23*300.15/1.602176634e-19*math.Log1p(i/1e-20) + i
			if 1000*i+drop < volts {
				lo = i
			} else {
				hi = i
			}
		}
		want := (lo + hi) / 2
		got := state.result
		if math.Abs(got.LEDs[0].Current-want) > 1e-10 || math.Abs(got.Resistors[0].Current-want) > 1e-10 || got.KCLResidual > 1e-10 || math.Abs(got.PowerResidual) > 1e-8 {
			t.Fatal("high-rail LED violates independent load line/conservation", volts, got, want)
		}
	}
}

func TestConstraintSeedUsesOnlyGroundedIdealPaths(t *testing.T) {
	g := highRailLEDGraph(24)
	cap, _ := MakeComponent("C1", "capacitor", Point{X: 500}, "1uF")
	g.Components = append(g.Components, cap)
	g.Wires = append(g.Wires, Wire{"W8", "C1.1", "D1.anode", true}, Wire{"W9", "C1.2", "GND1.reference", true})
	c, status, d := g.Compile()
	if d != nil {
		t.Fatal(status, d)
	}
	n := newMixedNetwork(c)
	tr := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	for _, dt := range []float64{0, .001} {
		seed := n.initialConstraintGuess([]float64{1.25}, dt, tr, 0)
		for pin, want := range map[string]float64{"V1.positive": 24, "V2.positive": 12, "D1.anode": 0} {
			if pin == "D1.anode" && dt == 0 {
				want = 1.25
			}
			if got := voltage(seed, n.index[c.TerminalNet[pin]]); got != want {
				t.Fatal("seed violated an ideal constraint or guessed a finite-branch voltage", pin, dt, got, want)
			}
		}
		for _, current := range seed[len(n.nodes):] {
			if current != 0 {
				t.Fatal("seed invented a branch-current solution", current)
			}
		}
	}
	// The LED anode is the common reference; neither end of V1 has an ideal
	// path to it. A zero guess must not be mistaken for a known source anchor.
	g = ExampleGraph()
	g.Wires[2].From, g.Wires[2].To = "D1.cathode", "V1.negative"
	g.Wires[3].From, g.Wires[3].To = "GND.reference", "D1.anode"
	c, status, d = g.Compile()
	if d != nil {
		t.Fatal(status, d)
	}
	for _, v := range newMixedNetwork(c).initialConstraintGuess(nil, 0, tr, 0) {
		if v != 0 {
			t.Fatal("unanchored ideal source received an invented absolute voltage", v)
		}
	}
}

func floatingGatePairGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct{ id, kind, value string }{
		{"V1", "dc-source", "0V"}, {"RL", "resistor", "1k"}, {"RG", "resistor", "1k"},
		{"M1", "nmos", ""}, {"M2", "nmos", ""}, {"GND1", "ground", ""},
	} {
		c, _ := MakeComponent(p.id, p.kind, Point{}, p.value)
		g.Components = append(g.Components, c)
	}
	for i, pair := range [][2]string{
		{"V1.positive", "RL.1"}, {"RL.2", "M1.drain"}, {"M1.drain", "M2.drain"},
		{"M1.source", "GND1.reference"}, {"M2.source", "GND1.reference"}, {"V1.negative", "GND1.reference"},
		{"M1.gate", "RG.1"}, {"M2.gate", "RG.2"},
	} {
		g.Wires = append(g.Wires, Wire{fmt.Sprintf("W%d", i+1), pair[0], pair[1], true})
	}
	return g
}

func TestZeroExcitationFloatingMOSGatesMustNotPublishArbitraryZero(t *testing.T) {
	c, status, d := floatingGatePairGraph().Compile()
	if d != nil {
		t.Fatal(status, d)
	}
	n := newMixedNetwork(c)
	_, err := n.solve(nil, nil, 0, &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}, 0)
	if err == nil || !strings.Contains(err.Error(), "singular") {
		t.Fatal("floating insulated gates at zero excitation were accepted", err)
	}
}

func TestOrdinaryPMOSStartAndWarmGuessRetainTheirBasin(t *testing.T) {
	tr := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	for _, gate := range []float64{0, 2.5, 5} {
		c, status, d := pmosDCGraph(gate).Compile()
		if d != nil {
			t.Fatal(status, d)
		}
		n := newMixedNetwork(c)
		ordinary, err := n.newton(t.Context(), nil, nil, 0, tr, 0)
		if err != nil {
			t.Fatal(err)
		}
		got, err := n.solve(nil, nil, 0, tr, 0)
		if err != nil || !reflect.DeepEqual(got.result, ordinary.result) {
			t.Fatal("a successful ordinary start was replaced", gate, err)
		}
		warm, err := n.newton(t.Context(), got.x, nil, 0, tr, 0)
		if err != nil {
			t.Fatal(err)
		}
		continued, err := n.solve(got.x, nil, 0, tr, 0)
		if err != nil || !reflect.DeepEqual(continued.result, warm.result) {
			t.Fatal("accepted warm guess was replaced", gate, err)
		}
	}
	c, _, _ := highRailLEDGraph(31).Compile()
	if _, err := newMixedNetwork(c).solve(nil, nil, 0, tr, 0); err == nil || !strings.Contains(err.Error(), "±30 V") {
		t.Fatal("constraint seed bypassed the node limit", err)
	}
}

func TestConstraintSeedUsesTheActualStimulusSideAtAPulseEdge(t *testing.T) {
	c, status, d := NMOSExampleGraph().Compile()
	if d != nil {
		t.Fatal(status, d)
	}
	n := newMixedNetwork(c)
	tr := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	for _, dt := range []float64{0, .001} {
		seed := n.initialConstraintGuess(nil, dt, tr, .05)
		want := 0.
		if dt > 0 {
			want = 5 // integration uses the interval's left-hand square phase
		}
		if seed[n.index[c.TerminalNet["VIN.positive"]]] != want || seed[n.index[c.TerminalNet["VCC.positive"]]] != 5 {
			t.Fatal("seed used a different source definition/time side from MNA", dt, seed)
		}
	}
}
