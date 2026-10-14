package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type saveCapture struct {
	document                             Document
	generation                           uint64
	sourcePath                           string
	disk, graph, preferences, recordings [32]byte
	overwrite                            bool
}
type projectSaveFunc func(context.Context, saveCapture, string, func(string, int)) (*preparedFile, error)

func (s *Service) startProjectSaveLocked(p SaveParams) error {
	if s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Wait for pending filesystem workers to finish"}
	}
	if s.activeProjectWorkLocked() && s.projectJob.work.Command != "project.load" {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	if s.verificationActiveLocked() {
		return &actionError{"busy", "Finish or cancel the active analysis before saving the project"}
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	path := p.Path
	if path == "" {
		path = s.documentPath
	}
	if path == "" {
		return &actionError{"unavailable", "Choose a destination for this schematic."}
	}
	path, err := documentPath(path)
	if err != nil {
		return err
	}
	capture := saveCapture{document: s.documentLocked(), generation: s.documentGeneration, sourcePath: s.documentPath, disk: s.diskHash,
		graph: contentHash(s.graph, s.transientSettings), preferences: preferencesHash(s.pins, s.displayPreferences, s.exampleID), recordings: s.recordingMembershipHashLocked(), overwrite: p.Overwrite}
	// A new explicit Save supersedes a pending Open. Its later file failure does
	// not silently revive that older replacement request.
	s.cancelProjectWorkLocked("superseded")
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), save: &capture,
		work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "project.save", Status: "queued", Phase: "queued", Cancelable: true, Path: path, InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	s.retainProjectJobLocked(job)
	prepare, publish := s.projectSave, s.projectPublish
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go s.runProjectSave(job, prepare, publish)
	return nil
}

func prepareProjectSave(ctx context.Context, capture saveCapture, path string, progress func(string, int)) (_ *preparedFile, err error) {
	progress("serializing", 0)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(capture.document, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) > MaxProjectBytes {
		return nil, &actionError{"invalid-document", "project file exceeds 8 MiB"}
	}
	return prepareFileData(ctx, data, path, capture.sourcePath, capture.disk, capture.overwrite, ".elab-save-*.tmp", progress)
}

func (s *Service) saveCurrentLocked(job *projectJob) bool {
	return !s.closed && s.projectJob == job && s.documentGeneration == job.save.generation && s.documentPath == job.save.sourcePath && s.diskHash == job.save.disk
}
func (s *Service) runProjectSave(job *projectJob, prepare projectSaveFunc, publish projectPublishFunc) {
	defer s.projectWorkerDone()
	prepared, err := prepare(job.ctx, *job.save, job.work.Path, func(phase string, bytes int) { s.projectProgress(job, phase, bytes) })
	s.mu.Lock()
	if !s.saveCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason == "" && err == nil && job.ctx.Err() == nil {
		job.publishing = true
		job.work.Phase = "publishing"
		job.work.Status = "running"
		job.work.Cancelable = false
		s.revision++
		s.signalLocked("jobs", "project", "recovery")
		s.mu.Unlock()
		err = publish(prepared)
		s.mu.Lock()
	}
	if prepared != nil {
		job.work.Phase = "cleanup"
		s.revision++
		s.signalLocked("jobs", "project", "recovery")
		s.mu.Unlock()
		removeErr := os.Remove(prepared.temp)
		s.mu.Lock()
		if removeErr != nil && !os.IsNotExist(removeErr) {
			job.work.CleanupWarning = removeErr.Error()
		}
	}

	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Project save was discarded before publication; the previous destination was preserved"}
	} else if err != nil || !job.publishing {
		if err == nil {
			err = job.ctx.Err()
		}
		s.failProjectWorkLocked(job, err)
	} else {
		associated := s.saveCurrentLocked(job)
		if associated {
			s.documentPath = job.work.Path
			s.diskHash = sha256.Sum256(prepared.data)
			s.savedGraph = job.save.graph
			s.savedPreferences = job.save.preferences
			s.savedRecordings = job.save.recordings
			s.recoveredFrom = ""
		}
		job.work.ResultGeneration = job.save.generation
		digest := sha256.Sum256(prepared.data)
		job.work.ArtifactSHA256 = hex.EncodeToString(digest[:])
		if associated && s.recoveryStore != nil {
			s.revision++
			s.signalLocked("jobs", "project", "recovery")
			capture := s.recoveryCaptureLocked()
			store := s.recoveryStore
			job.work.Phase = "recovery-cleanup"
			s.mu.Unlock()
			id, savedAt, cleanupErr := store.checkpoint(capture)
			candidates, problems, scanErr := store.scan()
			if cleanupErr == nil {
				cleanupErr = scanErr
			}
			s.mu.Lock()
			if !s.closed && s.recoveryStore == store && s.documentGeneration == capture.generation && s.revision == capture.stateRevision {
				if scanErr == nil {
					s.recoveryCatalogRevision++
					s.recovery.Candidates = candidates
					s.recovery.Problems = problems
					s.normalizeRecoverySelectionLocked()
				}
				if cleanupErr != nil {
					s.recovery.Error = cleanupErr.Error()
				} else {
					s.recovery.Error = ""
					if id != "" {
						s.recovery.LastSnapshot = id
						s.recovery.LastSavedAt = savedAt
					}
				}
			}
		}
		// Recovery cleanup runs unlocked too. A replacement during that phase
		// must not leave the terminal outcome claiming current association.
		associated = !s.closed && s.documentGeneration == job.save.generation && s.documentPath == job.work.Path && s.diskHash == digest
		job.work.Associated = &associated
		job.work.Status = "completed"
		job.work.Phase = "completed"
	}
	job.save.document = Document{}
	s.finishProjectWorkLocked(job)
	s.mu.Unlock()
}
