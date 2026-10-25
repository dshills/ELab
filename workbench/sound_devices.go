package workbench

import "fmt"

func soundStorage(id string) string { return "$" + id + "_piezo" }
func lowerSound(c *Compiled, p Component) []Component {
	if p.Kind != "piezo" {
		return []Component{p}
	}
	cap := Component{ID: soundStorage(p.ID), Kind: "capacitor", Terminals: []string{"1", "2"}, Value: p.Device.Sound.Capacitance}
	zero := 0.
	cap.InitialVoltage = &zero
	c.TerminalNet[cap.ID+".1"] = c.TerminalNet[p.ID+".positive"]
	c.TerminalNet[cap.ID+".2"] = c.TerminalNet[p.ID+".negative"]
	return []Component{p, cap}
}
func soundCurrent(p Component, v float64) (i, g float64) {
	g = 1e-7
	if p.Kind == "active-buzzer" {
		en, de := smoothLevel(v, 2, 3)
		return v * (g + en/1000), g + (en+v*de)/1000
	}
	return v * g, g
}
func (n *mixedNetwork) stampSound(p Component, x []float64, add func(int, int, float64), f []float64) {
	a, b := n.pin(p, "positive"), n.pin(p, "negative")
	i, g := soundCurrent(p, voltage(x, a)-voltage(x, b))
	if a >= 0 {
		f[a] += i
	}
	if b >= 0 {
		f[b] -= i
	}
	add(a, a, g)
	add(b, b, g)
	add(a, b, -g)
	add(b, a, -g)
}

type SoundResult struct {
	ID      string  `json:"id"`
	Kind    string  `json:"kind"`
	Voltage float64 `json:"voltage_v"`
	Current float64 `json:"current_a"`
	Energy  float64 `json:"stored_energy_j"`
	Power   float64 `json:"dissipation_w"`
	Active  bool    `json:"active"`
}

func (n *mixedNetwork) soundResult(p Component, x []float64) SoundResult {
	v := voltage(x, n.pin(p, "positive")) - voltage(x, n.pin(p, "negative"))
	i, _ := soundCurrent(p, v)
	r := SoundResult{ID: p.ID, Kind: p.Kind, Voltage: v, Current: i, Power: v * i, Active: p.Kind == "active-buzzer" && v >= 3}
	if p.Kind == "piezo" {
		r.Energy = .5 * p.Device.Sound.Capacitance * v * v
		for k, c := range n.caps {
			if c.ID == soundStorage(p.ID) {
				r.Current += x[n.capacitorIndex(k)]
			}
		}
	}
	return r
}
func validateRecordedSounds(c Compiled, g Graph, r *MixedResult) error {
	ids := map[string][]string{}
	for _, s := range r.Sounds {
		ids[s.Kind] = append(ids[s.Kind], s.ID)
	}
	for _, k := range []string{"piezo", "active-buzzer"} {
		if err := recordedObjectIDs(g, k, ids[k]); err != nil {
			return err
		}
	}
	if len(r.Sounds) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, s := range r.Sounds {
		p, ok := g.Component(s.ID)
		if !ok || !isSoundKind(p.Kind) || s.Kind != p.Kind {
			return fmt.Errorf("invalid sounder identity")
		}
		e := n.soundResult(p, x)
		if !recordingNear(s.Voltage, e.Voltage) || !recordingNear(s.Current, e.Current) || !recordingNear(s.Energy, e.Energy) || !recordingNear(s.Power, e.Power) || s.Active != e.Active {
			return fmt.Errorf("sounder readings disagree with terminals")
		}
	}
	return nil
}
func (n *mixedNetwork) restoreSounds(r *MixedResult, x []float64) error {
	for _, p := range n.authored.Components {
		if p.Kind != "piezo" {
			continue
		}
		found := false
		for _, s := range r.Sounds {
			if s.ID == p.ID {
				v := voltage(x, n.pin(p, "positive")) - voltage(x, n.pin(p, "negative"))
				leak, _ := soundCurrent(p, v)
				for k, c := range n.caps {
					if c.ID == soundStorage(p.ID) {
						x[n.capacitorIndex(k)] = s.Current - leak
					}
				}
				found = true
			}
		}
		if !found {
			return fmt.Errorf("missing piezo current")
		}
	}
	return nil
}
