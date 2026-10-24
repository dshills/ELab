package workbench

import "fmt"

func validateRecordedSequential(c Compiled, g Graph, r *MixedResult) error {
	ids := map[string][]string{}
	for _, u := range r.Sequential {
		p, ok := g.Component(u.ID)
		if !ok || !IsSequentialKind(p.Kind) || u.Kind != p.Kind {
			return fmt.Errorf("invalid recorded sequential identity")
		}
		ids[p.Kind] = append(ids[p.Kind], u.ID)
	}
	for _, kind := range []string{"d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter"} {
		if err := recordedObjectIDs(g, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(r.Sequential) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, actual := range r.Sequential {
		p, _ := g.Component(actual.ID)
		m := actual.Memory
		if !m.ClockKnown && m.ClockHigh || m.Value < 0 || m.Value >= sequentialModulus(p.Kind) || !m.Powered && m != (SequentialMemory{}) || m.ResetHigh && m.Value != 0 {
			return fmt.Errorf("invalid recorded sequential memory")
		}
		// Validate stable guard memory without replaying a clock edge or inferring
		// historical data from the present data pins.
		next, err := sequentialDecision(p.Kind, n.sequentialVoltages(p, x), m, false)
		if err != nil || next != m {
			return fmt.Errorf("recorded sequential guards disagree with terminal voltages")
		}
		for k, u := range n.sequential {
			if u.ID == p.ID {
				n.sequenceMemory[k] = m
			}
		}
		expected := n.sequentialResult(p, x)
		if actual.Model != expected.Model || actual.State != expected.State || actual.Clock != expected.Clock || actual.Reset != expected.Reset {
			return fmt.Errorf("invalid recorded sequential model or levels")
		}
		pairs := [][2]float64{{actual.SupplyVoltage, expected.SupplyVoltage}, {actual.OutputVoltage, expected.OutputVoltage}, {actual.ComplementVoltage, expected.ComplementVoltage}, {actual.Power, expected.Power}}
		for k, v := range actual.BitVoltages {
			pairs = append(pairs, [2]float64{v, expected.BitVoltages[k]})
		}
		for k, i := range actual.PinCurrents {
			pairs = append(pairs, [2]float64{i, expected.PinCurrents[k]})
		}
		for _, v := range pairs {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("recorded sequential readings disagree with terminal voltages")
			}
		}
	}
	return nil
}
