package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

type recoveryDismissCapture struct {
	store           *recoveryStore
	choice          RecoveryChoiceParams
	catalogRevision uint64
}
type preparedRecoveryDismiss struct{ info os.FileInfo }
type recoveryDismissResult struct {
	removed    bool
	candidates []RecoveryCandidate
	problems   []RecoveryProblem
	scanError  error
}
type recoveryDismissPrepareFunc func(context.Context, recoveryDismissCapture, func(string, int)) (*preparedRecoveryDismiss, error)
type recoveryDismissExecuteFunc func(recoveryDismissCapture, *preparedRecoveryDismiss) (recoveryDismissResult, error)

func validateRecoveryDismiss(capture recoveryDismissCapture) (os.FileInfo, error) {
	path := capture.store.candidatePath(capture.choice.ID)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fileError(err)
	}
	data, err := readRegularBounded(path, MaxRecoveryBytes)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if capture.choice.SHA256 == "" || hex.EncodeToString(digest[:]) != capture.choice.SHA256 {
		return nil, &actionError{"file-conflict", "Recovery snapshot changed; scan again"}
	}
	// Ownership and hash are checked against the same bytes. Damaged capsules
	// may still be explicitly dismissed by hash, as in the existing workflow.
	var capsule recoveryCapsule
	_ = params(data, &capsule, "format", "version", "id", "document_key", "session", "sequence", "command_revision", "saved_at", "document_sha256", "document")
	_, sessionErr := hex.DecodeString(capsule.Session)
	if sessionErr == nil && len(capsule.Session) == 24 && capsule.Session != capture.store.session {
		status := capture.store.sessionOwnerStatus(capsule.Session)
		if status == RecoveryOwnerUnknown {
			return nil, &actionError{"unavailable", "Another session's ownership could not be verified; its recovery copy is preserved. Restore a copy or refresh after the owner closes."}
		}
		if status == RecoveryOwnerActive {
			return nil, &actionError{"unavailable", "Another active session owns this snapshot; its recovery data is preserved"}
		}
	}
	current, err := os.Lstat(path)
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(info, current) {
		return nil, &actionError{"file-conflict", "Recovery snapshot changed while validating; scan again"}
	}
	return current, nil
}
func prepareRecoveryDismiss(ctx context.Context, capture recoveryDismissCapture, progress func(string, int)) (*preparedRecoveryDismiss, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	progress("validating-recovery-dismissal", 0)
	info, err := validateRecoveryDismiss(capture)
	if err != nil {
		return nil, err
	}
	progress("prepared-recovery-dismissal", int(info.Size()))
	return &preparedRecoveryDismiss{info: info}, ctx.Err()
}
func executeRecoveryDismiss(capture recoveryDismissCapture, prepared *preparedRecoveryDismiss) (recoveryDismissResult, error) {
	result := recoveryDismissResult{}
	// Serialize with this store's checkpoint/pruning paths, without holding the
	// editor lock. Recheck exact content, file identity and active ownership.
	capture.store.mu.Lock()
	current, err := validateRecoveryDismiss(capture)
	if err == nil && !os.SameFile(current, prepared.info) {
		err = &actionError{"file-conflict", "Recovery snapshot was replaced after preparation; scan again"}
	}
	if err == nil {
		err = os.Remove(capture.store.candidatePath(capture.choice.ID))
		if err != nil {
			err = fileError(err)
		} else {
			result.removed = true
		}
	}
	capture.store.mu.Unlock()
	if err != nil {
		return result, err
	}
	result.candidates, result.problems, result.scanError = capture.store.scan()
	return result, nil
}
func (s *Service) startRecoveryDismissLocked(p RecoveryChoiceParams) error {
	if s.recoveryStore == nil || !recoveryID(p.ID) {
		return fmt.Errorf("choose a listed recovery snapshot")
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if s.activeProjectWorkLocked() || s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	capture := recoveryDismissCapture{store: s.recoveryStore, choice: p, catalogRevision: s.recoveryCatalogRevision}
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	removed := false
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), recoveryDismiss: &capture, work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "recovery.dismiss", Status: "queued", Phase: "queued", Cancelable: true, Path: capture.store.candidatePath(p.ID), InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration, RecoveryID: p.ID, RecoveryDismissed: &removed}}
	s.retainProjectJobLocked(job)
	prepare, execute := s.recoveryDismissPrepare, s.recoveryDismissExecute
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go s.runRecoveryDismiss(job, prepare, execute)
	return nil
}
func (s *Service) recoveryDismissCurrentLocked(job *projectJob) bool {
	return !s.closed && s.projectJob == job && s.recoveryStore == job.recoveryDismiss.store
}
func (s *Service) runRecoveryDismiss(job *projectJob, prepare recoveryDismissPrepareFunc, execute recoveryDismissExecuteFunc) {
	defer s.projectWorkerDone()
	prepared, err := prepare(job.ctx, *job.recoveryDismiss, func(phase string, bytes int) { s.projectProgress(job, phase, bytes) })
	s.mu.Lock()
	if !s.recoveryDismissCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	result := recoveryDismissResult{}
	if job.reason == "" && err == nil && job.ctx.Err() == nil && prepared != nil && prepared.info != nil {
		job.publishing = true
		job.work.Status = "running"
		job.work.Phase = "dismissing-recovery"
		job.work.Cancelable = false
		job.work.ResultGeneration = s.documentGeneration
		s.revision++
		s.signalLocked("jobs", "recovery")
		s.mu.Unlock()
		result, err = execute(*job.recoveryDismiss, prepared)
		s.mu.Lock()
	}
	*job.work.RecoveryDismissed = result.removed
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Recovery dismissal discarded before removal"}
	} else if err != nil || !job.publishing || !result.removed {
		if err == nil {
			err = job.ctx.Err()
		}
		if err == nil {
			err = fmt.Errorf("recovery dismissal returned no confirmed removal outcome")
		}
		s.failProjectWorkLocked(job, err)
	} else {
		current := s.recoveryDismissCurrentLocked(job)
		catalogApplied := current && s.recoveryCatalogRevision == job.recoveryDismiss.catalogRevision && result.scanError == nil
		if catalogApplied {
			s.recovery.Candidates, s.recovery.Problems = result.candidates, result.problems
			s.recovery.Error = ""
			s.normalizeRecoverySelectionLocked()
			s.recoveryCatalogRevision++
		}
		if result.scanError != nil {
			job.work.CleanupWarning = result.scanError.Error()
			if current {
				s.recovery.Error = result.scanError.Error()
			}
		}
		// A newer catalog must not continue offering the file we just removed.
		if current && result.removed && !catalogApplied {
			id := job.recoveryDismiss.choice.ID
			candidates := s.recovery.Candidates[:0]
			for _, c := range s.recovery.Candidates {
				if c.ID != id {
					candidates = append(candidates, c)
				}
			}
			s.recovery.Candidates = candidates
			problems := s.recovery.Problems[:0]
			for _, p := range s.recovery.Problems {
				if p.ID != id {
					problems = append(problems, p)
				}
			}
			s.recovery.Problems = problems
			s.normalizeRecoverySelectionLocked()
			s.recoveryCatalogRevision++
		}
		job.work.CatalogApplied = &catalogApplied
		job.work.SourceSHA256 = job.recoveryDismiss.choice.SHA256
		job.work.Status = "completed"
		job.work.Phase = "completed"
	}
	s.finishProjectWorkLocked(job)
	s.mu.Unlock()
}
