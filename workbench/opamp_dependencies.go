package workbench

import "fmt"

func opAmpModelForGraph(g Graph) *OpAmpModel {
	for _, part := range g.Components {
		if part.Kind == "opamp" {
			model := OpAmpDeviceModel()
			return &model
		}
	}
	return nil
}

func validateOpAmpModel(g Graph, model *OpAmpModel) error {
	if model != nil && *model != OpAmpDeviceModel() {
		return fmt.Errorf("unsupported op-amp model parameters/version")
	}
	if opAmpModelForGraph(g) != nil && model == nil {
		return fmt.Errorf("op-amp schematics require supported opamp_model parameters")
	}
	return nil
}

func validateRecordedOpAmps(c Compiled, g Graph, result *MixedResult) error {
	ids := make([]string, 0, len(result.OpAmps))
	for _, u := range result.OpAmps {
		ids = append(ids, u.ID)
	}
	if err := recordedObjectIDs(g, "opamp", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if ground, ok := result.Nodes[c.Ground]; !ok || ground != 0 {
		return fmt.Errorf("invalid recorded op-amp ground reference")
	}
	n := newMixedNetwork(c)
	x := make([]float64, len(n.nodes))
	for k, node := range n.nodes {
		v, ok := result.Nodes[node]
		if !ok || !finite(v) || v < -maxMixedNodeVoltage || v > maxMixedNodeVoltage {
			return fmt.Errorf("invalid recorded op-amp node voltages")
		}
		x[k] = v
	}
	for _, actual := range result.OpAmps {
		p, _ := g.Component(actual.ID)
		expected := n.opAmpResult(p, x)
		if actual.Model != expected.Model || actual.State != expected.State {
			return fmt.Errorf("invalid recorded op-amp model or supply state")
		}
		for _, pair := range [][2]float64{
			{actual.InputDifference, expected.InputDifference}, {actual.OutputVoltage, expected.OutputVoltage},
			{actual.SupplyVoltage, expected.SupplyVoltage}, {actual.NoninvertingCurrent, expected.NoninvertingCurrent},
			{actual.InvertingCurrent, expected.InvertingCurrent}, {actual.OutputCurrent, expected.OutputCurrent},
			{actual.PositiveSupplyCurrent, expected.PositiveSupplyCurrent}, {actual.NegativeSupplyCurrent, expected.NegativeSupplyCurrent},
			{actual.Power, expected.Power},
		} {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("recorded op-amp readings disagree with its terminal voltages")
			}
		}
	}
	return nil
}
