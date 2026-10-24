package workbench

import "fmt"

func Timer555FlasherExampleGraph() Graph    { return timer555Example(false, false) }
func Timer555OscillatorExampleGraph() Graph { return timer555Example(true, false) }
func Timer555OneShotExampleGraph() Graph    { return timer555Example(false, true) }

func timer555Example(adjustable, oneShot bool) Graph {
	g := EmptyGraph()
	add := func(id, kind, value string, at Point, rotation int) {
		p, err := MakeComponent(id, kind, at, value)
		if err != nil {
			panic(err)
		}
		p.Rotation = rotation
		g.Components = append(g.Components, p)
	}
	wire := func(from, to string) {
		g.Wires = append(g.Wires, Wire{ID: fmt.Sprintf("W%d", len(g.Wires)+1), From: from, To: to, Connected: true})
	}
	add("V1", "dc-source", "5", Point{-540, 0}, 0)
	add("GNDV", "ground", "", Point{-540, 150}, 0)
	add("U1", "timer555", "", Point{80, 0}, 0)
	add("C1", "capacitor", "10u", Point{-300, 200}, 0)
	add("GND1", "ground", "", Point{-300, 340}, 0)
	add("CC", "capacitor", "10n", Point{40, 230}, 0)
	// Start the control bypass at its nominal divider level; C1 starts discharged.
	control := 10. / 3
	g.Components[len(g.Components)-1].InitialVoltage = &control
	add("GNDC", "ground", "", Point{40, 340}, 0)
	add("GNDU", "ground", "", Point{300, 180}, 0)
	add("RLED", "resistor", "1k", Point{390, 0}, 0)
	add("D1", "led", "", Point{530, 100}, 0)
	add("GNDL", "ground", "", Point{530, 230}, 0)
	add("JV", "junction", "", Point{-300, -350}, 0)
	add("JT", "junction", "", Point{-300, 100}, 0)
	wire("V1.negative", "GNDV.reference")
	wire("V1.positive", "JV.node")
	wire("JV.node", "U1.vcc")
	wire("U1.vcc", "U1.reset")
	wire("U1.gnd", "GNDU.reference")
	wire("U1.control", "CC.1")
	wire("CC.2", "GNDC.reference")
	wire("U1.out", "RLED.1")
	wire("RLED.2", "D1.anode")
	wire("D1.cathode", "GNDL.reference")
	wire("C1.1", "JT.node")
	wire("C1.2", "GND1.reference")
	wire("JT.node", "U1.threshold")
	if oneShot {
		g.Components[0].Position = Point{-540, -220}
		g.Components[1].Position = Point{-540, -80}
		add("R1", "resistor", "47k", Point{-300, -150}, 90)
		add("VIN", "finite-pulse-source", "5", Point{-560, 60}, 0)
		g.Components[len(g.Components)-1].Pulse = &PulseWaveform{HighVoltage: 5, LowVoltage: 0, HighSeconds: 1.98, LowSeconds: .02, RiseSeconds: .005, FallSeconds: .005}
		add("GNDIN", "ground", "", Point{-560, 220}, 0)
		wire("JV.node", "R1.1")
		wire("R1.2", "JT.node")
		wire("U1.discharge", "JT.node")
		wire("VIN.positive", "U1.trigger")
		wire("VIN.negative", "GNDIN.reference")
	} else {
		add("RA", "resistor", "10k", Point{-300, -220}, 90)
		add("JD", "junction", "", Point{-300, -110}, 0)
		wire("JV.node", "RA.1")
		wire("RA.2", "JD.node")
		wire("JD.node", "U1.discharge")
		wire("JT.node", "U1.trigger")
		if adjustable {
			add("P1", "potentiometer", "100k", Point{-300, -20}, 90)
			wire("JD.node", "P1.1")
			wire("P1.wiper", "JT.node")
			wire("P1.2", "P1.wiper")
		} else {
			add("RB", "resistor", "47k", Point{-300, -20}, 90)
			wire("JD.node", "RB.1")
			wire("RB.2", "JT.node")
		}
	}
	return g
}
