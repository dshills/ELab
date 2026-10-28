package workbench

import (
	"context"
	"encoding/json"
	"fmt"
)

type ShutdownWork struct {
	ID                 string   `json:"id"`
	Status             string   `json:"status"`
	Phase              string   `json:"phase"`
	Actor              string   `json:"actor"`
	InputRevision      uint64   `json:"input_revision"`
	CommandRevision    uint64   `json:"command_revision"`
	DocumentGeneration uint64   `json:"document_generation"`
	RecoveryDirectory  string   `json:"recovery_directory,omitempty"`
	DiscardRecovery    bool     `json:"discard_recovery"`
	CheckpointRecovery bool     `json:"checkpoint_recovery"`
	Cancelable         bool     `json:"cancelable"`
	RecoveryFinalized  bool     `json:"recovery_finalized"`
	WorkersJoined      bool     `json:"workers_joined"`
	ExitReady          bool     `json:"exit_ready"`
	RecoveryError      string   `json:"recovery_error,omitempty"`
	AppearanceError    string   `json:"appearance_error,omitempty"`
	Error              *Failure `json:"error,omitempty"`
}

type shutdownJob struct {
	work ShutdownWork
	done chan struct{}
}

func cloneShutdown(in *ShutdownWork) *ShutdownWork {
	if in == nil {
		return nil
	}
	out := *in
	if in.Error != nil {
		e := *in.Error
		out.Error = &e
	}
	return &out
}

// ReadyToExit reads lifecycle readiness without constructing a full snapshot.
func (s *Service) ReadyToExit() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed && s.shutdown != nil && s.shutdown.work.ExitReady
}

// Closed freezes semantic mutations immediately. ExitReady additionally means
// background finalization/joining completed, or its failure was acknowledged.
func (state Snapshot) ReadyToExit() bool {
	return state.Closed && state.Shutdown != nil && state.Shutdown.ExitReady
}

func (s *Service) shutdownViewLocked() *ShutdownWork {
	if s.shutdown == nil {
		return nil
	}
	return cloneShutdown(&s.shutdown.work)
}

func (s *Service) runShutdown(job *shutdownJob, store *recoveryStore, capture recoveryCapture) {
	defer close(job.done)
	s.mu.Lock()
	job.work.Status, job.work.Phase = "running", "finalizing-recovery"
	s.revision++
	s.signalLocked("lifecycle", "jobs")
	s.mu.Unlock()
	var err error
	if store != nil {
		err = store.finish(capture, job.work.DiscardRecovery, job.work.CheckpointRecovery)
	}
	s.mu.Lock()
	job.work.RecoveryFinalized = err == nil
	if err != nil {
		job.work.RecoveryError = err.Error()
		s.recovery.Error = err.Error()
	}
	job.work.Phase = "joining-workers"
	s.revision++
	s.signalLocked("lifecycle", "jobs", "recovery")
	s.mu.Unlock()
	// All admissions and worker Adds are guarded by the editor mutex and refuse
	// a closed service. The shutdown worker itself is intentionally not in it.
	s.liveWorkers.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	job.work.WorkersJoined = true
	if s.appearanceStore != nil && s.appearance.Status == "error" {
		job.work.AppearanceError = s.appearance.Error
	}
	job.work.Status, job.work.Phase = "completed", "completed"
	job.work.ExitReady = true
	if job.work.RecoveryError != "" || job.work.AppearanceError != "" {
		job.work.Status, job.work.Phase = "failed", "failed"
		job.work.ExitReady = false
		job.work.Error = &Failure{Code: "shutdown-failed", Message: "Shutdown cleanup or preference persistence failed; inspect recovery_error/appearance_error before acknowledging exit"}
	}
	s.revision++
	s.signalLocked("lifecycle", "jobs", "recovery", "preferences")
}

func (s *Service) shutdownQueryLocked(raw json.RawMessage) (*ShutdownWork, error) {
	var p IDParams
	if err := params(raw, &p, "id"); err != nil {
		return nil, err
	}
	if s.shutdown == nil || s.shutdown.work.ID != p.ID {
		return nil, &actionError{"not-found", "Unknown shutdown job"}
	}
	return cloneShutdown(&s.shutdown.work), nil
}

func (s *Service) acknowledgeShutdownLocked(raw json.RawMessage) error {
	var p IDParams
	if err := params(raw, &p, "id"); err != nil {
		return err
	}
	if s.shutdown == nil || s.shutdown.work.ID != p.ID {
		return &actionError{"not-found", "Unknown shutdown job"}
	}
	if s.shutdown.work.Status != "failed" {
		return &actionError{"unavailable", "Only a failed terminal shutdown needs acknowledgement"}
	}
	s.shutdown.work.ExitReady = true
	return nil
}

func (s *Service) awaitShutdown(ctx context.Context, out Response) Response {
	if out.Error != nil {
		return out
	}
	s.mu.Lock()
	job := s.shutdown
	s.mu.Unlock()
	if job == nil {
		return out
	}
	select {
	case <-job.done:
		s.mu.Lock()
		out.Revision, out.Result = s.revision, s.snapshotLocked()
		s.mu.Unlock()
	case <-ctx.Done():
		out.Error = &Failure{Code: "canceled", Message: fmt.Sprintf("Close was admitted as %s and continues; inspect the shutdown job: %v", job.work.ID, ctx.Err())}
	}
	return out
}
