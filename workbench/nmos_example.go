package workbench

import "fmt"

// NMOSExampleGraph uses a pulse gate and a grounded source. RG/RPD make the
// gate's external DC return explicit without inventing gate leakage.
func NMOSExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		position        Point
	}{
		{"VCC", "dc-source", "5 V", Point{-300, -120}},
		{"VIN", "pulse-source", "5 V", Point{-300, 60}},
		{"RG", "resistor", "100 Ω", Point{-140, 60}},
		{"RPD", "resistor", "100k Ω", Point{10, 190}},
		{"RL", "resistor", "1k Ω", Point{0, -200}},
		{"D1", "led", "", Point{150, -80}},
		{"M1", "nmos", "", Point{150, 60}},
		{"GND1", "ground", "", Point{260, 220}},
	} {
		c, err := MakeComponent(p.id, p.kind, p.position, p.value)
		if err != nil {
			panic(err)
		}
		if p.id == "RPD" {
			c.Rotation = 90
		}
		g.Components = append(g.Components, c)
	}
	for i, p := range [][2]string{
		{"VCC.positive", "RL.1"}, {"RL.2", "D1.anode"}, {"D1.cathode", "M1.drain"},
		{"VIN.positive", "RG.1"}, {"RG.2", "M1.gate"}, {"M1.gate", "RPD.1"},
		{"RPD.2", "GND1.reference"}, {"M1.source", "GND1.reference"},
		{"VCC.negative", "GND1.reference"}, {"VIN.negative", "GND1.reference"},
	} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", i+1), From: p[0], To: p[1], Connected: true})
	}
	return g
}
