package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type sweepImportFunc func(context.Context, string, func(string, int)) (*SweepRecord, [32]byte, error)

func prepareSweepImport(ctx context.Context, path string, progress func(string, int)) (*SweepRecord, [32]byte, error) {
	progress("reading", 0)
	if err := ctx.Err(); err != nil {
		return nil, [32]byte{}, err
	}
	data, err := readRegularBounded(path, MaxProjectBytes)
	if err != nil {
		return nil, [32]byte{}, err
	}
	if err = ctx.Err(); err != nil {
		return nil, [32]byte{}, err
	}
	progress("validating-sweep", len(data))
	record, err := validateSweepRecord(data)
	if err != nil {
		return nil, [32]byte{}, &actionError{"invalid-sweep", recoveryMessage(err)}
	}
	return record, sha256.Sum256(data), ctx.Err()
}
func (s *Service) startSweepImportLocked(p RecordingImportParams) error {
	if s.activeProjectWorkLocked() || s.projectWorkers >= MaxProjectOperations {
		return &actionError{"busy", "Finish or cancel the active project file operation first"}
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if err := s.requireSweepSlotLocked(); err != nil {
		return err
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	s.projectSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &projectJob{ctx: ctx, cancel: cancel, done: make(chan struct{}), importing: &recordingImportCapture{generation: s.documentGeneration, viewerRevision: s.viewerRevision, sweepViewerRevision: s.sweepViewerRevision}, work: ProjectWork{ID: fmt.Sprintf("project-%d", s.projectSerial), Command: "sweep.import", Status: "queued", Phase: "queued", Cancelable: true, Path: path, InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	s.retainProjectJobLocked(job)
	load := s.sweepImport
	s.projectWorkers++
	s.liveWorkers.Add(1)
	go runAssetImport(s, job, load, s.publishSweepImportLocked)
	return nil
}
func (s *Service) publishSweepImportLocked(job *projectJob, record *SweepRecord, digest [32]byte) error {
	if err := s.requireSweepSlotLocked(); err != nil {
		return err
	}
	selected := s.sweepViewerRevision == job.importing.sweepViewerRevision && s.viewerRevision == job.importing.viewerRevision
	s.sweepSerial++
	id := fmt.Sprintf("imported-sweep-%d", s.sweepSerial)
	sourceHash := hex.EncodeToString(digest[:])
	operation := &sweepJob{keep: true, record: record, metadata: record.Metadata, sourceHash: sweepGraphHash(record.Metadata.Source.Graph), summary: SweepSummary{ID: id, Kind: "sweep-import", Status: "completed", Actor: job.work.Actor, SourcePath: job.work.Path, SourceSHA256: sourceHash, Completed: len(record.Result.Points), Total: len(record.Result.Points)}}
	s.retainSweepJobLocked(operation)
	if selected {
		store := s.ensureSweepsLocked()
		store.selected, store.visible, store.traceID, store.cursor = id, true, record.Result.Probes[0].ID, nil
		s.sweepViewerRevision++
		s.dismissPrintPreviewLocked()
	}
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
