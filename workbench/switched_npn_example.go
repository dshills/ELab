package workbench

// SwitchedNPNExampleGraph adds a manual base switch and pull-down to the
// transistor LED fixture. The load current still flows through the collector.
func SwitchedNPNExampleGraph() Graph {
	g := NPNExampleGraph()
	for i := range g.Components {
		if g.Components[i].ID == "VIN" {
			g.Components[i].Position = Point{-360, 60}
		}
		if g.Components[i].ID == "GND1" {
			g.Components[i].Position = Point{100, 240}
		}
	}
	switchPart, _ := MakeComponent("S1", "switch", Point{-240, 60}, "")
	pullDown, _ := MakeComponent("RPD", "resistor", Point{-40, 180}, "100k")
	pullDown.Rotation = 90
	g.Components = append(g.Components, switchPart, pullDown)
	for i := range g.Wires {
		if g.Wires[i].From == "VIN.positive" {
			g.Wires[i].To = "S1.1"
		}
	}
	g.Wires = append(g.Wires,
		Wire{ID: "WS", From: "S1.2", To: "RB.1", Connected: true},
		Wire{ID: "WPD1", From: "Q1.base", To: "RPD.1", Connected: true},
		Wire{ID: "WPD2", From: "RPD.2", To: "GND1.reference", Connected: true},
	)
	return g
}
