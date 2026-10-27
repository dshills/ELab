package workbench

import "fmt"

const JFETModelID = "elab.generic-jfet.v1"
const jfetDescription = JFETModelID + ": Educational symmetric depletion-mode square-law channel with configurable IDSS, signed gate cutoff voltage and channel-length modulation. N-channel cutoff is negative, P-channel positive. Explicit gate-source and gate-drain generic silicon PN junctions each use silicon-diode-v1 plus 1 Gohm bleed. No MOS body diode. Constant 27 C parameters, no breakdown, intrinsic capacitance, diffusion charge, noise, thermal evolution, ratings or manufacturer identity. External capacitors set transient/AC timing."

type JFETConfig struct {
	Version string  `json:"version"`
	IDSS    float64 `json:"idss_a"`
	Cutoff  float64 `json:"cutoff_v"`
	Lambda  float64 `json:"lambda_per_v"`
}
type JFETParams struct {
	ID         string     `json:"id"`
	Parameters JFETConfig `json:"parameters"`
}
type JFETResult struct {
	ID            string     `json:"id"`
	Model         string     `json:"model"`
	Parameters    JFETConfig `json:"parameters"`
	GateCurrent   float64    `json:"gate_current_a"`
	DrainCurrent  float64    `json:"drain_current_a"`
	SourceCurrent float64    `json:"source_current_a"`
	VGS           float64    `json:"vgs_v"`
	VDS           float64    `json:"vds_v"`
	Power         float64    `json:"power_w"`
	State         string     `json:"state"`
}

func IsJFETKind(k string) bool { return k == "njfet" || k == "pjfet" }
func DefaultJFETConfig(kind string) JFETConfig {
	v := JFETConfig{JFETModelID, .005, -2, .01}
	if kind == "pjfet" {
		v.Cutoff = 2
	}
	return v
}
func jfetFields() []string { return []string{"version", "idss_a", "cutoff_v", "lambda_per_v"} }
func validateJFET(kind string, c JFETConfig) error {
	v := c.Cutoff
	if kind == "pjfet" {
		v = -v
	}
	if !IsJFETKind(kind) || c.Version != JFETModelID || !finite(c.IDSS) || c.IDSS < 1e-5 || c.IDSS > .1 || !finite(v) || v > -.1 || v < -12 || !finite(c.Lambda) || c.Lambda < 0 || c.Lambda > .1 {
		return fmt.Errorf("JFET requires supported version, IDSS 10 µA–100 mA, cutoff -12 to -0.1 V (N) or +0.1 to +12 V (P), lambda 0–0.1 per V")
	}
	return nil
}
func jfetSign(k string) float64 {
	if k == "pjfet" {
		return -1
	}
	return 1
}
func jfetPositive(c JFETConfig, cutoff, vg, vd float64) (i, gm, gd float64) {
	u := vg - cutoff
	if u <= 0 {
		return
	}
	beta := 2 * c.IDSS / (cutoff * cutoff)
	mod := 1 + c.Lambda*vd
	if vd < u {
		base := beta * (u*vd - .5*vd*vd)
		return base * mod, beta * vd * mod, beta*(u-vd)*mod + base*c.Lambda
	}
	base := .5 * beta * u * u
	return base * mod, beta * u * mod, base * c.Lambda
}
func jfetChannel(p Component, vgs, vds float64) (i, gm, gd float64) {
	sign := jfetSign(p.Kind)
	c := *p.Device.JFET
	vg, vd := sign*vgs, sign*vds
	cutoff := sign * c.Cutoff
	if vd >= 0 {
		i, gm, gd = jfetPositive(c, cutoff, vg, vd)
	} else {
		a, b, d := jfetPositive(c, cutoff, vg-vd, -vd)
		i, gm, gd = -a, -b, b+d
	}
	i *= sign
	return
}

var jfetPins = []string{"gate", "drain", "source"}

func jfetEvaluation(p Component, v [3]float64) (i [3]float64, j [3][3]float64) {
	id, gm, gd := jfetChannel(p, v[0]-v[2], v[1]-v[2])
	i[1], i[2] = id, -id
	j[1] = [3]float64{gm, gd, -gm - gd}
	j[2] = [3]float64{-gm, -gd, gm + gd}
	sign := jfetSign(p.Kind)
	for _, other := range []int{1, 2} {
		drop := v[0] - v[other]
		a, g := siliconDiodeCurrent(sign * drop)
		a = sign*a + drop*1e-9
		g += 1e-9
		i[0] += a
		i[other] -= a
		j[0][0] += g
		j[0][other] -= g
		j[other][0] -= g
		j[other][other] += g
	}
	return
}
func (n *mixedNetwork) stampJFET(p Component, x []float64, add func(int, int, float64), f []float64) {
	v := [3]float64{}
	for k, pin := range jfetPins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	i, j := jfetEvaluation(p, v)
	for row, pin := range jfetPins {
		node := n.pin(p, pin)
		if node >= 0 {
			f[node] += i[row]
		}
		for col, other := range jfetPins {
			add(node, n.pin(p, other), j[row][col])
		}
	}
}
func (n *mixedNetwork) jfetResult(p Component, x []float64) JFETResult {
	v := [3]float64{}
	for k, pin := range jfetPins {
		v[k] = voltage(x, n.pin(p, pin))
	}
	i, _ := jfetEvaluation(p, v)
	r := JFETResult{ID: p.ID, Model: JFETModelID, Parameters: *p.Device.JFET, GateCurrent: i[0], DrainCurrent: i[1], SourceCurrent: -i[2], VGS: v[0] - v[2], VDS: v[1] - v[2], State: "saturation"}
	for k, current := range i {
		r.Power += (v[k] - v[2]) * current
	}
	vg, vd := jfetSign(p.Kind)*r.VGS, jfetSign(p.Kind)*r.VDS
	if vd < 0 {
		r.State = "reverse"
	} else if vg <= jfetSign(p.Kind)*r.Parameters.Cutoff {
		r.State = "cutoff"
	} else if vd < vg-jfetSign(p.Kind)*r.Parameters.Cutoff {
		r.State = "triode"
	}
	if jfetSign(p.Kind)*(v[0]-v[2]) > .3 || jfetSign(p.Kind)*(v[0]-v[1]) > .3 {
		r.State += " · gate forward biased"
	}
	return r
}
func validateRecordedJFETs(c Compiled, g Graph, r *MixedResult) error {
	for _, kind := range []string{"njfet", "pjfet"} {
		ids := []string{}
		for _, q := range r.JFETs {
			p, ok := g.Component(q.ID)
			if !ok || !IsJFETKind(p.Kind) {
				return fmt.Errorf("invalid JFET result identity")
			}
			if p.Kind == kind {
				ids = append(ids, q.ID)
			}
		}
		if err := recordedObjectIDs(g, kind, ids); err != nil {
			return err
		}
	}
	if len(r.JFETs) == 0 {
		return nil
	}
	n, x, err := recordedDeviceNodes(c, r)
	if err != nil {
		return err
	}
	for _, q := range r.JFETs {
		p, _ := g.Component(q.ID)
		e := n.jfetResult(p, x)
		if q.Model != e.Model || q.Parameters != e.Parameters || q.State != e.State {
			return fmt.Errorf("recorded JFET model/state mismatch")
		}
		for _, v := range [][2]float64{{q.GateCurrent, e.GateCurrent}, {q.DrainCurrent, e.DrainCurrent}, {q.SourceCurrent, e.SourceCurrent}, {q.VGS, e.VGS}, {q.VDS, e.VDS}, {q.Power, e.Power}} {
			if !recordingNear(v[0], v[1]) {
				return fmt.Errorf("recorded JFET electrical law mismatch")
			}
		}
	}
	return nil
}
func hasJFETs(g Graph) bool {
	for _, p := range g.Components {
		if IsJFETKind(p.Kind) {
			return true
		}
	}
	return false
}
func JFETSymbolPaths(kind string) [][]Point {
	paths := [][]Point{{{-40, 0}, {-8, 0}}, {{-8, -20}, {-8, 20}}, {{-8, -16}, {25, -16}, {25, -35}}, {{-8, 16}, {25, 16}, {25, 35}}}
	if kind == "pjfet" {
		return append(paths, []Point{{-20, -6}, {-28, 0}, {-20, 6}})
	}
	return append(paths, []Point{{-24, -6}, {-16, 0}, {-24, 6}})
}
