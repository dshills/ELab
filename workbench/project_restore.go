package workbench

import (
	"context"
	"fmt"
	"time"
)

type recoveryRestoreCapture struct {
	store  *recoveryStore
	choice RecoveryChoiceParams
}
type recoveryRestoreFunc func(context.Context, *recoveryStore, RecoveryChoiceParams, func(string, int)) (recoveryCapsule, Document, string, error)

func prepareRecoveryRestore(ctx context.Context, store *recoveryStore, p RecoveryChoiceParams, progress func(string, int)) (recoveryCapsule, Document, string, error) {
	progress("reading-recovery", 0)
	if err := ctx.Err(); err != nil {
		return recoveryCapsule{}, Document{}, "", err
	}
	capsule, document, hash, err := store.read(p.ID)
	if err != nil {
		return capsule, document, hash, &actionError{"invalid-recovery", err.Error()}
	}
	progress("validated-recovery", len(capsule.Document))
	if p.SHA256 == "" || hash != p.SHA256 {
		return capsule, document, hash, &actionError{"file-conflict", "Recovery snapshot changed; scan again before choosing it"}
	}
	return capsule, document, hash, ctx.Err()
}
func (s *Service) startRecoveryRestoreLocked(p RecoveryChoiceParams) error {
	if s.activeProjectWorkLocked() || s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if err := s.requireCleanLocked(p.DiscardChanges); err != nil {
		return err
	}
	if s.recoveryStore == nil {
		return &actionError{"unavailable", "Configure recovery first"}
	}
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	capture := &recoveryRestoreCapture{store: s.recoveryStore, choice: p}
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), restoring: capture, guard: s.loadGuardLocked(), work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "recovery.restore", Status: "queued", Phase: "queued", Cancelable: true, Path: s.recoveryStore.candidatePath(p.ID), InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	s.retainProjectJobLocked(job)
	load := s.recoveryRestore
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go s.runRecoveryRestore(job, load)
	return nil
}
func (s *Service) restoreCurrentLocked(job *projectJob) bool {
	return !s.closed && s.projectJob == job && s.recoveryStore == job.restoring.store && s.loadGuardLocked() == job.guard
}
func (s *Service) runRecoveryRestore(job *projectJob, load recoveryRestoreFunc) {
	defer s.projectWorkerDone()
	capsule, document, hash, err := load(job.ctx, job.restoring.store, job.restoring.choice, func(phase string, bytes int) { s.projectProgress(job, phase, bytes) })
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.restoreCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Recovery restore discarded; newer project/edits and saved files were preserved"}
	} else if err != nil || job.ctx.Err() != nil {
		if err == nil {
			err = job.ctx.Err()
		}
		s.failProjectWorkLocked(job, err)
	} else {
		before := s.historyLocked()
		before.Actor = job.work.Actor
		before.CommandRevision = job.work.CommandRevision
		before.Name = historyName("recovery.restore")
		s.installDocumentLocked(document)
		s.documentPath = ""
		s.recoveredFrom = capsule.SourcePath
		if s.recoveredFrom == "" {
			s.recoveredFrom = "Unsaved schematic"
		}
		s.diskHash = [32]byte{}
		s.savedGraph = [32]byte{}
		if contentHash(before.Graph, before.Settings) != contentHash(s.graph, s.transientSettings) || preferencesHash(before.Pins, before.Display, before.ExampleID) != preferencesHash(s.pins, s.displayPreferences, s.exampleID) {
			s.undo = append(s.undo, before)
			s.redo = nil
			if len(s.undo) > 100 {
				s.undo = append([]historyEntry(nil), s.undo[len(s.undo)-100:]...)
			}
		}
		job.work.Status = "completed"
		job.work.Phase = "completed"
		job.work.SourceSHA256 = hash
		job.work.RecoveryID = capsule.ID
		job.work.RecoverySavedAt = capsule.SavedAt
		s.electrical++
		s.startLocked()
		job.work.ResultGeneration = s.documentGeneration
		job.work.ResultRun = s.run
	}
	s.finishProjectWorkLocked(job)
}
