package workbench

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type recoveryConfigurationCapture struct {
	params          RecoveryConfigureParams
	previous        *recoveryStore
	catalogRevision uint64
}
type recoveryConfigurationResult struct {
	store      *recoveryStore
	candidates []RecoveryCandidate
	problems   []RecoveryProblem
}
type recoveryConfigurationPrepareFunc func(context.Context, recoveryConfigurationCapture, func(string, int)) (recoveryConfigurationResult, error)

func prepareRecoveryConfiguration(ctx context.Context, capture recoveryConfigurationCapture, progress func(string, int)) (recoveryConfigurationResult, error) {
	result := recoveryConfigurationResult{}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	progress("opening-recovery-directory", 0)
	var err error
	if capture.previous != nil && capture.previous.directory == capture.params.Directory {
		result.store = capture.previous
	} else {
		result.store, err = newRecoveryStore(capture.params.Directory)
	}
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	progress("scanning-recovery", 0)
	result.candidates, result.problems, err = result.store.scan()
	if err != nil {
		// Preserve the legacy configure contract for an overfull catalog; OS
		// failures already carry their explicit file-error classification.
		var action *actionError
		if !errors.As(err, &action) {
			err = &actionError{"invalid-params", err.Error()}
		}
		return result, err
	}
	return result, ctx.Err()
}
func (s *Service) startRecoveryConfigurationLocked(p RecoveryConfigureParams) error {
	if !finite(p.Interval) || p.Interval < 1 || p.Interval > 60 {
		return fmt.Errorf("autosave interval must be 1–60 seconds")
	}
	directory, err := documentPath(p.Directory)
	if err != nil {
		return err
	}
	p.Directory = directory
	if s.activeProjectWorkLocked() || s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	capture := recoveryConfigurationCapture{params: p, previous: s.recoveryStore, catalogRevision: s.recoveryCatalogRevision}
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	applied := false
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), recoveryConfigure: &capture, work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "recovery.configure", Status: "queued", Phase: "queued", Cancelable: true, Path: directory, InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration, RecoveryConfigured: &applied}}
	s.retainProjectJobLocked(job)
	prepare := s.recoveryConfigure
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go s.runRecoveryConfiguration(job, prepare)
	return nil
}
func (s *Service) recoveryConfigurationCurrentLocked(job *projectJob) bool {
	// Recovery is a session setting. Editing or replacing the schematic does not
	// invalidate a setup for the same existing recovery store.
	return !s.closed && s.projectJob == job && s.recoveryStore == job.recoveryConfigure.previous
}
func (s *Service) installRecoveryConfigurationLocked(job *projectJob, result recoveryConfigurationResult) *recoveryStore {
	capture := job.recoveryConfigure
	previous := s.recoveryStore
	same := previous == result.store
	catalogApplied := !same || s.recoveryCatalogRevision == capture.catalogRevision
	if same {
		s.recovery.Enabled, s.recovery.Interval = capture.params.Enabled, capture.params.Interval
		if catalogApplied {
			s.recovery.Candidates, s.recovery.Problems = result.candidates, result.problems
			s.recovery.Error = ""
			s.normalizeRecoverySelectionLocked()
		}
	} else {
		if s.recoveryCancel != nil {
			s.recoveryCancel()
		}
		s.recoveryStore = result.store
		s.recoverySelectionCleared = false
		s.recovery = RecoveryState{Directory: capture.params.Directory, Enabled: capture.params.Enabled, Interval: capture.params.Interval, Candidates: result.candidates, Problems: result.problems}
		s.normalizeRecoverySelectionLocked()
		ctx, cancel := context.WithCancel(context.Background())
		s.recoveryCancel = cancel
		go s.recoveryLoop(ctx, result.store)
	}
	s.recoveryCatalogRevision++
	job.work.CatalogApplied = &catalogApplied
	*job.work.RecoveryConfigured = true
	job.work.ResultGeneration = s.documentGeneration
	if !same {
		return previous
	}
	return nil
}
func (s *Service) runRecoveryConfiguration(job *projectJob, prepare recoveryConfigurationPrepareFunc) {
	defer s.projectWorkerDone()
	result, err := prepare(job.ctx, *job.recoveryConfigure, func(phase string, bytes int) { s.projectProgress(job, phase, bytes) })
	s.mu.Lock()
	if !s.recoveryConfigurationCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	var cleanup *recoveryStore
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Recovery setup discarded; previous recovery configuration and saved files were preserved"}
	} else if err != nil || job.ctx.Err() != nil || result.store == nil {
		if err == nil {
			err = job.ctx.Err()
		}
		if err == nil {
			err = fmt.Errorf("recovery setup returned no validated store")
		}
		s.failProjectWorkLocked(job, err)
	} else {
		// Installation is an atomic in-memory publication. Lease cleanup may block
		// on an older background write, so it must run after releasing the editor.
		job.publishing = true
		job.work.Cancelable = false
		job.work.Phase = "closing-previous-recovery"
		job.work.Status = "running"
		cleanup = s.installRecoveryConfigurationLocked(job, result)
		s.revision++
		s.signalLocked("jobs", "recovery")
	}
	if !job.publishing && result.store != nil && result.store != job.recoveryConfigure.previous {
		cleanup = result.store
	}
	s.mu.Unlock()
	var cleanupErr error
	if cleanup != nil {
		cleanupErr = cleanup.closeLease()
	}
	s.mu.Lock()
	if cleanupErr != nil {
		job.work.CleanupWarning = cleanupErr.Error()
	}
	if job.publishing {
		job.work.Status = "completed"
		job.work.Phase = "completed"
	}
	s.finishProjectWorkLocked(job)
	s.mu.Unlock()
}
