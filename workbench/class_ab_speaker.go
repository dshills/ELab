package workbench

import "strings"

// ClassABSpeakerExampleGraph preserves the original 1 nF compensation and
// biased startup. A 10 mV input keeps this educational output stage near its
// linear range while the passive speaker's electrical states evolve.
func ClassABSpeakerExampleGraph() Graph {
	g := DiscreteClassABExampleGraph()
	replaced := false
	for i, p := range g.Components {
		if p.ID == "RL" {
			speaker, err := MakeComponent("SP1", "speaker", p.Position, "")
			if err != nil {
				panic(err)
			} // Static, validated example definition.
			speaker.Rotation = p.Rotation
			g.Components[i] = speaker
			replaced = true
		}
		if p.ID == "VIN" {
			g.Components[i].Sine.Amplitude = .01
			g.Components[i].Value = .01
		}
	}
	for i, w := range g.Wires {
		if w.From == "RL.1" {
			g.Wires[i].From = "SP1.positive"
		}
		if w.From == "RL.2" {
			g.Wires[i].From = "SP1.negative"
		}
		if w.To == "RL.1" {
			g.Wires[i].To = "SP1.positive"
		}
		if w.To == "RL.2" {
			g.Wires[i].To = "SP1.negative"
		}
	}
	if !replaced {
		panic("class-AB template is missing RL")
	}
	for _, w := range g.Wires {
		if strings.HasPrefix(w.From, "RL.") || strings.HasPrefix(w.To, "RL.") {
			panic("class-AB template has an unrecognized RL terminal")
		}
	}
	return g
}
