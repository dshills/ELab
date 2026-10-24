package workbench

import "fmt"

// Pin currents enter the device in A, B, OUT, VCC, GND order. B is zero for NOT.
type LogicResult struct {
	ID            string     `json:"id"`
	Kind          string     `json:"kind"`
	Model         LogicModel `json:"model"`
	InputA        string     `json:"input_a"`
	InputB        string     `json:"input_b"`
	State         string     `json:"state"`
	OutputVoltage float64    `json:"output_voltage_v"`
	SupplyVoltage float64    `json:"supply_voltage_v"`
	PinCurrents   [5]float64 `json:"pin_currents_a"`
	Power         float64    `json:"power_w"`
}

func (n *mixedNetwork) logicVoltages(p Component, x []float64) (v [5]float64) {
	for k, pin := range logicPinOrder {
		if pin == "b" && p.Kind == "logic-not" {
			continue
		}
		v[k] = voltage(x, n.pin(p, pin))
	}
	return
}
func (n *mixedNetwork) stampLogic(p Component, x []float64, add func(int, int, float64), f []float64) {
	e := logicCurrents(p.Kind, n.logicVoltages(p, x))
	for row, pin := range logicPinOrder {
		if pin == "b" && p.Kind == "logic-not" {
			continue
		}
		node := n.pin(p, pin)
		if node >= 0 {
			f[node] += e.Current[row]
		}
		for col, other := range logicPinOrder {
			if other == "b" && p.Kind == "logic-not" {
				continue
			}
			add(node, n.pin(p, other), e.Jacobian[row][col])
		}
	}
}
func logicLevel(v, span float64) string {
	if span <= 0 {
		return "unpowered"
	}
	m := LogicDeviceModel()
	if v <= m.LowFraction*span {
		return "low"
	}
	if v >= m.HighFraction*span {
		return "high"
	}
	return "intermediate"
}
func (n *mixedNetwork) logicResult(p Component, x []float64) LogicResult {
	v := n.logicVoltages(p, x)
	e := logicCurrents(p.Kind, v)
	span := v[3] - v[4]
	r := LogicResult{ID: p.ID, Kind: p.Kind, Model: LogicDeviceModel(), InputA: logicLevel(v[0]-v[4], span), InputB: logicLevel(v[1]-v[4], span), State: logicLevel(v[2]-v[4], span), OutputVoltage: v[2] - v[4], SupplyVoltage: span, PinCurrents: e.Current}
	if p.Kind == "logic-not" {
		r.InputB = "unused"
	}
	for k, i := range e.Current {
		r.Power += (v[k] - v[4]) * i
	}
	return r
}

func validateRecordedLogic(c Compiled, g Graph, r *MixedResult) error {
	ids := map[string][]string{}
	for _, u := range r.LogicGates {
		p, ok := g.Component(u.ID)
		if !ok || !IsLogicKind(p.Kind) || u.Kind != p.Kind {
			return fmt.Errorf("invalid recorded logic identity")
		}
		ids[p.Kind] = append(ids[p.Kind], u.ID)
	}
	for _, kind := range []string{"logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor"} {
		if err := recordedObjectIDs(g, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(r.LogicGates) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, actual := range r.LogicGates {
		p, _ := g.Component(actual.ID)
		expected := n.logicResult(p, x)
		if actual.Model != expected.Model || actual.State != expected.State || actual.InputA != expected.InputA || actual.InputB != expected.InputB {
			return fmt.Errorf("invalid recorded logic model or level")
		}
		for _, pair := range [][2]float64{{actual.OutputVoltage, expected.OutputVoltage}, {actual.SupplyVoltage, expected.SupplyVoltage}, {actual.Power, expected.Power}} {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("recorded logic readings disagree with terminal voltages")
			}
		}
		for k, i := range actual.PinCurrents {
			if !recordingNear(i, expected.PinCurrents[k]) {
				return fmt.Errorf("recorded logic current disagrees with terminal voltages")
			}
		}
	}
	return nil
}
