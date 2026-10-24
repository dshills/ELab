package workbench

func RegulatorSupplyExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "9", -600, 0, 0)
	b.ground("GNDV", "V1.negative", -600, 220)
	b.part("U1", "linear-regulator", "5", 0, 0, 0)
	b.part("RIN", "resistor", "10", -420, -120, 0)
	b.wire("V1.positive", "RIN.1")
	b.wire("RIN.2", "U1.in")
	b.ground("GNDU", "U1.gnd", 0, 240)
	b.part("CIN", "capacitor", "10u", -300, 140, 0)
	b.wire("U1.in", "CIN.1")
	b.ground("GNDI", "CIN.2", -300, 320)
	b.part("COUT", "capacitor", "10u", 300, 140, 0)
	b.wire("U1.out", "COUT.1")
	b.ground("GNDO", "COUT.2", 300, 320)
	b.part("R1", "resistor", "1k", 580, 0, 90)
	b.wire("U1.out", "R1.1")
	b.ground("GND1", "R1.2", 580, 240)
	b.part("S1", "switch", "", 800, 0, 90)
	b.wire("U1.out", "S1.1")
	b.part("RLOAD", "resistor", "100", 800, 200, 90)
	b.wire("S1.2", "RLOAD.1")
	b.ground("GNDL", "RLOAD.2", 800, 400)
	return b.graph
}
func RegulatorDropoutExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VIN", "sine-source", "2.5", -600, 0, 0)
	b.graph.Components[0].Sine = &SineWaveform{Amplitude: 2.5, Offset: 6.5, Frequency: 1}
	b.ground("GNDV", "VIN.negative", -600, 200)
	for k, kind := range []string{"linear-regulator", "ldo-regulator"} {
		id := "U1"
		r := "R1"
		ground := "GND1"
		x, y := 0., -200.
		if k == 1 {
			id = "U2"
			r = "R2"
			ground = "GND2"
			y = 220
		}
		b.part(id, kind, "5", x, y, 0)
		b.wire("VIN.positive", id+".in")
		b.ground(ground, id+".gnd", x, y+150)
		b.part(r, "resistor", "1k", 380, y, 90)
		b.wire(id+".out", r+".1")
		b.ground(ground+"R", r+".2", 380, y+150)
	}
	return b.graph
}
