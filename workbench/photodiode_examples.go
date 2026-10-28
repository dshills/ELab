package workbench

func PhotodiodeLoadExampleGraph() Graph {
	b := exampleBuilder{}
	b.part("V1", "dc-source", "5", -200, 0, 0)
	b.part("RL", "resistor", "100k", 0, 0, 0)
	b.part("PD1", "photodiode", "", 200, 0, 0)
	b.wire("V1.positive", "RL.1")
	b.wire("RL.2", "PD1.cathode")
	b.ground("GND", "V1.negative", -200, 200)
	b.wire("PD1.anode", "GND.reference")
	for k, p := range b.graph.Components {
		if p.ID == "PD1" {
			b.graph.Components[k].Device.Photo.Capacitance = 10e-9
		}
	}
	return b.graph
}
func PhotodiodeTIAExampleGraph() Graph {
	b := exampleBuilder{}
	b.part("VP", "dc-source", "5", -400, -200, 0)
	b.part("VN", "dc-source", "5", -400, 200, 0)
	b.part("PD1", "photodiode", "", -200, 0, 0)
	b.part("U1", "opamp", "", 100, 0, 0)
	b.part("RF", "resistor", "100k", 100, -200, 0)
	b.part("CF", "capacitor", "100p", 100, -350, 0)
	b.part("RL", "resistor", "10k", 400, 0, 0)
	b.ground("GND", "VP.negative", -400, 400)
	b.wire("VN.positive", "GND.reference")
	b.wire("U1.vplus", "VP.positive")
	b.wire("U1.vminus", "VN.negative")
	b.wire("U1.noninverting", "GND.reference")
	b.wire("PD1.anode", "GND.reference")
	b.wire("PD1.cathode", "U1.inverting")
	b.wire("RF.1", "U1.out")
	b.wire("RF.2", "U1.inverting")
	b.wire("CF.1", "U1.out")
	b.wire("CF.2", "U1.inverting")
	b.wire("RL.1", "U1.out")
	b.wire("RL.2", "GND.reference")
	for k, p := range b.graph.Components {
		if p.ID == "PD1" {
			b.graph.Components[k].Device.Photo.Light = 20
		}
	}
	return b.graph
}
