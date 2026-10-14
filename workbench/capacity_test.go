package workbench

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"
)

// Each track adds an unloaded wiper node. Fifty tracks plus a source produce
// 101 nodes; tying the final wiper to pin 1 yields exactly 100 valid nodes.
func capacityChain(t *testing.T, sourceKind string, pots int) Graph {
	t.Helper()
	g := EmptyGraph()
	add := func(id, kind string, p Point) {
		c, err := MakeComponent(id, kind, p, "")
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, c)
	}
	add("V1", sourceKind, Point{-800, 0})
	add("GND1", "ground", Point{800, 0})
	previous := "V1.positive"
	for i := 1; i <= pots; i++ {
		id := fmt.Sprintf("P%d", i)
		add(id, "potentiometer", Point{float64((i-1)%7)*180 - 540, float64((i-1)/7)*180 - 540})
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i), From: previous, To: id + ".1", Connected: true})
		previous = id + ".2"
	}
	if pots == 50 {
		g.Wires = append(g.Wires, Wire{ID: "WTIE", From: "P50.wiper", To: "P50.1", Connected: true})
	}
	g.Wires = append(g.Wires, Wire{ID: "WG", From: previous, To: "GND1.reference", Connected: true}, Wire{ID: "WV", From: "V1.negative", To: "GND1.reference", Connected: true})
	return g
}
func checkCapacityReadings(t *testing.T, state Snapshot, pots int, source float64) {
	t.Helper()
	if state.Mixed == nil || state.Mixed.Stale || state.SimulatedComponents != pots+1 || len(state.Nets) != pots*2+1-boolCount(pots == 50) {
		t.Fatalf("invalid capacity state: status=%s parts=%d nets=%d", state.Status, state.SimulatedComponents, len(state.Nets))
	}
	resistance := float64(pots) * 10000
	if pots == 50 {
		resistance -= 5000
	}
	current := source / resistance
	for _, p := range state.Mixed.Potentiometers {
		var number int
		fmt.Sscanf(p.ID, "P%d", &number)
		expected := source - current*((float64(number)-1)*10000+5000)
		wiperCurrent, terminalCurrent := 0., current
		if pots == 50 && number == 50 {
			expected = source - current*490000
			wiperCurrent = current
			terminalCurrent = 0
		}
		if math.Abs(p.WiperVoltage-expected) > 1e-7 || math.Abs(p.Terminal1Current-terminalCurrent) > 1e-10 || math.Abs(p.WiperCurrent-wiperCurrent) > 1e-10 {
			t.Fatalf("%s divider reference mismatch: %+v expected %.9g V %.9g A", p.ID, p, expected, current)
		}
	}
	if state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
		t.Fatal("capacity conservation failed")
	}
}
func TestHundredNodesAndFourTraces(t *testing.T) {
	for _, kind := range []string{"dc-source", "pulse-source"} {
		t.Run(kind, func(t *testing.T) {
			s := newGraphService(SolveLED, capacityChain(t, kind, 50))
			defer s.Close()
			state := settled(t, s)
			checkCapacityReadings(t, state, 50, 5)
			if kind == "pulse-source" {
				call(t, s, "simulation.pause", struct{}{}, "pause")
				call(t, s, "simulation.restart", struct{}{}, "restart")
				call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:V1", "v:P1.wiper", "v:P25.wiper", "i:P49.1"}}, "probes")
				call(t, s, "simulation.step", StepParams{Duration: .06}, "step")
				duration := .06
				_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:V1", "v:P1.wiper", "v:P25.wiper", "i:P49.1"}}, "verify-capacity")
				if record.Metadata.Mixed == nil || len(record.Metadata.Mixed.Nodes) != 100 {
					t.Fatal("capacity Verify lost nodes")
				}
				state = s.Snapshot()
				checkCapacityReadings(t, state, 50, 0)
				if len(state.Scope.Traces) != 4 || state.Transient.Stale {
					t.Fatal("four live capacity traces unavailable")
				}
			}
			path := filepath.Join(t.TempDir(), "capacity.elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			fresh := New()
			defer fresh.Close()
			call(t, fresh, "project.load", LoadParams{Path: path}, "load")
			checkCapacityReadings(t, settled(t, fresh), 50, 5)
		})
	}
}
func TestExcessNodeCapacityPreservesEditingAndSave(t *testing.T) {
	g := capacityChain(t, "dc-source", 50)
	g.Wires = g.Wires[:len(g.Wires)-3]
	g.Wires = append(g.Wires, Wire{ID: "WG", From: "P50.2", To: "GND1.reference", Connected: true}, Wire{ID: "WV", From: "V1.negative", To: "GND1.reference", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	state := s.Snapshot()
	if state.Status != "unsupported" || len(state.Diagnostics) != 1 || state.Diagnostics[0].Code != "mixed-limit" || state.SimulatedComponents != 51 {
		t.Fatal("excess capacity not diagnosed")
	}
	call(t, s, "component.move", MoveParams{ID: "P1", Position: Point{0, 2000}}, "move")
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "excess.elab")}, "save")
	if s.Snapshot().Document.Dirty {
		t.Fatal("unsupported drawing could not be saved")
	}
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func resistorCapacityChain(t *testing.T, count int) Graph {
	t.Helper()
	g := EmptyGraph()
	for _, p := range []PlaceParams{{ID: "V1", Kind: "dc-source", Position: Point{-800, 0}}, {ID: "GND1", Kind: "ground", Position: Point{800, 0}}} {
		c, err := MakeComponent(p.ID, p.Kind, p.Position, "")
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, c)
	}
	previous := "V1.positive"
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("R%d", i)
		c, err := MakeComponent(id, "resistor", Point{float64((i-1)%8)*160 - 560, float64((i-1)/8)*160 - 560}, "1k")
		if err != nil {
			t.Fatal(err)
		}
		g.Components = append(g.Components, c)
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i), From: previous, To: id + ".1", Connected: true})
		previous = id + ".2"
	}
	g.Wires = append(g.Wires, Wire{ID: "WG", From: previous, To: "GND1.reference", Connected: true}, Wire{ID: "WV", From: "V1.negative", To: "GND1.reference", Connected: true})
	return g
}
func TestSixtyFourPartBoundaryAndExcessRemainsSavable(t *testing.T) {
	for _, count := range []int{63, 64} {
		s := newGraphService(SolveLED, resistorCapacityChain(t, count))
		state := settled(t, s)
		if count == 63 {
			if state.SimulatedComponents != 64 || state.Mixed == nil || state.Status != "running" {
				t.Fatal("64-part envelope unavailable")
			}
			for _, r := range state.Mixed.Resistors {
				if math.Abs(r.Current-5./63000) > 1e-10 {
					t.Fatal("64-part resistor reference mismatch")
				}
			}
		} else {
			if state.Status != "unsupported" || state.Diagnostics[0].Code != "mixed-limit" {
				t.Fatal("65th physical part not diagnosed")
			}
			call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "excess-parts.elab")}, "save")
		}
		s.Close()
	}
}
