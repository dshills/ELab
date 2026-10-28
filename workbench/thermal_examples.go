package workbench

func SelfHeatingExampleGraph() Graph {
	b := exampleBuilder{}
	b.part("V1", "dc-source", "5", -200, 0, 0)
	b.part("R1", "resistor", "100", 200, 0, 0)
	b.wire("V1.positive", "R1.1")
	b.ground("GND", "V1.negative", -200, 200)
	b.wire("R1.2", "GND.reference")
	cfg := DefaultThermalConfig("resistor")
	cfg.Alpha = 0
	cfg.SelfHeating = true
	cfg.Resistance = 100
	cfg.Capacitance = .001
	for k, p := range b.graph.Components {
		if p.ID == "R1" {
			b.graph.Components[k].Thermal = &cfg
		}
	}
	return b.graph
}
func ThermalClassABExampleGraph() Graph {
	g := DiscreteClassABExampleGraph()
	for k, p := range g.Components {
		if p.ID == "Q6" || p.ID == "Q7" {
			cfg := DefaultThermalConfig(p.Kind)
			g.Components[k].Thermal = &cfg
		}
	}
	return g
}
