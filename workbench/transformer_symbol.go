package workbench

import "math"

func TransformerTerminalLocal(pin string) (Point, bool) {
	switch pin {
	case "primary1":
		return Point{-70, -50}, true
	case "primary2":
		return Point{-70, 50}, true
	case "secondary1":
		return Point{70, -50}, true
	case "secondary2":
		return Point{70, 50}, true
	}
	return Point{}, false
}
func TransformerSymbolPaths(reversed bool) [][]Point {
	out := [][]Point{{{-70, -50}, {-25, -50}, {-25, -40}}, {{-70, 50}, {-25, 50}, {-25, 40}}, {{70, -50}, {25, -50}, {25, -40}}, {{70, 50}, {25, 50}, {25, 40}}, {{-4, -44}, {-4, 44}}, {{4, -44}, {4, 44}}}
	for _, side := range []float64{-1, 1} {
		for turn := 0; turn < 5; turn++ {
			path := []Point{}
			for j := 0; j <= 12; j++ {
				a := -math.Pi/2 + float64(j)*math.Pi/12
				path = append(path, Point{side*25 - side*8*math.Cos(a), float64(turn)*16 - 32 + 8*math.Sin(a)})
			}
			out = append(out, path)
		}
	}
	dot := func(x, y float64) {
		p := []Point{}
		for j := 0; j <= 16; j++ {
			a := float64(j) * 2 * math.Pi / 16
			p = append(p, Point{x + 2.5*math.Cos(a), y + 2.5*math.Sin(a)})
		}
		out = append(out, p)
	}
	dot(-39, -38)
	y := -38.
	if reversed {
		y = 38
	}
	dot(39, y)
	return out
}
func TransformerExampleGraph(supply bool) Graph {
	b := newExampleBuilder()
	b.part("VIN", "sine-source", "12", -500, 0, 0)
	b.graph.Components[0].Sine.Frequency = 50
	b.graph.Components[0].Sine.Offset = 0
	b.graph.Components[0].Sine.Amplitude = 12
	b.graph.Components[0].Value = 12
	b.ground("GNDIN", "VIN.negative", -500, 240)
	b.part("T1", "transformer", "", -100, 0, 0)
	b.wire("VIN.positive", "T1.primary1")
	b.ground("GNDP", "T1.primary2", -230, 240)
	if !supply {
		b.part("RL", "resistor", "100", 200, 0, 90)
		b.wire("T1.secondary1", "RL.1")
		b.wire("T1.secondary2", "RL.2")
		b.ground("GNDS", "T1.secondary2", 20, 240)
		return b.graph
	}
	c := b.graph.Components[2].Power.Transformer
	c.Ratio = .75
	c.Inductance = 1
	c.PrimaryResistance = 10
	c.SecondaryResistance = 10
	b.label("LA", "AC_A", "T1.secondary1", 40, -140)
	b.label("LB", "AC_B", "T1.secondary2", 40, 140)
	b.part("LBA", "net-label", "AC_A", 400, 0, 0)
	b.part("LBB", "net-label", "AC_B", 800, 0, 0)
	b.part("D1", "diode", "", 400, -120, 180)
	b.part("D2", "diode", "", 800, -120, 180)
	b.part("D3", "diode", "", 400, 120, 180)
	b.part("D4", "diode", "", 800, 120, 180)
	b.part("JP", "junction", "", 1000, -240, 0)
	b.part("JN", "junction", "", 1000, 240, 0)
	for _, w := range [][2]string{{"LBA.node", "D1.anode"}, {"D1.cathode", "JP.node"}, {"LBB.node", "D2.anode"}, {"D2.cathode", "JP.node"}, {"JN.node", "D3.anode"}, {"D3.cathode", "LBA.node"}, {"JN.node", "D4.anode"}, {"D4.cathode", "LBB.node"}} {
		b.wire(w[0], w[1])
	}
	b.ground("GNDB", "JN.node", 1000, 430)
	b.part("RREF", "resistor", "100k", 600, 420, 90)
	b.label("LRR", "AC_B", "RREF.1", 600, 280)
	b.ground("GNDR", "RREF.2", 600, 590)
	b.part("C1", "capacitor", "1m", 1200, 0, 0)
	b.wire("C1.1", "JP.node")
	b.wire("C1.2", "JN.node")
	b.part("U1", "ldo-regulator", "3.3", 1500, -240, 0)
	b.wire("U1.in", "JP.node")
	b.ground("GNDU", "U1.gnd", 1500, 240)
	b.part("RLOAD", "resistor", "1k", 1750, 0, 90)
	b.wire("U1.out", "RLOAD.1")
	b.ground("GNDL", "RLOAD.2", 1750, 240)
	b.part("RSN", "resistor", "100", 600, -450, 0)
	b.part("CSN", "capacitor", "10u", 900, -450, 0)
	b.wire("RSN.2", "CSN.1")
	b.label("LSA", "AC_A", "RSN.1", 545, -600)
	b.label("LSB", "AC_B", "CSN.2", 900, -300)
	return b.graph
}
