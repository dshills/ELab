package workbench

// LEDFadeExampleGraph places a capacitor across an LED after its current
// limiter. Pulse-source low discharges it through the resistor and the LED.
func LEDFadeExampleGraph() Graph {
	g := ExampleGraph()
	g.Components[0], _ = MakeComponent("V1", "pulse-source", Point{-230, 20}, "5 V")
	for i := range g.Components {
		if g.Components[i].ID == "D1" {
			g.Components[i].Position = Point{X: 120, Y: 20}
		}
	}
	cap, _ := MakeComponent("C1", "capacitor", Point{X: 330, Y: 20}, "100uF")
	g.Components = append(g.Components, cap)
	g.Wires = append(g.Wires, Wire{ID: "W5", From: "D1.anode", To: "C1.1", Connected: true}, Wire{ID: "W6", From: "C1.2", To: "GND.reference", Connected: true})
	return g
}
