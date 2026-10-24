package workbench

import "fmt"

func validateRecordedBipolars(c Compiled, graph Graph, result *MixedResult) error {
	ids := map[string][]string{"npn": {}, "pnp": {}}
	for _, device := range result.Transistors {
		part, ok := graph.Component(device.TransistorID)
		if !ok || !IsBipolarKind(part.Kind) {
			return fmt.Errorf("invalid recorded bipolar transistor identity")
		}
		ids[part.Kind] = append(ids[part.Kind], part.ID)
	}
	for _, kind := range []string{"npn", "pnp"} {
		if err := recordedObjectIDs(graph, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(result.Transistors) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, result)
	if err != nil {
		return err
	}
	for _, actual := range result.Transistors {
		part, _ := graph.Component(actual.TransistorID)
		expected := n.bipolarResult(part, x)
		validModel := actual.Model == expected.Model
		if part.Kind == "npn" {
			validModel = supportedNPNModel(actual.Model)
		}
		if !validModel || actual.State != expected.State || !recordingNear(actual.EmitterCurrent, actual.BaseCurrent+actual.CollectorCurrent) {
			return fmt.Errorf("invalid recorded bipolar model, region or terminal-current convention")
		}
		for _, pair := range [][2]float64{
			{actual.VBE, expected.VBE}, {actual.VCE, expected.VCE},
			{actual.BaseCurrent, expected.BaseCurrent}, {actual.CollectorCurrent, expected.CollectorCurrent},
			{actual.EmitterCurrent, expected.EmitterCurrent}, {actual.TransistorPower, expected.TransistorPower},
		} {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("recorded bipolar readings disagree with its terminal voltages")
			}
		}
	}
	return nil
}

// Reconstruct only the saved node voltages, without rerunning a solve. A
// captured device reading must agree with its own graph and terminal state
// even when none of that device's readings was selected as a scope probe.
func recordedDeviceNodes(c Compiled, result *MixedResult) (*mixedNetwork, []float64, error) {
	if ground, ok := result.Nodes[c.Ground]; !ok || ground != 0 {
		return nil, nil, fmt.Errorf("invalid recorded device ground reference")
	}
	n := newMixedNetwork(c)
	if len(result.Nodes) != len(n.nodes)+1 {
		return nil, nil, fmt.Errorf("recorded device node inventory disagrees with its graph")
	}
	x := make([]float64, len(n.nodes))
	for k, node := range n.nodes {
		v, ok := result.Nodes[node]
		if !ok || !finite(v) || v < -maxMixedNodeVoltage || v > maxMixedNodeVoltage {
			return nil, nil, fmt.Errorf("invalid recorded device node voltages")
		}
		x[k] = v
	}
	return n, x, nil
}

func validateRecordedMOSFETs(c Compiled, graph Graph, result *MixedResult) error {
	ids := map[string][]string{"nmos": {}, "pmos": {}}
	for _, device := range result.MOSFETs {
		part, ok := graph.Component(device.ID)
		if !ok || !IsMOSFETKind(part.Kind) {
			return fmt.Errorf("invalid recorded MOSFET identity")
		}
		ids[part.Kind] = append(ids[part.Kind], device.ID)
	}
	for _, kind := range []string{"nmos", "pmos"} {
		if err := recordedObjectIDs(graph, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(result.MOSFETs) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, result)
	if err != nil {
		return err
	}
	for _, actual := range result.MOSFETs {
		part, _ := graph.Component(actual.ID)
		expected := n.mosfetResult(part, x)
		if actual.Model != expected.Model || actual.State != expected.State || actual.GateCurrent != 0 || actual.SourceCurrent != actual.DrainCurrent {
			return fmt.Errorf("invalid recorded MOSFET model, region or terminal-current convention")
		}
		for _, pair := range [][2]float64{
			{actual.VGS, expected.VGS}, {actual.VDS, expected.VDS},
			{actual.DrainCurrent, expected.DrainCurrent}, {actual.Power, expected.Power},
		} {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("recorded MOSFET readings disagree with its terminal voltages")
			}
		}
	}
	return nil
}
