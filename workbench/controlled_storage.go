package workbench

import "fmt"

func validateRecordedControls(c Compiled, g Graph, r *MixedResult) error {
	ids := map[string][]string{}
	for _, d := range r.Controls {
		p, ok := g.Component(d.ID)
		if !ok || !IsControlKind(p.Kind) || p.Kind != d.Kind {
			return fmt.Errorf("invalid recorded control identity")
		}
		ids[d.Kind] = append(ids[d.Kind], d.ID)
	}
	for _, kind := range []string{"pushbutton", "bouncing-pushbutton", "comparator", "relay"} {
		if err := recordedObjectIDs(g, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(r.Controls) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, actual := range r.Controls {
		p, _ := g.Component(actual.ID)
		m := actual.Memory
		if !finite(m.StartedAt) || !finite(m.ReleaseAt) || m.StartedAt < 0 || m.ReleaseAt < 0 || m.StartedAt > r.Time {
			return fmt.Errorf("invalid control event timestamps")
		}
		if IsButtonKind(p.Kind) && (m.High || m.Powered || m.ReleaseAt > 0 && m.ReleaseAt <= m.StartedAt) {
			return fmt.Errorf("invalid button memory")
		}
		if p.Kind == "comparator" && (m.Pressed || m.ContactClosed || m.Bouncing || m.StartedAt != 0 || m.ReleaseAt != 0 || !m.Powered && m.High) {
			return fmt.Errorf("invalid comparator memory")
		}
		next, err := controlDecision(p, n.controlVoltages(p, x), m, r.Time, actual.Current)
		if err != nil || next != m {
			return fmt.Errorf("recorded control guard/contact state disagrees with terminal readings")
		}
		for k, q := range n.controls {
			if p.ID == q.ID {
				n.controlMemory[k] = m
			}
		}
		if p.Kind == "relay" {
			if m.Powered || m.Pressed || m.ContactClosed || m.Bouncing || m.StartedAt != 0 || m.ReleaseAt != 0 {
				return fmt.Errorf("invalid relay memory")
			}
			if len(x) < n.matrixSize(0) {
				x = append(x, make([]float64, n.matrixSize(0)-len(x))...)
			}
			for k, l := range n.inductors {
				if l.ID == p.ID {
					x[n.inductorIndex(k)] = actual.Current
				}
			}
		}
		expected := n.controlResult(p, x)
		if actual.Model != expected.Model || actual.State != expected.State {
			return fmt.Errorf("invalid recorded control model/state")
		}
		pairs := [][2]float64{{actual.Voltage, expected.Voltage}, {actual.Current, expected.Current}, {actual.Power, expected.Power}, {actual.Energy, expected.Energy}}
		for k, v := range actual.PinCurrents {
			pairs = append(pairs, [2]float64{v, expected.PinCurrents[k]})
		}
		for _, v := range pairs {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("recorded control current/power disagrees with terminal readings")
			}
		}
	}
	return nil
}
