package workbench

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

func TestFlasherBiasLabelsClearNeighboringTerminals(t *testing.T) {
	g := FlasherExampleGraph()
	for _, id := range []string{"RB1", "RB2"} {
		part, _ := g.Component(id)
		x, y, text := schematicPartLabel(part)
		for _, neighbor := range g.Components {
			if neighbor.ID == id {
				continue
			}
			for _, pin := range neighbor.Terminals {
				at, _ := TerminalPosition(neighbor, pin)
				if at.X >= x-7 && at.X <= x+schematicLabelWidth(text)+4 && at.Y >= y-19 && at.Y <= y+8 {
					t.Errorf("%s label crowds %s.%s at %v", id, neighbor.ID, pin, at)
				}
			}
		}
	}
}

func TestFlasherLayoutPreservesTheOriginalCircuitAndTimedResponse(t *testing.T) {
	updated := FlasherExampleGraph()
	data, err := os.ReadFile("testdata/flasher-layout-095.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Graph    Graph             `json:"graph"`
		Settings TransientSettings `json:"transient_settings"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	original := fixture.Graph
	if !ElectricalEqual(updated, original) {
		t.Fatal("template layout changed electrical inputs or named connectivity")
	}
	kernels := []*simulationKernel{}
	for _, graph := range []Graph{original, updated} {
		c, status, d := graph.Compile()
		if d != nil {
			t.Fatal(status, d)
		}
		kernel, err := newSimulationKernel(t.Context(), kernelPlan{compiled: c, settings: fixture.Settings, mode: "auto", run: 1, revision: 1})
		if err != nil {
			t.Fatal(err)
		}
		kernels = append(kernels, kernel)
	}
	for sample := 0; sample <= 600; sample++ {
		if sample > 0 {
			for _, kernel := range kernels {
				if err := kernel.advance(t.Context(), float64(sample)*.001); err != nil {
					t.Fatal(err)
				}
			}
		}
		if !reflect.DeepEqual(kernels[0].mixed.result, kernels[1].mixed.result) || !reflect.DeepEqual(kernels[0].trace.Sample, kernels[1].trace.Sample) {
			t.Fatalf("layout changed accepted voltages/currents/charge at %.3f s", float64(sample)*.001)
		}
	}
}

func TestFlasherBiasLabelMasksLeaveRoutedWiresContinuous(t *testing.T) {
	g := FlasherExampleGraph()
	crosses := func(a, b Point, box routeRect) bool {
		lo, hi := 0., 1.
		for _, axis := range [][4]float64{{a.X, b.X, box.left, box.right}, {a.Y, b.Y, box.top, box.bottom}} {
			delta := axis[1] - axis[0]
			if delta == 0 {
				if axis[0] < axis[2] || axis[0] > axis[3] {
					return false
				}
				continue
			}
			x, y := (axis[2]-axis[0])/delta, (axis[3]-axis[0])/delta
			lo, hi = math.Max(lo, math.Min(x, y)), math.Min(hi, math.Max(x, y))
			if lo > hi {
				return false
			}
		}
		return true
	}
	for _, id := range []string{"RB1", "RB2"} {
		p, _ := g.Component(id)
		x, y, text := schematicPartLabel(p)
		// Enclose the actual white mask plus half the 2.2-unit wire stroke.
		mask := routeRect{x - 4.1, x - 3 + schematicLabelWidth(text) + 1.1, y - 16.1, y + 5.1}
		for _, wire := range g.Wires {
			path := WirePath(g, wire)
			for i := 1; i < len(path); i++ {
				if crosses(path[i-1], path[i], mask) {
					t.Errorf("%s mask cuts %s segment %v → %v", id, wire.ID, path[i-1], path[i])
				}
			}
		}
	}
}
