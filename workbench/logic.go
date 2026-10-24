package workbench

import "fmt"

const LogicModelID = "elab.generic-cmos-logic.v1"

// LogicModel describes single functional gates, not package-level 4000/74xx ICs.
// Inputs interpolate smoothly between valid low and high levels. Conductive
// output branches conserve current and power, including intermediate inputs.
type LogicModel struct {
	ID                string  `json:"id"`
	InputResistance   float64 `json:"input_resistance_ohm"`
	OutputResistance  float64 `json:"output_resistance_ohm"`
	OffResistance     float64 `json:"off_resistance_ohm"`
	SupplyResistance  float64 `json:"supply_resistance_ohm"`
	SupplySoftVoltage float64 `json:"supply_soft_voltage_v"`
	LowFraction       float64 `json:"low_supply_fraction"`
	HighFraction      float64 `json:"high_supply_fraction"`
	Limitations       string  `json:"limitations"`
}

func LogicDeviceModel() LogicModel {
	return LogicModel{LogicModelID, 1e9, 50, 1e9, 1e6, .1, .3, .7, "Generic powered combinational gates, not fitted hardware or package pinouts. Inputs <=30% of local supply are low; >=70% are high. The interval uses smoothstep interpolation, not a valid digital level. Complementary weighted 50 Ω output paths, 1 GΩ input/output bleeds, 1 MΩ supply load and a smooth supply enable are explicit. Reversed/zero supply disables active output paths. No propagation delay, intrinsic capacitance, hysteresis, protection, current limit, ratings, noise or thermal behavior. Feedback may settle at an intermediate analog equilibrium; this model does not qualify digital latches, ring oscillators or hardware timing."}
}
func IsLogicKind(kind string) bool {
	switch kind {
	case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		return true
	}
	return false
}
func logicPins(kind string) []string {
	if kind == "logic-not" {
		return []string{"a", "out", "vcc", "gnd"}
	}
	return []string{"a", "b", "out", "vcc", "gnd"}
}

// Internal stamp order is fixed; the unused B row is zero for an inverter.
var logicPinOrder = [...]string{"a", "b", "out", "vcc", "gnd"}

type logicEvaluation struct {
	Current  [5]float64
	Jacobian [5][5]float64
	Weight   float64
}

func logicInput(v, span float64) (q, dv, ds float64) {
	m := LogicDeviceModel()
	x := (v/span - m.LowFraction) / (m.HighFraction - m.LowFraction)
	if x <= 0 {
		return 0, 0, 0
	}
	if x >= 1 {
		return 1, 0, 0
	}
	d := 6 * x * (1 - x) / (m.HighFraction - m.LowFraction)
	// Return derivatives scaled by span. The stamp multiplies by enable/span
	// directly, keeping the zero-supply limit finite even for subnormal rails.
	return x * x * (3 - 2*x), d, -d * (v / span)
}
func logicWeight(kind string, a, b float64) (q, da, db float64) {
	switch kind {
	case "logic-not":
		return 1 - a, -1, 0
	case "logic-and":
		return a * b, b, a
	case "logic-nand":
		return 1 - a*b, -b, -a
	case "logic-or":
		return a + b - a*b, 1 - b, 1 - a
	case "logic-nor":
		return (1 - a) * (1 - b), b - 1, a - 1
	case "logic-xor":
		return a*(1-b) + (1-a)*b, 1 - 2*b, 1 - 2*a
	case "logic-xnor":
		return a*b + (1-a)*(1-b), 2*b - 1, 2*a - 1
	}
	panic("unsupported logic kind")
}
func logicCurrents(kind string, v [5]float64) logicEvaluation {
	m := LogicDeviceModel()
	e := logicEvaluation{}
	branch := func(a, b int, g float64, dg [5]float64) {
		drop := v[a] - v[b]
		i := drop * g
		e.Current[a] += i
		e.Current[b] -= i
		for k := range dg {
			d := drop * dg[k]
			if k == a {
				d += g
			}
			if k == b {
				d -= g
			}
			e.Jacobian[a][k] += d
			e.Jacobian[b][k] -= d
		}
	}
	branch(0, 4, 1/m.InputResistance, [5]float64{})
	if kind != "logic-not" {
		branch(1, 4, 1/m.InputResistance, [5]float64{})
	}
	branch(2, 4, 1/m.OffResistance, [5]float64{})
	branch(3, 4, 1/m.SupplyResistance, [5]float64{})
	span := v[3] - v[4]
	if span <= 0 {
		return e
	}
	a, av, as := logicInput(v[0]-v[4], span)
	b, bv, bs := 0., 0., 0.
	if kind != "logic-not" {
		b, bv, bs = logicInput(v[1]-v[4], span)
	}
	q, qa, qb := logicWeight(kind, a, b)
	e.Weight = q
	dq := [5]float64{qa * av, qb * bv, 0, qa*as + qb*bs, -qa*(av+as) - qb*(bv+bs)}
	soft := m.SupplySoftVoltage * m.SupplySoftVoltage
	den := span*span + soft
	enable := span * span / den
	de := 2 * span * soft / (den * den)
	up, down := [5]float64{}, [5]float64{}
	for k := range dq {
		d := 0.
		if k == 3 {
			d = de
		}
		if k == 4 {
			d = -de
		}
		up[k] = ((span/den)*dq[k] + q*d) / m.OutputResistance
		down[k] = (-(span/den)*dq[k] + (1-q)*d) / m.OutputResistance
	}
	branch(2, 3, enable*q/m.OutputResistance, up)
	branch(2, 4, enable*(1-q)/m.OutputResistance, down)
	return e
}
func logicModelForGraph(g Graph) *LogicModel {
	for _, p := range g.Components {
		if IsLogicKind(p.Kind) {
			m := LogicDeviceModel()
			return &m
		}
	}
	return nil
}
func validateLogicModel(g Graph, m *LogicModel) error {
	if m != nil && *m != LogicDeviceModel() {
		return fmt.Errorf("unsupported logic model parameters/version")
	}
	if logicModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("logic schematics require supported logic_model parameters")
	}
	return nil
}
