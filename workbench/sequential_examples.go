package workbench

import "fmt"

func DFlipFlopExampleGraph() Graph     { return sequentialExample("d-flipflop") }
func JKFlipFlopExampleGraph() Graph    { return sequentialExample("jk-flipflop") }
func TFlipFlopExampleGraph() Graph     { return sequentialExample("t-flipflop") }
func BinaryCounterExampleGraph() Graph { return sequentialExample("binary-counter") }
func DecadeCounterExampleGraph() Graph { return sequentialExample("decade-counter") }
func sequentialExample(kind string) Graph {
	g := EmptyGraph()
	add := func(id, kind, value string, x, y float64, rotation int) {
		p, err := MakeComponent(id, kind, Point{x, y}, value)
		if err != nil {
			panic(err)
		}
		p.Rotation = rotation
		g.Components = append(g.Components, p)
	}
	wire := func(a, b string) {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", len(g.Wires)+1), From: a, To: b, Connected: true})
	}
	add("VCC", "dc-source", "5", -820, -300, 0)
	add("GNDV", "ground", "", -820, -140, 0)
	wire("VCC.negative", "GNDV.reference")
	add("JV", "junction", "", -300, -390, 0)
	wire("VCC.positive", "JV.node")
	add("U1", kind, "", 0, 0, 0)
	add("GNDU", "ground", "", 0, 250, 0)
	wire("JV.node", "U1.vcc")
	wire("U1.gnd", "GNDU.reference")
	add("CLK", "finite-pulse-source", "5", -380, -80, 0)
	g.Components[len(g.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .1, LowSeconds: .1, RiseSeconds: .02, FallSeconds: .02}
	add("GNDC", "ground", "", -380, 50, 0)
	wire("CLK.negative", "GNDC.reference")
	clockAt, _ := SequentialTerminalLocal(kind, "clk")
	add("JCLK1", "junction", "", -240, -114, 0)
	add("JCLK2", "junction", "", -240, clockAt.Y, 0)
	wire("CLK.positive", "JCLK1.node")
	wire("JCLK1.node", "JCLK2.node")
	wire("JCLK2.node", "U1.clk")
	add("SRESET", "switch", "", -480, 170, 90)
	add("RR", "resistor", "100k", -230, 300, 90)
	add("GNDR", "ground", "", -230, 460, 0)
	add("JPOWER", "junction", "", -300, 130, 0)
	wire("JV.node", "JPOWER.node")
	wire("JPOWER.node", "SRESET.1")
	wire("SRESET.2", "RR.1")
	resetAt, _ := SequentialTerminalLocal(kind, "reset")
	add("JR1", "junction", "", -180, 225, 0)
	add("JR2", "junction", "", -180, resetAt.Y, 0)
	wire("SRESET.2", "JR1.node")
	wire("JR1.node", "JR2.node")
	wire("JR2.node", "U1.reset")
	wire("RR.2", "GNDR.reference")
	switch kind {
	case "d-flipflop":
		add("DATA", "finite-pulse-source", "5", -640, 80, 0)
		g.Components[len(g.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: .3, LowSeconds: .3, RiseSeconds: .01, FallSeconds: .01}
		add("GNDD", "ground", "", -640, 240, 0)
		wire("DATA.negative", "GNDD.reference")
		add("JD1", "junction", "", -180, -220, 0)
		add("JD2", "junction", "", -180, -45, 0)
		wire("DATA.positive", "JD1.node")
		wire("JD1.node", "JD2.node")
		wire("JD2.node", "U1.d")
	case "jk-flipflop":
		wire("JV.node", "U1.j")
		wire("JV.node", "U1.k")
	case "t-flipflop":
		wire("JV.node", "U1.t")
	default:
		add("SENABLE", "switch", "", -600, 390, 90)
		closed := true
		g.Components[len(g.Components)-1].SwitchClosed = &closed
		add("RE", "resistor", "100k", -460, 490, 90)
		add("GNDE", "ground", "", -460, 640, 0)
		add("JP2", "junction", "", -300, 330, 0)
		wire("JPOWER.node", "JP2.node")
		wire("JP2.node", "SENABLE.1")
		wire("SENABLE.2", "RE.1")
		wire("RE.2", "GNDE.reference")
		add("JE1", "junction", "", -270, 445, 0)
		add("JE2", "junction", "", -270, 0, 0)
		wire("SENABLE.2", "JE1.node")
		wire("JE1.node", "JE2.node")
		wire("JE2.node", "U1.enable")
	}
	pins := []string{"q", "nq"}
	if IsCounterKind(kind) {
		pins = []string{"q0", "q1", "q2", "q3"}
	}
	for k, pin := range pins {
		y := -140. + float64(k)*220
		resistorX := 400.
		if IsCounterKind(kind) {
			resistorX = 500
		}
		column := 160. + 60*float64(len(pins)-1-k)
		add(fmt.Sprintf("R%d", k+1), "resistor", "1k", resistorX, y, 0)
		add(fmt.Sprintf("D%d", k+1), "led", "", resistorX+200, y, 270)
		add(fmt.Sprintf("GND%d", k+1), "ground", "", resistorX+350, y+80, 0)
		pinAt, _ := SequentialTerminalLocal(kind, pin)
		add(fmt.Sprintf("JQ%d", k+1), "junction", "", column, pinAt.Y, 0)
		add(fmt.Sprintf("JO%d", k+1), "junction", "", column, y, 0)
		wire("U1."+pin, fmt.Sprintf("JQ%d.node", k+1))
		wire(fmt.Sprintf("JQ%d.node", k+1), fmt.Sprintf("JO%d.node", k+1))
		wire(fmt.Sprintf("JO%d.node", k+1), fmt.Sprintf("R%d.1", k+1))
		wire(fmt.Sprintf("R%d.2", k+1), fmt.Sprintf("D%d.anode", k+1))
		wire(fmt.Sprintf("D%d.cathode", k+1), fmt.Sprintf("GND%d.reference", k+1))
	}
	return g
}
