package workbench

// DiscreteClassABExampleGraph is a closed-loop, entirely discrete amplifier.
// Q1/Q2 are the differential input, Q3/Q4 its mirror load, Q5 the voltage-gain
// stage, Q6/Q7 the complementary output, and Q8 the tail-current sink.
// All devices retain their published generic models and work budgets.
func DiscreteClassABExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VIN", "sine-source", "0.1", -1030, -180, 0)
	b.graph.Components[0].Sine = &SineWaveform{Amplitude: .1, Frequency: 10}
	b.ground("GI", "VIN.negative", -1030, -60)
	b.label("LI", "INPUT", "VIN.positive", -1030, -300)
	b.part("VP", "dc-source", "9", 1100, 420, 0)
	b.part("VN", "dc-source", "9", 1100, 620, 0)
	b.ground("GS", "VP.negative", 1100, 520)
	b.wire("VN.positive", "GS.reference")
	b.label("LP", "VCC", "VP.positive", 1100, 300)
	b.label("LN", "VEE", "VN.negative", 1100, 740)

	b.part("Q1", "npn", "", -600, 0, 0)
	b.part("Q2", "npn", "", -240, 0, 0)
	b.part("Q3", "pnp", "", -600, -260, 0)
	b.part("Q4", "pnp", "", -240, -260, 0)
	b.label("PF1", "VCC", "Q3.emitter", -575, -390)
	b.label("PF2", "VCC", "Q4.emitter", -215, -390)
	b.wire("Q3.collector", "Q3.base")
	b.wire("Q3.base", "Q4.base")
	b.wire("Q1.collector", "Q3.collector")
	b.wire("Q2.collector", "Q4.collector")
	b.part("RDI1", "resistor", "100", -575, 100, 90)
	b.part("RDI2", "resistor", "100", -215, 100, 90)
	b.wire("Q1.emitter", "RDI1.1")
	b.wire("Q2.emitter", "RDI2.1")
	b.part("JT", "junction", "", -420, 190, 0)
	b.wire("RDI1.2", "JT.node")
	b.wire("RDI2.2", "JT.node")
	b.label("IF", "INPUT", "Q2.base", -330, -80)
	b.label("FB", "FEEDBACK", "Q1.base", -850, 0)

	b.part("Q8", "npn", "", -420, 280, 0)
	b.wire("Q8.collector", "JT.node")
	b.part("RT", "resistor", "680", -395, 430, 90)
	b.wire("Q8.emitter", "RT.1")
	b.label("NT", "VEE", "RT.2", -395, 550)
	b.part("RREF", "resistor", "2.2k", -900, 130, 90)
	b.ground("GR", "RREF.1", -900, 10)
	b.part("D3", "diode", "", -900, 300, 0)
	b.part("D4", "diode", "", -900, 430, 0)
	b.wire("RREF.2", "D3.anode")
	b.wire("D3.cathode", "D4.anode")
	b.wire("D3.anode", "Q8.base")
	b.label("ND", "VEE", "D4.cathode", -900, 550)

	b.part("Q5", "pnp", "", 100, -90, 0)
	b.wire("Q2.collector", "Q5.base")
	b.part("RV", "resistor", "47", 125, -260, 90)
	b.wire("RV.2", "Q5.emitter")
	b.label("PV", "VCC", "RV.1", 125, -390)
	b.part("D1", "diode", "", 300, 30, 0)
	b.part("D2", "diode", "", 300, 150, 0)
	b.wire("Q5.collector", "D1.anode")
	b.wire("D1.cathode", "D2.anode")
	b.part("RPULL", "resistor", "3.3k", 300, 390, 90)
	b.wire("D2.cathode", "RPULL.1")
	b.label("NP", "VEE", "RPULL.2", 300, 530)

	b.part("Q6", "npn", "", 570, -60, 0)
	b.part("Q7", "pnp", "", 570, 240, 0)
	b.wire("D1.anode", "Q6.base")
	b.wire("D2.cathode", "Q7.base")
	b.label("PO", "VCC", "Q6.collector", 595, -200)
	b.label("NO", "VEE", "Q7.collector", 595, 390)
	b.part("RE1", "resistor", "22", 730, 20, 90)
	b.part("RE2", "resistor", "22", 730, 160, 90)
	b.wire("Q6.emitter", "RE1.1")
	b.wire("Q7.emitter", "RE2.2")
	b.part("JO", "junction", "", 860, 90, 0)
	b.wire("RE1.2", "JO.node")
	b.wire("RE2.1", "JO.node")
	b.label("LO", "OUTPUT", "JO.node", 980, 90)
	b.part("RL", "resistor", "100", 1010, 190, 90)
	b.wire("JO.node", "RL.1")
	b.ground("GL", "RL.2", 1010, 320)

	b.part("RF", "resistor", "1k", -650, 690, 0)
	b.label("FO", "OUTPUT", "RF.2", -500, 690)
	b.label("FF", "FEEDBACK", "RF.1", -1000, 690)
	b.part("RG", "resistor", "100", -1000, 830, 90)
	b.wire("RF.1", "RG.1")
	b.ground("GF", "RG.2", -1000, 960)

	// An explicit biased initial condition starts the compensation capacitor
	// near the zero-input operating point, rather than claiming a power-on test.
	b.part("C1", "capacitor", "1n", 280, 690, 0)
	initial := 7.55
	b.graph.Components[len(b.graph.Components)-1].InitialVoltage = &initial
	b.label("VB", "VAS_BASE", "Q5.base", 0, -170)
	b.label("VD", "DRIVE", "Q5.collector", 190, -120)
	b.label("CVB", "VAS_BASE", "C1.1", 280, 570)
	b.label("CVD", "DRIVE", "C1.2", 280, 810)
	return b.graph
}
