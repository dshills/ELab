package workbench

import (
	"fmt"
	"math"
)

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
	if len(result.Nodes) != len(c.Nets) {
		return nil, nil, fmt.Errorf("recorded device node inventory disagrees with its graph")
	}
	x := make([]float64, n.matrixSize(0))
	for k, node := range n.nodes {
		if privatePart(node) {
			continue
		}
		v, ok := result.Nodes[node]
		if !ok || !finite(v) || v < -maxMixedNodeVoltage || v > maxMixedNodeVoltage {
			return nil, nil, fmt.Errorf("invalid recorded device node voltages")
		}
		x[k] = v
	}
	if err := n.restorePowerNodes(result, x); err != nil {
		return nil, nil, err
	}
	for i := range n.nodes {
		if !finite(x[i]) || math.Abs(x[i]) > maxMixedNodeVoltage {
			return nil, nil, fmt.Errorf("invalid reconstructed device state")
		}
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

func validateRecordedDiodes(c Compiled, graph Graph, result *MixedResult) error {
	ids := map[string][]string{"diode": {}, "zener": {}, "schottky": {}}
	for _, d := range result.Diodes {
		p, ok := graph.Component(d.ID)
		if !ok || !IsDiodeKind(p.Kind) {
			return fmt.Errorf("invalid recorded diode identity")
		}
		ids[p.Kind] = append(ids[p.Kind], d.ID)
	}
	for _, kind := range []string{"diode", "zener", "schottky"} {
		if err := recordedObjectIDs(graph, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(result.Diodes) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, result)
	if err != nil {
		return err
	}
	for _, actual := range result.Diodes {
		p, _ := graph.Component(actual.ID)
		v := voltage(x, n.pin(p, "anode")) - voltage(x, n.pin(p, "cathode"))
		i, _ := deviceDiodeCurrent(p, v)
		if !recordingNear(actual.Voltage, v) || !recordingNear(actual.Current, i) || !recordingNear(actual.Power, v*i) {
			return fmt.Errorf("recorded diode readings disagree with its terminal voltages")
		}
	}
	return nil
}

func validateRecordedTimers555(c Compiled, g Graph, result *MixedResult) error {
	ids := []string{}
	for _, u := range result.Timers555 {
		ids = append(ids, u.ID)
	}
	if err := recordedObjectIDs(g, "timer555", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, result)
	if err != nil {
		return err
	}
	for _, actual := range result.Timers555 {
		p, _ := g.Component(actual.ID)
		memory := timer555Memory{High: actual.High, Powered: actual.Powered}
		next, err := timer555Decision(n.timer555Voltages(p, x), memory)
		if err != nil || next != memory || (!actual.Powered && actual.High) {
			return fmt.Errorf("recorded 555 latch disagrees with its inputs")
		}
		for k, part := range n.timers {
			if part.ID == p.ID {
				n.timerMemory[k] = memory
			}
		}
		expected := n.timer555Result(p, x)
		if actual.Model != expected.Model || actual.State != expected.State {
			return fmt.Errorf("invalid recorded 555 model or state")
		}
		for _, pair := range [][2]float64{{actual.SupplyVoltage, expected.SupplyVoltage}, {actual.OutputVoltage, expected.OutputVoltage}, {actual.TriggerLevel, expected.TriggerLevel}, {actual.ThresholdLevel, expected.ThresholdLevel}, {actual.Power, expected.Power}} {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("recorded 555 readings disagree with terminal voltages")
			}
		}
		for k, i := range actual.PinCurrents {
			if !recordingNear(i, expected.PinCurrents[k]) {
				return fmt.Errorf("recorded 555 terminal current disagrees with terminal voltages")
			}
		}
	}
	return nil
}
