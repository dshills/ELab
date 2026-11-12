package workbench

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"
)

// Each track adds an unloaded wiper node. Tying the final wiper to pin 1
// reduces the chain from 2*pots+1 electrical nodes to 2*pots.
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
	last := fmt.Sprintf("P%d", pots)
	g.Wires = append(g.Wires, Wire{ID: "WTIE", From: last + ".wiper", To: last + ".1", Connected: true})
	g.Wires = append(g.Wires, Wire{ID: "WG", From: previous, To: "GND1.reference", Connected: true}, Wire{ID: "WV", From: "V1.negative", To: "GND1.reference", Connected: true})
	return g
}
func checkCapacityReadings(t *testing.T, state Snapshot, pots int, source float64) {
	t.Helper()
	if state.Mixed == nil || state.Mixed.Stale || state.SimulatedComponents != pots+1 || len(state.Nets) != pots*2 || len(state.Mixed.Potentiometers) != pots {
		t.Fatalf("invalid capacity state: status=%s parts=%d nets=%d", state.Status, state.SimulatedComponents, len(state.Nets))
	}
	resistance := float64(pots)*10000 - 5000
	current := source / resistance
	for _, p := range state.Mixed.Potentiometers {
		var number int
		fmt.Sscanf(p.ID, "P%d", &number)
		expected := source - current*((float64(number)-1)*10000+5000)
		wiperCurrent, terminalCurrent := 0., current
		if number == pots {
			expected = current * 5000
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
	pots := MaxElectricalNodes / 2
	g := capacityChain(t, "dc-source", pots)
	g.Wires = append(g.Wires[:pots], g.Wires[pots+1:]...) // Remove the final wiper tie.
	s := newGraphService(SolveLED, g)
	defer s.Close()
	state := s.Snapshot()
	if state.Status != "unsupported" || len(state.Diagnostics) != 1 || state.Diagnostics[0].Code != "mixed-limit" || state.SimulatedComponents != pots+1 || len(state.Nets) != MaxElectricalNodes+1 {
		t.Fatal("excess capacity not diagnosed")
	}
	call(t, s, "component.move", MoveParams{ID: "P1", Position: Point{0, 2000}}, "move")
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "excess.elab")}, "save")
	if s.Snapshot().Document.Dirty {
		t.Fatal("unsupported drawing could not be saved")
	}
}

func TestExpandedNodeBoundaryPulsesAndRetainsCapture(t *testing.T) {
	pots := MaxElectricalNodes / 2
	s := newGraphService(SolveLED, capacityChain(t, "pulse-source", pots))
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	checkCapacityReadings(t, settled(t, s), pots, 5)
	probes := []string{"v:V1", "v:P1.wiper", "v:P50.wiper", "i:P99.1"}
	call(t, s, "scope.probes.set", ProbeParams{IDs: probes}, "probes")
	call(t, s, "simulation.step", StepParams{Duration: .06}, "step")
	checkCapacityReadings(t, s.Snapshot(), pots, 0)
	duration, step := .06, .002
	_, record := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: probes}, "capture")
	if len(record.Metadata.Mixed.Nodes) != MaxElectricalNodes || len(record.Metadata.Probes) != len(probes) {
		t.Fatal("capture lost nodes or probes")
	}
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: record.Operation.ID, Keep: true}, "keep")
	path := filepath.Join(t.TempDir(), "200-nodes.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	t.Cleanup(fresh.Close)
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	state := settled(t, fresh)
	checkCapacityReadings(t, state, pots, 5)
	if len(state.Operations) != 1 || !state.Paused || state.Document.Dirty {
		t.Fatal("200-node project did not reopen with its kept capture")
	}
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
func TestExpandedPartBoundaryAndExcessRemainsSavable(t *testing.T) {
	for _, count := range []int{MaxSimulatedComponents - 1, MaxSimulatedComponents} {
		s := newGraphService(SolveLED, resistorCapacityChain(t, count))
		state := settled(t, s)
		if count+1 == MaxSimulatedComponents {
			if state.SimulatedComponents != MaxSimulatedComponents || state.Mixed == nil || state.Status != "running" {
				t.Fatal("expanded part envelope unavailable")
			}
			for _, r := range state.Mixed.Resistors {
				if math.Abs(r.Current-5/(1000*float64(count))) > 1e-10 {
					t.Fatal("resistor reference mismatch at the part boundary")
				}
			}
		} else {
			if state.Status != "unsupported" || state.Diagnostics[0].Code != "mixed-limit" {
				t.Fatal("excess physical part not diagnosed")
			}
			call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "excess-parts.elab")}, "save")
		}
		s.Close()
	}
}

func TestEightCapacitorBranchesAgainstRCReference(t *testing.T) {
	b := newExampleBuilder()
	b.part("V1", "pulse-source", "5 V", -400, 0, 0)
	b.ground("GND", "V1.negative", -400, 200)
	for i := 1; i <= MaxNetworkCapacitors+1; i++ {
		r, c := fmt.Sprintf("R%d", i), fmt.Sprintf("C%d", i)
		x := float64(i) * 160
		b.part(r, "resistor", "1k", x, -100, 0)
		b.part(c, "capacitor", "1uF", x, 100, 0)
		b.wire("V1.positive", r+".1")
		b.wire(r+".2", c+".1")
		b.wire(c+".2", "GND.reference")
	}
	if _, _, d := b.graph.Compile(); d == nil || d.Code != "mixed-limit" {
		t.Fatal("excess capacitor not diagnosed", d)
	}
	g := b.graph
	g.Components = g.Components[:len(g.Components)-2]
	g.Wires = g.Wires[:len(g.Wires)-3]
	s := newGraphService(SolveLED, g)
	t.Cleanup(s.Close)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	const at = .005
	call(t, s, "simulation.step", StepParams{Duration: at}, "step")
	result := s.Snapshot().Mixed
	if result == nil || result.Stale || len(result.Capacitors) != MaxNetworkCapacitors || result.KCLResidual > 1e-8 {
		t.Fatal("missing accepted capacitor result", result)
	}
	want := 5 * (1 - math.Exp(-at/.001))
	for _, c := range result.Capacitors {
		if math.Abs(c.Voltage-want) > .002 {
			t.Fatalf("%s got %g V, want %g V", c.ID, c.Voltage, want)
		}
	}
}
