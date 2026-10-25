package workbench

import (
	"fmt"
	"math"
)

// SourceSamplingAdvice describes configured output-grid guidance, not internal
// solver accuracy or measured circuit timing. Constant waveforms need no hint.
type SourceSamplingAdvice struct {
	ID              string  `json:"id"`
	SourceKind      string  `json:"source_kind"`
	TargetStep      float64 `json:"target_step_s"`
	RecommendedStep float64 `json:"recommended_step_s"`
	MinimumStep     float64 `json:"minimum_step_s"`
	Unresolved      bool    `json:"unresolved"`
	MinimumLimited  bool    `json:"minimum_limited"`
	CanFit          bool    `json:"can_fit"`
}

func waveformTargetStep(p Component) float64 {
	p = withDeviceWaveform(p)
	target := math.Inf(1)
	if p.Sine != nil && p.Sine.Amplitude > 0 {
		target = 1 / (32 * p.Sine.Frequency)
	}
	if p.Pulse != nil && p.Pulse.HighVoltage != p.Pulse.LowVoltage {
		target = math.Min(p.Pulse.RiseSeconds, p.Pulse.FallSeconds) / 4
	}
	return target
}

func sourceSamplingAdvice(g Graph, cfg TransientSettings) []SourceSamplingAdvice {
	out := []SourceSamplingAdvice{}
	for _, p := range g.Components {
		target := waveformTargetStep(p)
		if !finite(target) || target <= 0 {
			continue
		}
		recommended := math.Min(cfg.StepSeconds, math.Max(minimumTransientSampleStep, target))
		out = append(out, SourceSamplingAdvice{ID: p.ID, SourceKind: p.Kind, TargetStep: target, RecommendedStep: recommended, MinimumStep: minimumTransientSampleStep, Unresolved: cfg.StepSeconds > target, MinimumLimited: target < minimumTransientSampleStep, CanFit: recommended < cfg.StepSeconds})
	}
	return out
}

func sourceSamplingWarning(advice SourceSamplingAdvice, step float64) Diagnostic {
	message := fmt.Sprintf("%s: output sample step %.6g s may hide waveform detail; use %.6g s or less.", advice.ID, step, advice.TargetStep)
	if advice.MinimumLimited {
		message = fmt.Sprintf("%s: four-samples-per-ramp target %.6g s is below the supported %.6g s minimum. Fitting cannot meet this target; use longer rise/fall times for a slower experiment.", advice.ID, advice.TargetStep, advice.MinimumStep)
	}
	return Diagnostic{"source-sampling", message, advice.ID}
}

func waveformSamplingWarnings(g Graph, cfg TransientSettings) []Diagnostic {
	out := []Diagnostic{}
	for _, a := range sourceSamplingAdvice(g, cfg) {
		if a.Unresolved {
			out = append(out, sourceSamplingWarning(a, cfg.StepSeconds))
		}
	}
	return out
}

func sourceSamplingHelp(g Graph, c Compiled, advice SourceSamplingAdvice, step float64) DiagnosticHelp {
	help := makeDiagnosticHelp(g, c, sourceSamplingWarning(advice, step), "sampling", nil)
	if advice.MinimumLimited {
		help.Summary = "The scope cannot meet this source's sampling target."
		help.Action = "The 1 µs minimum cannot meet the four-samples-per-ramp target. Use longer rise/fall times; fitting can only reach the supported limit."
	}
	return help
}
