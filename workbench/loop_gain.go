package workbench

import (
	"fmt"
	"math"
	"reflect"
)

const LoopGainEngineID = "elab.controlled-source-return-ratio.v1"

func IsFrequencySweep(parameter string) bool {
	return parameter == "frequency" || parameter == "loop-frequency"
}
func loopProbe(id string) Probe {
	return Probe{ID: "loop:" + id, Label: id + " differential return ratio", Unit: "ratio", Positive: id + ".noninverting", Reference: id + ".inverting"}
}
func loopAssumptions() []string {
	return []string{
		"Closed-circuit DC bias is solved before opening only the selected dynamic op-amp differential-to-compensation controlled-current increment; input/output loading, compensation storage, rails and all other paths remain in place",
		"At fixed bias Aclosed=Aopen+u*v^T, where v^T selects V(noninverting)-V(inverting). Inject one equivalent volt into this control path: T=v^T*Aopen^-1*u. The returned differential voltage is -T and the characteristic factor is 1+T; this is not closed-loop signal Vout/Vin",
		"All independent voltage sources are AC grounded; configured current devices keep their bias Jacobians. Only dynamic op-amps expose this injection target; unsupported/event-dependent/singular breaks refuse explicitly",
		"Margins are all sampled/interpolated unity and odd-180-degree crossings in an ascending positive band. Missing crossings are unavailable, not infinite. Phase margin is signed to (-180,180]. Coarse grids can miss crossings; refine/expand the band",
		"Other loops remain closed. Open-loop right-half-plane poles, multiple-loop Nyquist winding, nonlinear slew/clipping, uncertainty and hardware stability are not certified. Margins alone are not a stability verdict",
	}
}

type LoopCrossing struct {
	Frequency float64 `json:"frequency_hz"`
	Margin    float64 `json:"margin"`
}
type LoopMargins struct {
	Method      string         `json:"method"`
	Injection   string         `json:"injection_component"`
	Unity       []LoopCrossing `json:"unity_crossings_phase_margin_deg"`
	Phase       []LoopCrossing `json:"phase_crossings_gain_margin_db"`
	UnityReason string         `json:"unity_reason"`
	PhaseReason string         `json:"phase_reason"`
}

func cloneLoopMargins(in *LoopMargins) *LoopMargins {
	if in == nil {
		return nil
	}
	out := *in
	out.Unity = append([]LoopCrossing{}, in.Unity...)
	out.Phase = append([]LoopCrossing{}, in.Phase...)
	return &out
}
func loopMargins(r *SweepResult) (*LoopMargins, error) {
	if r == nil || len(r.Points) < 2 || len(r.Points) > MaxDCSweepPoints {
		return nil, fmt.Errorf("loop margins require 2–201 points")
	}
	for k, p := range r.Points {
		if len(p.Values) != 2 || len(p.AC) != 1 || !finite(p.Input) || p.Input <= 0 || !finite(p.Values[0]) || !finite(p.Values[1]) || math.Abs(p.Values[1]) > 360*(MaxDCSweepPoints+1) {
			return nil, fmt.Errorf("invalid loop margin point")
		}
		if k > 0 {
			a := r.Points[k-1]
			if p.Input <= a.Input || p.AC[0].PhaseDefined && a.AC[0].PhaseDefined && math.Abs(p.Values[1]-a.Values[1]) > 180+1e-9 {
				return nil, fmt.Errorf("loop phase/grid continuity is unresolved")
			}
		}
	}

	out := &LoopMargins{Method: LoopGainEngineID, Injection: r.Target.ComponentID, Unity: []LoopCrossing{}, Phase: []LoopCrossing{}}
	add := func(dst *[]LoopCrossing, c LoopCrossing) {
		if len(*dst) == 0 || (*dst)[len(*dst)-1].Frequency != c.Frequency {
			*dst = append(*dst, c)
		}
	}
	for k := 1; k < len(r.Points); k++ {
		a, b := r.Points[k-1], r.Points[k]
		if len(a.AC) == 0 || len(b.AC) == 0 || !a.AC[0].PhaseDefined || !b.AC[0].PhaseDefined {
			continue
		}
		interpolate := func(t float64) float64 {
			if t == 0 {
				return a.Input
			}
			if t == 1 {
				return b.Input
			}
			return a.Input * math.Exp(math.Log(b.Input/a.Input)*t)
		}
		if a.Values[0] != b.Values[0] && (a.Values[0] == 0 || b.Values[0] == 0 || a.Values[0]*b.Values[0] < 0) {
			t := -a.Values[0] / (b.Values[0] - a.Values[0])
			phase := a.Values[1] + t*(b.Values[1]-a.Values[1])
			pm := math.Mod(180+phase, 360)
			if pm <= -180 {
				pm += 360
			}
			if pm > 180 {
				pm -= 360
			}
			add(&out.Unity, LoopCrossing{interpolate(t), pm})
		}
		lo, hi := math.Min(a.Values[1], b.Values[1]), math.Max(a.Values[1], b.Values[1])
		if lo == hi {
			continue
		}
		for turn := math.Ceil((lo + 180) / 360); -180+turn*360 <= hi; turn++ {
			level := -180 + turn*360
			t := (level - a.Values[1]) / (b.Values[1] - a.Values[1])
			db := a.Values[0] + t*(b.Values[0]-a.Values[0])
			add(&out.Phase, LoopCrossing{interpolate(t), -db})
		}
	}
	if len(out.Unity) == 0 {
		out.UnityReason = "No resolved unity-gain crossing in the sampled band"
	}
	if len(out.Phase) == 0 {
		out.PhaseReason = "No resolved odd-180-degree phase crossing in the sampled band"
	}
	return out, nil
}
func (n *mixedNetwork) openLoopControl(id string, x []float64, jac [][]float64) ([]float64, error) {
	p, ok := n.authored.Component(id)
	if !ok || !hasDynamics(p) {
		return nil, fmt.Errorf("loop injection requires an op-amp with configured dynamics")
	}
	v := n.opAmpVoltages(p, x)
	pole, ok := n.index[poleNode(id)]
	if !ok || pole < 0 {
		return nil, fmt.Errorf("loop compensation node missing")
	}
	if n.pin(p, "noninverting") == n.pin(p, "inverting") {
		return nil, fmt.Errorf("selected op-amp inputs share a net; no differential return path")
	}
	_, d, rail := poleCurrent(p, v, voltage(x, pole))
	if !finite(d[0]) || d[0] <= 0 {
		return nil, fmt.Errorf("selected differential control is disabled or locally saturated")
	}
	u := make([]float64, len(jac))
	u[pole] -= d[0]
	if r := n.pin(p, []string{"noninverting", "inverting", "out", "vplus", "vminus"}[rail]); r >= 0 {
		u[r] += d[0]
	}
	for row, value := range u {
		if col := n.pin(p, "noninverting"); col >= 0 {
			jac[row][col] -= value
		}
		if col := n.pin(p, "inverting"); col >= 0 {
			jac[row][col] += value
		}
	}
	return u, nil
}
func validateLoopMargins(plan sweepPlan, r *SweepResult) error {
	if plan.settings.Parameter != "loop-frequency" {
		if r.Loop != nil {
			return fmt.Errorf("non-loop sweep contains loop margins")
		}
		return nil
	}
	expected, err := loopMargins(r)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(r.Loop, expected) {
		return fmt.Errorf("loop margin estimates disagree with captured return-ratio points")
	}
	return nil
}
