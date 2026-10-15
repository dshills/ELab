package workbench

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func recoveryQueryFixture(t *testing.T) (*Service, RecoveryChoiceParams, VerificationPage, string) {
	t.Helper()
	source := pausedRC(t)
	defer source.Close()
	duration := .01
	_, record := verify(t, source, VerifyParams{Duration: &duration}, "record")
	page := call(t, source, "operation.samples", map[string]any{"id": record.Operation.ID, "offset": 1, "limit": 2}, "page").Result.(VerificationPage)
	directory := recoverySetupDirectory(t)
	configureRecovery(t, source, directory, false)
	call(t, source, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, source)
	source.Close()
	target := pausedRC(t)
	configureRecovery(t, target, directory, false)
	return target, choice, page, directory
}
func blockRecoveryQuery(t *testing.T, s *Service) (<-chan *recoveryStore, func()) {
	t.Helper()
	entered := make(chan *recoveryStore, MaxRecoveryReaders)
	release := make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.recoveryQueryRead = func(ctx context.Context, store *recoveryStore, p RecoveryChoiceParams, progress func(string, int)) (recoveryCapsule, Document, string, error) {
		entered <- store
		<-release
		return prepareRecoveryRestore(ctx, store, p, progress)
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func recoveryQueryRequest(t *testing.T, s *Service, ctx context.Context, command string, p any) <-chan Response {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan Response, 1)
	go func() {
		done <- s.Dispatch(ctx, Request{Version: 1, ID: command, Command: command, Params: raw}, "reader")
	}()
	return done
}
func awaitRecoveryQuery(t *testing.T, done <-chan Response) Response {
	t.Helper()
	select {
	case response := <-done:
		return response
	case <-time.After(time.Second):
		t.Fatal("recovery query did not finish")
		return Response{}
	}
}
func TestRecoveryQueriesAllowEditingAndReturnCapturedFileData(t *testing.T) {
	for _, command := range []string{"recovery.inspect", "recovery.samples"} {
		t.Run(command, func(t *testing.T) {
			s, choice, want, directory := recoveryQueryFixture(t)
			defer s.Close()
			entered, release := blockRecoveryQuery(t, s)
			defer release()
			var p any = choice
			if command == "recovery.samples" {
				p = RecoverySamplesParams{ID: choice.ID, SHA256: choice.SHA256, Recording: 0, Offset: 1, Limit: 2}
			}
			done := recoveryQueryRequest(t, s, context.Background(), command, p)
			<-entered
			promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 4, "dy": 8})
			call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
			promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
			before := s.Snapshot()
			release()
			response := awaitRecoveryQuery(t, done)
			if response.Error != nil {
				t.Fatal(response.Error)
			}
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("read changes later editor/physics state")
			}
			if command == "recovery.inspect" {
				result := response.Result.(RecoveryInspection)
				if result.Directory != directory || result.Candidate.SHA256 != choice.SHA256 || len(result.Document.Recordings) != 1 {
					t.Fatal("inspection capture identity missing")
				}
				component, _ := result.Document.Graph.Component("R1")
				if component.Value != 10000 {
					t.Fatal("inspection substitutes newer live graph", component.Value)
				}
			} else {
				page := response.Result.(VerificationPage)
				if page.ID != choice.ID || !reflect.DeepEqual(page.Rows, want.Rows) || !reflect.DeepEqual(page.Probes, want.Probes) {
					t.Fatal("sample query substitutes live rows")
				}
			}
		})
	}
}
func TestRecoveryQueryBoundedReadersAndCancellationArePure(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	before := s.Snapshot()
	entered, release := blockRecoveryQuery(t, s)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	readers := []<-chan Response{}
	for i := 0; i < MaxRecoveryReaders; i++ {
		readers = append(readers, recoveryQueryRequest(t, s, ctx, "recovery.inspect", choice))
		<-entered
	}
	refused := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice))
	if refused.Error == nil || refused.Error.Code != "busy" {
		t.Fatal("unbounded recovery reader admission", refused.Error)
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("reader bookkeeping publishes mutations")
	}
	cancel()
	release()
	for _, reader := range readers {
		response := awaitRecoveryQuery(t, reader)
		if response.Error == nil || response.Error.Code != "canceled" {
			t.Fatal("query cancellation omitted", response.Error)
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("canceled read changes state")
	}
	s.mu.Lock()
	count := len(s.recoveryReaders)
	s.recoveryQueryRead = prepareRecoveryRestore
	s.mu.Unlock()
	if count != 0 {
		t.Fatal("reader slots not released")
	}
	if r := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice)); r.Error != nil {
		t.Fatal("canceled readers exhausted later reads", r.Error)
	}
}
func TestRecoveryQueryCapturesStoreAcrossReconfiguration(t *testing.T) {
	s, choice, _, directory := recoveryQueryFixture(t)
	defer s.Close()
	entered, release := blockRecoveryQuery(t, s)
	defer release()
	done := recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice)
	captured := <-entered
	newer := filepath.Join(t.TempDir(), "new-recovery")
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: newer, Enabled: false, Interval: 2}, "new-directory")
	before := s.Snapshot()
	release()
	response := awaitRecoveryQuery(t, done)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	result := response.Result.(RecoveryInspection)
	if result.Directory != directory || result.Directory != captured.directory || s.Snapshot().Recovery.Directory != newer || result.Candidate.SHA256 != choice.SHA256 {
		t.Fatal("query mixed recovery-store identity")
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("late read overwrites current configuration")
	}
}
func TestRecoveryCloseCancelsAndJoinsReadWithoutLateMutation(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	entered, release := blockRecoveryQuery(t, s)
	defer release()
	query := recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice)
	<-entered
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Status != "closed" {
		if time.Now().After(deadline) {
			t.Fatal("close did not cancel read")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-closed:
		t.Fatal("close did not join active query")
	default:
	}
	before := s.Snapshot()
	release()
	response := awaitRecoveryQuery(t, query)
	if response.Error == nil || response.Error.Code != "canceled" {
		t.Fatal("close did not cancel query result", response.Error)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not drain read")
	}
	after := s.Snapshot()
	// Shutdown publishes its own progress while the canceled reader drains.
	before.Revision, before.Shutdown = after.Revision, after.Shutdown
	if !reflect.DeepEqual(before, after) {
		t.Fatal("late canceled read mutates closed circuit/catalog state")
	}
	refused := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice))
	if refused.Error == nil || refused.Error.Code != "unavailable" {
		t.Fatal("closed service accepts new filesystem readers")
	}
}
func TestRecoveryQueryFailuresRemainPureAndReleaseSlots(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	before := s.Snapshot()
	s.mu.Lock()
	s.recoveryQueryRead = func(context.Context, *recoveryStore, RecoveryChoiceParams, func(string, int)) (recoveryCapsule, Document, string, error) {
		return recoveryCapsule{}, Document{}, "", context.DeadlineExceeded
	}
	s.mu.Unlock()
	response := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice))
	if response.Error == nil || response.Error.Code != "timeout" || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("read timeout changes state or loses classification")
	}
	s.mu.Lock()
	count := len(s.recoveryReaders)
	s.recoveryQueryRead = prepareRecoveryRestore
	s.mu.Unlock()
	if count != 0 {
		t.Fatal("timeout leaks reader slot")
	}
	bad := choice
	bad.SHA256 = "old"
	response = awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", bad))
	if response.Error == nil || response.Error.Code != "file-conflict" || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("stale hash read changes state")
	}
	result := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice)).Result.(RecoveryInspection)
	result.Document.Recordings[0].Graph.Components[0].Value = 999
	again := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice)).Result.(RecoveryInspection)
	if again.Document.Recordings[0].Graph.Components[0].Value == 999 {
		t.Fatal("inspection aliases later captured metadata")
	}
}
