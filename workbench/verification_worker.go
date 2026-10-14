package workbench

import (
	"context"
	"errors"
	"math"
	"time"
)

func (s *Service) runVerification(operation *verificationOperation, input verificationInput) {
	defer operation.cancel()
	defer close(operation.done)
	s.mu.Lock()
	if s.closed || s.operations[operation.summary.ID] != operation || operation.ctx.Err() != nil {
		s.finishVerificationLocked(operation, nil, operation.ctx.Err())
		s.mu.Unlock()
		return
	}
	operation.summary.Status = "running"
	s.revision++
	s.signalLocked("jobs", "recordings")
	s.mu.Unlock()
	record, err := computeVerification(operation.ctx, input, func(step int, at float64) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.operations[operation.summary.ID] != operation || operation.summary.Status == "canceling" {
			return
		}
		operation.summary.CompletedSteps = step
		operation.summary.SimulatedTime = at
		operation.summary.Progress = float64(step) / float64(input.steps)
		s.revision++
		s.signalLocked("jobs", "recordings")
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if operation.ctx.Err() != nil {
		err = operation.ctx.Err()
	}
	s.finishVerificationLocked(operation, record, err)
}
func (s *Service) finishVerificationLocked(operation *verificationOperation, record *verificationRecord, err error) {
	if s.closed || s.operations[operation.summary.ID] != operation {
		return
	}
	switch {
	case errors.Is(err, context.Canceled):
		operation.summary.Status = "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		operation.summary.Status = "failed"
		operation.summary.Error = &Failure{Code: "timeout", Message: "Verification exceeded its 30 second wall-clock budget"}
	case err != nil:
		operation.summary.Status = "failed"
		operation.summary.Error = &Failure{Code: "solve-failed", Message: err.Error()}
	default:
		operation.record = record
		operation.keep = true
		operation.summary.Status = "completed"
		operation.summary.Progress = 1
		operation.summary.CompletedSteps = operation.summary.TotalSteps
		operation.summary.SimulatedTime = inputDuration(record)
	}
	s.revision++
	s.signalLocked("jobs", "recordings")
}
func inputDuration(record *verificationRecord) float64 {
	if record == nil {
		return 0
	}
	return record.metadata.Duration
}
func computeVerification(ctx context.Context, input verificationInput, progress func(int, float64)) (*verificationRecord, error) {
	kernel, err := newSimulationKernel(ctx, kernelPlan{compiled: input.compiled, settings: input.settings, mode: input.mode, run: input.serial, revision: input.metadata.ElectricalRevision})
	if err != nil {
		return nil, err
	}
	row, err := verificationRow(kernel.trace.Sample, input.probes)
	if err != nil {
		return nil, err
	}
	rows := []VerificationRow{row}
	lastProgress := time.Now()
	if IsTransientAnalysis(input.compiled.Analysis) {
		for step := 1; step <= input.steps; step++ {
			previous := kernel.trace.Sample.Time
			end := math.Min(input.duration, float64(step)*input.settings.StepSeconds)
			if err := kernel.advance(ctx, end); err != nil {
				return nil, err
			}
			// A step spans at most one pulse edge because high/low >= sample step.
			// Preserve both edge sides and duplicate timestamps, not interpolated values.
			for _, sample := range kernel.trace.Samples {
				if sample.Time > previous {
					row, err := verificationRow(sample, input.probes)
					if err != nil {
						return nil, err
					}
					rows = append(rows, row)
				}
			}
			if len(rows) > MaxVerificationFrames {
				return nil, errors.New("verification exceeded its recorded frame bound")
			}
			if time.Since(lastProgress) >= 100*time.Millisecond || step == input.steps {
				progress(step, kernel.trace.Sample.Time)
				lastProgress = time.Now()
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	metadata := input.metadata
	metadata.Engine = kernel.trace.Engine
	metadata.SampleCount = len(rows)
	metadata.LED = kernel.led
	metadata.NPN = kernel.npn
	if kernel.mixed != nil {
		metadata.Mixed = kernel.mixed.result
	}
	for i, probe := range input.probes {
		points := make([]ScopePoint, len(rows))
		for j, row := range rows {
			points[j] = ScopePoint{Time: row.Time, Value: row.Values[i]}
		}
		metadata.Statistics = append(metadata.Statistics, scopeStatistics(points, probe.ID))
	}
	return &verificationRecord{metadata: metadata, rows: rows}, nil
}
