package workbench

import (
	"fmt"
	"math"
)

const minimumTransientSampleStep = 1e-6
const lcSamplesPerReferencePeriod = 32

// LCSamplingAdvice is a value-based display hint, not a circuit-pole estimate.
// It is derived session data; only the actual transient settings are saved.
type LCSamplingAdvice struct {
	InductorID      string  `json:"inductor_id"`
	CapacitorID     string  `json:"capacitor_id"`
	ReferencePeriod float64 `json:"reference_period_s"`
	TargetStep      float64 `json:"target_step_s"`
	RecommendedStep float64 `json:"recommended_step_s"`
	MinimumStep     float64 `json:"minimum_step_s"`
	Unresolved      bool    `json:"unresolved"`
	MinimumLimited  bool    `json:"minimum_limited"`
	CanFit          bool    `json:"can_fit"`
	Note            string  `json:"note"`
}

func samplingRecommendation(g Graph, cfg TransientSettings) (float64, *LCSamplingAdvice) {
	step := cfg.StepSeconds
	var coil, cap Component
	for _, part := range g.Components {
		step = math.Min(step, waveformRecommendedStep(part))
		if !finite(part.Value) || part.Value <= 0 {
			continue
		}
		if part.Kind == "inductor" && (coil.ID == "" || part.Value < coil.Value) {
			coil = part
		}
		if part.Kind == "capacitor" && (cap.ID == "" || part.Value < cap.Value) {
			cap = part
		}
	}
	if coil.ID == "" || cap.ID == "" {
		return step, nil
	}
	period := 2 * math.Pi * math.Sqrt(coil.Value*cap.Value)
	if !finite(period) || period <= 0 || !finite(step) || step < minimumTransientSampleStep || step > 1 {
		return step, nil
	}
	target := period / lcSamplesPerReferencePeriod
	step = math.Min(step, math.Max(minimumTransientSampleStep, target))
	note := "Based on the smallest L/C values, not a measured period or circuit-pole estimate. Damping, connections and active devices can change timing."
	if target < minimumTransientSampleStep {
		note += " The supported minimum cannot resolve this reference scale."
	} else {
		note += " Compare finer sample steps."
	}
	return step, &LCSamplingAdvice{
		InductorID: coil.ID, CapacitorID: cap.ID, ReferencePeriod: period,
		TargetStep: target, RecommendedStep: step, MinimumStep: minimumTransientSampleStep,
		Unresolved: cfg.StepSeconds > target, MinimumLimited: target < minimumTransientSampleStep,
		CanFit: step < cfg.StepSeconds,
		Note:   note,
	}
}

func lcSamplingAdvice(g Graph, cfg TransientSettings) *LCSamplingAdvice {
	_, advice := samplingRecommendation(g, cfg)
	return advice
}

func cloneLCSamplingAdvice(in *LCSamplingAdvice) *LCSamplingAdvice {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func (s *Service) fitSamplingLocked() error {
	next := s.transientSettings
	next.StepSeconds, _ = samplingRecommendation(s.graph, next)
	next.PlaybackSpeed = math.Min(next.PlaybackSpeed, MaxLiveSamplesPerWallSecond*next.StepSeconds)
	if err := next.validateForGraph(s.graph, true); err != nil {
		return err
	}
	s.transientSettings = next
	return nil
}

func lcSamplingHelp(g Graph, c Compiled, advice *LCSamplingAdvice) *DiagnosticHelp {
	if advice == nil || !advice.Unresolved {
		return nil
	}
	detail := fmt.Sprintf("%s / %s: LC reference period %.6g s; target %.6g s per sample (32 points per reference period). %s", advice.InductorID, advice.CapacitorID, advice.ReferencePeriod, advice.TargetStep, advice.Note)
	d := Diagnostic{Code: "lc-sampling", Message: detail, ObjectID: advice.InductorID}
	help := makeDiagnosticHelp(g, c, d, "sampling", []string{advice.CapacitorID})
	help.Severity = "warning"
	help.Summary = "Live output sampling may hide this LC reference scale."
	help.Action = "Use Fit sampling or reduce the output sample interval; compare finer steps for your circuit."
	if advice.MinimumLimited {
		help.Summary = "This LC timing scale is faster than the scope's sampling limit."
		help.Action = "The 1 µs minimum cannot resolve this reference scale. Fitting can only reach that limit; consider a slower experiment with larger L/C values."
	}
	return &help
}
