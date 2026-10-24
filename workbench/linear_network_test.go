package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
)

func linearStackGraph(total float64) Graph {
	g := bottomReferencedStack()
	if total <= 24 {
		g.Components[0].Value, g.Components[1].Value = total/2, total/2
		return g
	}
	g.Components[0].Value, g.Components[1].Value = 10, 10
	v, _ := MakeComponent("V3", "dc-source", Point{X: -400}, FormatValue(total-20, "V"))
	g.Components = append(g.Components, v)
	g.Wires[0].To = "V3.negative"
	g.Wires = append(g.Wires, Wire{"W6", "V3.positive", "R1.1", true})
	return g
}

func TestLinearStacksReachDeclaredVoltageEnvelope(t *testing.T) {
	for _, volts := range []float64{0, 18, 24, 30} {
		g := linearStackGraph(volts)
		state, c := solveFloatingDC(t, g)
		if math.Abs(state.result.Nodes[c.TerminalNet["R1.1"]]-volts) > 1e-10 || math.Abs(state.result.Resistors[0].Current-volts/1000) > 1e-12 || math.Abs(state.result.Resistors[0].Power-volts*volts/1000) > 1e-10 || state.result.KCLResidual > 1e-10 || math.Abs(state.result.PowerResidual) > 1e-10 {
			t.Fatal("linear stack violates independent Ohm/Kirchhoff reference", volts, state.result)
		}
	}
}

func TestLinearStackStillRefusesOverRange(t *testing.T) {
	g := linearStackGraph(31)
	c, status, d := g.Compile()
	if d != nil {
		t.Fatalf("%s %+v", status, d)
	}
	_, err := newMixedNetwork(c).solve(nil, nil, 0, &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}, 0)
	if err == nil || !strings.Contains(err.Error(), "±30 V") {
		t.Fatal("out-of-range stack did not produce its range error", err)
	}
}

func TestLinearNegativeRailAndCapacitorStep(t *testing.T) {
	g := linearStackGraph(24)
	g.Wires[4].From = "V1.positive"
	state, c := solveFloatingDC(t, g)
	if math.Abs(state.result.Nodes[c.TerminalNet["V2.negative"]]+24) > 1e-12 || math.Abs(state.result.Resistors[0].Current-.024) > 1e-12 {
		t.Fatal("negative 24V rail changed current/reference", state.result)
	}
	g = linearStackGraph(24)
	g.Components[2].Value = 10000
	cap, _ := MakeComponent("C1", "capacitor", Point{X: 400}, "1uF")
	g.Components = append(g.Components, cap)
	g.Wires[1].From = "C1.2"
	g.Wires = append(g.Wires, Wire{"W6", "R1.2", "C1.1", true})
	state, _ = solveFloatingDC(t, g)
	if state.result.Capacitors[0].Voltage != 0 {
		t.Fatal("initial charge was not respected")
	}
	const dt = .001
	state, err := state.net.solve(state.x, state.capV, dt, &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}, dt)
	if err != nil {
		t.Fatal(err)
	}
	// Independent one-step backward-Euler capacitor law: C*v/dt=(24-v)/R.
	want := 24 * dt / (.01 + dt)
	if math.Abs(state.result.Capacitors[0].Voltage-want) > 1e-11 || math.Abs(state.result.Resistors[0].Current-(24-want)/10000) > 1e-12 || math.Abs(state.result.PowerResidual) > 1e-10 {
		t.Fatal("linear high-voltage capacitor stamp differs from BE law", state.result, want)
	}
}

func TestNetworkLinearityIsConservativeForEveryPart(t *testing.T) {
	nonlinear := map[string]bool{"led": true, "diode": true, "zener": true, "schottky": true, "npn": true, "pnp": true, "nmos": true, "pmos": true, "opamp": true, "timer555": true, "logic-not": true, "logic-and": true, "logic-or": true, "logic-nand": true, "logic-nor": true, "logic-xor": true, "logic-xnor": true}
	for _, part := range Library() {
		n := newMixedNetwork(Compiled{MixedGraph: Graph{Components: []Component{{Kind: part.Kind}}}})
		if n.nonlinear != nonlinear[part.Kind] {
			t.Fatal("incorrect model linearity", part.Kind)
		}
	}
	if !newMixedNetwork(Compiled{MixedGraph: Graph{Components: []Component{{Kind: "future-device"}}}}).nonlinear {
		t.Fatal("unknown future device silently selected full linear corrections")
	}
}

func TestLinearZeroExcitationStillChecksRankAndCancellation(t *testing.T) {
	resistor, _ := MakeComponent("R1", "resistor", Point{}, "1k")
	// Deliberately bypass compilation to check the kernel's defensive rank gate:
	// this floating resistor has infinitely many zero-current voltage solutions.
	n := newMixedNetwork(Compiled{Ground: "ground", Nets: []Net{{ID: "ground"}, {ID: "a"}, {ID: "b"}}, TerminalNet: map[string]string{"R1.1": "a", "R1.2": "b"}, MixedGraph: Graph{Components: []Component{resistor}}})
	tr := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	if _, err := n.solve(nil, nil, 0, tr, 0); err == nil || !strings.Contains(err.Error(), "singular") {
		t.Fatal("zero residual was mistaken for a unique linear solution", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := n.solveContext(ctx, nil, nil, 0, tr, 0); !errors.Is(err, context.Canceled) {
		t.Fatal("linear solve ignored cancellation", err)
	}
}

func TestLinearNumericalDiscoveryMatchesRetainedBounds(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	caps, err := detailedCapabilities("component.place")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
		t.Fatal(err)
	}
	numerics := schema["x-network-numerics"].(map[string]any)
	if numerics["max_newton_iterations"] != float64(100) || numerics["max_nonlinear_node_update_v"] != .2 || numerics["max_abs_node_voltage_v"] != float64(30) {
		t.Fatal("numerical bounds discovery changed", numerics)
	}
	if s.Snapshot().CommandRevision != before.CommandRevision {
		t.Fatal("numerical discovery mutated the service")
	}
}
