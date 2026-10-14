package workbench

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

// ProjectWork reports file preparation separately from replacing the document.
// Progress is a named phase; file validation has no trustworthy percentage.
type ProjectWork struct {
	ID                 string   `json:"id"`
	Command            string   `json:"command"`
	Status             string   `json:"status"`
	Phase              string   `json:"phase"`
	Actor              string   `json:"actor"`
	Path               string   `json:"path"`
	InputRevision      uint64   `json:"input_revision"`
	CommandRevision    uint64   `json:"command_revision"`
	DocumentGeneration uint64   `json:"document_generation"`
	Bytes              int      `json:"bytes"`
	ResultGeneration   uint64   `json:"result_document_generation,omitempty"`
	ResultRun          uint64   `json:"result_run_id,omitempty"`
	Error              *Failure `json:"error,omitempty"`
}

func cloneProjectWork(in *ProjectWork) *ProjectWork {
	if in == nil {
		return nil
	}
	out := *in
	if in.Error != nil {
		v := *in.Error
		out.Error = &v
	}
	return &out
}

func (s *Service) projectWorkLocked() *ProjectWork {
	if s.projectJob == nil {
		return nil
	}
	return cloneProjectWork(&s.projectJob.work)
}

type loadGuard struct {
	generation, electrical, actuation, operationSerial uint64
	graph, preferences, recordings, disk               [32]byte
	path, mode, pending                                string
}

func (s *Service) loadGuardLocked() loadGuard {
	return loadGuard{
		generation:      s.documentGeneration,
		electrical:      s.electrical,
		actuation:       s.actuationRevision,
		operationSerial: s.operationSerial,
		graph:           contentHash(s.graph, s.transientSettings),
		preferences:     preferencesHash(s.pins, s.displayPreferences, s.exampleID),
		recordings:      s.recordingMembershipHashLocked(),
		disk:            s.diskHash,
		path:            s.documentPath,
		mode:            s.inputMode,
		pending:         s.pendingInputMode,
	}
}

type projectLoadFunc func(context.Context, string, func(string, int)) (Document, [32]byte, error)
type projectJob struct {
	work                         ProjectWork
	ctx                          context.Context
	cancel                       context.CancelFunc
	done                         chan struct{}
	guard                        loadGuard
	reason, legacyKey, requestID string
	final                        Response
}

func prepareProjectLoad(ctx context.Context, path string, progress func(string, int)) (Document, [32]byte, error) {
	progress("reading", 0)
	if err := ctx.Err(); err != nil {
		return Document{}, [32]byte{}, err
	}
	data, err := readDocumentFile(path)
	if err != nil {
		return Document{}, [32]byte{}, err
	}
	if err := ctx.Err(); err != nil {
		return Document{}, [32]byte{}, err
	}
	progress("validating", len(data))
	d, err := decodeDocument(data)
	if err != nil {
		return Document{}, [32]byte{}, &actionError{"invalid-document", err.Error()}
	}
	return d, sha256.Sum256(data), ctx.Err()
}
func (s *Service) activeProjectWorkLocked() bool {
	return s.projectJob != nil && operationActive(s.projectJob.work.Status)
}
func (s *Service) startProjectLoadLocked(p LoadParams) error {
	if s.activeProjectWorkLocked() {
		return &actionError{"busy", "Finish or cancel the active project operation first"}
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if err := s.requireCleanLocked(p.DiscardChanges); err != nil {
		return err
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), guard: s.loadGuardLocked(), work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "project.load", Status: "queued", Phase: "queued", Path: path, InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	s.projectJob = job
	load := s.projectLoad
	s.liveWorkers.Add(1)
	go s.runProjectLoad(job, load)
	return nil
}
func (s *Service) cancelProjectWorkLocked(reason string) {
	if s.activeProjectWorkLocked() {
		if s.projectJob.reason == "" {
			s.projectJob.reason = reason
		}
		s.projectJob.work.Status = "canceling"
		s.projectJob.cancel()
	}
}
func (s *Service) runProjectLoad(job *projectJob, load projectLoadFunc) {
	defer s.liveWorkers.Done()
	progress := func(phase string, bytes int) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.projectJob == job && job.ctx.Err() == nil {
			job.work.Status = "running"
			job.work.Phase = phase
			job.work.Bytes = bytes
			s.revision++
			s.signalLocked()
		}
	}
	document, digest, err := load(job.ctx, job.work.Path, progress)
	s.mu.Lock()
	defer s.mu.Unlock()
	if (s.closed || s.projectJob != job || s.loadGuardLocked() != job.guard) && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Project open was discarded; the current document was preserved"}
	} else if err != nil || job.ctx.Err() != nil {
		if err == nil {
			err = job.ctx.Err()
		}
		code := "file-error"
		var action *actionError
		if errors.As(err, &action) {
			code = action.code
		}
		if errors.Is(err, context.DeadlineExceeded) {
			code = "timeout"
		}
		job.work.Status = "failed"
		job.work.Error = &Failure{code, err.Error()}
	} else {
		before := s.historyLocked()
		before.Actor = job.work.Actor
		before.CommandRevision = job.work.CommandRevision
		before.Name = historyName("project.load")
		s.installDocumentLocked(document)
		s.recoveredFrom = ""
		s.documentPath = job.work.Path
		s.savedGraph = contentHash(s.graph, s.transientSettings)
		s.savedPreferences = preferencesHash(s.pins, s.displayPreferences, s.exampleID)
		s.savedRecordings = s.recordingMembershipHashLocked()
		s.diskHash = digest
		if contentHash(before.Graph, before.Settings) != s.savedGraph || preferencesHash(before.Pins, before.Display, before.ExampleID) != s.savedPreferences {
			s.undo = append(s.undo, before)
			s.redo = nil
			if len(s.undo) > 100 {
				s.undo = append([]historyEntry(nil), s.undo[len(s.undo)-100:]...)
			}
		}
		job.work.Status = "completed"
		job.work.Phase = "completed"
		s.electrical++
		s.startLocked()
		job.work.ResultGeneration = s.documentGeneration
		job.work.ResultRun = s.run
	}
	job.cancel()
	s.revision++
	s.signalLocked()
	job.final = Response{Version: ProtocolVersion, ID: job.requestID, Revision: s.revision, Result: s.snapshotLocked()}
	if job.work.Error != nil {
		v := *job.work.Error
		job.final.Error = &v
		job.final.Help = s.requestDiagnosticLocked(Request{Command: "project.load"}, v.Code, v.Message)
	}
	if old, ok := s.cache[job.legacyKey]; ok && job.legacyKey != "" {
		old.response = cloneMutationResponse(job.final)
		s.cache[job.legacyKey] = old
	}
	close(job.done)
}
func (s *Service) awaitProjectLoad(ctx context.Context, r Request, actor string, accepted Response) Response {
	if accepted.Error != nil {
		return accepted
	}
	s.mu.Lock()
	key := actor + "\x00" + r.Key
	if cached, ok := s.cache[key]; ok {
		state := cached.response.Result.(Snapshot)
		if state.ProjectWork != nil && !operationActive(state.ProjectWork.Status) {
			out := cloneMutationResponse(cached.response)
			out.ID = r.ID
			out.Duplicate = accepted.Duplicate
			s.mu.Unlock()
			return out
		}
	}
	job := s.projectJob
	if job == nil || job.work.ID != accepted.Result.(Snapshot).ProjectWork.ID {
		s.mu.Unlock()
		return s.failure(r, "operation-unavailable", "Original project operation has expired; inspect current state")
	}
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		return s.failure(r, "canceled", "Request wait ended; admitted project open continues. Inspect state or retry the original key")
	case <-job.done:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := cloneMutationResponse(job.final)
	out.ID = r.ID
	out.Duplicate = accepted.Duplicate
	return out
}
