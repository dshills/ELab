package workbench

import (
	"context"
	"fmt"
	"math"
	"time"
)

type StepWork struct {
	ID                 string   `json:"id"`
	Status             string   `json:"status"`
	Actor              string   `json:"actor"`
	RunID              uint64   `json:"run_id"`
	ElectricalRevision uint64   `json:"electrical_revision"`
	ActuationRevision  uint64   `json:"actuation_revision"`
	CommandRevision    uint64   `json:"command_revision"`
	StartTime          float64  `json:"start_time_s"`
	Duration           float64  `json:"duration_s"`
	CompletedSamples   int      `json:"completed_samples"`
	TotalSamples       int      `json:"total_samples"`
	Progress           float64  `json:"progress"`
	Error              *Failure `json:"error,omitempty"`
}

func cloneStepWork(work *StepWork) *StepWork {
	if work == nil {
		return nil
	}
	out := *work
	if work.Error != nil {
		err := *work.Error
		out.Error = &err
	}
	return &out
}

type stepJob struct {
	work                         StepWork
	ctx                          context.Context
	cancel                       context.CancelFunc
	done                         chan struct{}
	input                        *Service
	compiled                     Compiled
	originalTrace                *Transient
	originalState                *mixedState
	actuation                    uint64
	mode, pending                string
	end                          float64
	reason, legacyKey, requestID string
	final                        Response
}

func cloneMixedState(state *mixedState) *mixedState {
	if state == nil {
		return nil
	}
	return &mixedState{net: state.net, x: append([]float64{}, state.x...), capV: append([]float64{}, state.capV...), result: cloneMixed(state.result), inductorStep: state.inductorStep, inductorCeiling: state.inductorCeiling}
}

func manualStepTargetTime(start, end, step float64, sample int) float64 {
	next := math.Min(end, start+float64(sample)*step)
	if end-next < 1e-12 {
		return end
	}
	return next
}

// Count the worker's actual absolute-time grid, including its existing tiny
// final-tail snapping. Admission validates the interval and 1000-sample bound.
// A quotient ceiling can overestimate the count at an adjacent endpoint or
// after a nonzero origin, leaving a completed job reporting fewer than its total.
func manualStepTargetCount(start, end, step float64) int {
	for sample := 1; sample < MaxManualStepSamples; sample++ {
		if manualStepTargetTime(start, end, step, sample) == end {
			return sample
		}
	}
	return MaxManualStepSamples
}

func (s *Service) startStepLocked(p StepParams) error {
	if s.activeInputWorkLocked() {
		return &actionError{"busy", "finish or cancel applying the current input before stepping"}
	}
	if s.stepJob != nil && operationActive(s.stepJob.work.Status) {
		return &actionError{"busy", "finish or cancel the current manual step"}
	}
	c, _, d := s.graph.Compile()
	if !s.paused || d != nil || !IsTransientAnalysis(c.Analysis) || s.transient == nil || s.transient.Stale {
		return &actionError{"unavailable", "pause a valid transient run before stepping"}
	}
	if !finite(p.Duration) || p.Duration <= 0 || p.Duration > MaxManualStepSeconds || p.Duration > float64(MaxManualStepSamples)*s.transientSettings.StepSeconds {
		return fmt.Errorf("step duration must be positive, at most %g s and %d sample steps", MaxManualStepSeconds, MaxManualStepSamples)
	}
	end := s.transient.Sample.Time + p.Duration
	if end <= s.transient.Sample.Time {
		return fmt.Errorf("duration is too small to advance the current floating-point time")
	}
	if IsNetworkAnalysis(c.Analysis) && s.mixedState == nil {
		return &actionError{"unavailable", "wait for a fresh network result before stepping"}
	}
	if IsNetworkAnalysis(c.Analysis) && len(s.mixedState.net.caps) > 0 && p.Duration < MinCapacitorStepSeconds {
		return &actionError{"solve-failed", "mixed timestep too small; manual step must advance at least 1e-10 s"}
	}
	// This isolated state holder only runs time/input helpers under one worker.
	// It is never dispatched, started or closed, and owns all mutable solver data.
	input := &Service{graph: s.graph.Clone(), transient: cloneTransient(s.transient), transientSettings: s.transientSettings,
		mixedState: cloneMixedState(s.mixedState), mixed: cloneMixed(s.mixed), npn: cloneNPN(s.npn), npnHigh: cloneNPN(s.npnHigh), npnLow: cloneNPN(s.npnLow),
		inputMode: s.inputMode, pendingInputMode: s.pendingInputMode, networkAdvance: s.networkAdvance}
	ctx, cancel := context.WithTimeout(context.Background(), ManualStepDeadline)
	s.stepSerial++
	job := &stepJob{work: StepWork{ID: fmt.Sprintf("step-%d", s.stepSerial), Status: "queued", RunID: s.run, ElectricalRevision: s.electrical, ActuationRevision: s.actuationRevision, CommandRevision: s.commandRevision + 1, StartTime: s.transient.Sample.Time, Duration: p.Duration, TotalSamples: manualStepTargetCount(s.transient.Sample.Time, end, s.transientSettings.StepSeconds)},
		ctx: ctx, cancel: cancel, done: make(chan struct{}), input: input, compiled: c, originalTrace: s.transient, originalState: s.mixedState, actuation: s.actuationRevision,
		mode: s.transient.Mode, pending: s.transient.PendingMode, end: end}
	s.stepJob = job
	s.liveWorkers.Add(1)
	go s.runStep(job)
	return nil
}
func (s *Service) cancelStepLocked(reason string) {
	if s.stepJob != nil && operationActive(s.stepJob.work.Status) {
		if s.stepJob.reason == "" {
			s.stepJob.reason = reason
		}
		s.stepJob.work.Status = "canceling"
		s.stepJob.cancel()
	}
}
func (s *Service) cancelStepCommandLocked(p IDParams) error {
	if s.inputJob != nil && s.inputJob.work.ID == p.ID && s.activeInputWorkLocked() {
		s.cancelInputWorkLocked("canceled")
		return nil
	}
	if s.stepJob == nil || s.stepJob.work.ID != p.ID || !operationActive(s.stepJob.work.Status) {
		return &actionError{"operation-unavailable", "choose the active manual step or input work to cancel"}
	}
	s.cancelStepLocked("canceled")
	return nil
}
func (s *Service) stepCurrentLocked(job *stepJob) bool {
	return s.stepJob == job && !s.closed && s.paused && s.run == job.work.RunID && s.actuationRevision == job.actuation && s.transient == job.originalTrace && s.mixedState == job.originalState && !s.transient.Stale && s.transient.Mode == job.mode && s.transient.PendingMode == job.pending
}
func (s *Service) runStep(job *stepJob) {
	defer s.liveWorkers.Done()
	s.mu.Lock()
	if job.ctx.Err() == nil {
		job.work.Status = "running"
		s.revision++
		s.signalLocked("simulation", "jobs", "diagnostics")
	}
	s.mu.Unlock()
	input := job.input
	var err error
	completed := 0
	lastProgress := time.Now()
	if job.ctx.Err() == nil && input.transient.PendingMode != "" {
		err = input.applyTransientInputLocked(job.compiled, input.transient.PendingMode)
	}
	for err == nil && input.transient.Sample.Time < job.end {
		if err = job.ctx.Err(); err != nil {
			break
		}
		// Anchor every requested output to the captured origin. Repeated
		// addition can drift far enough to require a 1001st final sample.
		next := manualStepTargetTime(job.work.StartTime, job.end, input.transientSettings.StepSeconds, completed+1)
		err = input.advanceTransientLocked(job.compiled, next)
		if err != nil {
			break
		}
		completed++
		if completed == 1 || time.Since(lastProgress) >= 50*time.Millisecond {
			s.mu.Lock()
			if job.ctx.Err() == nil && s.stepCurrentLocked(job) {
				job.work.CompletedSamples = completed
				job.work.Progress = math.Min(1, (input.transient.Sample.Time-job.work.StartTime)/job.work.Duration)
				s.revision++
				s.signalLocked("simulation", "jobs", "diagnostics")
			}
			s.mu.Unlock()
			lastProgress = time.Now()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stepCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Manual step discarded; committed time and charge were preserved"}
	} else if err != nil || job.ctx.Err() != nil {
		if err == nil {
			err = job.ctx.Err()
		}
		job.work.Status = "failed"
		job.work.Error = &Failure{Code: simulationFailureCode(err, "solve-failed"), Message: err.Error()}
	} else {
		s.transient, s.mixedState, s.mixed, s.npn = input.transient, input.mixedState, input.mixed, input.npn
		s.inputMode, s.pendingInputMode = input.inputMode, input.pendingInputMode
		job.work.Status = "completed"
		job.work.Progress = 1
	}
	job.work.CompletedSamples = completed
	if job.work.Status != "completed" {
		job.work.Progress = math.Min(1, (input.transient.Sample.Time-job.work.StartTime)/job.work.Duration)
	}
	job.cancel()
	s.revision++
	s.signalLocked("simulation", "jobs", "diagnostics")
	job.final = Response{Version: ProtocolVersion, ID: job.requestID, Revision: s.revision, Result: s.snapshotLocked()}
	if job.work.Error != nil {
		failure := *job.work.Error
		job.final.Error = &failure
	}
	// The waiting compatibility command has not delivered its outcome yet.
	// Finalize that response once; immediate-start retries keep their admission.
	if old, ok := s.cache[job.legacyKey]; ok && job.legacyKey != "" {
		old.response = cloneMutationResponse(job.final)
		s.cache[job.legacyKey] = old
	}
	close(job.done)
}
func (s *Service) awaitStep(ctx context.Context, r Request, actor string, accepted Response) Response {
	if accepted.Error != nil {
		return accepted
	}
	s.mu.Lock()
	key := actor + "\x00" + r.Key
	if cached, ok := s.cache[key]; ok {
		state := cached.response.Result.(Snapshot)
		if state.StepWork != nil && !operationActive(state.StepWork.Status) {
			out := cloneMutationResponse(cached.response)
			out.ID = r.ID
			out.Duplicate = accepted.Duplicate
			s.mu.Unlock()
			return out
		}
	}
	job := s.stepJob
	if job == nil || job.work.ID != accepted.Result.(Snapshot).StepWork.ID {
		s.mu.Unlock()
		return s.failure(r, "operation-unavailable", "original step has expired; inspect current state")
	}
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		return s.failure(r, "canceled", "Request wait ended; admitted step continues. Inspect state or retry the original key")
	case <-job.done:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := cloneMutationResponse(job.final)
	out.ID = r.ID
	out.Duplicate = accepted.Duplicate
	return out
}

func (s *Service) stepWorkLocked() *StepWork {
	if s.stepJob == nil {
		return nil
	}
	return cloneStepWork(&s.stepJob.work)
}
