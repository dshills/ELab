package workbench

func CurrentLEDExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("I1", "current-source", "", -220, 0, 0)
	b.part("D1", "led", "", 220, 0, 0)
	b.wire("I1.positive", "D1.anode")
	b.ground("GI", "I1.negative", -220, 180)
	b.ground("GD", "D1.cathode", 220, 180)
	for i, p := range b.graph.Components {
		if p.ID == "I1" {
			v := DefaultCurrentConfig()
			v.Timing.Pulsed = true
			v.Low = .002
			v.Current = .015
			b.graph.Components[i].Device = &DeviceParameters{Current: &v}
		}
	}
	return b.graph
}
func ElectronicLoadExampleGraph(power bool) Graph {
	b := newExampleBuilder()
	b.part("V1", "dc-source", "5", -400, 0, 0)
	b.ground("GV", "V1.negative", -400, 180)
	b.part("RS", "resistor", "10", 0, -100, 90)
	b.wire("V1.positive", "RS.1")
	b.part("LOAD1", "electronic-load", "", 380, 0, 0)
	b.wire("RS.2", "LOAD1.positive")
	b.ground("GL", "LOAD1.negative", 380, 180)
	b.part("C1", "capacitor", "10u", 150, 160, 0)
	b.wire("LOAD1.positive", "C1.1")
	b.ground("GC", "C1.2", 150, 350)
	for i, p := range b.graph.Components {
		if p.ID == "LOAD1" {
			v := DefaultLoadConfig()
			v.Level = .08
			v.Low = .01
			v.Timing.Pulsed = true
			if power {
				v.Mode = "cp"
				v.Level = .25
				v.Low = .05
			}
			b.graph.Components[i].Device = &DeviceParameters{Load: &v}
		}
	}
	return b.graph
}
