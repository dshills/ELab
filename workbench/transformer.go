package workbench

import (
	"fmt"
	"math"
)

var transformerPins = []string{"primary1", "primary2", "secondary1", "secondary2"}

type TransformerParams struct {
	ID         string            `json:"id"`
	Parameters TransformerConfig `json:"parameters"`
}

func transformerDefault() TransformerConfig {
	return TransformerConfig{Inductance: .1, Ratio: .5, Coupling: .98, PrimaryResistance: 1, SecondaryResistance: 1}
}
func transformerParameterFields() []string {
	return []string{"primary_inductance_h", "secondary_primary_turns_ratio", "coupling", "primary_resistance_ohm", "secondary_resistance_ohm", "reversed_secondary"}
}
func validateTransformer(c TransformerConfig) error {
	for _, v := range []struct{ v, min, max float64 }{{c.Inductance, 1e-6, 100}, {c.Ratio, .1, 10}, {c.Coupling, 0, .9999}, {c.PrimaryResistance, .01, 1000}, {c.SecondaryResistance, .01, 1000}} {
		if !finite(v.v) || v.v < v.min || v.v > v.max {
			return fmt.Errorf("transformer parameters exceed the declared domains; exact coupling 1 is not supported")
		}
	}
	secondary := c.Inductance * c.Ratio * c.Ratio
	if secondary < 1e-6 || secondary > 100 {
		return fmt.Errorf("derived secondary inductance must be 1 µH–100 H (Lp × turns ratio²)")
	}
	return nil
}
func (s *Service) configureTransformerLocked(v TransformerParams) error {
	if err := validateTransformer(v.Parameters); err != nil {
		return err
	}
	for i, p := range s.graph.Components {
		if p.ID == v.ID && p.Kind == "transformer" {
			p.Power = &PowerParameters{Transformer: &v.Parameters}
			s.graph.Components[i] = p
			return nil
		}
	}
	return fmt.Errorf("choose a transformer / coupled winding")
}
func transformerWindingID(id string, secondary bool) string {
	suffix := "_primary"
	if secondary {
		suffix = "_secondary"
	}
	return "$" + id + suffix
}
func lowerTransformer(c *Compiled, p Component) []Component {
	cfg := p.Power.Transformer
	parts := []Component{}
	for _, secondary := range []bool{false, true} {
		id, pin, L := transformerWindingID(p.ID, secondary), "primary", cfg.Inductance
		if secondary {
			pin = "secondary"
			L *= cfg.Ratio * cfg.Ratio
		}
		zero := 0.
		q := Component{ID: id, Kind: "inductor", Terminals: []string{"1", "2"}, Value: L, InitialCurrent: &zero, Power: clonePower(p.Power)}
		c.TerminalNet[id+".1"] = c.TerminalNet[p.ID+"."+pin+"1"]
		c.TerminalNet[id+".2"] = c.TerminalNet[p.ID+"."+pin+"2"]
		parts = append(parts, q)
	}
	return parts
}

type windingCoupling struct {
	first, second int
	mutual        float64
}

func (n *mixedNetwork) prepareCouplings() {
	for _, p := range n.authored.Components {
		if p.Kind != "transformer" {
			continue
		}
		first, second := -1, -1
		for k, q := range n.inductors {
			if q.ID == transformerWindingID(p.ID, false) {
				first = k
			}
			if q.ID == transformerWindingID(p.ID, true) {
				second = k
			}
		}
		if first < 0 || second < 0 {
			continue
		}
		c := p.Power.Transformer
		M := c.Coupling * c.Inductance * c.Ratio
		if c.Reversed {
			M = -M
		}
		n.couplings = append(n.couplings, windingCoupling{first, second, M})
	}
}
func (n *mixedNetwork) windingPartner(k int) (int, float64) {
	for _, c := range n.couplings {
		if c.first == k {
			return c.second, c.mutual
		}
		if c.second == k {
			return c.first, c.mutual
		}
	}
	return -1, 0
}
func (n *mixedNetwork) inverseInductance() [][]float64 {
	a := denseMatrix(len(n.inductors))
	for k, p := range n.inductors {
		a[k][k] = 1 / p.Value
	}
	for _, c := range n.couplings {
		Lp, Ls := n.inductors[c.first].Value, n.inductors[c.second].Value
		det := Lp*Ls - c.mutual*c.mutual
		a[c.first][c.first] = Ls / det
		a[c.second][c.second] = Lp / det
		a[c.first][c.second] = -c.mutual / det
		a[c.second][c.first] = -c.mutual / det
	}
	return a
}

// Open-winding startup still requires dI/dt=0 through its cut. For coupled
// windings that derivative depends on both winding voltages through L inverse.
func (n *mixedNetwork) stampCoupledStartup(x []float64, a [][]float64, f []float64) {
	inverse := n.inverseInductance()
	for _, cut := range n.inductorCuts {
		row := cut.row
		clear(a[row])
		f[row] = 0
		weights := make([]float64, len(n.inductors))
		for _, edge := range cut.edges {
			for j, v := range inverse[edge.index] {
				weights[j] += edge.sign * v
			}
		}
		scale := 0.
		for _, v := range weights {
			scale += math.Abs(v)
		}
		for j, weight := range weights {
			if weight == 0 {
				continue
			}
			weight /= scale
			p := n.inductors[j]
			one, two := n.pin(p, "1"), n.pin(p, "2")
			R := inductorWindingResistance(p)
			f[row] += weight * (voltage(x, one) - voltage(x, two) - R*x[n.inductorIndex(j)])
			a[row][n.inductorIndex(j)] -= weight * R
			if one >= 0 {
				a[row][one] += weight
			}
			if two >= 0 {
				a[row][two] -= weight
			}
		}
	}
}
func (n *mixedNetwork) minimumWindingInductance() float64 {
	minimum := math.Inf(1)
	for _, p := range n.inductors {
		minimum = math.Min(minimum, p.Value)
	}
	for _, c := range n.couplings {
		Lp, Ls := n.inductors[c.first].Value, n.inductors[c.second].Value
		largest := (Lp + Ls + math.Hypot(Lp-Ls, 2*c.mutual)) / 2
		smallest := (Lp*Ls - c.mutual*c.mutual) / largest
		minimum = math.Min(minimum, smallest)
	}
	return minimum
}

type TransformerResult struct {
	ID                   string  `json:"id"`
	Model                string  `json:"model"`
	PrimaryVoltage       float64 `json:"primary_voltage_v"`
	SecondaryVoltage     float64 `json:"secondary_voltage_v"`
	PrimaryCurrent       float64 `json:"primary_current_a"`
	SecondaryCurrent     float64 `json:"secondary_current_a"`
	PrimaryFluxLinkage   float64 `json:"primary_flux_linkage_wb_turn"`
	SecondaryFluxLinkage float64 `json:"secondary_flux_linkage_wb_turn"`
	Energy               float64 `json:"magnetic_energy_j"`
	WindingLoss          float64 `json:"winding_loss_w"`
	PrimaryPower         float64 `json:"primary_power_w"`
	SecondaryPower       float64 `json:"secondary_power_w"`
	Power                float64 `json:"terminal_power_w"`
}

func (n *mixedNetwork) transformerCurrentIndex(id string, secondary bool) int {
	for k, p := range n.inductors {
		if p.ID == transformerWindingID(id, secondary) {
			return n.inductorIndex(k)
		}
	}
	return -1
}
func (n *mixedNetwork) transformerResult(p Component, x []float64) TransformerResult {
	c := p.Power.Transformer
	Lp, Ls, M := c.Inductance, c.Inductance*c.Ratio*c.Ratio, c.Coupling*c.Inductance*c.Ratio
	if c.Reversed {
		M = -M
	}
	ip, is := x[n.transformerCurrentIndex(p.ID, false)], x[n.transformerCurrentIndex(p.ID, true)]
	vp := voltage(x, n.pin(p, "primary1")) - voltage(x, n.pin(p, "primary2"))
	vs := voltage(x, n.pin(p, "secondary1")) - voltage(x, n.pin(p, "secondary2"))
	// Complete the square to avoid subtracting large almost-equal energies.
	energy := .5*Lp*math.Pow(ip+M/Lp*is, 2) + .5*Ls*(1-c.Coupling*c.Coupling)*is*is
	return TransformerResult{p.ID, PowerModelID, vp, vs, ip, is, Lp*ip + M*is, M*ip + Ls*is, energy, c.PrimaryResistance*ip*ip + c.SecondaryResistance*is*is, vp * ip, vs * is, vp*ip + vs*is}
}
func (n *mixedNetwork) restoreTransformerNodes(p Component, r *MixedResult, x []float64) error {
	for _, u := range r.Transformers {
		if u.ID == p.ID {
			if !finite(u.PrimaryCurrent) || !finite(u.SecondaryCurrent) {
				return fmt.Errorf("invalid winding currents")
			}
			x[n.transformerCurrentIndex(p.ID, false)] = u.PrimaryCurrent
			x[n.transformerCurrentIndex(p.ID, true)] = u.SecondaryCurrent
			return nil
		}
	}
	return fmt.Errorf("missing transformer winding state")
}
func validateRecordedTransformers(c Compiled, g Graph, r *MixedResult) error {
	ids := []string{}
	for _, u := range r.Transformers {
		ids = append(ids, u.ID)
	}
	if err := recordedObjectIDs(g, "transformer", ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, u := range r.Transformers {
		p, _ := g.Component(u.ID)
		e := n.transformerResult(p, x)
		if u.Model != PowerModelID {
			return fmt.Errorf("invalid transformer model")
		}
		for _, v := range [][2]float64{{u.PrimaryVoltage, e.PrimaryVoltage}, {u.SecondaryVoltage, e.SecondaryVoltage}, {u.PrimaryFluxLinkage, e.PrimaryFluxLinkage}, {u.SecondaryFluxLinkage, e.SecondaryFluxLinkage}, {u.Energy, e.Energy}, {u.WindingLoss, e.WindingLoss}, {u.PrimaryPower, e.PrimaryPower}, {u.SecondaryPower, e.SecondaryPower}, {u.Power, e.Power}} {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("transformer readings disagree with mutual flux and winding laws")
			}
		}
	}
	return nil
}
