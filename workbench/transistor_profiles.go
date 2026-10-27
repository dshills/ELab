package workbench

import (
	"fmt"
	"math"
	"strings"
)

const TransistorProfileModelID = "elab.bipolar-profile.v1"

// A deliberately bounded educational law. These are not manufacturer SPICE cards.
type TransistorProfile struct {
	Version           string  `json:"version"`
	ForwardBeta       float64 `json:"forward_beta"`
	ReverseBeta       float64 `json:"reverse_beta"`
	SaturationCurrent float64 `json:"saturation_current_a"`
	EarlyVoltage      float64 `json:"early_voltage_v"`
	CBE               float64 `json:"base_emitter_capacitance_f"`
	CBC               float64 `json:"base_collector_capacitance_f"`
}
type TransistorProfileParams struct {
	ID         string            `json:"id"`
	Enabled    bool              `json:"enabled"`
	Parameters TransistorProfile `json:"parameters"`
}

func DefaultTransistorProfile() TransistorProfile {
	return TransistorProfile{TransistorProfileModelID, 100, 1, 1e-14, 100, 20e-12, 5e-12}
}
func hasTransistorProfile(p Component) bool {
	return IsBipolarKind(p.Kind) && p.Device != nil && p.Device.Bipolar != nil
}
func validateTransistorProfile(v TransistorProfile) error {
	if v.Version != TransistorProfileModelID {
		return fmt.Errorf("unsupported transistor profile version")
	}
	if !finite(v.ForwardBeta) || v.ForwardBeta < 1 || v.ForwardBeta > 1000 || !finite(v.ReverseBeta) || v.ReverseBeta < .1 || v.ReverseBeta > 100 || !finite(v.SaturationCurrent) || v.SaturationCurrent < 1e-18 || v.SaturationCurrent > 1e-10 || !finite(v.EarlyVoltage) || v.EarlyVoltage < 0 || v.EarlyVoltage > 1000 || v.EarlyVoltage > 0 && v.EarlyVoltage < 1 || !finite(v.CBE) || v.CBE < 0 || v.CBE > 1e-6 || !finite(v.CBC) || v.CBC < 0 || v.CBC > 1e-6 {
		return fmt.Errorf("profile bounds: beta 1–1000, reverse beta 0.1–100, IS 1e-18–1e-10 A, Early voltage 0 (off) or 1–1000 V, each capacitance 0–1 µF")
	}
	return nil
}
func componentBipolarModel(p Component) NPNModel {
	m := bipolarModel(p.Kind)
	if !hasTransistorProfile(p) {
		return m
	}
	v := p.Device.Bipolar
	m.ID = TransistorProfileModelID + "." + p.Kind
	m.ForwardBeta, m.ReverseBeta, m.SaturationCurrent = v.ForwardBeta, v.ReverseBeta, v.SaturationCurrent
	m.EarlyVoltage, m.CBE, m.CBC = v.EarlyVoltage, v.CBE, v.CBC
	m.Limitations = "Educational configurable reciprocal transport model at 27 C; forward Early term IS*max(expm1(VBE/VT),0)*max(VCE,0)/VA, mirrored for PNP. Constant linear junction capacitors, biased startup. No voltage-dependent depletion charge, diffusion storage, high-injection gain rolloff, breakdown, thermal/noise effects or manufacturer identity. Each nonzero junction capacitor consumes existing storage/part budget."
	return m
}
func componentBipolarLinear(p Component, vbe, vce float64) (ib, ic, bb, bc, cb, cc float64) {
	if !hasTransistorProfile(p) {
		return bipolarLinear(p.Kind, vbe, vce)
	}
	sign := bipolarPolarity(p.Kind)
	b, c := sign*vbe, sign*vce
	m := componentBipolarModel(p)
	vt := 1.380649e-23 * (273.15 + m.Temperature) / 1.602176634e-19
	eb, ec := math.Exp(math.Max(-80, math.Min(80, b/vt))), math.Exp(math.Max(-80, math.Min(80, (b-c)/vt)))
	f, r := m.SaturationCurrent*(eb-1), m.SaturationCurrent*(ec-1)
	gf, gr := m.SaturationCurrent*eb/vt, m.SaturationCurrent*ec/vt
	if b/vt <= -80 || b/vt >= 80 {
		gf = 0
	}
	if (b-c)/vt <= -80 || (b-c)/vt >= 80 {
		gr = 0
	}
	ib, ic = f/m.ForwardBeta+r/m.ReverseBeta, f-(1+1/m.ReverseBeta)*r
	bb, bc = gf/m.ForwardBeta+gr/m.ReverseBeta, -gr/m.ReverseBeta
	cb, cc = gf-(1+1/m.ReverseBeta)*gr, (1+1/m.ReverseBeta)*gr
	if m.EarlyVoltage > 0 && f > 0 && c > 0 {
		ic += f * c / m.EarlyVoltage
		cb += gf * c / m.EarlyVoltage
		cc += f / m.EarlyVoltage
	}
	return sign * ib, sign * ic, bb, bc, cb, cc
}
func junctionCapID(id, side string) string { return "$" + id + "_junction_" + side }
func isJunctionCap(p Component) bool {
	return p.Kind == "capacitor" && strings.HasPrefix(p.ID, "$") && strings.Contains(p.ID, "_junction_")
}
func lowerTransistorProfile(c *Compiled, p Component) []Component {
	parts := []Component{p}
	v := p.Device.Bipolar
	for _, j := range []struct {
		side, pin string
		value     float64
	}{{"be", "emitter", v.CBE}, {"bc", "collector", v.CBC}} {
		if j.value == 0 {
			continue
		}
		cap := Component{ID: junctionCapID(p.ID, j.side), Kind: "capacitor", Terminals: []string{"1", "2"}, Value: j.value}
		c.TerminalNet[cap.ID+".1"] = c.TerminalNet[p.ID+".base"]
		c.TerminalNet[cap.ID+".2"] = c.TerminalNet[p.ID+"."+j.pin]
		parts = append(parts, cap)
	}
	return parts
}
func (n *mixedNetwork) junctionReading(p Component, x []float64) (be, bc, energy float64) {
	if !hasTransistorProfile(p) {
		return
	}
	for k, c := range n.caps {
		if c.ID != junctionCapID(p.ID, "be") && c.ID != junctionCapID(p.ID, "bc") {
			continue
		}
		current := 0.
		if n.capacitorIndex(k) >= len(x) && !n.dc {
			return math.NaN(), math.NaN(), math.NaN()
		}
		if n.capacitorIndex(k) < len(x) {
			current = x[n.capacitorIndex(k)]
		}
		if c.ID == junctionCapID(p.ID, "be") {
			be = current
		} else {
			bc = current
		}
		v := voltage(x, n.pin(c, "1")) - voltage(x, n.pin(c, "2"))
		energy += .5 * c.Value * v * v
	}
	return
}
func (n *mixedNetwork) restoreJunctions(r *MixedResult, x []float64) error {
	for _, p := range n.authored.Components {
		if !hasTransistorProfile(p) {
			continue
		}
		found := false
		for _, q := range r.Transistors {
			if q.TransistorID != p.ID {
				continue
			}
			found = true
			if !finite(q.JunctionBECurrent) || !finite(q.JunctionBCCurrent) {
				return fmt.Errorf("nonfinite junction current")
			}
			for k, c := range n.caps {
				if (c.ID == junctionCapID(p.ID, "be") || c.ID == junctionCapID(p.ID, "bc")) && n.capacitorIndex(k) >= len(x) {
					return fmt.Errorf("missing junction current state slots")
				}
				if c.ID == junctionCapID(p.ID, "be") {
					x[n.capacitorIndex(k)] = q.JunctionBECurrent
				}
				if c.ID == junctionCapID(p.ID, "bc") {
					x[n.capacitorIndex(k)] = q.JunctionBCCurrent
				}
			}
		}
		if !found {
			return fmt.Errorf("missing profiled transistor state")
		}
	}
	return nil
}

func hasTransistorProfiles(g Graph) bool {
	for _, p := range g.Components {
		if hasTransistorProfile(p) {
			return true
		}
	}
	return false
}

func hasJunctionStorage(p Component) bool {
	return hasTransistorProfile(p) && (p.Device.Bipolar.CBE > 0 || p.Device.Bipolar.CBC > 0)
}
