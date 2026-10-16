package workbench

import "fmt"

func FlasherExampleGraph() Graph {
	g := EmptyGraph()
	specs := []struct {
		id, kind, value string
		pos             Point
		rotation        int
	}{
		{"V1", "dc-source", "5 V", Point{-400, 0}, 0},
		{"RL1", "resistor", "1k", Point{-140, -200}, 90}, {"RL2", "resistor", "1k", Point{140, -200}, 90},
		{"RB1", "resistor", "47k", Point{-330, -100}, 90}, {"RB2", "resistor", "47k", Point{280, -100}, 90},
		{"D1", "led", "", Point{-140, -70}, 0}, {"D2", "led", "", Point{140, -70}, 0},
		{"Q1", "npn", "", Point{-140, 70}, 0}, {"Q2", "npn", "", Point{140, 70}, 0},
		{"C1", "capacitor", "10uF", Point{-20, 10}, 270}, {"C2", "capacitor", "10uF", Point{20, 160}, 90},
		{"GND1", "ground", "", Point{0, 240}, 0},
	}
	for _, p := range specs {
		c, _ := MakeComponent(p.id, p.kind, p.pos, p.value)
		c.Rotation = p.rotation
		if c.ID == "C1" {
			v := .1
			c.InitialVoltage = &v
		}
		g.Components = append(g.Components, c)
	}
	for i, w := range [][2]string{{"V1.positive", "RL1.1"}, {"V1.positive", "RL2.1"}, {"V1.positive", "RB1.1"}, {"V1.positive", "RB2.1"}, {"RL1.2", "D1.anode"}, {"RL2.2", "D2.anode"}, {"D1.cathode", "Q1.collector"}, {"D2.cathode", "Q2.collector"}, {"RB1.2", "Q1.base"}, {"RB2.2", "Q2.base"}, {"Q1.collector", "C1.1"}, {"C1.2", "Q2.base"}, {"Q2.collector", "C2.1"}, {"C2.2", "Q1.base"}, {"Q1.emitter", "GND1.reference"}, {"Q2.emitter", "GND1.reference"}, {"V1.negative", "GND1.reference"}} {
		g.Wires = append(g.Wires, Wire{fmt.Sprintf("W%d", i+1), w[0], w[1], true})
	}
	return g
}
