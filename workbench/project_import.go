package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type recordingImportCapture struct {
	generation          uint64
	viewerRevision      uint64
	sweepViewerRevision uint64
}
type projectImportFunc func(context.Context, string, func(string, int)) (*verificationRecord, [32]byte, error)

func prepareRecordingImport(ctx context.Context, path string, progress func(string, int)) (*verificationRecord, [32]byte, error) {
	progress("reading", 0)
	if err := ctx.Err(); err != nil {
		return nil, [32]byte{}, err
	}
	data, err := readRegularBounded(path, MaxRecordingBytes)
	if err != nil {
		return nil, [32]byte{}, err
	}
	if err := ctx.Err(); err != nil {
		return nil, [32]byte{}, err
	}
	progress("validating-recording", len(data))
	record, err := decodeRecording(data)
	if err != nil {
		return nil, [32]byte{}, &actionError{"invalid-recording", recoveryMessage(err)}
	}
	return record, sha256.Sum256(data), ctx.Err()
}
func (s *Service) startRecordingImportLocked(p RecordingImportParams) error {
	if s.activeProjectWorkLocked() || s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if err := s.requireRecordingSlotLocked(); err != nil {
		return err
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), importing: &recordingImportCapture{generation: s.documentGeneration, viewerRevision: s.viewerRevision, sweepViewerRevision: s.sweepViewerRevision}, work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "verification.import", Status: "queued", Phase: "queued", Cancelable: true, Path: path, InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	s.retainProjectJobLocked(job)
	load := s.projectImport
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go s.runRecordingImport(job, load)
	return nil
}
func (s *Service) importCurrentLocked(job *projectJob) bool {
	return !s.closed && s.projectJob == job && s.documentGeneration == job.importing.generation
}

// Asset readers and publishers remain typed while sharing cancellation,
// generation checks and filesystem worker lifecycle.
func runAssetImport[T any](s *Service, job *projectJob, load func(context.Context, string, func(string, int)) (*T, [32]byte, error), publish func(*projectJob, *T, [32]byte) error) {
	defer s.projectWorkerDone()
	record, digest, err := load(job.ctx, job.work.Path, func(phase string, bytes int) { s.projectProgress(job, phase, bytes) })
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.importCurrentLocked(job) && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{job.reason, "Analysis import discarded; current project and live simulation were preserved"}
	} else if err != nil || job.ctx.Err() != nil || record == nil {
		if err == nil {
			err = job.ctx.Err()
		}
		if err == nil {
			err = fmt.Errorf("reader returned no validated analysis asset")
		}
		s.failProjectWorkLocked(job, err)
	} else if err = publish(job, record, digest); err != nil {
		s.failProjectWorkLocked(job, err)
	}
	s.finishProjectWorkLocked(job)
}
func (s *Service) runRecordingImport(job *projectJob, load projectImportFunc) {
	runAssetImport(s, job, load, s.publishRecordingImportLocked)
}
func (s *Service) publishRecordingImportLocked(job *projectJob, record *verificationRecord, digest [32]byte) error {
	if err := s.requireRecordingSlotLocked(); err != nil {
		return err
	}
	s.operationSerial++
	id := fmt.Sprintf("record-%d", s.operationSerial)
	sourceHash := hex.EncodeToString(digest[:])
	summary := OperationSummary{ID: id, Kind: "record-import", Status: "completed", Actor: job.work.Actor, InputRevision: record.metadata.InputRevision, ElectricalRevision: record.metadata.ElectricalRevision, DocumentGeneration: record.metadata.DocumentGeneration, RequestedDuration: record.metadata.Duration, SimulatedTime: record.metadata.Duration, CompletedSteps: len(record.rows), TotalSteps: len(record.rows), Progress: 1, SourcePath: job.work.Path, SourceSHA256: sourceHash}
	operation := &verificationOperation{keep: true, summary: summary, record: record, inputMetadata: cloneVerificationMetadata(record.metadata)}
	s.retainOperationLocked(operation)
	selected := s.viewerRevision == job.importing.viewerRevision && s.sweepViewerRevision == job.importing.sweepViewerRevision
	if selected {
		s.selectRecordingLocked(id)
		s.recording.viewing = true
	} else if s.selectedOperation != "" && s.operations[s.selectedOperation] == nil {
		s.selectRecordingLocked("")
	}
	s.lastOperation = id
	associated := true
	job.work.Associated = &associated
	job.work.ViewerSelected = &selected
	job.work.ResultOperationID = id
	job.work.SourceSHA256 = sourceHash
	job.work.ResultGeneration = s.documentGeneration
	job.work.Status = "completed"
	job.work.Phase = "completed"
	return nil
}
