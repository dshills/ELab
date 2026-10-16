package workbench

// RCHighPassExampleGraph couples an ideal square source through C1 to R1.
// Output is R1 pin 1 relative to ground; capacitor voltage remains pin 1 minus 2.
func RCHighPassExampleGraph() Graph {
	source, _ := MakeComponent("V1", "pulse-source", Point{-200, 0}, "5 V")
	cap, _ := MakeComponent("C1", "capacitor", Point{0, -100}, "1uF")
	cap.Rotation = 270
	load, _ := MakeComponent("R1", "resistor", Point{200, 0}, "10k")
	load.Rotation = 90
	ground, _ := MakeComponent("GND1", "ground", Point{0, 150}, "")
	return Graph{Components: []Component{source, cap, load, ground}, Wires: []Wire{
		{ID: "W1", From: "V1.positive", To: "C1.1", Connected: true},
		{ID: "W2", From: "C1.2", To: "R1.1", Connected: true},
		{ID: "W3", From: "R1.2", To: "GND1.reference", Connected: true},
		{ID: "W4", From: "V1.negative", To: "GND1.reference", Connected: true},
	}}
}
