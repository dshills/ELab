package workbench

import (
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func bottomReferencedStack() Graph {
	g := SplitSupplyExampleGraph()
	g.Wires[4].From = "V2.negative"
	return g
}

func solveFloatingDC(t *testing.T, g Graph) (*mixedState, Compiled) {
	t.Helper()
	c, status, d := g.Compile()
	if d != nil {
		t.Fatalf("compile %s: %+v", status, d)
	}
	n := newMixedNetwork(c)
	state, err := n.solve(nil, make([]float64, len(n.caps)), 0, &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return state, c
}

func TestFloatingDCSplitSupplyKirchhoffReference(t *testing.T) {
	state, c := solveFloatingDC(t, SplitSupplyExampleGraph())
	r := state.result
	for pin, expected := range map[string]float64{"V1.positive": 3, "V1.negative": 0, "V2.positive": 0, "V2.negative": -3} {
		if math.Abs(r.Nodes[c.TerminalNet[pin]]-expected) > 1e-12 {
			t.Fatal("incorrect rail", pin, r.Nodes)
		}
	}
	if math.Abs(r.Resistors[0].Current-.006) > 1e-12 || math.Abs(r.Resistors[0].Voltage-6) > 1e-12 || math.Abs(r.Resistors[0].Power-.036) > 1e-12 || r.KCLResidual > 1e-12 || math.Abs(r.PowerResidual) > 1e-12 {
		t.Fatal("split supply violates Ohm/Kirchhoff reference", r)
	}
	for _, source := range r.Sources {
		if math.Abs(source.Current-.006) > 1e-12 || math.Abs(source.Voltage-3) > 1e-12 || math.Abs(source.Power-.018) > 1e-12 {
			t.Fatal("source delivered signs or differential voltage wrong", source)
		}
	}
}

func TestFloatingDCSingleLEDReferenceShiftUsesNetwork(t *testing.T) {
	g := ExampleGraph()
	// Keep the same electrical loop; choose LED anode as the common reference.
	for i := range g.Wires {
		switch g.Wires[i].ID {
		case "W3":
			g.Wires[i].From, g.Wires[i].To = "D1.cathode", "V1.negative"
		case "W4":
			g.Wires[i].From, g.Wires[i].To = "GND.reference", "D1.anode"
		}
	}
	state, c := solveFloatingDC(t, g)
	if c.Analysis != NetworkDCAnalysis {
		t.Fatal("shifted LED used grounded analytic path")
	}
	reference, err := SolveLED(t.Context(), DefaultCircuit())
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(state.result.LEDs[0].Current-reference.Current) > 1e-10 || math.Abs(state.result.Nodes[c.TerminalNet["V1.negative"]]+reference.LEDVoltage) > 1e-8 {
		t.Fatal("changing common reference changed the electrical loop", state.result, reference)
	}
}

func TestFloatingDCSeriesAndNegativeRails(t *testing.T) {
	g := bottomReferencedStack()
	g.Components[0].Value = 5
	state, c := solveFloatingDC(t, g)
	if math.Abs(state.result.Nodes[c.TerminalNet["V1.positive"]]-8) > 1e-12 || math.Abs(state.result.Nodes[c.TerminalNet["V1.negative"]]-3) > 1e-12 || math.Abs(state.result.Resistors[0].Current-.008) > 1e-12 {
		t.Fatal("series sources did not add", state.result)
	}
}

func TestFloatingDCShiftedRCCapacitorTiming(t *testing.T) {
	for _, returned := range []bool{false, true} {
		t.Run(map[bool]string{false: "output reference", true: "return resistance"}[returned], func(t *testing.T) {
			testFloatingDCRC(t, returned)
		})
	}
}

func testFloatingDCRC(t *testing.T, returned bool) {
	t.Helper()
	g := RCExampleGraph()
	g.Components[0].Kind, g.Components[0].Model = "dc-source", "ideal-dc.v1"
	g.Wires = []Wire{{"W1", "V1.positive", "R1.1", true}, {"W2", "R1.2", "C1.1", true}, {"W3", "C1.2", "V1.negative", true}, {"W4", "C1.1", "GND.reference", true}}
	if returned {
		g.Wires = []Wire{{"W1", "V1.positive", "C1.1", true}, {"W2", "C1.2", "GND.reference", true}, {"W3", "V1.negative", "R1.1", true}, {"W4", "R1.2", "GND.reference", true}}
	}
	state, c := solveFloatingDC(t, g)
	tr := &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}
	for i := 1; i <= 50; i++ {
		var err error
		state, err = state.advance(tr, float64(i)*.001)
		if err != nil {
			t.Fatal(err)
		}
		expected := 5 * -math.Expm1(-float64(i)/10) // 10k × 1uF = 10 ms.
		got := state.result.Capacitors[0].Voltage
		negative := -got
		if returned {
			negative = got - 5
		}
		if math.Abs(got-expected) > .003 || math.Abs(state.result.Nodes[c.TerminalNet["V1.negative"]]-negative) > 1e-10 || state.result.KCLResidual > 1e-9 || math.Abs(state.result.PowerResidual) > 1e-8 {
			t.Fatal("shifted RC changed time constant/reference/conservation", i, got, expected, state.result)
		}
	}
}

func TestFloatingDCCaptureSaveReopenAndSweep(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "split-supply"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	_, captured := verify(t, s, VerifyParams{ProbeIDs: []string{"v:V2.negative", "v:V2", "i:R1"}}, "verify")
	if captured.Metadata.Mixed == nil || math.Abs(captured.Metadata.Mixed.Resistors[0].Current-.006) > 1e-12 || !reflect.DeepEqual(before.Mixed, s.Snapshot().Mixed) {
		t.Fatal("captured split supply differs or changes live circuit")
	}
	path := filepath.Join(t.TempDir(), "split.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	other := New()
	defer other.Close()
	call(t, other, "project.load", LoadParams{Path: path}, "open")
	reopened := settled(t, other)
	if reopened.Mixed == nil || math.Abs(reopened.Mixed.Resistors[0].Current-.006) > 1e-12 || len(reopened.Operations) != 1 {
		t.Fatal("saved split circuit/capture did not reopen", reopened.Diagnostics)
	}
	sweep := sweepForTest(t, SplitSupplyExampleGraph(), SweepSettings{ComponentID: "V2", Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 6, ProbeIDs: []string{"v:V2.negative", "v:V2", "i:R1"}})
	for _, point := range sweep.Points {
		if math.Abs(point.Values[0]+point.Input) > 1e-12 || math.Abs(point.Values[1]-point.Input) > 1e-12 || math.Abs(point.Values[2]-(3+point.Input)/1000) > 1e-12 {
			t.Fatal("sweep lost relative source polarity", point)
		}
	}
}

func TestFloatingDCConstraintsAndWaveformRestrictions(t *testing.T) {
	for _, mode := range []string{"direct-cap", "stack-cap-loop", "source-loop", "pulse", "sine", "finite-pulse"} {
		t.Run(mode, func(t *testing.T) {
			g := bottomReferencedStack()
			expected := "ideal-constraint-loop"
			switch mode {
			case "direct-cap", "stack-cap-loop":
				cap, _ := MakeComponent("C1", "capacitor", Point{}, "1uF")
				g.Components = append(g.Components, cap)
				negative := "V2.negative"
				if mode == "direct-cap" {
					negative, expected = "V1.negative", "ideal-source-capacitor"
				}
				g.Wires = append(g.Wires, Wire{"W6", "C1.1", "V1.positive", true}, Wire{"W7", "C1.2", negative, true})
			case "source-loop":
				source, _ := MakeComponent("V3", "dc-source", Point{}, "6 V")
				g.Components = append(g.Components, source)
				g.Wires = append(g.Wires, Wire{"W6", "V3.positive", "V1.positive", true}, Wire{"W7", "V3.negative", "V2.negative", true})
			default:
				kind := map[string]string{"pulse": "pulse-source", "sine": "sine-source", "finite-pulse": "finite-pulse-source"}[mode]
				g.Components[0], _ = MakeComponent("V1", kind, Point{}, "3 V")
				expected = "unsupported-topology"
			}
			_, status, d := g.Compile()
			if status != "unsupported" || d == nil || d.Code != expected {
				t.Fatalf("%s: %s %+v", mode, status, d)
			}
		})
	}
}

func TestFloatingDCAcceptedStateRespectsRange(t *testing.T) {
	g := bottomReferencedStack()
	g.Components[0].Value, g.Components[1].Value = 10, 10
	source, _ := MakeComponent("V3", "dc-source", Point{}, "11 V")
	g.Components = append(g.Components, source)
	g.Wires[0].To = "V3.negative"
	g.Wires = append(g.Wires, Wire{"W6", "V3.positive", "R1.1", true})
	c, status, d := g.Compile()
	if d != nil {
		t.Fatalf("%s %+v", status, d)
	}
	n := newMixedNetwork(c)
	guess := make([]float64, len(n.nodes)+len(n.sources))
	for pin, v := range map[string]float64{"V1.positive": 20, "V2.positive": 10, "V3.positive": 31} {
		guess[n.index[c.TerminalNet[pin]]] = v
	}
	for i := range n.sources {
		guess[len(n.nodes)+i] = -.031
	}
	_, err := n.newton(t.Context(), guess, nil, 0, &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}, 0)
	if err == nil || !strings.Contains(err.Error(), "±30 V") {
		t.Fatal("out-of-range converged seed was accepted", err, guess)
	}
}
