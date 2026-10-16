package workbench

import "fmt"

// SplitSupplyExampleGraph stacks two ideal 3 V sources around a common reference.
func SplitSupplyExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		position        Point
	}{
		{"V1", "dc-source", "3 V", Point{-180, -100}},
		{"V2", "dc-source", "3 V", Point{-180, 100}},
		{"R1", "resistor", "1k Ω", Point{180, 0}},
		{"J1", "junction", "", Point{-180, 0}},
		{"GND1", "ground", "", Point{0, 60}},
	} {
		c, err := MakeComponent(p.id, p.kind, p.position, p.value)
		if err != nil {
			panic(err)
		}
		if c.Kind == "resistor" {
			c.Rotation = 90
		}
		g.Components = append(g.Components, c)
	}
	for i, pair := range [][2]string{
		{"V1.positive", "R1.1"}, {"R1.2", "V2.negative"},
		{"V1.negative", "J1.node"}, {"V2.positive", "J1.node"}, {"J1.node", "GND1.reference"},
	} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i+1), From: pair[0], To: pair[1], Connected: true})
	}
	return g
}
