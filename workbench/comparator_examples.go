package workbench

func ComparatorThresholdExampleGraph() Graph {
	b := newExampleBuilder()
	b.part("VCC", "dc-source", "5", -600, -180, 0)
	b.ground("GNDV", "VCC.negative", -600, -20)
	b.part("VIN", "sine-source", "400mV", -600, 220, 0)
	w := SineWaveform{Amplitude: .4, Offset: 2.5, Frequency: 1, Phase: 0}
	b.graph.Components[len(b.graph.Components)-1].Sine = &w
	b.ground("GNDI", "VIN.negative", -600, 420)
	b.part("VREF", "dc-source", "2.5", -280, 240, 0)
	b.ground("GNDR", "VREF.negative", -280, 420)
	b.part("U1", "comparator", "200mV", 0, 0, 0)
	b.wire("VIN.positive", "U1.noninverting")
	b.wire("VREF.positive", "U1.inverting")
	b.wire("VCC.positive", "U1.vcc")
	b.ground("GNDU", "U1.gnd", 0, 200)
	b.part("R1", "resistor", "1k", 250, 0, 0)
	b.part("D1", "led", "", 450, 0, 270)
	b.wire("U1.out", "R1.1")
	b.wire("R1.2", "D1.anode")
	b.ground("GNDD", "D1.cathode", 620, 100)
	return b.graph
}
func ComparatorCounterExampleGraph() Graph {
	base := ButtonCounterExampleGraph(false)
	b := newExampleBuilder()
	removed := map[string]bool{"SCOUNT": true, "PBPOWER": true, "RCLK": true, "GNDP": true}
	for _, p := range base.Components {
		if !removed[p.ID] {
			b.graph.Components = append(b.graph.Components, p)
		}
	}
	for _, w := range base.Wires {
		from, _, _ := base.Terminal(w.From)
		to, _, _ := base.Terminal(w.To)
		if !removed[from.ID] && !removed[to.ID] {
			b.wire(w.From, w.To)
		}
	}
	b.part("U1", "comparator", "200mV", -360, 0, 0)
	b.label("PC", "VCC", "U1.vcc", -360, -150)
	b.ground("GNDC", "U1.gnd", -360, 200)
	b.part("VIN", "sine-source", "400mV", -920, 200, 0)
	b.graph.Components[len(b.graph.Components)-1].Sine = &SineWaveform{Amplitude: .4, Offset: 2.5, Frequency: 1}
	b.ground("GNDI", "VIN.negative", -920, 390)
	b.part("VREF", "dc-source", "2.5", -630, 230, 0)
	b.ground("GNDREF", "VREF.negative", -630, 390)
	b.wire("VIN.positive", "U1.noninverting")
	b.wire("VREF.positive", "U1.inverting")
	b.wire("U1.out", "U2.clk")
	return b.graph
}
