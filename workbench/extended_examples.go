package workbench

import "fmt"

func BuckExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VIN", "dc-source", "12", -700, -300, 0)
	b.ground("GNDV", "VIN.negative", -700, -100)
	b.label("PV", "VM", "VIN.positive", -700, -480)
	b.part("PWM", "pwm-controller", "", -380, 80, 0)
	b.label("PP", "VM", "PWM.vcc", -380, -140)
	b.ground("GP", "PWM.gnd", -380, 270)
	b.part("U1", "hbridge", "", 0, 0, 0)
	b.label("PU", "VM", "U1.vm", 0, -230)
	b.ground("GU", "U1.gnd", 0, 260)
	b.label("PI1", "VM", "U1.in1", -180, -180)
	b.ground("GI2", "U1.in2", -180, 150)
	b.wire("PWM.out", "U1.enable")
	b.part("RW", "resistor", "10", 250, -30, 0)
	b.wire("U1.out1", "RW.1")
	b.part("L1", "inductor", "50m", 460, -30, 90)
	b.wire("RW.2", "L1.1")
	b.part("C1", "capacitor", "470u", 660, 160, 0)
	b.wire("L1.2", "C1.1")
	b.ground("GC", "C1.2", 660, 340)
	b.part("RL", "resistor", "50", 900, 100, 90)
	b.wire("C1.1", "RL.1")
	b.ground("GL", "RL.2", 900, 340)
	b.part("D1", "schottky", "", 230, 200, 180)
	b.wire("U1.out1", "D1.cathode")
	b.ground("GD", "D1.anode", 230, 380)
	b.part("SLOAD", "switch", "", 900, -210, 0)
	b.wire("C1.1", "SLOAD.1")
	b.part("RSTEP", "resistor", "50", 1150, 0, 90)
	b.wire("SLOAD.2", "RSTEP.1")
	b.ground("GS", "RSTEP.2", 1150, 270)
	return b.graph
}
func OpAmpDynamicsExampleGraph(pulse bool) Graph {
	g := OpAmpFollowerExampleGraph()
	for k, p := range g.Components {
		switch p.ID {
		case "U1":
			g.Components[k].Device = &DeviceParameters{Dynamics: &OpAmpDynamics{1000, 1000}}
		case "VIN":
			if pulse {
				v, _ := MakeComponent(p.ID, "finite-pulse-source", p.Position, "1")
				v.Rotation = p.Rotation
				v.Pulse = &PulseWaveform{HighVoltage: 1, LowVoltage: -1, HighSeconds: .01, LowSeconds: .01, RiseSeconds: .0001, FallSeconds: .0001}
				g.Components[k] = v
			} else {
				g.Components[k].Sine.Amplitude = .01
				g.Components[k].Sine.Frequency = 1000
				g.Components[k].Value = .01
			}
		}
	}
	return g
}
func ShiftRegisterExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -750, -450, 0)
	b.ground("GV", "VCC.negative", -750, -270)
	b.label("PV", "VCC", "VCC.positive", -750, -620)
	b.part("U1", "shift-register", "", 0, 0, 0)
	b.label("PU", "VCC", "U1.vcc", 0, -360)
	b.ground("GU", "U1.gnd", 0, 360)
	b.part("CLK", "finite-pulse-source", "5", -700, -20, 0)
	b.graph.Components[len(b.graph.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .1, LowSeconds: .1, RiseSeconds: .01, FallSeconds: .01}
	b.ground("GK", "CLK.negative", -700, 150)
	b.label("CK", "CLOCK", "CLK.positive", -700, -190)
	b.label("CU", "CLOCK", "U1.clk", -250, -60)
	for j, id := range []string{"SDATA", "SLATCH", "SCLEAR", "SOE"} {
		pin := []string{"data", "latch", "reset_n", "oe_n"}[j]
		y := float64(j)*250 + 330
		b.part(id, "switch", "", -550, y, 0)
		b.part("R"+id, "resistor", "100k", -260, y+110, 90)
		net := "CTRL" + fmt.Sprint(j)
		b.label(id+"U", net, "U1."+pin, -220, float64(j)*75-130)
		b.label(id+"O", net, id+".2", -330, y)
		b.wire(id+".2", "R"+id+".1")
		if j == 2 {
			b.ground("G"+id, id+".1", -700, y+150)
			b.label("P"+id, "VCC", "R"+id+".2", -260, y+310)
		} else {
			b.label("P"+id, "VCC", id+".1", -760, y)
			b.ground("G"+id, "R"+id+".2", -260, y+310)
		}
		if j == 0 {
			v := true
			for k, p := range b.graph.Components {
				if p.ID == id {
					b.graph.Components[k].SwitchClosed = &v
				}
			}
		}
	}
	b.part("DS1", "seven-segment", "", 850, 20, 0)
	b.ground("GDISPLAY", "DS1.common", 850, 400)
	for k, pin := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		id := fmt.Sprintf("R%d", k)
		y := float64(k)*110 - 330
		b.part(id, "resistor", "1k", 450, y, 0)
		net := fmt.Sprintf("BIT%d", k)
		b.label(id+"Q", net, fmt.Sprintf("U1.q%d", k), 210, float64(k)*35-140)
		b.label(id+"I", net, id+".1", 330, y)
		b.wire(id+".2", "DS1."+pin)
	}
	b.part("R7", "resistor", "1k", 450, 530, 0)
	b.label("RQ7", "BIT7", "U1.q7", 210, 105)
	b.label("RI7", "BIT7", "R7.1", 330, 530)
	b.part("D7", "led", "", 850, 530, 270)
	b.wire("R7.2", "D7.anode")
	b.ground("G7", "D7.cathode", 1050, 730)
	return b.graph
}
func Piezo555ExampleGraph() Graph {
	g := Timer555FlasherExampleGraph()
	for k, p := range g.Components {
		if p.ID == "C1" {
			g.Components[k].Value = 10e-9
		}
		if p.ID == "RA" {
			g.Components[k].Value = 10e3
		}
		if p.ID == "RB" {
			g.Components[k].Value = 68e3
		}
	}
	b := exampleBuilder{graph: g}
	b.part("RS", "resistor", "220", 760, 0, 0)
	b.wire("U1.out", "RS.1")
	b.part("BZ1", "piezo", "", 950, 100, 0)
	b.wire("RS.2", "BZ1.positive")
	b.ground("GBZ", "BZ1.negative", 950, 300)
	return b.graph
}
func BuzzerAlarmExampleGraph() Graph {
	b := exampleBuilder{graph: SensorExampleGraph(false)}
	b.part("BZ1", "active-buzzer", "", 500, 420, 0)
	b.wire("U1.out", "BZ1.positive")
	b.ground("GBZ", "BZ1.negative", 500, 620)
	return b.graph
}
