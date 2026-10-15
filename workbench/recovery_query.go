package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const MaxRecoveryReaders = 4

// Called with s.mu held; it returns with that same lock held. Only admission,
// reader bookkeeping and response publication use the editor lock. Reading,
// model validation, lease checks and result projection run outside it.
func (s *Service) recoveryQueryLocked(ctx context.Context, command string, raw json.RawMessage) (any, error) {
	var choice RecoveryChoiceParams
	var samples RecoverySamplesParams
	if command == "recovery.samples" {
		if err := params(raw, &samples, "id", "sha256", "recording_index"); err != nil {
			return nil, err
		}
		choice = RecoveryChoiceParams{ID: samples.ID, SHA256: samples.SHA256}
	} else {
		if err := params(raw, &choice, "id", "sha256"); err != nil {
			return nil, err
		}
	}
	if s.recoveryStore == nil {
		return nil, &actionError{"unavailable", "Configure recovery first"}
	}
	if s.closed {
		return nil, &actionError{"unavailable", "Recovery reads are unavailable after Close; open a new session to inspect retained files"}
	}
	if len(s.recoveryReaders) >= MaxRecoveryReaders {
		return nil, &actionError{"busy", "Four recovery reads are active; cancel or finish one before reading another"}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store, read := s.recoveryStore, s.recoveryQueryRead
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	s.recoveryReadSerial++
	id := s.recoveryReadSerial
	if s.recoveryReaders == nil {
		s.recoveryReaders = map[uint64]context.CancelFunc{}
	}
	s.recoveryReaders[id] = cancel
	s.liveWorkers.Add(1)
	defer func() { delete(s.recoveryReaders, id); cancel(); s.liveWorkers.Done() }()
	s.mu.Unlock()
	defer s.mu.Lock()
	var page *RecoverySamplesParams
	if command == "recovery.samples" {
		page = &samples
	}
	return projectRecoveryRead(bounded, store, choice, page, read, func(string, int) {})
}

// Shared immutable file projection for pure queries and observable reads.
func projectRecoveryRead(bounded context.Context, store *recoveryStore, choice RecoveryChoiceParams, samples *RecoverySamplesParams, read recoveryRestoreFunc, progress func(string, int)) (any, error) {
	if err := bounded.Err(); err != nil {
		return nil, err
	}
	capsule, document, hash, err := read(bounded, store, choice, progress)
	if bounded.Err() != nil {
		return nil, bounded.Err()
	}
	if err != nil {
		return nil, err
	}
	if samples != nil {
		if document.Recordings == nil || samples.Recording < 0 || samples.Recording >= len(document.Recordings.Entries) {
			return nil, fmt.Errorf("choose an embedded recovery recording index")
		}
		data := document.Recordings.Entries[samples.Recording].Data
		page, err := recordedSamplePage(choice.ID, data.Metadata.Probes, data.Rows, samples.Offset, samples.Limit)
		if bounded.Err() != nil {
			return nil, bounded.Err()
		}
		return page, err
	}
	candidate := RecoveryCandidate{ID: capsule.ID, DocumentKey: capsule.DocumentKey, SavedAt: capsule.SavedAt, SourcePath: capsule.SourcePath, SHA256: hash, CurrentSession: capsule.Session == store.session, ActiveSession: store.sessionActive(capsule.Session), Recordings: documentRecordingCount(document), Components: len(document.Graph.Components)}
	result := RecoveryInspection{Directory: store.directory, Candidate: candidate, Document: inspectDocument(document)}
	if bounded.Err() != nil {
		return nil, bounded.Err()
	}
	return result, nil
}
