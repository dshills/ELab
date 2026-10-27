package workbench

func FeedbackLoopExampleGraph() Graph {
	g := OpAmpFollowerExampleGraph()
	for k, p := range g.Components {
		if p.ID == "U1" {
			g.Components[k].Device = &DeviceParameters{Dynamics: &OpAmpDynamics{GBW: 1000, Slew: 1e6}}
		}
		if p.ID == "VIN" {
			g.Components[k].Sine.Amplitude = .1
			g.Components[k].Value = .1
		}
	}
	b := exampleBuilder{graph: g}
	b.part("CL", "capacitor", "1u", 400, 330, 0)
	b.wire("U1.out", "CL.1")
	b.ground("GC", "CL.2", 400, 500)
	return b.graph
}

func StepResponseExampleGraph() Graph {
	g := ACRCExampleGraph()
	for k, p := range g.Components {
		if p.ID == "V1" {
			g.Components[k].Value = 5
		}
	}
	return g
}
