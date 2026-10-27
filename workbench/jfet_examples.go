package workbench

func JFETCurrentExampleGraph(pchannel bool) Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "9", -420, -40, 0)
	b.ground("GV", "V1.negative", -420, 150)
	kind := "njfet"
	if pchannel {
		kind = "pjfet"
	}
	b.part("J1", kind, "", 100, 0, 0)
	b.part("RSET", "resistor", "220", 100, 180, 0)
	b.part("RL", "resistor", "470", 100, -210, 0)
	if !pchannel {
		b.wire("V1.positive", "RL.1")
		b.wire("RL.2", "J1.drain")
		b.wire("J1.source", "RSET.1")
		b.ground("GS", "RSET.2", 100, 360)
		b.ground("GG", "J1.gate", -120, 110)
	} else {
		b.wire("V1.positive", "RSET.1")
		b.wire("RSET.2", "J1.source")
		b.wire("V1.positive", "J1.gate")
		b.wire("J1.drain", "RL.1")
		b.ground("GL", "RL.2", 350, 100)
	}
	return b.graph
}
func JFETPreampExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "9", -550, -250, 0)
	b.ground("GV", "VCC.negative", -550, -80)
	b.part("VIN", "sine-source", ".02", -550, 220, 0)
	b.graph.Components[len(b.graph.Components)-1].Sine = &SineWaveform{Amplitude: .02, Frequency: 100}
	b.ground("GI", "VIN.negative", -550, 400)
	b.part("RG", "resistor", "100k", -220, 150, 90)
	b.wire("VIN.positive", "RG.1")
	b.part("J1", "njfet", "", 100, 20, 0)
	b.wire("RG.2", "J1.gate")
	b.part("RS", "resistor", "470", 100, 220, 0)
	b.wire("J1.source", "RS.1")
	b.ground("GS", "RS.2", 100, 410)
	b.part("RD", "resistor", "1k", 100, -200, 0)
	b.wire("VCC.positive", "RD.1")
	b.wire("RD.2", "J1.drain")
	b.part("COUT", "capacitor", "1u", 420, -30, 90)
	b.wire("J1.drain", "COUT.1")
	b.part("RL", "resistor", "10k", 670, 180, 0)
	b.wire("COUT.2", "RL.1")
	b.ground("GL", "RL.2", 670, 400)
	return b.graph
}
