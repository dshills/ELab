package workbench

import (
	"context"
	"fmt"
	"time"
)

// ExperimentCapture contains exact accepted samples for one probe on an
// isolated copy of a circuit. It does not create a retained recording.
type ExperimentCapture struct {
	Probe Probe             `json:"probe"`
	Rows  []VerificationRow `json:"rows"`
}

func CaptureExperiment(ctx context.Context, graph Graph, settings TransientSettings, probeID string, duration float64) (ExperimentCapture, error) {
	compiled, _, diagnostic := graph.Compile()
	if diagnostic != nil {
		return ExperimentCapture{}, fmt.Errorf("%s", diagnostic.Message)
	}
	if !finite(duration) || duration < 0 || duration > 10 {
		return ExperimentCapture{}, fmt.Errorf("choose a duration between 0 and 10 seconds")
	}
	steps := 1
	if IsTransientAnalysis(compiled.Analysis) {
		if duration == 0 {
			return ExperimentCapture{}, fmt.Errorf("choose a positive duration for this circuit")
		}
		if err := settings.validateForGraph(graph, false); err != nil {
			return ExperimentCapture{}, err
		}
		steps = verificationOutputSteps(duration, settings.StepSeconds)
		if steps > MaxVerificationSteps {
			return ExperimentCapture{}, fmt.Errorf("shorten the duration or increase the sample interval to stay within %d steps", MaxVerificationSteps)
		}
	} else {
		duration = 0
	}
	var probe Probe
	for _, candidate := range probeInventory(graph) {
		if candidate.ID == probeID {
			probe = candidate
			break
		}
	}
	if probe.ID == "" {
		return ExperimentCapture{}, fmt.Errorf("choose a measurement available on this circuit")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	record, err := computeVerification(ctx, verificationInput{
		compiled: compiled, settings: settings, mode: "auto", duration: duration, steps: steps,
		probes: []Probe{probe}, metadata: VerificationMetadata{Probes: []Probe{probe}},
	}, func(int, float64) {})
	if err != nil {
		return ExperimentCapture{}, err
	}
	return ExperimentCapture{Probe: probe, Rows: record.rows}, nil
}
