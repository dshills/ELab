package workbench

import "strconv"

// PNPExampleGraph switches the positive supply. A low base drive lights D1.
func PNPExampleGraph() Graph {
	g := EmptyGraph()
	for _, p := range []struct {
		id, kind, value string
		at              Point
		rotation        int
	}{
		{"VCC", "dc-source", "5 V", Point{-300, -260}, 0},
		{"VIN", "pulse-source", "5 V", Point{-300, 100}, 0},
		{"RB", "resistor", "10k", Point{-110, -100}, 0},
		{"RPU", "resistor", "100k", Point{-40, -240}, 90},
		{"Q1", "pnp", "", Point{100, -100}, 0},
		{"RL", "resistor", "1k", Point{130, 30}, 90},
		{"D1", "led", "", Point{130, 140}, 0},
		{"GND1", "ground", "", Point{130, 250}, 0},
		{"GNDV", "ground", "", Point{-440, -150}, 0},
		{"GNDIN", "ground", "", Point{-300, 240}, 0},
	} {
		part, err := MakeComponent(p.id, p.kind, p.at, p.value)
		if err != nil {
			panic(err)
		}
		part.Rotation = p.rotation
		g.Components = append(g.Components, part)
	}
	for i, p := range [][2]string{
		{"VCC.positive", "Q1.emitter"}, {"VCC.positive", "RPU.1"},
		{"RPU.2", "Q1.base"}, {"VIN.positive", "RB.1"}, {"RB.2", "Q1.base"},
		{"Q1.collector", "RL.1"}, {"RL.2", "D1.anode"}, {"D1.cathode", "GND1.reference"},
		{"VCC.negative", "GNDV.reference"}, {"VIN.negative", "GNDIN.reference"},
	} {
		g.Wires = append(g.Wires, Wire{ID: "W" + strconv.Itoa(i+1), From: p[0], To: p[1], Connected: true})
	}
	return g
}

func EmitterFollowerExampleGraph() Graph      { return bipolarFollowerExample(false) }
func ComplementaryBipolarExampleGraph() Graph { return bipolarFollowerExample(true) }

// The complementary version uses split ±5 V rails and one resistor at each
// base. The unbiased crossover dead band is intentional and visible at 10 Hz.
func bipolarFollowerExample(complementary bool) Graph {
	g := EmptyGraph()
	add := func(id, kind, value string, at Point, rotation int) {
		part, err := MakeComponent(id, kind, at, value)
		if err != nil {
			panic(err)
		}
		part.Rotation = rotation
		g.Components = append(g.Components, part)
	}
	add("VIN", "sine-source", "1 V", Point{-300, 0}, 0)
	g.Components[0].Sine = &SineWaveform{Amplitude: 1, Offset: 2.5, Frequency: 10}
	add("VCC", "dc-source", "5 V", Point{420, -160}, 0)
	add("Q1", "npn", "", Point{80, -90}, 0)
	add("RB", "resistor", "1k", Point{-100, -90}, 0)
	add("RL", "resistor", "1k", Point{250, 100}, 90)
	add("JO", "junction", "", Point{250, 0}, 0)
	add("GNDIN", "ground", "", Point{-300, 100}, 0)
	add("GNDL", "ground", "", Point{250, 200}, 0)
	add("GNDP", "ground", "", Point{420, -60}, 0)
	connect := func(one, two string) {
		g.Wires = append(g.Wires, Wire{ID: "W" + strconv.Itoa(len(g.Wires)+1), From: one, To: two, Connected: true})
	}
	for _, ends := range [][2]string{
		{"VIN.positive", "RB.1"}, {"RB.2", "Q1.base"},
		{"Q1.collector", "VCC.positive"}, {"Q1.emitter", "JO.node"},
		{"JO.node", "RL.1"}, {"RL.2", "GNDL.reference"},
		{"VIN.negative", "GNDIN.reference"}, {"VCC.negative", "GNDP.reference"},
	} {
		connect(ends[0], ends[1])
	}
	if complementary {
		g.Components[0].Value = 2.5
		g.Components[0].Sine = &SineWaveform{Amplitude: 2.5, Frequency: 10}
		// Leave the negative-supply caption clear of the output load/ground.
		for i := range g.Components {
			if g.Components[i].ID == "VCC" || g.Components[i].ID == "GNDP" {
				g.Components[i].Position.X = 610
			}
		}
		add("Q2", "pnp", "", Point{80, 90}, 0)
		add("RB2", "resistor", "1k", Point{-100, 90}, 0)
		add("VEE", "dc-source", "5 V", Point{610, 160}, 0)
		add("GNDN", "ground", "", Point{610, 60}, 180)
		for _, ends := range [][2]string{
			{"VIN.positive", "RB2.1"}, {"RB2.2", "Q2.base"},
			{"Q2.emitter", "JO.node"}, {"Q2.collector", "VEE.negative"},
			{"VEE.positive", "GNDN.reference"},
		} {
			connect(ends[0], ends[1])
		}
	}
	return g
}
