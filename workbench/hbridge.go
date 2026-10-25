package workbench

import (
	"fmt"
	"math"
)

const HBridgeModelID = "elab.generic-hbridge.v1"

// One functional bridge, not a packaged IC or four discrete MOSFET models.
// All currents enter the device in hbridgePins order, including both outputs.
var hbridgePins = [...]string{"in1", "in2", "enable", "out1", "out2", "vm", "gnd"}

type HBridgeModel struct {
	ID               string  `json:"id"`
	OnResistance     float64 `json:"on_resistance_per_leg_ohm"`
	OffResistance    float64 `json:"off_resistance_per_leg_ohm"`
	InputResistance  float64 `json:"input_pulldown_ohm"`
	SupplyResistance float64 `json:"supply_resistance_ohm"`
	LowVoltage       float64 `json:"input_low_v"`
	HighVoltage      float64 `json:"input_high_v"`
	DiodeDrop        float64 `json:"clamp_diode_knee_v"`
	DiodeResistance  float64 `json:"clamp_diode_resistance_ohm"`
	Limitations      string  `json:"limitations"`
}

func HBridgeDeviceModel() HBridgeModel {
	return HBridgeModel{HBridgeModelID, .25, 1e9, 1e6, 1e5, .8, 2, .65, .25,
		"Generic single H-bridge with IN1/IN2 and active-high ENABLE/PWM. Enabled 10 drives forward, 01 reverse, 00 brakes to GND and 11 brakes to VM; ENABLE low coasts. Inputs are local to GND: <=0.8 V low, >=2 V high. Intermediate levels use a smooth nonoverlapping conductance transition, not a valid digital state. Each active leg is 0.25 ohm with a 1 Gohm off path; four passive piecewise-linear clamp diodes remain active during coast and power-off (0.65 V knee plus 0.25 ohm). Input pulldowns are 1 Mohm; supply load is 100 kohm. Supply enable rises smoothly from 2 to 3 V; <=2 V disables active legs. Bidirectional leg conduction and diode paths allow regenerative current into VM. No temporal dead time, propagation delay, gate charge, current limiting, thermal protection, switching loss, physical ratings or fitted manufacturer identity. Connect all controls and both supply pins; the supply must accommodate returned energy. The IC counts as one functional part within existing network bounds."}
}

func hbridgeModelForGraph(g Graph) *HBridgeModel {
	for _, p := range g.Components {
		if p.Kind == "hbridge" {
			m := HBridgeDeviceModel()
			return &m
		}
	}
	return nil
}

func validateHBridgeModel(g Graph, m *HBridgeModel) error {
	if m != nil && *m != HBridgeDeviceModel() {
		return fmt.Errorf("unsupported H-bridge model parameters/version")
	}
	if hbridgeModelForGraph(g) != nil && m == nil {
		return fmt.Errorf("h-bridge schematics require hbridge_model parameters")
	}
	return nil
}

func smoothLevel(v, lo, hi float64) (value, slope float64) {
	x := (v - lo) / (hi - lo)
	if x <= 0 {
		return 0, 0
	}
	if x >= 1 {
		return 1, 0
	}
	return x * x * (3 - 2*x), 6 * x * (1 - x) / (hi - lo)
}

type hbridgeEvaluation struct {
	Current  [7]float64
	Jacobian [7][7]float64
	Clamps   [4]float64 // OUT1->VM, GND->OUT1, OUT2->VM, GND->OUT2.
}

func hbridgeCurrents(v [7]float64) (e hbridgeEvaluation) {
	m := HBridgeDeviceModel()
	branch := func(a, b int, current, slope float64, control [7]float64) {
		e.Current[a] += current
		e.Current[b] -= current
		for k := range control {
			d := control[k]
			if k == a {
				d += slope
			}
			if k == b {
				d -= slope
			}
			e.Jacobian[a][k] += d
			e.Jacobian[b][k] -= d
		}
	}
	resistor := func(a, b int, resistance float64) {
		branch(a, b, (v[a]-v[b])/resistance, 1/resistance, [7]float64{})
	}
	for _, pin := range []int{0, 1, 2} {
		resistor(pin, 6, m.InputResistance)
	}
	resistor(5, 6, m.SupplyResistance)
	supply, ds := smoothLevel(v[5]-v[6], 2, 3)
	enable, de := smoothLevel(v[2]-v[6], m.LowVoltage, m.HighVoltage)
	for j, input := range []int{0, 1} {
		out := 3 + j
		q, dq := smoothLevel(v[input]-v[6], m.LowVoltage, m.HighVoltage)
		// Up and down never conduct actively at the same input voltage. This
		// algebraic transition has no invented time or stored gate energy.
		for side, rail := range []int{5, 6} {
			weight, dw := math.Max(2*q-1, 0), 2*dq
			if side == 1 {
				weight, dw = math.Max(1-2*q, 0), -2*dq
			}
			dw *= 2 * weight
			weight *= weight
			g := 1/m.OffResistance + supply*enable*weight/m.OnResistance
			drop := v[out] - v[rail]
			d := [7]float64{}
			d[input] += drop * supply * enable * dw / m.OnResistance
			d[2] += drop * supply * de * weight / m.OnResistance
			d[5] += drop * ds * enable * weight / m.OnResistance
			d[6] -= drop * (supply*enable*dw + supply*de*weight + ds*enable*weight) / m.OnResistance
			branch(out, rail, drop*g, g, d)
		}
		for side, pair := range [][2]int{{out, 5}, {6, out}} {
			drop := v[pair[0]] - v[pair[1]]
			if drop > m.DiodeDrop {
				i := (drop - m.DiodeDrop) / m.DiodeResistance
				e.Clamps[j*2+side] = i
				branch(pair[0], pair[1], i, 1/m.DiodeResistance, [7]float64{})
			}
		}
	}
	return
}

func (n *mixedNetwork) hbridgeVoltages(p Component, x []float64) (v [7]float64) {
	for k, pin := range hbridgePins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	return
}

func (n *mixedNetwork) stampHBridge(p Component, x []float64, add func(int, int, float64), f []float64) {
	e := hbridgeCurrents(n.hbridgeVoltages(p, x))
	for row, pin := range hbridgePins {
		at := n.pin(p, pin)
		if at >= 0 {
			f[at] += e.Current[row]
		}
		for col, other := range hbridgePins {
			add(at, n.pin(p, other), e.Jacobian[row][col])
		}
	}
}

type HBridgeResult struct {
	ID            string     `json:"id"`
	Model         string     `json:"model"`
	State         string     `json:"state"`
	Input1        string     `json:"input1"`
	Input2        string     `json:"input2"`
	Enable        string     `json:"enable"`
	SupplyVoltage float64    `json:"supply_voltage_v"`
	OutputVoltage float64    `json:"output_voltage_v"`
	PinCurrents   [7]float64 `json:"pin_currents_a"`
	ClampCurrents [4]float64 `json:"clamp_currents_a"`
	Power         float64    `json:"dissipation_w"`
}

func hbridgeLevel(v float64) string {
	m := HBridgeDeviceModel()
	if v <= m.LowVoltage {
		return "low"
	}
	if v >= m.HighVoltage {
		return "high"
	}
	return "intermediate"
}

func (n *mixedNetwork) hbridgeResult(p Component, x []float64) HBridgeResult {
	v := n.hbridgeVoltages(p, x)
	e := hbridgeCurrents(v)
	r := HBridgeResult{ID: p.ID, Model: HBridgeModelID, Input1: hbridgeLevel(v[0] - v[6]), Input2: hbridgeLevel(v[1] - v[6]), Enable: hbridgeLevel(v[2] - v[6]), SupplyVoltage: v[5] - v[6], OutputVoltage: v[3] - v[4], PinCurrents: e.Current, ClampCurrents: e.Clamps}
	r.State = "intermediate control"
	switch {
	case r.SupplyVoltage <= 2:
		r.State = "unpowered / coast"
	case r.Enable == "low":
		r.State = "coast"
	case r.SupplyVoltage < 3 || r.Enable != "high" || r.Input1 == "intermediate" || r.Input2 == "intermediate":
	case r.Input1 == "high" && r.Input2 == "low":
		r.State = "forward"
	case r.Input1 == "low" && r.Input2 == "high":
		r.State = "reverse"
	case r.Input1 == "low":
		r.State = "brake low"
	default:
		r.State = "brake high"
	}
	for k, i := range e.Current {
		r.Power += (v[k] - v[6]) * i
	}
	return r
}

func validateRecordedHBridges(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, u := range r.HBridges {
		ids = append(ids, u.ID)
	}
	if err := recordedObjectIDs(g, "hbridge", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, u := range r.HBridges {
		p, _ := g.Component(u.ID)
		e := n.hbridgeResult(p, x)
		if u.Model != e.Model || u.State != e.State || u.Input1 != e.Input1 || u.Input2 != e.Input2 || u.Enable != e.Enable {
			return fmt.Errorf("invalid H-bridge model or control state")
		}
		pairs := [][2]float64{{u.SupplyVoltage, e.SupplyVoltage}, {u.OutputVoltage, e.OutputVoltage}, {u.Power, e.Power}}
		for k, i := range u.PinCurrents {
			pairs = append(pairs, [2]float64{i, e.PinCurrents[k]})
		}
		for k, i := range u.ClampCurrents {
			pairs = append(pairs, [2]float64{i, e.ClampCurrents[k]})
		}
		for _, v := range pairs {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("h-bridge readings disagree with terminal voltages")
			}
		}
	}
	return nil
}
