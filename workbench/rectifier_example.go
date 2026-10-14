package workbench

// RectifierExampleGraph is a low-voltage half-wave supply experiment. R1 limits
// charging current; RL discharges the capacitor while the diode blocks.
func RectifierExampleGraph() Graph {
	source, _ := MakeComponent("V1", "sine-source", Point{-300, 20}, "")
	source.Sine = &SineWaveform{Amplitude: 5, Frequency: 10}
	source.Value = 5
	series, _ := MakeComponent("R1", "resistor", Point{-160, -100}, "100")
	diode, _ := MakeComponent("D1", "diode", Point{0, -100}, "")
	diode.Rotation = 270
	cap, _ := MakeComponent("C1", "capacitor", Point{170, 20}, "10u")
	load, _ := MakeComponent("RL", "resistor", Point{340, 20}, "10k")
	load.Rotation = 90
	ground, _ := MakeComponent("GND1", "ground", Point{170, 180}, "")
	return Graph{Components: []Component{source, series, diode, cap, load, ground}, Wires: []Wire{
		{ID: "W1", From: "V1.positive", To: "R1.1", Connected: true},
		{ID: "W2", From: "R1.2", To: "D1.anode", Connected: true},
		{ID: "W3", From: "D1.cathode", To: "C1.1", Connected: true},
		{ID: "W4", From: "C1.1", To: "RL.1", Connected: true},
		{ID: "W5", From: "RL.2", To: "C1.2", Connected: true},
		{ID: "W6", From: "C1.2", To: "GND1.reference", Connected: true},
		{ID: "W7", From: "V1.negative", To: "GND1.reference", Connected: true},
	}}
}
