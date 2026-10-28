package workbench

func SCRLatchExampleGraph() Graph {
	b := exampleBuilder{}
	b.part("V1", "dc-source", "5", -400, 0, 0)
	b.part("RESET", "switch", "", -200, -150, 0)
	b.part("RL", "resistor", "330", 0, -150, 0)
	b.part("D1", "led", "", 200, -150, 0)
	b.part("SCR1", "scr", "", 200, 100, 0)
	b.part("TRIGGER", "pushbutton", "", -200, 100, 0)
	b.part("RG", "resistor", "1k", 0, 100, 0)
	b.part("RPD", "resistor", "100k", -50, 300, 0)
	b.ground("GND", "V1.negative", -400, 350)
	b.wire("V1.positive", "RESET.1")
	b.wire("RESET.2", "RL.1")
	b.wire("RL.2", "D1.anode")
	b.wire("D1.cathode", "SCR1.anode")
	b.wire("SCR1.cathode", "GND.reference")
	b.wire("V1.positive", "TRIGGER.1")
	b.wire("TRIGGER.2", "RG.1")
	b.wire("RG.2", "SCR1.gate")
	b.wire("RPD.1", "SCR1.gate")
	b.wire("RPD.2", "GND.reference")
	for k, p := range b.graph.Components {
		if p.ID == "RESET" {
			closed := true
			b.graph.Components[k].SwitchClosed = &closed
		}
	}
	return b.graph
}
func TRIACPhaseExampleGraph() Graph {
	b := exampleBuilder{}
	b.part("VAC", "sine-source", "8", -350, -100, 0)
	b.part("RL", "resistor", "100", 0, -100, 0)
	b.part("TR1", "triac", "", 200, 100, 0)
	b.part("VG", "sine-source", "5", -350, 250, 0)
	b.part("DG", "diode", "", -200, 150, 270)
	b.part("RG", "resistor", "1k", 0, 150, 0)
	b.ground("GND", "VAC.negative", -350, 450)
	b.wire("VAC.positive", "RL.1")
	b.wire("RL.2", "TR1.mt2")
	b.wire("TR1.mt1", "GND.reference")
	b.wire("VG.negative", "GND.reference")
	b.wire("VG.positive", "DG.anode")
	b.wire("DG.cathode", "RG.1")
	b.wire("RG.2", "TR1.gate")
	for k, p := range b.graph.Components {
		if p.ID == "VAC" {
			b.graph.Components[k].Sine = &SineWaveform{Amplitude: 8, Frequency: 50}
		}
		if p.ID == "VG" {
			b.graph.Components[k].Sine = &SineWaveform{Amplitude: 5, Offset: -3, Frequency: 100, Phase: -90}
		}
	}
	return b.graph
}
