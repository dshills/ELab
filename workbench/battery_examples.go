package workbench

func BatteryExampleGraph(regulator bool) Graph {
	b := newExampleBuilder()
	b.part("B1", "battery", "9", -400, 0, 0)
	b.graph.Components[0].Power = &PowerParameters{Battery: &BatteryConfig{.00002, 120, 1}}
	b.ground("GND", "B1.negative", -400, 200)
	if regulator {
		b.graph.Components[0].Power.Battery = &BatteryConfig{.0002, 10, 1}
		b.part("U1", "linear-regulator", "5", 0, 0, 0)
		b.wire("B1.positive", "U1.in")
		b.ground("GNDU", "U1.gnd", 0, 190)
		b.part("RL", "resistor", "100", 300, 0, 90)
		b.wire("U1.out", "RL.1")
		b.ground("GNDL", "RL.2", 300, 220)
		b.part("C1", "capacitor", "10u", 120, 280, 0)
		b.wire("C1.1", "U1.out")
		b.wire("C1.2", "GND.reference")
	} else {
		b.part("R1", "resistor", "1k", -80, 0, 0)
		b.wire("B1.positive", "R1.1")
		b.part("D1", "led", "", 160, 0, 270)
		b.wire("R1.2", "D1.anode")
		b.ground("GNDD", "D1.cathode", 300, 170)
		b.part("S1", "switch", "", -220, -180, 0)
		b.wire("B1.positive", "S1.1")
		b.part("RLOAD", "resistor", "100", 0, -180, 0)
		b.wire("S1.2", "RLOAD.1")
		b.wire("RLOAD.2", "GND.reference")
	}
	return b.graph
}
