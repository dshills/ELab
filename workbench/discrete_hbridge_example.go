package workbench

import "strings"

// Each half-bridge uses two real MOSFETs with common drains. The inverter
// drives the opposite half from the same direction pulse.
func discreteHBridgeExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -760, -180, 0)
	b.ground("GNDS", "VCC.negative", -760, -40)
	b.label("SUPPLY", "VDD", "VCC.positive", -760, -330)
	b.part("DIR", "pulse-source", "5", -760, 220, 0)
	b.ground("GNDD", "DIR.negative", -760, 360)
	b.part("U1", "logic-not", "", -490, 130, 0)
	b.part("JDIR", "junction", "", -625, 130, 0)
	b.wire("DIR.positive", "JDIR.node")
	b.wire("JDIR.node", "U1.a")
	b.label("CTRL", "A", "JDIR.node", -625, 25)
	b.label("INV", "B", "U1.out", -340, 130)
	b.label("LOGICV", "VDD", "U1.vcc", -490, -20)
	b.ground("GNDI", "U1.gnd", -490, 285)
	b.part("RL", "resistor", "100", 160, 0, 0)
	for i, side := range []string{"A", "B"} {
		x := float64(i)*440 - 60
		high, low, junction := "MP"+side, "MN"+side, "J"+side
		b.part(high, "pmos", "", x, -145, 180)
		b.part(low, "nmos", "", x, 145, 0)
		b.label("V"+side, "VDD", high+".source", x, -300)
		b.ground("GND"+side, low+".source", x, 300)
		b.label("P"+side, side, high+".gate", x+115, -145)
		b.label("N"+side, side, low+".gate", x-115, 145)
		b.part(junction, "junction", "", x, 0, 0)
		b.wire(high+".drain", junction+".node")
		b.wire(low+".drain", junction+".node")
		pin := "RL.1"
		if i == 1 {
			pin = "RL.2"
		}
		b.wire(junction+".node", pin)
	}
	return b.graph
}

// The lower bridge has its own direction waveform and load. VDD and ground
// are common to both bridges; the two pairs of gate signals are independent.
func dualHBridgeExampleGraph() Graph {
	first := discreteHBridgeExampleGraph()
	b := exampleBuilder{graph: first.Clone()}
	for _, part := range first.Components {
		switch part.ID {
		case "VCC", "GNDS", "SUPPLY":
			continue
		case "DIR":
			b.part("B_DIR", "finite-pulse-source", "5 V", part.Position.X, part.Position.Y+800, 0)
			b.graph.Components[len(b.graph.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, HighSeconds: .03, LowSeconds: .03, RiseSeconds: .001, FallSeconds: .001}
			continue
		case "RL":
			part.Value = 150
		}
		if part.Label != "" && part.Label != "VDD" {
			part.Label = "B_" + part.Label
		}
		part.ID = "B_" + part.ID
		part.Position.Y += 800
		b.graph.Components = append(b.graph.Components, part)
	}
	for _, wire := range first.Wires {
		if !strings.HasPrefix(wire.From, "VCC.") {
			b.wire("B_"+wire.From, "B_"+wire.To)
		}
	}
	return b.graph
}
