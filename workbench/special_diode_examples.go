package workbench

// ZenerExampleGraph is a 10 Hz, 8–12 V input with a 470 Ω ballast and
// 4.7 kΩ load. The reverse-biased 5.1 V knee holds the output near 5.2 V.
func ZenerExampleGraph() Graph {
	source, _ := MakeComponent("V1", "sine-source", Point{-300, 20}, "2")
	source.Sine = &SineWaveform{Amplitude: 2, Offset: 10, Frequency: 10}
	series, _ := MakeComponent("R1", "resistor", Point{-130, -100}, "470")
	diode, _ := MakeComponent("D1", "zener", Point{70, 20}, "5.1")
	diode.Rotation = 180
	load, _ := MakeComponent("RL", "resistor", Point{310, 20}, "4.7k")
	load.Rotation = 90
	ground, _ := MakeComponent("GND1", "ground", Point{70, 180}, "")
	return Graph{Components: []Component{source, series, diode, load, ground}, Wires: []Wire{
		{ID: "W1", From: "V1.positive", To: "R1.1", Connected: true},
		{ID: "W2", From: "R1.2", To: "D1.cathode", Connected: true},
		{ID: "W3", From: "D1.cathode", To: "RL.1", Connected: true},
		{ID: "W4", From: "D1.anode", To: "GND1.reference", Connected: true},
		{ID: "W5", From: "RL.2", To: "GND1.reference", Connected: true},
		{ID: "W6", From: "V1.negative", To: "GND1.reference", Connected: true},
	}}
}

// SchottkyExampleGraph compares separate, equally loaded half-wave branches.
// Two capacitors allow the scope to show both forward loss and reverse hold.
func SchottkyExampleGraph() Graph {
	source, _ := MakeComponent("V1", "sine-source", Point{-360, 100}, "1")
	source.Sine = &SineWaveform{Amplitude: 1, Frequency: 10}
	r1, _ := MakeComponent("R1", "resistor", Point{-180, -140}, "100")
	d1, _ := MakeComponent("D1", "schottky", Point{-10, -140}, "")
	d1.Rotation = 270
	c1, _ := MakeComponent("C1", "capacitor", Point{180, -40}, "10u")
	load1, _ := MakeComponent("RL1", "resistor", Point{380, -40}, "10k")
	load1.Rotation = 90
	r2, _ := MakeComponent("R2", "resistor", Point{-180, 180}, "100")
	d2, _ := MakeComponent("D2", "diode", Point{-10, 180}, "")
	d2.Rotation = 270
	c2, _ := MakeComponent("C2", "capacitor", Point{180, 280}, "10u")
	load2, _ := MakeComponent("RL2", "resistor", Point{380, 280}, "10k")
	load2.Rotation = 90
	ground, _ := MakeComponent("GND1", "ground", Point{180, 100}, "")
	ground2, _ := MakeComponent("GND2", "ground", Point{180, 430}, "")
	groundSource, _ := MakeComponent("GNDV", "ground", Point{-360, 240}, "")
	return Graph{Components: []Component{source, r1, d1, c1, load1, r2, d2, c2, load2, ground, ground2, groundSource}, Wires: []Wire{
		{ID: "W1", From: "V1.positive", To: "R1.1", Connected: true},
		{ID: "W2", From: "V1.positive", To: "R2.1", Connected: true},
		{ID: "W3", From: "R1.2", To: "D1.anode", Connected: true},
		{ID: "W4", From: "R2.2", To: "D2.anode", Connected: true},
		{ID: "W5", From: "D1.cathode", To: "C1.1", Connected: true},
		{ID: "W6", From: "D2.cathode", To: "C2.1", Connected: true},
		{ID: "W7", From: "C1.1", To: "RL1.1", Connected: true},
		{ID: "W8", From: "C2.1", To: "RL2.1", Connected: true},
		{ID: "W9", From: "C1.2", To: "GND1.reference", Connected: true},
		{ID: "W10", From: "C2.2", To: "GND2.reference", Connected: true},
		{ID: "W11", From: "RL1.2", To: "GND1.reference", Connected: true},
		{ID: "W12", From: "RL2.2", To: "GND2.reference", Connected: true},
		{ID: "W13", From: "V1.negative", To: "GNDV.reference", Connected: true},
	}}
}
