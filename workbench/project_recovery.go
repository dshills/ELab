package workbench

import (
	"context"
	"fmt"
	"time"
)

type RecoveryWorkParams struct {
	Kind string `json:"kind"`
}
type manualRecoveryCapture struct {
	kind            string
	store           *recoveryStore
	capture         recoveryCapture
	catalogRevision uint64
}
type recoveryWorkResult struct {
	id, savedAt string
	candidates  []RecoveryCandidate
	problems    []RecoveryProblem
	scanError   error
}
type recoveryPrepareFunc func(context.Context, manualRecoveryCapture, func(string, int)) ([]byte, error)
type recoveryExecuteFunc func(manualRecoveryCapture, []byte) (recoveryWorkResult, error)

func prepareManualRecovery(ctx context.Context, capture manualRecoveryCapture, progress func(string, int)) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if capture.kind == "scan" {
		progress("scanning", 0)
		return nil, nil
	}
	progress("serializing-recovery", 0)
	data, err := encodeRecoveryCapture(capture.capture)
	if err != nil {
		return nil, err
	}
	progress("prepared-recovery", len(data))
	return data, ctx.Err()
}
func executeManualRecovery(capture manualRecoveryCapture, data []byte) (recoveryWorkResult, error) {
	result := recoveryWorkResult{}
	if capture.kind == "checkpoint" {
		id, savedAt, err := capture.store.checkpointEncoded(capture.capture, data)
		result.id, result.savedAt = id, savedAt
		if err != nil {
			return result, err
		}
	}
	var err error
	result.candidates, result.problems, err = capture.store.scan()
	if capture.kind == "checkpoint" {
		result.scanError = err
		return result, nil
	}
	return result, err
}
func (s *Service) startRecoveryWorkLocked(p RecoveryWorkParams) error {
	if p.Kind != "scan" && p.Kind != "checkpoint" {
		return fmt.Errorf("recovery work kind must be scan or checkpoint")
	}
	if s.recoveryStore == nil {
		return &actionError{"unavailable", "Configure a recovery directory first"}
	}
	if s.activeProjectWorkLocked() || s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	capture := manualRecoveryCapture{kind: p.Kind, store: s.recoveryStore, capture: s.recoveryCaptureLocked(), catalogRevision: s.recoveryCatalogRevision}
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), recoveryWork: &capture, work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "recovery." + p.Kind, Status: "queued", Phase: "queued", Cancelable: true, Path: s.recoveryStore.directory, InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	s.retainProjectJobLocked(job)
	prepare, execute := s.recoveryPrepare, s.recoveryExecute
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go s.runRecoveryWork(job, prepare, execute)
	return nil
}
func (s *Service) recoveryWorkCurrentLocked(job *projectJob) bool {
	return !s.closed && s.projectJob == job && s.recoveryStore == job.recoveryWork.store && (job.recoveryWork.kind == "scan" || s.documentGeneration == job.recoveryWork.capture.generation)
}
func (s *Service) runRecoveryWork(job *projectJob, prepare recoveryPrepareFunc, execute recoveryExecuteFunc) {
	defer s.projectWorkerDone()
	data, err := prepare(job.ctx, *job.recoveryWork, func(phase string, bytes int) { s.projectProgress(job, phase, bytes) })
	s.mu.Lock()
	if !s.recoveryWorkCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	result := recoveryWorkResult{}
	if job.reason == "" && err == nil && job.ctx.Err() == nil {
		if job.recoveryWork.kind == "checkpoint" {
			job.publishing = true
			job.work.Phase = "publishing-recovery"
			job.work.Cancelable = false
		} else {
			job.work.Phase = "scanning"
		}
		job.work.Status = "running"
		s.revision++
		s.signalLocked()
		s.mu.Unlock()
		result, err = execute(*job.recoveryWork, data)
		s.mu.Lock()
	}
	if !job.publishing && !s.recoveryWorkCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Recovery work discarded; current project and saved files were preserved"}
	} else if err != nil || !job.publishing && job.ctx.Err() != nil {
		if err == nil {
			err = job.ctx.Err()
		}
		s.failProjectWorkLocked(job, err)
	} else {
		current := s.recoveryWorkCurrentLocked(job)
		catalogApplied := current && s.recoveryCatalogRevision == job.recoveryWork.catalogRevision && result.scanError == nil
		// Preserve a more recent recovery publication and current user selection.
		if current && s.recoveryCatalogRevision == job.recoveryWork.catalogRevision {
			if result.scanError == nil {
				s.recovery.Candidates = result.candidates
				s.recovery.Problems = result.problems
				s.normalizeRecoverySelectionLocked()
				s.recoveryCatalogRevision++
			}
			if result.scanError != nil {
				s.recovery.Error = result.scanError.Error()
			} else {
				s.recovery.Error = ""
			}
			if result.id != "" {
				s.recovery.LastSnapshot = result.id
				s.recovery.LastSavedAt = result.savedAt
			}
		}
		job.work.RecoveryID = result.id
		job.work.CatalogApplied = &catalogApplied
		job.work.RecoverySavedAt = result.savedAt
		job.work.Associated = &current
		job.work.ResultGeneration = job.recoveryWork.capture.generation
		job.work.Status = "completed"
		job.work.Phase = "completed"
	}
	job.recoveryWork.capture.document = Document{}
	s.finishProjectWorkLocked(job)
	s.mu.Unlock()
}
