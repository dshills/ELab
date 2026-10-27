package workbench

// A finite-resistance powered amplifier drives a passive dynamic load.
func SpeakerAmplifierExampleGraph() Graph {
	g := OpAmpFollowerExampleGraph()
	for i, p := range g.Components {
		if p.ID == "RL" {
			speaker, err := MakeComponent("SP1", "speaker", p.Position, "")
			if err != nil {
				panic(err)
			}
			speaker.Rotation = p.Rotation
			g.Components[i] = speaker
		}
		if p.ID == "VIN" {
			g.Components[i].Sine.Frequency = 400
			g.Components[i].Sine.Amplitude = .05
			g.Components[i].Value = .05
		}
	}
	for i, w := range g.Wires {
		switch w.From {
		case "RL.1":
			g.Wires[i].From = "SP1.positive"
		case "RL.2":
			g.Wires[i].From = "SP1.negative"
		}
		switch w.To {
		case "RL.1":
			g.Wires[i].To = "SP1.positive"
		case "RL.2":
			g.Wires[i].To = "SP1.negative"
		}
	}
	return g
}
