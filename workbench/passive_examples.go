package workbench

func LossyCapacitorExampleGraph() Graph {
	g := RCExampleGraph()
	for i, p := range g.Components {
		if p.Kind == "capacitor" {
			g.Components[i].Device = &DeviceParameters{Passive: &PassiveConfig{Version: PassiveLossModelID, ESR: 1000, Leakage: 1e5}}
		}
	}
	return g
}
func LossyWindingExampleGraph() Graph {
	g := RLCExampleGraph()
	for i, p := range g.Components {
		if p.Kind == "inductor" {
			g.Components[i].Device = &DeviceParameters{Passive: &PassiveConfig{Version: PassiveLossModelID, Winding: 20}}
		}
	}
	return g
}
