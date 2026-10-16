package workbench

import "fmt"

// CMOSExampleGraph joins complementary drains to invert the shared gate drive.
// RL defines the output even if both channels are off; it is an external load,
// not a hidden gate leak or a physical CMOS input model.
func CMOSExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		position        Point
		rotation        int
	}{
		{"VCC", "dc-source", "5 V", Point{-300, -200}, 0},
		{"VIN", "pulse-source", "5 V", Point{-300, 60}, 0},
		{"MP", "pmos", "", Point{100, -140}, 180},
		{"MN", "nmos", "", Point{100, 140}, 0},
		{"RL", "resistor", "10k Ω", Point{300, 0}, 0},
		{"GND1", "ground", "", Point{-400, 280}, 0},
		{"JIN", "junction", "", Point{-100, -40}, 0},
		{"JOUT", "junction", "", Point{100, 0}, 0},
	} {
		part, err := MakeComponent(p.id, p.kind, p.position, p.value)
		if err != nil {
			panic(err)
		}
		part.Rotation = p.rotation
		g.Components = append(g.Components, part)
	}
	for i, pins := range [][2]string{
		{"VCC.positive", "MP.source"}, {"VIN.positive", "JIN.node"},
		{"JIN.node", "MP.gate"}, {"JIN.node", "MN.gate"},
		{"MP.drain", "JOUT.node"}, {"JOUT.node", "MN.drain"},
		{"JOUT.node", "RL.1"}, {"RL.2", "GND1.reference"},
		{"MN.source", "GND1.reference"}, {"VCC.negative", "GND1.reference"},
		{"VIN.negative", "GND1.reference"},
	} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i+1), From: pins[0], To: pins[1], Connected: true})
	}
	return g
}
