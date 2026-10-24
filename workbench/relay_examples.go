package workbench

func RelayExampleGraph(manual bool) Graph {
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -700, -250, 0)
	b.ground("GNDV", "VCC.negative", -700, -80)
	b.part("K1", "relay", "100mH", 0, 0, 0)
	b.wire("VCC.positive", "K1.coil1")
	b.part("Q1", "npn", "", -100, 300, 0)
	b.wire("K1.coil2", "Q1.collector")
	b.ground("GNDQ", "Q1.emitter", -70, 470)
	b.part("RB", "resistor", "4.7k", -360, 300, 0)
	b.wire("RB.2", "Q1.base")
	b.part("RBE", "resistor", "100k", -300, 480, 90)
	b.wire("RBE.1", "Q1.base")
	b.wire("RBE.2", "GNDQ.reference")
	b.part("D2", "diode", "", -320, -60, 0)
	b.wire("K1.coil2", "D2.anode")
	b.wire("K1.coil1", "D2.cathode")
	if manual {
		b.part("S1", "pushbutton", "", -650, 240, 0)
		b.wire("VCC.positive", "S1.1")
		b.wire("S1.2", "RB.1")
	} else {
		b.part("VIN", "finite-pulse-source", "5", -650, 260, 0)
		b.graph.Components[len(b.graph.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .02, LowSeconds: .02, RiseSeconds: .001, FallSeconds: .001}
		b.wire("VIN.positive", "RB.1")
		b.ground("GNDI", "VIN.negative", -650, 490)
	}
	b.part("VLOAD", "dc-source", "9", 350, -290, 0)
	b.ground("GNDL", "VLOAD.negative", 350, -130)
	b.wire("VLOAD.positive", "K1.common")
	b.part("RL", "resistor", "1k", 420, 0, 0)
	b.part("D1", "led", "", 650, 0, 270)
	b.wire("K1.no", "RL.1")
	b.wire("RL.2", "D1.anode")
	b.ground("GNDD", "D1.cathode", 810, 100)
	b.part("RNC", "resistor", "10k", 400, 220, 0)
	b.wire("K1.nc", "RNC.1")
	b.ground("GNDNC", "RNC.2", 600, 320)
	return b.graph
}
