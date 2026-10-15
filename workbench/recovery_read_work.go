package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const MaxRecoveryReadJobs = 8
const MaxRecoveryReadResultBytes = 2 * MaxProjectBytes
const MaxRecoveryReadResponseBytes = 1024*1024 - 1024 // Leave room for the response envelope.

type RecoveryReadParams struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	SHA256    string `json:"sha256"`
	Recording *int   `json:"recording_index,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}
type RecoveryReadWork struct {
	ID                 string             `json:"id"`
	Input              RecoveryReadParams `json:"input"`
	Directory          string             `json:"directory"`
	Status             string             `json:"status"`
	Phase              string             `json:"phase"`
	Actor              string             `json:"actor"`
	InputRevision      uint64             `json:"input_revision"`
	CommandRevision    uint64             `json:"command_revision"`
	DocumentGeneration uint64             `json:"document_generation"`
	Bytes              int                `json:"document_bytes"`
	ResultBytes        int                `json:"result_bytes"`
	ResultSHA256       string             `json:"result_sha256,omitempty"`
	Error              *Failure           `json:"error,omitempty"`
}
type RecoveryReadResult struct {
	ID         string              `json:"id"`
	Kind       string              `json:"kind"`
	Inspection *RecoveryInspection `json:"inspection,omitempty"`
	Samples    *VerificationPage   `json:"samples,omitempty"`
}
type RecoveryReadChunkParams struct {
	ID     string `json:"id"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type RecoveryReadChunk struct {
	ID     string `json:"id"`
	Offset int    `json:"offset"`
	Next   int    `json:"next_offset"`
	Total  int    `json:"total"`
	SHA256 string `json:"sha256"`
	JSON   []byte `json:"json_base64"`
}
type recoveryReadJob struct {
	work     RecoveryReadWork
	store    *recoveryStore
	ctx      context.Context
	cancel   context.CancelFunc
	readerID uint64
	result   json.RawMessage // Immutable, encoded off the editor lock; never exposed directly.
}

func cloneRecoveryReadWork(in RecoveryReadWork) RecoveryReadWork {
	out := in
	if in.Input.Recording != nil {
		v := *in.Input.Recording
		out.Input.Recording = &v
	}
	if in.Error != nil {
		e := *in.Error
		out.Error = &e
	}
	return out
}
func (s *Service) recoveryStateLocked() RecoveryState {
	out := cloneRecovery(s.recovery)
	for _, id := range s.recoveryReadOrder {
		out.Reads = append(out.Reads, cloneRecoveryReadWork(s.recoveryReadJobs[id].work))
	}
	return out
}
func (s *Service) startRecoveryReadLocked(raw json.RawMessage) error {
	var p RecoveryReadParams
	if err := params(raw, &p, "kind", "id", "sha256"); err != nil {
		return err
	}
	if p.ID == "" || len(p.ID) > 128 || len(p.SHA256) != 64 {
		return fmt.Errorf("choose a recovery ID up to 128 bytes and a 64-character SHA-256")
	}
	if _, err := hex.DecodeString(p.SHA256); err != nil {
		return fmt.Errorf("SHA-256 must be hexadecimal")
	}
	if p.Kind != "inspection" && p.Kind != "samples" {
		return fmt.Errorf("kind must be inspection or samples")
	}
	if p.Kind == "inspection" {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		for _, name := range []string{"recording_index", "offset", "limit"} {
			if _, ok := fields[name]; ok {
				return fmt.Errorf("%s applies only to sample reads", name)
			}
		}
	} else {
		if p.Recording == nil || *p.Recording < 0 || p.Offset < 0 || p.Limit < 0 || p.Limit > 200 {
			return fmt.Errorf("samples require recording_index >=0, offset >=0 and limit 0 (100) or 1–200")
		}
	}
	if s.recoveryStore == nil {
		return &actionError{"unavailable", "Configure recovery first"}
	}
	if len(s.recoveryReaders) >= MaxRecoveryReaders {
		return &actionError{"busy", "Four recovery reads are active; finish or cancel one first"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.recoveryReadSerial++
	serial := s.recoveryReadSerial
	job := &recoveryReadJob{ctx: ctx, cancel: cancel, readerID: serial, store: s.recoveryStore, work: RecoveryReadWork{ID: fmt.Sprintf("recovery-read-%d", serial), Input: p, Directory: s.recoveryStore.directory, Status: "queued", Phase: "queued", InputRevision: s.revision, CommandRevision: s.commandRevision + 1, DocumentGeneration: s.documentGeneration}}
	if s.recoveryReaders == nil {
		s.recoveryReaders = map[uint64]context.CancelFunc{}
	}
	if s.recoveryReadJobs == nil {
		s.recoveryReadJobs = map[string]*recoveryReadJob{}
	}
	s.recoveryReaders[serial] = cancel
	s.recoveryReadJobs[job.work.ID] = job
	s.recoveryReadOrder = append(s.recoveryReadOrder, job.work.ID)
	for len(s.recoveryReadOrder) > MaxRecoveryReadJobs {
		for i, id := range s.recoveryReadOrder {
			if !operationActive(s.recoveryReadJobs[id].work.Status) {
				delete(s.recoveryReadJobs, id)
				s.recoveryReadOrder = append(s.recoveryReadOrder[:i], s.recoveryReadOrder[i+1:]...)
				break
			}
		}
	}
	s.liveWorkers.Add(1)
	read := s.recoveryQueryRead
	go s.runRecoveryRead(job, read)
	return nil
}
func (s *Service) runRecoveryRead(job *recoveryReadJob, read recoveryRestoreFunc) {
	defer s.liveWorkers.Done()
	defer job.cancel()
	progress := func(phase string, bytes int) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if job.ctx.Err() != nil || !operationActive(job.work.Status) {
			return
		}
		job.work.Status, job.work.Phase, job.work.Bytes = "running", phase, bytes
		s.revision++
		s.signalLocked("recovery", "jobs")
	}
	p := job.work.Input
	choice := RecoveryChoiceParams{ID: p.ID, SHA256: p.SHA256}
	var samples *RecoverySamplesParams
	if p.Kind == "samples" {
		samples = &RecoverySamplesParams{ID: p.ID, SHA256: p.SHA256, Recording: *p.Recording, Offset: p.Offset, Limit: p.Limit}
	}
	progress("reading-recovery", 0)
	value, err := projectRecoveryRead(job.ctx, job.store, choice, samples, read, progress)
	result := RecoveryReadResult{ID: job.work.ID, Kind: p.Kind}
	var encoded []byte
	var digest string
	if err == nil {
		progress("preparing-result", job.work.Bytes)
		if p.Kind == "inspection" {
			v := value.(RecoveryInspection)
			result.Inspection = &v
		} else {
			v := value.(VerificationPage)
			result.Samples = &v
		}
		encoded, err = json.Marshal(result)
		if err == nil && len(encoded) > MaxRecoveryReadResultBytes {
			err = &actionError{"result-limit", "Recovery read result exceeds its response budget"}
		}
		if err == nil {
			sum := sha256.Sum256(encoded)
			digest = hex.EncodeToString(sum[:])
		}
	}
	if job.ctx.Err() != nil {
		err = job.ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.ctx.Err() != nil {
		err = job.ctx.Err()
	}
	delete(s.recoveryReaders, job.readerID)
	if err != nil {
		code := "invalid-params"
		var action *actionError
		switch {
		case errors.As(err, &action):
			code = action.code
		case errors.Is(err, context.Canceled):
			code = "canceled"
		case errors.Is(err, context.DeadlineExceeded):
			code = "timeout"
		}
		job.work.Status = "failed"
		if code == "canceled" {
			job.work.Status = "canceled"
		}
		job.work.Error = &Failure{Code: code, Message: err.Error()}
	} else {
		job.work.Status, job.work.Phase = "completed", "completed"
		job.result = encoded
		job.work.ResultBytes = len(encoded)
		job.work.ResultSHA256 = digest
	}
	s.revision++
	s.signalLocked("recovery", "jobs")
}
func (s *Service) cancelRecoveryReadLocked(raw json.RawMessage) error {
	var p IDParams
	if err := params(raw, &p, "id"); err != nil {
		return err
	}
	job := s.recoveryReadJobs[p.ID]
	if job == nil {
		return &actionError{"not-found", "Recovery read expired or is unknown"}
	}
	if !operationActive(job.work.Status) {
		return &actionError{"unavailable", "Recovery read is already terminal"}
	}
	job.work.Status = "canceling"
	job.cancel()
	return nil
}

// The immutable encoded result remains alive through eviction; decoding owns all
// response arrays/pointers and runs outside the editor lock, without filesystem IO.
func (s *Service) recoveryReadQueryLocked(ctx context.Context, command string, raw json.RawMessage) (any, error) {
	var p IDParams
	var chunk RecoveryReadChunkParams
	var err error
	if command == "recovery.read.chunk" {
		err = params(raw, &chunk, "id")
		p.ID = chunk.ID
	} else {
		err = params(raw, &p, "id")
	}
	if err != nil {
		return nil, err
	}
	job := s.recoveryReadJobs[p.ID]
	if job == nil {
		return nil, &actionError{"not-found", "Recovery read expired or is unknown"}
	}
	if command == "recovery.read.inspect" {
		return cloneRecoveryReadWork(job.work), nil
	}
	if job.work.Status != "completed" || len(job.result) == 0 {
		return nil, &actionError{"not-ready", "Recovery read has no complete result"}
	}
	data := job.result
	if command == "recovery.read.chunk" {
		if chunk.Limit == 0 {
			chunk.Limit = 65536
		}
		if chunk.Offset < 0 || chunk.Offset > len(data) || chunk.Limit < 1 || chunk.Limit > 65536 {
			return nil, &actionError{"invalid-params", "Result chunks require offset 0–total and limit 1–65536 (zero defaults)"}
		}
		end := min(len(data), chunk.Offset+chunk.Limit)
		return RecoveryReadChunk{ID: p.ID, Offset: chunk.Offset, Next: end, Total: len(data), SHA256: job.work.ResultSHA256, JSON: append([]byte{}, data[chunk.Offset:end]...)}, nil
	}
	if len(data) > MaxRecoveryReadResponseBytes {
		return nil, &actionError{"result-too-large", "Use recovery.read.chunk for this complete result"}
	}
	s.mu.Unlock()
	defer s.mu.Lock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result RecoveryReadResult
	err = json.Unmarshal(data, &result)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return result, err
}
