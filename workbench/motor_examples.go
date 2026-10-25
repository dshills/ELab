package workbench

func MotorExampleGraph(pwm bool) Graph {
	if !pwm {
		return motorGraphExample()
	}
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -800, -300, 0)
	b.ground("GNDV", "VCC.negative", -800, -100)
	b.label("PWR", "VCC", "VCC.positive", -800, -450)
	b.part("U1", "timer555", "", -200, 0, 0)
	b.label("PV", "VCC", "U1.vcc", -200, -300)
	b.label("PR", "VCC", "U1.reset", -370, -300)
	b.ground("GNDT", "U1.gnd", -200, 250)
	b.part("RA", "resistor", "1k", -600, -180, 90)
	b.label("PA", "VCC", "RA.1", -600, -340)
	b.wire("RA.2", "U1.discharge")
	b.part("P1", "potentiometer", "10k", -600, 110, 90)
	b.wire("P1.1", "U1.discharge")
	b.wire("P1.wiper", "U1.threshold")
	b.wire("P1.2", "U1.threshold")
	b.wire("U1.trigger", "U1.threshold")
	b.part("C1", "capacitor", "1u", -450, 360, 0)
	b.wire("C1.1", "U1.threshold")
	b.ground("GNDC", "C1.2", -450, 520)
	b.part("RG", "resistor", "1k", 80, 0, 0)
	b.wire("U1.out", "RG.1")
	b.part("M1", "nmos", "", 300, 0, 0)
	b.wire("RG.2", "M1.gate")
	b.ground("GNDM", "M1.source", 330, 200)
	b.part("VLOAD", "dc-source", "12", 750, -420, 0)
	b.ground("GNDL", "VLOAD.negative", 750, -240)
	b.part("MTR1", "dc-motor", "", 500, -100, 0)
	b.wire("VLOAD.positive", "MTR1.positive")
	b.wire("MTR1.negative", "M1.drain")
	b.part("D1", "diode", "", 650, -100, 180)
	b.wire("D1.anode", "MTR1.negative")
	b.wire("D1.cathode", "MTR1.positive")
	b.part("RSN", "resistor", "100", 900, -100, 90)
	b.wire("MTR1.positive", "RSN.1")
	b.part("CSN", "capacitor", "1u", 900, 100, 0)
	b.wire("RSN.2", "CSN.1")
	b.wire("CSN.2", "MTR1.negative")
	return b.graph
}
func motorGraphExample() Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "12", -250, 0, 0)
	b.ground("GND", "V1.negative", -250, 200)
	b.part("MTR1", "dc-motor", "", 150, 0, 0)
	b.wire("V1.positive", "MTR1.positive")
	b.ground("GNDM", "MTR1.negative", 150, 200)
	return b.graph
}
