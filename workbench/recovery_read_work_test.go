package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func readWork(t *testing.T, s *Service, id string) RecoveryReadWork {
	t.Helper()
	return call(t, s, "recovery.read.inspect", IDParams{ID: id}, "inspect-read").Result.(RecoveryReadWork)
}
func awaitReadWork(t *testing.T, s *Service, id, status string) RecoveryReadWork {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		w := readWork(t, s, id)
		if !operationActive(w.Status) {
			if w.Status != status {
				t.Fatalf("read status %+v", w)
			}
			return w
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("recovery read did not finish")
	return RecoveryReadWork{}
}
func startRead(t *testing.T, s *Service, p RecoveryReadParams, key string) (Response, string) {
	t.Helper()
	r := call(t, s, "recovery.read.start", p, key)
	reads := r.Result.(Snapshot).Recovery.Reads
	return r, reads[len(reads)-1].ID
}

func TestRecoveryReadJobsPreserveLaterEditsAndOwnResults(t *testing.T) {
	s, choice, want, _ := recoveryQueryFixture(t)
	defer s.Close()
	entered, release := blockRecoveryQuery(t, s)
	defer release()
	p := RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}
	accepted, id := startRead(t, s, p, "inspection")
	<-entered
	refuse(t, s, "recovery.read.result", IDParams{ID: id}, "not-ready")
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	// A blocked read must not monopolize project file work.
	path := filepath.Join(t.TempDir(), "live.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save-live")
	before := s.Snapshot()
	release()
	w := awaitReadWork(t, s, id, "completed")
	if w.Actor != "test" || w.Input.ID != choice.ID || w.Input.SHA256 != choice.SHA256 || w.ResultBytes == 0 || w.ResultSHA256 == "" {
		t.Fatal("captured read identity absent", w)
	}
	result := call(t, s, "recovery.read.result", IDParams{ID: id}, "result").Result.(RecoveryReadResult)
	part, _ := result.Inspection.Document.Graph.Component("R1")
	if part.Value != 10000 {
		t.Fatal("inspection substituted newer editor graph")
	}
	after := s.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || before.Recovery.Selected != after.Recovery.Selected {
		t.Fatal("read completion changes source/catalog/selection")
	}
	result.Inspection.Document.Graph.Components[0].Value = 999
	result.Inspection.Document.Recordings[0].Probes[0].Label = "corrupt"
	fresh := call(t, s, "recovery.read.result", IDParams{ID: id}, "owned").Result.(RecoveryReadResult)
	if fresh.Inspection.Document.Graph.Components[0].Value == 999 || fresh.Inspection.Document.Recordings[0].Probes[0].Label == "corrupt" {
		t.Fatal("result aliases retained data")
	}
	retry := call(t, s, "recovery.read.start", p, "inspection")
	if !retry.Duplicate || len(retry.Result.(Snapshot).Recovery.Reads) != len(accepted.Result.(Snapshot).Recovery.Reads) {
		t.Fatal("retry duplicated read")
	}
	i := 0
	_, samplesID := startRead(t, s, RecoveryReadParams{Kind: "samples", ID: choice.ID, SHA256: choice.SHA256, Recording: &i, Offset: 1, Limit: 2}, "samples")
	awaitReadWork(t, s, samplesID, "completed")
	page := call(t, s, "recovery.read.result", IDParams{ID: samplesID}, "rows").Result.(RecoveryReadResult).Samples
	want.ID = choice.ID
	if !reflect.DeepEqual(*page, want) {
		t.Fatal("observable sample page differs from exact source")
	}
	page.Rows[0].Values[0] = 999
	if got := call(t, s, "recovery.read.result", IDParams{ID: samplesID}, "owned-page").Result.(RecoveryReadResult).Samples; got.Rows[0].Values[0] == 999 {
		t.Fatal("sample result aliases client")
	}
}

func TestRecoveryReadJobsSharePureReadSlotsAndCancelWithoutPartialResult(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	entered, release := blockRecoveryQuery(t, s)
	defer release()
	var ids []string
	for i := 0; i < MaxRecoveryReaders; i++ {
		_, id := startRead(t, s, RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}, fmt.Sprint("read-", i))
		ids = append(ids, id)
		<-entered
	}
	refuse(t, s, "recovery.read.start", RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}, "busy")
	response := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice))
	if response.Error == nil || response.Error.Code != "busy" {
		t.Fatal("pure reads bypass shared pool")
	}
	call(t, s, "recovery.read.cancel", IDParams{ID: ids[0]}, "cancel")
	refuse(t, s, "recovery.read.start", RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}, "busy")
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 2, "dy": 3})
	release()
	awaitReadWork(t, s, ids[0], "canceled")
	// Refusal checks compare complete snapshots; let the other released readers
	// finish their legitimate publications before asserting query purity.
	for _, id := range ids[1:] {
		awaitReadWork(t, s, id, "completed")
	}
	refuse(t, s, "recovery.read.result", IDParams{ID: ids[0]}, "not-ready")
	refuse(t, s, "recovery.read.cancel", IDParams{ID: ids[0]}, "unavailable")
}

func TestRecoveryReadResultsRemainCapturedAcrossReconfigurationAndEviction(t *testing.T) {
	s, choice, _, directory := recoveryQueryFixture(t)
	defer s.Close()
	entered, release := blockRecoveryQuery(t, s)
	defer release()
	_, id := startRead(t, s, RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}, "read")
	<-entered
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: recoverySetupDirectory(t), Interval: 1}, "configure-other")
	newCatalog := s.Snapshot().Recovery
	release()
	awaitReadWork(t, s, id, "completed")
	s.mu.Lock()
	s.recoveryQueryRead = prepareRecoveryRestore
	s.mu.Unlock()
	if got := s.Snapshot().Recovery; got.Directory != newCatalog.Directory || got.Selected != newCatalog.Selected || !reflect.DeepEqual(got.Candidates, newCatalog.Candidates) {
		t.Fatal("old read published into new catalog")
	}
	result := call(t, s, "recovery.read.result", IDParams{ID: id}, "captured").Result.(RecoveryReadResult)
	if result.Inspection.Directory != directory {
		t.Fatal("completed result changed with configuration")
	}
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Interval: 1}, "configure-back")
	for i := 0; i < MaxRecoveryReadJobs; i++ {
		_, next := startRead(t, s, RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}, fmt.Sprint("later-", i))
		awaitReadWork(t, s, next, "completed")
	}
	refuse(t, s, "recovery.read.inspect", IDParams{ID: id}, "not-found")
	refuse(t, s, "recovery.read.result", IDParams{ID: id}, "not-found")
	if len(s.Snapshot().Recovery.Reads) != 8 {
		t.Fatal("retained job inventory unbounded")
	}
}

func TestRecoveryReadFailuresPublishNoPartialResults(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	p := RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: strings.Repeat("0", 64)}
	_, id := startRead(t, s, p, "wrong-hash")
	w := awaitReadWork(t, s, id, "failed")
	if w.Error == nil || w.Error.Code != "file-conflict" || w.ResultBytes != 0 || w.ResultSHA256 != "" {
		t.Fatal("hash failure exposed a partial result", w)
	}
	refuse(t, s, "recovery.read.chunk", RecoveryReadChunkParams{ID: id}, "not-ready")
	s.mu.Lock()
	s.recoveryQueryRead = func(context.Context, *recoveryStore, RecoveryChoiceParams, func(string, int)) (recoveryCapsule, Document, string, error) {
		return recoveryCapsule{}, Document{}, "", context.DeadlineExceeded
	}
	s.mu.Unlock()
	p.SHA256 = choice.SHA256
	_, id = startRead(t, s, p, "deadline")
	w = awaitReadWork(t, s, id, "failed")
	if w.Error == nil || w.Error.Code != "timeout" || w.ResultBytes != 0 {
		t.Fatal("deadline was not classified", w)
	}
}

func TestRecoveryReadChunkOwnsBytesAndSupportsLargeCompleteResults(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	_, id := startRead(t, s, RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}, "read")
	awaitReadWork(t, s, id, "completed")
	// A large retained result exercises transport chunking independently of
	// component-count fixtures; no disk read occurs during result delivery.
	data := []byte(strings.Repeat("x", MaxRecoveryReadResponseBytes+1))
	sum := sha256.Sum256(data)
	s.mu.Lock()
	job := s.recoveryReadJobs[id]
	job.result = data
	job.work.ResultBytes = len(data)
	job.work.ResultSHA256 = hex.EncodeToString(sum[:])
	s.mu.Unlock()
	refuse(t, s, "recovery.read.result", IDParams{ID: id}, "result-too-large")
	var collected []byte
	for len(collected) < len(data) {
		chunk := call(t, s, "recovery.read.chunk", RecoveryReadChunkParams{ID: id, Offset: len(collected)}, "chunk").Result.(RecoveryReadChunk)
		if chunk.Offset != len(collected) || chunk.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatal("chunk identity differs")
		}
		collected = append(collected, chunk.JSON...)
		chunk.JSON[0] = 'z'
	}
	if !bytes.Equal(data, collected) {
		t.Fatal("chunked result changed or aliased")
	}
	last := call(t, s, "recovery.read.chunk", RecoveryReadChunkParams{ID: id, Offset: len(data)}, "end").Result.(RecoveryReadChunk)
	if len(last.JSON) != 0 || last.Next != last.Total {
		t.Fatal("inclusive chunk end failed")
	}
}

func TestRecoveryReadCloseJoinsAndInputRefusalsArePure(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	for _, p := range []any{map[string]any{"kind": "inspection", "id": choice.ID, "sha256": choice.SHA256, "offset": 0}, RecoveryReadParams{Kind: "samples", ID: choice.ID, SHA256: choice.SHA256}, RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: "bad"}} {
		before := s.Snapshot()
		refuse(t, s, "recovery.read.start", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid read mutates state")
		}
	}
	entered, release := blockRecoveryQuery(t, s)
	defer release()
	_, id := startRead(t, s, RecoveryReadParams{Kind: "inspection", ID: choice.ID, SHA256: choice.SHA256}, "read")
	<-entered
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("Close failed to join read")
	case <-time.After(20 * time.Millisecond):
	}
	release()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not finish")
	}
	if readWork(t, s, id).Status != "canceled" {
		t.Fatal("Close did not cancel active read")
	}
	// Querying retained metadata after Close remains an in-memory operation.
	_, err := json.Marshal(s.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
}
