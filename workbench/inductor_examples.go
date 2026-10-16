package workbench

import "fmt"

// RLExampleGraph makes current continuity visible at an ideal source edge.
func RLExampleGraph() Graph {
	source, _ := MakeComponent("V1", "pulse-source", Point{-200, 0}, "5 V")
	resistor, _ := MakeComponent("R1", "resistor", Point{0, -100}, "100 Ω")
	coil, _ := MakeComponent("L1", "inductor", Point{200, 0}, "1 H")
	coil.Rotation = 90
	ground, _ := MakeComponent("GND1", "ground", Point{0, 160}, "")
	return Graph{Components: []Component{source, resistor, coil, ground}, Wires: []Wire{
		{ID: "W1", From: "V1.positive", To: "R1.1", Connected: true},
		{ID: "W2", From: "R1.2", To: "L1.1", Connected: true},
		{ID: "W3", From: "L1.2", To: "GND1.reference", Connected: true},
		{ID: "W4", From: "V1.negative", To: "GND1.reference", Connected: true},
	}}
}

// RLCExampleGraph uses explicit series resistance and zero stored startup state.
func RLCExampleGraph() Graph {
	source, _ := MakeComponent("V1", "dc-source", Point{-240, 0}, "5 V")
	resistor, _ := MakeComponent("R1", "resistor", Point{0, -160}, "50 Ω")
	coil, _ := MakeComponent("L1", "inductor", Point{200, -50}, "100 mH")
	coil.Rotation = 90
	cap, _ := MakeComponent("C1", "capacitor", Point{200, 140}, "10 µF")
	ground, _ := MakeComponent("GND1", "ground", Point{0, 280}, "")
	return Graph{Components: []Component{source, resistor, coil, cap, ground}, Wires: []Wire{
		{ID: "W1", From: "V1.positive", To: "R1.1", Connected: true},
		{ID: "W2", From: "R1.2", To: "L1.1", Connected: true},
		{ID: "W3", From: "L1.2", To: "C1.1", Connected: true},
		{ID: "W4", From: "C1.2", To: "GND1.reference", Connected: true},
		{ID: "W5", From: "V1.negative", To: "GND1.reference", Connected: true},
	}}
}

// FlybackExampleGraph uses explicit rail/drain/gate junctions and local
// ground symbols to keep the recirculation loop and drive return readable.
func FlybackExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		position        Point
		rotation        int
	}{
		{"VCC", "dc-source", "5 V", Point{-280, -150}, 0},
		{"VIN", "pulse-source", "5 V", Point{-480, 200}, 0},
		{"RG", "resistor", "100 Ω", Point{-160, 200}, 0},
		{"RPD", "resistor", "100k Ω", Point{0, 300}, 90},
		{"RL", "resistor", "100 Ω", Point{200, -170}, 90},
		{"D1", "diode", "", Point{400, -80}, 180},
		{"M1", "nmos", "", Point{200, 200}, 0},
		{"GND1", "ground", "", Point{200, 400}, 0},
		{"L1", "inductor", "1 H", Point{200, 10}, 90},
		{"GND2", "ground", "", Point{-280, -40}, 0},
		{"GND3", "ground", "", Point{-480, 320}, 0},
		{"JVP", "junction", "", Point{200, -280}, 0},
		{"JDR", "junction", "", Point{200, 110}, 0},
		{"JG", "junction", "", Point{0, 200}, 0},
		{"JD1", "junction", "", Point{400, -280}, 0},
		{"JD2", "junction", "", Point{400, 110}, 0},
	} {
		part, err := MakeComponent(p.id, p.kind, p.position, p.value)
		if err != nil {
			panic(err)
		}
		part.Rotation = p.rotation
		g.Components = append(g.Components, part)
	}
	for k, pins := range [][2]string{
		{"VCC.positive", "JVP.node"}, {"JVP.node", "RL.1"},
		{"RL.2", "L1.1"}, {"L1.2", "JDR.node"}, {"JDR.node", "M1.drain"},
		{"VIN.positive", "RG.1"}, {"RG.2", "JG.node"}, {"JG.node", "M1.gate"}, {"JG.node", "RPD.1"},
		{"RPD.2", "GND1.reference"}, {"M1.source", "GND1.reference"},
		{"VCC.negative", "GND2.reference"}, {"VIN.negative", "GND3.reference"},
		{"D1.anode", "JD2.node"}, {"JD2.node", "JDR.node"},
		{"D1.cathode", "JD1.node"}, {"JD1.node", "JVP.node"},
	} {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", k+1), From: pins[0], To: pins[1], Connected: true})
	}
	return g
}
