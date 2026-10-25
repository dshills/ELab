package workbench

// Switches connect actual input pins; no example-only driver override exists.
func HBridgeExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VM", "dc-source", "12", -650, -300, 0)
	b.ground("GNDV", "VM.negative", -650, -130)
	b.label("LVM", "VM", "VM.positive", -650, -450)
	b.part("U1", "hbridge", "", 0, 0, 0)
	b.label("LPOWER", "VM", "U1.vm", 0, -220)
	b.ground("GNDU", "U1.gnd", 0, 230)
	for j, id := range []string{"S1", "S2"} {
		y := float64(j)*190 - 220
		b.part(id, "switch", "", -400, y, 0)
		b.label(id+"P", "VM", id+".1", -540, y)
		pin := "in1"
		if j == 1 {
			pin = "in2"
		}
		b.wire(id+".2", "U1."+pin)
		b.part("R"+id, "resistor", "10k", -220, y+75, 90)
		b.wire(id+".2", "R"+id+".1")
		b.ground("GND"+id, "R"+id+".2", -220, y+200)
		closed := j == 0
		for k, p := range b.graph.Components {
			if p.ID == id {
				b.graph.Components[k].SwitchClosed = &closed
			}
		}
	}
	b.part("PWM", "finite-pulse-source", "5", -650, 300, 0)
	for k, p := range b.graph.Components {
		if p.ID == "PWM" {
			b.graph.Components[k].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .005, LowSeconds: .005, RiseSeconds: .0001, FallSeconds: .0001}
		}
	}
	b.ground("GNDP", "PWM.negative", -650, 450)
	b.part("SEN", "switch", "", -400, 240, 0)
	for k, p := range b.graph.Components {
		if p.ID == "SEN" {
			v := true
			b.graph.Components[k].SwitchClosed = &v
		}
	}
	b.wire("PWM.positive", "SEN.1")
	b.wire("SEN.2", "U1.enable")
	b.part("REN", "resistor", "10k", -170, 330, 90)
	b.wire("U1.enable", "REN.1")
	b.ground("GNDE", "REN.2", -170, 480)
	b.part("MTR1", "dc-motor", "", 350, 0, 0)
	b.wire("U1.out1", "MTR1.positive")
	b.wire("U1.out2", "MTR1.negative")
	return b.graph
}
