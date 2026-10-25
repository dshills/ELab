package workbench

import (
	"fmt"
	"math"
)

const compensationC = 1e-9

func poleNode(id string) string { return "$" + id + "_pole" }
func poleCap(id string) string  { return "$" + id + "_compensation" }
func lowerOpAmpDynamics(c *Compiled, p Component) []Component {
	node := poleNode(p.ID)
	c.Nets = append(c.Nets, Net{ID: node})
	cap := Component{ID: poleCap(p.ID), Kind: "capacitor", Terminals: []string{"1", "2"}, Value: compensationC}
	zero := 0.
	cap.InitialVoltage = &zero
	c.TerminalNet[cap.ID+".1"] = node
	c.TerminalNet[cap.ID+".2"] = c.TerminalNet[p.ID+".vminus"]
	return []Component{p, cap}
}

// A bounded transconductance charges a real private capacitor. Rail gating
// prevents integrator wind-up without adding hidden state or an ideal source.
func poleCurrent(p Component, v opAmpVoltages, pole float64) (j float64, d [6]float64, rail int) {
	cfg := p.Device.Dynamics
	w := 2 * math.Pi * cfg.GBW / OpAmpDeviceModel().OpenLoopGain
	a := w * (OpAmpDeviceModel().OpenLoopGain*(v[0]-v[1]) + .5*(v[3]+v[4]) - pole)
	q := math.Tanh(a / cfg.Slew)
	s := compensationC * w * (1 - q*q)
	j = compensationC * cfg.Slew * q
	d = [6]float64{s * OpAmpDeviceModel().OpenLoopGain, -s * OpAmpDeviceModel().OpenLoopGain, 0, .5 * s, .5 * s, -s}
	rail = 4
	if v[3] <= v[4] {
		j = compensationC * w * (v[4] - pole)
		d = [6]float64{0, 0, 0, 0, compensationC * w, -compensationC * w}
		return
	}
	g, dg := smoothLevel(pole-v[4], 0, .05)
	control := [6]float64{0, 0, 0, 0, -dg, dg}
	if j >= 0 {
		rail = 3
		g, dg = smoothLevel(v[3]-pole, 0, .05)
		control = [6]float64{0, 0, 0, dg, 0, -dg}
	}
	for k := range d {
		d[k] = d[k]*g + j*control[k]
	}
	j *= g
	return
}
func (n *mixedNetwork) opAmpDevice(p Component, x []float64) (opAmpEvaluation, float64) {
	v := n.opAmpVoltages(p, x)
	if !hasDynamics(p) {
		return opAmpCurrents(v), 0
	}
	pole := voltage(x, n.index[poleNode(p.ID)])
	e := opAmpCurrentsControlled(v, pole, true)
	j, _, rail := poleCurrent(p, v, pole)
	e.current[rail] += j
	e.current[4] -= j
	return e, j
}
func (n *mixedNetwork) stampDynamicOpAmp(p Component, x []float64, add func(int, int, float64), f []float64) {
	v := n.opAmpVoltages(p, x)
	pole := n.index[poleNode(p.ID)]
	vp := voltage(x, pole)
	e := opAmpCurrentsControlled(v, vp, true)
	pins := n.opAmpPins(p)
	for row, node := range pins {
		if node >= 0 {
			f[node] += e.current[row]
		}
		for col, other := range pins {
			add(node, other, e.jacobian[row][col])
		}
		add(node, pole, e.poleDerivative[row])
	}
	j, d, rail := poleCurrent(p, v, vp)
	if pins[rail] >= 0 {
		f[pins[rail]] += j
	}
	f[pole] -= j
	for k, node := range append(pins[:], pole) {
		add(pins[rail], node, d[k])
		add(pole, node, -d[k])
	}
}

type DynamicsResult struct {
	ID      string  `json:"id"`
	Voltage float64 `json:"pole_voltage_v"`
	Current float64 `json:"compensation_current_a"`
	Energy  float64 `json:"stored_energy_j"`
}

func (n *mixedNetwork) dynamicsResult(p Component, x []float64) DynamicsResult {
	v := n.opAmpVoltages(p, x)
	vp := voltage(x, n.index[poleNode(p.ID)])
	j, _, _ := poleCurrent(p, v, vp)
	relative := vp - v[4]
	return DynamicsResult{p.ID, relative, j, .5 * compensationC * relative * relative}
}
func (n *mixedNetwork) restoreDynamics(r *MixedResult, x []float64) error {
	for _, p := range n.authored.Components {
		if !hasDynamics(p) {
			continue
		}
		found := false
		for _, d := range r.Dynamics {
			if d.ID != p.ID {
				continue
			}
			if !finite(d.Voltage) || math.Abs(d.Voltage) > maxMixedNodeVoltage {
				return fmt.Errorf("invalid compensation state")
			}
			x[n.index[poleNode(p.ID)]] = voltage(x, n.pin(p, "vminus")) + d.Voltage
			for k, c := range n.caps {
				if c.ID == poleCap(p.ID) {
					x[n.capacitorIndex(k)] = d.Current
				}
			}
			found = true
		}
		if !found {
			return fmt.Errorf("missing op-amp compensation state")
		}
	}
	return nil
}
func validateRecordedDynamics(c Compiled, g Graph, r *MixedResult) error {
	expected := 0
	seen := map[string]bool{}
	for _, p := range g.Components {
		if hasDynamics(p) {
			expected++
		}
	}
	if len(r.Dynamics) != expected {
		return fmt.Errorf("wrong compensation state count")
	}
	if expected == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, d := range r.Dynamics {
		p, ok := g.Component(d.ID)
		if !ok || !hasDynamics(p) || seen[d.ID] {
			return fmt.Errorf("invalid compensation identity")
		}
		seen[d.ID] = true
		e := n.dynamicsResult(p, x)
		if !recordingNear(d.Current, e.Current) || !recordingNear(d.Energy, e.Energy) {
			return fmt.Errorf("compensation readings disagree with state")
		}
	}
	return nil
}
