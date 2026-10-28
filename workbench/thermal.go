package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const ThermalModelID = "elab.electrothermal.v1"
const thermalScale = 100.
const thermalReferenceC = 27.
const thermalDescription = "Optional generic resistor/silicon diode/NPN/PNP temperature model. Reference 27 C, resistor R=Rref*(1+alpha*(T-27)); junction VT=kT/q and IS(T)=ISref*(T/Tref)^(3/n)*exp(1.11eV/(n*k)*(1/Tref-1/T)), n=1 for bipolar transport, fixed beta/Early law. Optional one-node thermal RC: Cth*dT/dt=conductive heat-(T-ambient)/Rth, startup at ambient. No package fit, heat-sink geometry, mutual thermal coupling, ratings or protection. Electrical and thermal energy are reported separately. Each thermal RC reserves a capacitor, resistor, two primitives and node within existing capacities."

type ThermalConfig struct {
	Version     string  `json:"version"`
	Ambient     float64 `json:"ambient_c"`
	SelfHeating bool    `json:"self_heating"`
	Resistance  float64 `json:"thermal_resistance_k_per_w"`
	Capacitance float64 `json:"thermal_capacitance_j_per_k"`
	Alpha       float64 `json:"resistance_coefficient_per_k"`
}
type ThermalParams struct {
	ID         string        `json:"id"`
	Enabled    bool          `json:"enabled"`
	Parameters ThermalConfig `json:"parameters"`
}
type ThermalReading struct {
	ID             string  `json:"id"`
	Model          string  `json:"model"`
	Ambient        float64 `json:"ambient_c"`
	Temperature    float64 `json:"temperature_c"`
	ConductiveHeat float64 `json:"conductive_heat_w"`
	Transfer       float64 `json:"heat_to_ambient_w"`
	StorageRate    float64 `json:"heat_storage_rate_w"`
	Balance        float64 `json:"thermal_balance_residual_w"`
	StoredHeat     float64 `json:"stored_sensible_heat_j"`
	Resistance     float64 `json:"electrical_resistance_ohm,omitempty"`
}

func DefaultThermalConfig(kind string) ThermalConfig {
	c := ThermalConfig{Version: ThermalModelID, Ambient: 27, Resistance: 100, Capacitance: .01}
	if kind == "resistor" {
		c.Alpha = .0001
	}
	return c
}
func thermalKind(kind string) bool {
	return kind == "resistor" || kind == "diode" || IsBipolarKind(kind)
}
func hasThermal(p Component) bool { return p.Thermal != nil }
func hasThermals(g Graph) bool {
	for _, p := range g.Components {
		if hasThermal(p) {
			return true
		}
	}
	return false
}
func cloneThermal(v *ThermalConfig) *ThermalConfig {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}
func validateThermal(p Component) error {
	if p.Thermal == nil {
		return nil
	}
	v := *p.Thermal
	if !thermalKind(p.Kind) || v.Version != ThermalModelID || !finite(v.Ambient) || v.Ambient < -40 || v.Ambient > 125 || !finite(v.Resistance) || v.Resistance < .1 || v.Resistance > 2000 || !finite(v.Capacitance) || v.Capacitance < .001 || v.Capacitance > 10 || !finite(v.Alpha) || math.Abs(v.Alpha) > .005 || p.Kind != "resistor" && v.Alpha != 0 {
		return fmt.Errorf("thermal model: resistor/silicon diode/NPN/PNP; ambient -40..125 C, Rth 0.1..2000 K/W, Cth 0.001..10 J/K, resistor coefficient +/-0.005/K (zero on other kinds)")
	}
	return nil
}
func thermalFields() []string {
	return []string{"version", "ambient_c", "self_heating", "thermal_resistance_k_per_w", "thermal_capacitance_j_per_k", "resistance_coefficient_per_k"}
}
func (s *Service) configureThermalLocked(raw json.RawMessage) error {
	var v ThermalParams
	if err := params(raw, &v, "id", "enabled", "parameters"); err != nil {
		return err
	}
	if err := paramsNested(raw, "parameters", &ThermalConfig{}, thermalFields()...); err != nil {
		return err
	}
	p, ok := s.graph.Component(v.ID)
	if !ok || !thermalKind(p.Kind) {
		return fmt.Errorf("select a resistor, silicon diode or bipolar transistor")
	}
	p.Thermal = &v.Parameters
	if err := validateThermal(p); err != nil {
		return err
	}
	if !v.Enabled {
		p.Thermal = nil
	}
	for k, c := range s.graph.Components {
		if c.ID == p.ID {
			s.graph.Components[k] = p
			return nil
		}
	}
	return fmt.Errorf("component missing")
}
func thermalNode(id string) string { return "$" + id + "_temperature" }
func thermalCap(id string) string  { return "$" + id + "_thermal_storage" }
func isThermalCap(p Component) bool {
	return privatePart(p.ID) && strings.HasSuffix(p.ID, "_thermal_storage")
}
func isThermalNode(id string) bool { return privatePart(id) && strings.HasSuffix(id, "_temperature") }
func lowerThermal(c *Compiled, p Component) []Component {
	if !hasThermal(p) || !p.Thermal.SelfHeating {
		return nil
	}
	node := thermalNode(p.ID)
	c.Nets = append(c.Nets, Net{ID: node})
	zero := 0.
	cap := Component{ID: thermalCap(p.ID), Kind: "capacitor", Terminals: []string{"1", "2"}, Value: p.Thermal.Capacitance, InitialVoltage: &zero}
	c.TerminalNet[cap.ID+".1"] = node
	c.TerminalNet[cap.ID+".2"] = c.Ground
	return []Component{cap}
}
func (n *mixedNetwork) thermalTemperature(p Component, x []float64) float64 {
	t := p.Thermal.Ambient
	if p.Thermal.SelfHeating {
		t += thermalScale * voltage(x, n.index[thermalNode(p.ID)])
	}
	return t
}
func thermalSaturationScale(temperature, n float64) (factor, logDerivative float64) {
	k := 8.617333262145e-5
	t := 273.15 + temperature
	ref := 273.15 + thermalReferenceC
	factor = math.Pow(t/ref, 3/n) * math.Exp(1.11/(n*k)*(1/ref-1/t))
	logDerivative = (3/t + 1.11/(k*t*t)) / n
	return
}

type thermalEvaluation struct {
	count       int
	pins        [3]int
	currents    [3]float64
	jacobian    [3][3]float64
	derivativeT [3]float64
	heat        float64
	heatV       [3]float64
	heatT       float64
	resistance  float64
}

func (n *mixedNetwork) evaluateThermal(p Component, x []float64) thermalEvaluation {
	raw := n.thermalTemperature(p, x)
	temperature := math.Max(-40, math.Min(150, raw))
	within := raw >= -40 && raw <= 150
	e := thermalEvaluation{count: 2, pins: [3]int{n.pin(p, "1"), n.pin(p, "2"), -1}}
	if p.Kind == "diode" {
		e.pins = [3]int{n.pin(p, "anode"), n.pin(p, "cathode"), -1}
	}
	if IsBipolarKind(p.Kind) {
		e.count = 3
		e.pins = [3]int{n.pin(p, "base"), n.pin(p, "collector"), n.pin(p, "emitter")}
		vb := voltage(x, e.pins[0]) - voltage(x, e.pins[2])
		vc := voltage(x, e.pins[1]) - voltage(x, e.pins[2])
		ib, ic, bb, bc, cb, cc, bt, ct := thermalBipolarLinear(p, temperature, vb, vc)
		e.currents = [3]float64{ib, ic, -ib - ic}
		e.jacobian = [3][3]float64{{bb, bc, -bb - bc}, {cb, cc, -cb - cc}, {-bb - cb, -bc - cc, bb + bc + cb + cc}}
		e.derivativeT = [3]float64{bt, ct, -bt - ct}
	} else {
		v := voltage(x, e.pins[0]) - voltage(x, e.pins[1])
		current, g, it := 0., 0., 0.
		if p.Kind == "resistor" {
			r := p.Value * (1 + p.Thermal.Alpha*(temperature-thermalReferenceC))
			e.resistance = r
			g = 1 / r
			current = v * g
			it = -v * p.Value * p.Thermal.Alpha / (r * r)
		} else {
			m := SiliconDiodeModel()
			scale, l := thermalSaturationScale(temperature, m.EmissionCoefficient)
			m.SaturationCurrent *= scale
			m.Temperature = temperature
			current, g = junctionDiodeCurrent(m, v)
			nv := m.EmissionCoefficient * 1.380649e-23 * (273.15 + temperature) / 1.602176634e-19
			junction := v - current*m.SeriesResistance
			gj := (current + m.SaturationCurrent) / nv
			it = (current*l - (current+m.SaturationCurrent)*junction/(nv*(273.15+temperature))) / (1 + m.SeriesResistance*gj)
		}
		e.currents = [3]float64{current, -current, 0}
		e.jacobian = [3][3]float64{{g, -g, 0}, {-g, g, 0}, {}}
		e.derivativeT = [3]float64{it, -it, 0}
	}
	if !within {
		e.derivativeT = [3]float64{}
	}
	for i := 0; i < e.count; i++ {
		v := voltage(x, e.pins[i])
		e.heat += v * e.currents[i]
		e.heatT += v * e.derivativeT[i]
		e.heatV[i] += e.currents[i]
		for j := 0; j < e.count; j++ {
			e.heatV[j] += v * e.jacobian[i][j]
		}
	}
	return e
}
func thermalBipolarLinear(p Component, temperature, vbe, vce float64) (ib, ic, bb, bc, cb, cc, bt, ct float64) {
	m := componentBipolarModel(p)
	factor, l := thermalSaturationScale(temperature, 1)
	is := m.SaturationCurrent * factor
	t := 273.15 + temperature
	vt := 1.380649e-23 * t / 1.602176634e-19
	sign := bipolarPolarity(p.Kind)
	b, c := sign*vbe, sign*vce
	junction := func(v float64) (f, g, derivativeT float64) {
		z := v / vt
		limited := math.Max(-80, math.Min(80, z))
		ez := math.Exp(limited)
		f = is * (ez - 1)
		derivativeT = f * l
		if z > -80 && z < 80 {
			g = is * ez / vt
			derivativeT -= is * ez * z / t
		}
		return
	}
	f, gf, ft := junction(b)
	r, gr, rt := junction(b - c)
	ib, ic = f/m.ForwardBeta+r/m.ReverseBeta, f-(1+1/m.ReverseBeta)*r
	bb, bc = gf/m.ForwardBeta+gr/m.ReverseBeta, -gr/m.ReverseBeta
	cb, cc = gf-(1+1/m.ReverseBeta)*gr, (1+1/m.ReverseBeta)*gr
	bt, ct = ft/m.ForwardBeta+rt/m.ReverseBeta, ft-(1+1/m.ReverseBeta)*rt
	if m.EarlyVoltage > 0 && f > 0 && c > 0 {
		ic += f * c / m.EarlyVoltage
		cb += gf * c / m.EarlyVoltage
		cc += f / m.EarlyVoltage
		ct += ft * c / m.EarlyVoltage
	}
	return sign * ib, sign * ic, bb, bc, cb, cc, sign * bt, sign * ct
}
func (n *mixedNetwork) stampThermal(p Component, x []float64, add func(int, int, float64), f []float64) {
	e := n.evaluateThermal(p, x)
	theta := -1
	if p.Thermal.SelfHeating {
		theta = n.index[thermalNode(p.ID)]
	}
	for i := 0; i < e.count; i++ {
		if e.pins[i] >= 0 {
			f[e.pins[i]] += e.currents[i]
		}
		for j := 0; j < e.count; j++ {
			add(e.pins[i], e.pins[j], e.jacobian[i][j])
		}
		if theta >= 0 {
			add(e.pins[i], theta, thermalScale*e.derivativeT[i])
		}
	}
	if theta >= 0 {
		f[theta] += x[theta]/p.Thermal.Resistance - e.heat/thermalScale
		add(theta, theta, 1/p.Thermal.Resistance-e.heatT)
		for j := 0; j < e.count; j++ {
			add(theta, e.pins[j], -e.heatV[j]/thermalScale)
		}
	}
}
func (n *mixedNetwork) thermalReading(p Component, x []float64) ThermalReading {
	e := n.evaluateThermal(p, x)
	temperature := n.thermalTemperature(p, x)
	r := ThermalReading{ID: p.ID, Model: ThermalModelID, Ambient: p.Thermal.Ambient, Temperature: temperature, ConductiveHeat: e.heat, Resistance: e.resistance}
	r.Transfer = e.heat // Imposed-temperature bath removes the conductive heat.
	if p.Thermal.SelfHeating {
		rise := temperature - p.Thermal.Ambient
		r.Transfer = rise / p.Thermal.Resistance
		r.StoredHeat = p.Thermal.Capacitance * rise
	}
	return r
}
func (n *mixedNetwork) validateThermalState(x []float64) error {
	for _, p := range n.c.MixedGraph.Components {
		if hasThermal(p) {
			t := n.thermalTemperature(p, x)
			if !finite(t) || t < -40-1e-8 || t > 150+1e-8 {
				return fmt.Errorf("%s junction temperature exceeds -40..150 C model range; reduce power or thermal resistance", p.ID)
			}
		}
	}
	return nil
}

// Reconstruct the private thermal domain from its explicit capture reading.
// Thermal storage is never represented as electrical capacitor charge.
func (n *mixedNetwork) restoreThermals(r *MixedResult, x []float64) error {
	seen := map[string]bool{}
	for _, a := range r.Thermal {
		p, ok := n.authored.Component(a.ID)
		if !ok || !hasThermal(p) || seen[a.ID] || a.Model != ThermalModelID || a.Ambient != p.Thermal.Ambient || !finite(a.Temperature) || a.Temperature < -40-1e-8 || a.Temperature > 150+1e-8 || !finite(a.StorageRate) {
			return fmt.Errorf("invalid recorded thermal state")
		}
		seen[a.ID] = true
		if p.Thermal.SelfHeating {
			x[n.index[thermalNode(p.ID)]] = (a.Temperature - p.Thermal.Ambient) / thermalScale
			found := false
			for k, c := range n.caps {
				if c.ID == thermalCap(p.ID) {
					if n.capacitorIndex(k) >= len(x) {
						return fmt.Errorf("missing thermal storage current slot")
					}
					found = true
					x[n.capacitorIndex(k)] = a.StorageRate / thermalScale
				}
			}
			if !found {
				return fmt.Errorf("missing thermal storage capacitor")
			}
		} else if a.Temperature != a.Ambient || a.StorageRate != 0 {
			return fmt.Errorf("imposed temperature has no thermal storage")
		}
		if p.Kind == "resistor" || p.Kind == "diode" {
			branches := r.Resistors
			if p.Kind == "diode" {
				branches = r.Diodes
			}
			found := false
			evaluation := n.evaluateThermal(p, x)
			v := voltage(x, evaluation.pins[0]) - voltage(x, evaluation.pins[1])
			for _, b := range branches {
				if b.ID == p.ID {
					found = true
					if !recordingNear(b.Current, evaluation.currents[0]) || !recordingNear(b.Voltage, v) || !recordingNear(b.Power, evaluation.heat) {
						return fmt.Errorf("recorded thermal electrical branch disagrees with state")
					}
				}
			}
			if !found {
				return fmt.Errorf("missing recorded thermal electrical branch")
			}
		}
		e := n.thermalReading(p, x)
		e.Balance = e.ConductiveHeat - e.Transfer - a.StorageRate
		for _, q := range [][2]float64{{a.ConductiveHeat, e.ConductiveHeat}, {a.Transfer, e.Transfer}, {a.StoredHeat, e.StoredHeat}, {a.Resistance, e.Resistance}, {a.Balance, e.Balance}} {
			if !recordingNear(q[0], q[1]) {
				return fmt.Errorf("recorded thermal readings disagree with state")
			}
		}
		if math.Abs(a.Balance) > 1e-5 {
			return fmt.Errorf("recorded thermal heat balance exceeds solver tolerance")
		}
	}
	for _, p := range n.authored.Components {
		if hasThermal(p) && !seen[p.ID] {
			return fmt.Errorf("missing recorded thermal state")
		}
	}
	return nil
}
