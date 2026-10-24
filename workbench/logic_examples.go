package workbench

import "fmt"

func LogicInverterExampleGraph() Graph  { return logicExample("logic-inverter") }
func LogicSwitchesExampleGraph() Graph  { return logicExample("logic-switches") }
func LogicHalfAdderExampleGraph() Graph { return logicExample("logic-half-adder") }
func logicExample(id string) Graph {
	g := EmptyGraph()
	add := func(id, kind, value string, x, y float64) {
		p, err := MakeComponent(id, kind, Point{x, y}, value)
		if err != nil {
			panic(err)
		}
		g.Components = append(g.Components, p)
	}
	wire := func(a, b string) {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", len(g.Wires)+1), From: a, To: b, Connected: true})
	}
	add("VCC", "dc-source", "5", -560, -250)
	add("GNDV", "ground", "", -560, -120)
	wire("VCC.negative", "GNDV.reference")
	kind := "logic-not"
	if id == "logic-switches" {
		kind = "logic-and"
	}
	if id == "logic-half-adder" {
		kind = "logic-xor"
	}
	add("U1", kind, "", 0, 0)
	add("GNDU", "ground", "", 0, 180)
	wire("VCC.positive", "U1.vcc")
	wire("U1.gnd", "GNDU.reference")
	add("R1", "resistor", "1k", 200, 0)
	add("D1", "led", "", 350, 80)
	add("GND1", "ground", "", 350, 210)
	wire("U1.out", "R1.1")
	wire("R1.2", "D1.anode")
	wire("D1.cathode", "GND1.reference")
	if id == "logic-switches" {
		for k, name := range []string{"A", "B"} {
			y := -20. + float64(k)*260
			x := -350. + float64(k)*140
			add("S"+name, "switch", "", x-120, y)
			add("RP"+name, "resistor", "100k", x, y+100)
			g.Components[len(g.Components)-1].Rotation = 90
			groundY := y + 230
			if k == 0 {
				groundY = y + 190
			}
			add("GND"+name, "ground", "", x, groundY)
			wire("VCC.positive", "S"+name+".1")
			wire("S"+name+".2", "RP"+name+".1")
			wire("RP"+name+".2", "GND"+name+".reference")
			wire("S"+name+".2", "U1."+string('a'+rune(k)))
		}
	} else {
		for k, name := range []string{"A", "B"} {
			if k == 1 && id == "logic-inverter" {
				break
			}
			x, y := -360., -20.+float64(k)*310
			add(name, "finite-pulse-source", "5", x, y)
			period := .2 * float64(k+1)
			g.Components[len(g.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: period, LowSeconds: period, RiseSeconds: .01, FallSeconds: .01}
			add("GND"+name, "ground", "", x, y+130)
			wire(name+".negative", "GND"+name+".reference")
			wire(name+".positive", "U1."+string('a'+rune(k)))
		}
	}
	if id == "logic-half-adder" {
		// Explicit branch dots keep separate input/power buses away from pins.
		for _, p := range []struct {
			id   string
			x, y float64
		}{{"JV", -140, -350}, {"JVP", -140, 260}, {"JA", -200, -20}, {"JAP", -200, 350}, {"JB", -260, 20}, {"JBP", -260, 390}} {
			add(p.id, "junction", "", p.x, p.y)
		}
		for k := range g.Wires {
			w := &g.Wires[k]
			if w.From == "VCC.positive" && w.To == "U1.vcc" {
				w.To = "JV.node"
			}
			if w.From == "A.positive" && w.To == "U1.a" {
				w.To = "JA.node"
			}
			if w.From == "B.positive" && w.To == "U1.b" {
				w.To = "JB.node"
			}
		}
		wire("JV.node", "U1.vcc")
		wire("JA.node", "U1.a")
		wire("JB.node", "U1.b")
		add("U2", "logic-and", "", 0, 370)
		add("GND2", "ground", "", 0, 530)
		wire("JV.node", "JVP.node")
		wire("JVP.node", "U2.vcc")
		wire("U2.gnd", "GND2.reference")
		wire("JA.node", "JAP.node")
		wire("JAP.node", "U2.a")
		wire("JB.node", "JBP.node")
		wire("JBP.node", "U2.b")
		add("R2", "resistor", "1k", 200, 370)
		add("D2", "led", "", 350, 450)
		add("GNDL2", "ground", "", 350, 580)
		wire("U2.out", "R2.1")
		wire("R2.2", "D2.anode")
		wire("D2.cathode", "GNDL2.reference")
	}
	return g
}
