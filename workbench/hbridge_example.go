package workbench

// Switches connect actual input nets; labels keep the three control paths clear
// without introducing an example-only driver override.
func HBridgeExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VM", "dc-source", "12", -850, 0, 0)
	b.ground("GNDV", "VM.negative", -850, 200)
	b.label("LVM", "VM", "VM.positive", -850, -160)
	b.part("U1", "hbridge", "", 200, 0, 0)
	b.label("LPOWER", "VM", "U1.vm", 200, -240)
	b.ground("GNDU", "U1.gnd", 200, 240)
	for j, id := range []string{"S1", "S2"} {
		y := float64(j)*300 - 350
		b.part(id, "switch", "", -500, y, 0)
		b.label(id+"P", "VM", id+".1", -650, y)
		resistor, net, pin := "R1", "DIR1", "in1"
		if j == 1 {
			resistor, net, pin = "R2", "DIR2", "in2"
		}
		b.label(id+"OUT", net, id+".2", -180, y)
		b.label(id+"IN", net, "U1."+pin, -80, float64(j)*40-40)
		b.part(resistor, "resistor", "10k", -320, y+110, 90)
		b.wire(id+".2", resistor+".1")
		b.ground("GND"+id, resistor+".2", -320, y+230)
		closed := j == 0
		for k, p := range b.graph.Components {
			if p.ID == id {
				b.graph.Components[k].SwitchClosed = &closed
			}
		}
	}
	b.part("PWM", "finite-pulse-source", "5", -850, 420, 0)
	for k, p := range b.graph.Components {
		if p.ID == "PWM" {
			b.graph.Components[k].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .005, LowSeconds: .005, RiseSeconds: .0001, FallSeconds: .0001}
		}
	}
	b.ground("GNDP", "PWM.negative", -850, 600)
	b.part("SEN", "switch", "", -500, 330, 0)
	for k, p := range b.graph.Components {
		if p.ID == "SEN" {
			v := true
			b.graph.Components[k].SwitchClosed = &v
		}
	}
	b.wire("PWM.positive", "SEN.1")
	b.label("LENOUT", "ENABLE", "SEN.2", -180, 330)
	b.label("LENIN", "ENABLE", "U1.enable", -80, 40)
	b.part("REN", "resistor", "10k", -320, 440, 90)
	b.wire("SEN.2", "REN.1")
	b.ground("GNDE", "REN.2", -320, 580)
	b.part("MTR1", "dc-motor", "", 600, 0, 0)
	b.wire("U1.out1", "MTR1.positive")
	b.wire("U1.out2", "MTR1.negative")
	return b.graph
}
