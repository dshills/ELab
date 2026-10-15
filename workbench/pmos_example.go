package workbench

import "fmt"

// PMOSExampleGraph switches the supply side: a low gate lights the LED.
func PMOSExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		position        Point
		rotation        int
	}{
		{"VCC", "dc-source", "5 V", Point{-260, -150}, 0},
		{"VIN", "pulse-source", "5 V", Point{-260, 80}, 0},
		{"RG", "resistor", "100 Ω", Point{260, -130}, 0},
		{"RPU", "resistor", "100k Ω", Point{230, -40}, 0},
		{"M1", "pmos", "", Point{100, -130}, 180},
		{"RL", "resistor", "1k Ω", Point{100, 0}, 90},
		{"D1", "led", "", Point{100, 100}, 0},
		{"GND1", "ground", "", Point{100, 230}, 0},
		{"J1", "junction", "", Point{380, -180}, 0},
	} {
		c, err := MakeComponent(p.id, p.kind, p.position, p.value)
		if err != nil {
			panic(err)
		}
		c.Rotation = p.rotation
		g.Components = append(g.Components, c)
	}
	for i, p := range [][2]string{
		{"VCC.positive", "M1.source"}, {"M1.drain", "RL.1"}, {"RL.2", "D1.anode"}, {"D1.cathode", "GND1.reference"},
		{"VIN.positive", "J1.node"}, {"J1.node", "RG.2"}, {"RG.1", "M1.gate"}, {"M1.gate", "RPU.1"}, {"RPU.2", "M1.source"},
		{"VCC.negative", "GND1.reference"}, {"VIN.negative", "GND1.reference"},
	} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i+1), From: p[0], To: p[1], Connected: true})
	}
	return g
}
