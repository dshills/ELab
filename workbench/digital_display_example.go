package workbench

import "fmt"

// A shared builder keeps new teaching circuits explicit: every electrical join
// is a named pin/wire, not an inference from intersecting drawing geometry.
type exampleBuilder struct{ graph Graph }

func newExampleBuilder() *exampleBuilder { return &exampleBuilder{EmptyGraph()} }
func (b *exampleBuilder) part(id, kind, value string, x, y float64, rotation int) {
	p, err := MakeComponent(id, kind, Point{x, y}, value)
	if err != nil {
		panic(err)
	}
	p.Rotation = rotation
	b.graph.Components = append(b.graph.Components, p)
}
func (b *exampleBuilder) wire(from, to string) {
	b.graph.Wires = append(b.graph.Wires, Wire{ID: fmt.Sprintf("W%d", len(b.graph.Wires)+1), From: from, To: to, Connected: true})
}
func (b *exampleBuilder) ground(id, ref string, x, y float64) {
	b.part(id, "ground", "", x, y, 0)
	b.wire(ref, id+".reference")
}
func (b *exampleBuilder) label(id, name, ref string, x, y float64) {
	b.part(id, "net-label", name, x, y, 0)
	b.wire(ref, id+".node")
}

func DisplayCounterExampleGraph() Graph {
	b := newExampleBuilder()
	base := Timer555FlasherExampleGraph()
	removed := map[string]bool{"RLED": true, "D1": true, "GNDL": true}
	for _, p := range base.Components {
		if !removed[p.ID] {
			p.Position.X -= 650
			b.graph.Components = append(b.graph.Components, p)
		}
	}
	for _, w := range base.Wires {
		from, _, _ := base.Terminal(w.From)
		to, _, _ := base.Terminal(w.To)
		if !removed[from.ID] && !removed[to.ID] {
			b.wire(w.From, w.To)
		}
	}
	b.part("U2", "decade-counter", "", 50, 0, 0)
	b.part("U3", "bcd-decoder", "", 650, 0, 0)
	b.part("DS1", "seven-segment", "", 1460, 0, 0)
	b.label("POWER", "VCC", "JV.node", -850, -460)
	b.label("P2", "VCC", "U2.vcc", 50, -260)
	b.label("P3", "VCC", "U3.vcc", 650, -330)
	b.label("ENABLE", "VCC", "U2.enable", -140, 70)
	b.ground("GND2", "U2.gnd", 50, 290)
	b.ground("GNDR", "U2.reset", -140, 180)
	b.ground("GND3", "U3.gnd", 650, 350)
	b.ground("GNDD", "DS1.common", 1460, 350)
	b.wire("U1.out", "U2.clk")
	for k := range 4 {
		b.wire(fmt.Sprintf("U2.q%d", k), fmt.Sprintf("U3.b%d", k))
	}
	for k, pin := range segmentNames {
		y := -150. + 50*float64(k)
		id := fmt.Sprintf("RS%d", k+1)
		b.part(id, "resistor", "1k", 1050, y, 0)
		b.wire("U3."+pin, id+".1")
		b.wire(id+".2", "DS1."+pin)
	}
	return b.graph
}
