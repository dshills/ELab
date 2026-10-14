package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type saveCapture struct {
	document                             Document
	generation                           uint64
	sourcePath                           string
	disk, graph, preferences, recordings [32]byte
	overwrite                            bool
}
type preparedSave struct {
	path, temp      string
	data            []byte
	replace         bool
	destination     os.FileInfo
	destinationHash [32]byte
}
type projectSaveFunc func(context.Context, saveCapture, string, func(string, int)) (*preparedSave, error)
type projectPublishFunc func(*preparedSave) error

func (s *Service) startProjectSaveLocked(p SaveParams) error {
	if s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Wait for pending filesystem workers to finish"}
	}
	if s.activeProjectWorkLocked() && s.projectJob.work.Command == "project.save" {
		return &actionError{"busy", "Finish or cancel the active project save first"}
	}
	if s.verificationActiveLocked() {
		return &actionError{"busy", "Finish or cancel the active verification before saving the project"}
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

func prepareProjectSave(ctx context.Context, capture saveCapture, path string, progress func(string, int)) (_ *preparedSave, err error) {
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
	prepared := &preparedSave{path: path, data: data}
	progress("checking-destination", len(data))
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return nil, fileError(fmt.Errorf("save destination must be a regular file, not a link or directory"))
		}
		if path != capture.sourcePath && !capture.overwrite {
			return nil, &actionError{"file-exists", "Destination exists. Choose another path or explicitly allow overwrite."}
		}
		current, err := readDocumentFile(path)
		if err != nil {
			return nil, err
		}
		prepared.destinationHash = sha256.Sum256(current)
		if path == capture.sourcePath && !capture.overwrite && prepared.destinationHash != capture.disk {
			return nil, &actionError{"file-conflict", "The file changed outside ELab. Save As, or explicitly overwrite it."}
		}
		prepared.replace = true
		prepared.destination = info
	} else if !os.IsNotExist(err) {
		return nil, fileError(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	progress("writing", len(data))
	f, err := os.CreateTemp(filepath.Dir(path), ".elab-save-*.tmp")
	if err != nil {
		return nil, fileError(err)
	}
	prepared.temp = f.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(prepared.temp)
		}
	}()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return nil, fileError(err)
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return nil, fileError(err)
	}
	if err = f.Close(); err != nil {
		return nil, fileError(err)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return prepared, nil
}

// Once the service marks publishing, cancellation cannot claim to have stopped
// a filesystem operation that may already be replacing the destination.
func publishProjectSave(prepared *preparedSave) error {
	var err error
	if prepared.replace {
		current, readErr := readDocumentFile(prepared.path)
		if readErr != nil {
			return &actionError{"file-conflict", "Save destination changed during preparation; choose the destination again."}
		}
		info, statErr := os.Lstat(prepared.path)
		if statErr != nil || !info.Mode().IsRegular() || !os.SameFile(info, prepared.destination) || sha256.Sum256(current) != prepared.destinationHash {
			return &actionError{"file-conflict", "Save destination changed during preparation; choose the destination again."}
		}
		err = os.Rename(prepared.temp, prepared.path)
	} else {
		err = os.Link(prepared.temp, prepared.path)
	}
	if err != nil {
		if os.IsExist(err) {
			return &actionError{"file-exists", "Destination was created while saving; choose another path."}
		}
		return fileError(err)
	}
	if directory, err := os.Open(filepath.Dir(prepared.path)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
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
		s.signalLocked()
		s.mu.Unlock()
		err = publish(prepared)
		s.mu.Lock()
	}
	if prepared != nil {
		job.work.Phase = "cleanup"
		s.revision++
		s.signalLocked()
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
			s.signalLocked()
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
