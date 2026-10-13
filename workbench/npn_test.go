package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func npnSolved(t *testing.T, g Graph) (Compiled, NPNResult) {
	t.Helper()
	c, status, d := g.Compile()
	if d != nil {
		t.Fatalf("compile %s: %+v", status, d)
	}
	r, err := SolveNPN(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return c, r
}
func npnValue(g Graph, id string, value float64) Graph {
	g = g.Clone()
	for i, c := range g.Components {
		if c.ID == id {
			g.Components[i].Value = value
		}
	}
	return g
}
func resistiveNPN(g Graph) Graph {
	g = g.Clone()
	parts := []Component{}
	for _, p := range g.Components {
		if p.Kind != "led" {
			parts = append(parts, p)
		}
	}
	g.Components = parts
	wires := []Wire{}
	for _, w := range g.Wires {
		if w.From == "RL.2" {
			w.To = "Q1.collector"
		}
		if w.From != "D1.cathode" {
			wires = append(wires, w)
		}
	}
	g.Wires = wires
	return g
}
func TestNPNReferenceStatesAndConservation(t *testing.T) {
	g := NPNExampleGraph()
	var last float64
	for _, test := range []struct {
		drive float64
		state string
	}{{0, "cutoff"}, {.8, "active"}, {5, "saturation"}} {
		c, r := npnSolved(t, npnValue(g, "VIN", test.drive))
		t.Logf("drive %.1f V: %s, IC %.9g A VCE %.9g V IB %.9g A", test.drive, r.State, r.CollectorCurrent, r.VCE, r.BaseCurrent)
		if r.State != test.state || r.CollectorCurrent < last {
			t.Fatalf("incorrect switch state: %+v", r)
		}
		last = r.CollectorCurrent
		if math.Abs(r.EmitterCurrent-r.BaseCurrent-r.CollectorCurrent) > 1e-14 || r.KCLResidual > 1e-10 || math.Abs(r.KVLResidual) > 1e-10 {
			t.Fatal("KCL/KVL failure")
		}
		inputPower := c.Input.Value*r.BaseCurrent + c.Source.Value*r.CollectorCurrent
		dissipated := r.BaseResistorPower + r.LoadResistorPower + r.LEDPower + r.TransistorPower
		if math.Abs(inputPower-dissipated) > 1e-10 {
			t.Fatalf("power mismatch %g", inputPower-dissipated)
		}
		if r.Nodes[c.Supply] != c.Source.Value || r.Nodes[c.Collector] != r.VCE || r.Nodes[c.Base] != r.VBE {
			t.Fatal("bad node annotation")
		}
	}
	_, r := npnSolved(t, resistiveNPN(g))
	// Independent LTspice/KiCad measurement from the saved research fixture:
	// IS=1e-14, BF=100, BR default 1, 5 V / 1k load / 10k base.
	t.Logf("resistive SPICE reference: VCE %.12g V", r.VCE)
	if math.Abs(r.VCE-.070399299264) > .001 {
		t.Fatalf("SPICE benchmark VCE: %.9g", r.VCE)
	}
	_, off := npnSolved(t, npnValue(resistiveNPN(g), "VIN", 0))
	if math.Abs(off.VCE-5) > 1e-8 {
		t.Fatal("off collector does not approach supply")
	}
}
func TestNPNBoundsAndCancellation(t *testing.T) {
	for _, supply := range []float64{0, .1, 5, 12} {
		for _, drive := range []float64{0, .2, 1, 5, 12} {
			for _, r := range []float64{10, 1000, 100000} {
				g := npnValue(npnValue(npnValue(npnValue(NPNExampleGraph(), "VCC", supply), "VIN", drive), "RB", r), "RL", r)
				_, result := npnSolved(t, g)
				if !finite(result.VCE) || !finite(result.CollectorCurrent) || result.VCE < 0 {
					t.Fatal("invalid bound result")
				}
			}
		}
	}
	c, _, _ := NPNExampleGraph().Compile()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SolveNPN(ctx, c); err != context.Canceled {
		t.Fatal("ignored cancellation")
	}
}
func TestNPNTopologyAndGeometry(t *testing.T) {
	g := NPNExampleGraph()
	_, want := npnSolved(t, g)
	// Reverse both resistor pin identities; electrical connections remain valid.
	for i := range g.Wires {
		for _, field := range []*string{&g.Wires[i].From, &g.Wires[i].To} {
			for _, id := range []string{"RB", "RL"} {
				if *field == id+".1" {
					*field = id + ".2"
				} else if *field == id+".2" {
					*field = id + ".1"
				}
			}
		}
	}
	for i := range g.Components {
		g.Components[i].Rotation = 90
	}

	// A shared supply may feed both the base and collector resistor branches.
	shared := NPNExampleGraph()
	shared.Components = append(shared.Components[:1], shared.Components[2:]...)
	wires := []Wire{}
	for _, w := range shared.Wires {
		if w.From == "VIN.negative" {
			continue
		}
		if w.From == "VIN.positive" {
			w.From = "VCC.positive"
		}
		wires = append(wires, w)
	}
	shared.Wires = wires
	sc, sr := npnSolved(t, shared)
	if sc.Input.ID != sc.Source.ID || math.Abs(sr.CollectorCurrent-want.CollectorCurrent) > 1e-14 {
		t.Fatal("shared source topology differs")
	}
	_, got := npnSolved(t, g)
	if got.CollectorCurrent != want.CollectorCurrent {
		t.Fatal("rotation/pin order affected result")
	}
	q, _ := g.Component("Q1")
	if p, _ := TerminalPosition(q, "base"); p != (Point{q.Position.X, q.Position.Y - 40}) {
		t.Fatal("rotated base terminal wrong")
	}
	// Place LED before the load resistor; preserve correct absolute net voltages.
	g = NPNExampleGraph()
	g.Wires[0].To = "D1.anode"
	g.Wires[1].From = "D1.cathode"
	g.Wires[1].To = "RL.1"
	g.Wires[2].From = "RL.2"
	compiled, got := npnSolved(t, g)
	if math.Abs(got.CollectorCurrent-want.CollectorCurrent) > 1e-14 || got.Nodes[compiled.Supply] != 5 || math.Abs(got.Nodes[compiled.Cathode]-(5-got.LEDVoltage)) > 1e-14 {
		t.Fatal("LED first node annotation")
	}
	for _, test := range []struct {
		name   string
		edit   func(*Graph)
		status string
	}{
		{"open base", func(g *Graph) { g.Wires = g.Wires[:4] }, "incomplete"},
		{"reversed LED", func(g *Graph) { g.Wires[1].To = "D1.cathode"; g.Wires[2].From = "D1.anode" }, "unsupported"},
		{"short CE", func(g *Graph) { g.Wires = append(g.Wires, Wire{"short", "Q1.collector", "Q1.emitter", true}) }, "error"},
		{"extra base link", func(g *Graph) { g.Wires = append(g.Wires, Wire{"bad", "Q1.base", "VCC.positive", true}) }, "unsupported"},
		{"capacitor", func(g *Graph) {
			c, _ := MakeComponent("C1", "capacitor", Point{500, 0}, "")
			g.Components = append(g.Components, c)
		}, "incomplete"},
		{"pulsed supply", func(g *Graph) {
			c, _ := MakeComponent("VCC", "pulse-source", Point{260, -100}, "5 V")
			g.Components[0] = c
		}, "unsupported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := NPNExampleGraph()
			test.edit(&g)
			_, status, d := g.Compile()
			if d == nil || status != test.status {
				t.Fatalf("%s: %+v", status, d)
			}
		})
	}
}
func TestNPNSharedServicePersistenceAndStaleReadings(t *testing.T) {
	s := New()
	defer s.Close()
	g := NPNExampleGraph()
	// Build with arbitrary application commands, rather than relying on example IDs.
	for i, p := range g.Components {
		call(t, s, "component.place", PlaceParams{ID: p.ID, Kind: p.Kind, Position: p.Position, Value: func() string {
			if p.Unit != "" {
				return fmt.Sprintf("%g %s", p.Value, p.Unit)
			}
			return ""
		}()}, fmt.Sprintf("p%d", i))
	}
	for i, w := range g.Wires {
		call(t, s, "wire.connect", ConnectParams{ID: w.ID, From: w.From, To: w.To}, fmt.Sprintf("w%d", i))
	}
	on := settled(t, s)
	if on.NPN == nil || on.NPN.State != "saturation" || on.Result != nil || on.Transient != nil {
		t.Fatal("missing NPN result")
	}
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.set", ValueParams{ID: "VIN", Value: "0 V"}, "off")
	stale := s.Snapshot()
	if stale.NPN == nil || !stale.NPN.Stale || stale.Status != "paused" {
		t.Fatal("paused edit did not mark stale")
	}
	call(t, s, "simulation.resume", struct{}{}, "resume")
	off := settled(t, s)
	if off.NPN.State != "cutoff" || off.NPN.ElectricalRevision != off.ElectricalRevision {
		t.Fatal("off did not settle")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	restored := settled(t, s)
	if restored.NPN.State != "saturation" {
		t.Fatal("undo drive did not solve")
	}
	call(t, s, "component.move", MoveParams{ID: "Q1", Position: Point{140, 70}}, "move")
	moved := s.Snapshot()
	if moved.RunID != restored.RunID || !reflect.DeepEqual(moved.NPN, restored.NPN) {
		t.Fatal("geometry restarted solve")
	}
	path := filepath.Join(t.TempDir(), "NPN.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeDocument(data)
	if err != nil || d.NPNModel == nil {
		t.Fatalf("NPN save: %+v %v", d, err)
	}
	for _, bad := range []func(*Document){func(d *Document) { d.NPNModel = nil }, func(d *Document) { d.NPNModel.ForwardBeta = 200 }} {
		var modified Document
		_ = json.Unmarshal(data, &modified)
		bad(&modified)
		raw, _ := json.Marshal(modified)
		if _, err := decodeDocument(raw); err == nil {
			t.Fatal("accepted missing/changed model")
		}
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := settled(t, fresh)
	if loaded.NPN == nil || loaded.NPN.CollectorCurrent != restored.NPN.CollectorCurrent || loaded.Document.Dirty {
		t.Fatal("fresh session NPN differs")
	}
	copied := s.Snapshot()
	copied.NPN.Nodes["bad"] = 99
	if _, ok := s.Snapshot().NPN.Nodes["bad"]; ok {
		t.Fatal("snapshot aliases NPN nodes")
	}
	call(t, s, "object.delete", IDParams{ID: "W5"}, "disconnect")
	broken := s.Snapshot()
	if broken.Status != "incomplete" || !broken.NPN.Stale {
		t.Fatal("broken circuit reading not stale")
	}
}
