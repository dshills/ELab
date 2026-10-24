package workbench

import (
	"fmt"
	"math"
)

const RegulatorModelID = "elab.generic-regulators.v1"

type RegulatorModel struct {
	ID               string  `json:"id"`
	LinearDropout    float64 `json:"linear_dropout_v"`
	LDODropout       float64 `json:"ldo_dropout_v"`
	OutputResistance float64 `json:"output_resistance_ohm"`
	CurrentLimit     float64 `json:"current_limit_a"`
	SupplyResistance float64 `json:"supply_resistance_ohm"`
	OffResistance    float64 `json:"off_resistance_ohm"`
	Limitations      string  `json:"limitations"`
}

func VoltageRegulatorModel() RegulatorModel {
	return RegulatorModel{RegulatorModelID, 2, .3, .2, .1, 1e6, 1e9, "Generic educational non-sinking linear and LDO regulators. Fixed dropout headroom, 0.2 Ω small-signal pass resistance, smooth 100 mA asymptotic current limit, 1 MΩ input quiescent load and 1 GΩ output bleed. No thermal shutdown, reverse protection, noise, startup delay, feedback bandwidth, capacitor stability or physical package rating. External capacitors and loads determine circuit dynamics; not a fitted 78xx/LDO device."}
}
func IsRegulatorKind(kind string) bool { return kind == "linear-regulator" || kind == "ldo-regulator" }
func regulatorDropout(kind string) float64 {
	m := VoltageRegulatorModel()
	if kind == "ldo-regulator" {
		return m.LDODropout
	}
	return m.LinearDropout
}
func regulatorModelForGraph(g Graph) *RegulatorModel {
	for _, p := range g.Components {
		if IsRegulatorKind(p.Kind) {
			m := VoltageRegulatorModel()
			return &m
		}
	}
	return nil
}
func validateRegulatorModel(g Graph, m *RegulatorModel) error {
	if m != nil && *m != VoltageRegulatorModel() {
		return fmt.Errorf("unsupported voltage regulator model")
	}
	if regulatorModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("regulators require regulator_model parameters")
	}
	return nil
}

var regulatorPins = []string{"in", "out", "gnd"}

type regulatorEvaluation struct {
	Current             [3]float64
	Jacobian            [3][3]float64
	PassCurrent, Target float64
}

func regulatorCurrents(p Component, v [3]float64) regulatorEvaluation {
	m := VoltageRegulatorModel()
	e := regulatorEvaluation{}
	vin, vout := v[0]-v[2], v[1]-v[2]
	branch := func(a, b int, i float64, derivative [3]float64) {
		e.Current[a] += i
		e.Current[b] -= i
		for k, d := range derivative {
			e.Jacobian[a][k] += d
			e.Jacobian[b][k] -= d
		}
	}
	branch(0, 2, vin/m.SupplyResistance, [3]float64{1 / m.SupplyResistance, 0, -1 / m.SupplyResistance})
	branch(1, 2, vout/m.OffResistance, [3]float64{0, 1 / m.OffResistance, -1 / m.OffResistance})
	head := vin - regulatorDropout(p.Kind)
	e.Target = math.Max(0, math.Min(p.Value, head))
	if head <= 0 {
		return e
	}
	errorV := e.Target - vout
	if errorV < 0 {
		return e
	}
	z := errorV / (m.OutputResistance * m.CurrentLimit)
	root := math.Hypot(1, z)
	e.PassCurrent = errorV / m.OutputResistance / root
	derivative := 1 / (m.OutputResistance * root * root * root)
	targetDerivative := 0.
	if head < p.Value {
		targetDerivative = 1
	}
	branch(0, 1, e.PassCurrent, [3]float64{derivative * targetDerivative, -derivative, derivative * (1 - targetDerivative)})
	return e
}
func (n *mixedNetwork) regulatorVoltages(p Component, x []float64) (v [3]float64) {
	for k, pin := range regulatorPins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	return
}
func (n *mixedNetwork) stampRegulator(p Component, x []float64, add func(int, int, float64), f []float64) {
	e := regulatorCurrents(p, n.regulatorVoltages(p, x))
	for k, pin := range regulatorPins {
		row := n.pin(p, pin)
		if row >= 0 {
			f[row] += e.Current[k]
		}
		for j, col := range regulatorPins {
			add(row, n.pin(p, col), e.Jacobian[k][j])
		}
	}
}

type RegulatorResult struct {
	ID            string         `json:"id"`
	Kind          string         `json:"kind"`
	Model         RegulatorModel `json:"model"`
	State         string         `json:"state"`
	InputVoltage  float64        `json:"input_voltage_v"`
	OutputVoltage float64        `json:"output_voltage_v"`
	Current       float64        `json:"current_a"`
	PinCurrents   [3]float64     `json:"pin_currents_a"`
	Power         float64        `json:"power_w"`
}

func (n *mixedNetwork) regulatorResult(p Component, x []float64) RegulatorResult {
	v := n.regulatorVoltages(p, x)
	e := regulatorCurrents(p, v)
	r := RegulatorResult{ID: p.ID, Kind: p.Kind, Model: VoltageRegulatorModel(), State: "regulating", InputVoltage: v[0] - v[2], OutputVoltage: v[1] - v[2], Current: -e.Current[1], PinCurrents: e.Current}
	switch {
	case r.InputVoltage <= regulatorDropout(p.Kind):
		r.State = "unpowered"
	case e.PassCurrent >= .9*r.Model.CurrentLimit:
		r.State = "current limited"
	case e.Target < p.Value:
		r.State = "dropout"
	case e.PassCurrent == 0 && r.OutputVoltage > e.Target:
		r.State = "not sourcing"
	}
	for k, i := range e.Current {
		r.Power += (v[k] - v[2]) * i
	}
	return r
}
func validateRecordedRegulators(c Compiled, g Graph, r *MixedResult) error {
	ids := map[string][]string{}
	for _, d := range r.Regulators {
		p, ok := g.Component(d.ID)
		if !ok || !IsRegulatorKind(p.Kind) || p.Kind != d.Kind {
			return fmt.Errorf("invalid regulator identity")
		}
		ids[d.Kind] = append(ids[d.Kind], d.ID)
	}
	for _, kind := range []string{"linear-regulator", "ldo-regulator"} {
		if err := recordedObjectIDs(g, kind, ids[kind]); err != nil {
			return err
		}
	}
	if len(r.Regulators) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, actual := range r.Regulators {
		p, _ := g.Component(actual.ID)
		expected := n.regulatorResult(p, x)
		if actual.Model != expected.Model || actual.State != expected.State {
			return fmt.Errorf("recorded regulator model/state mismatch")
		}
		pairs := [][2]float64{{actual.InputVoltage, expected.InputVoltage}, {actual.OutputVoltage, expected.OutputVoltage}, {actual.Current, expected.Current}, {actual.Power, expected.Power}}
		for k, v := range actual.PinCurrents {
			pairs = append(pairs, [2]float64{v, expected.PinCurrents[k]})
		}
		for _, pair := range pairs {
			if !recordingNear(pair[0], pair[1]) {
				return fmt.Errorf("recorded regulator readings disagree with terminal voltages")
			}
		}
	}
	return nil
}
