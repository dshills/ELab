package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestShutdownFinalizationReleasesEditorMutex(t *testing.T) {
	s := pausedRC(t)
	configureRecovery(t, s, recoverySetupDirectory(t), false)
	store := configuredStore(t, s)
	store.mu.Lock()
	var once sync.Once
	release := func() { once.Do(store.mu.Unlock) }
	defer func() { release(); s.Close() }()
	raw, _ := json.Marshal(ReplaceParams{DiscardChanges: true})
	commandRevision := s.Snapshot().CommandRevision
	closed := make(chan Response, 1)
	go func() {
		closed <- s.Dispatch(context.Background(), Request{Version: 1, ID: "close", Key: "close", Command: "app.close", ExpectedCommandRevision: &commandRevision, Params: raw}, "user")
	}()
	// Let the close request enter its finalization before querying. Store lock
	// release is controlled by this test; the observation must not depend on it.
	time.Sleep(20 * time.Millisecond)
	observed := make(chan Snapshot, 1)
	go func() { observed <- s.Snapshot() }()
	select {
	case state := <-observed:
		if !state.Closed {
			t.Fatal("close has not admitted while recovery serialization is blocked")
		}
	case <-time.After(time.Second):
		t.Fatal("recovery shutdown holds the editor mutex")
	}
	select {
	case <-closed:
		t.Fatal("waiting close returned before finalization")
	default:
	}
	release()
	select {
	case response := <-closed:
		if response.Error != nil {
			t.Fatal(response.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("waiting close did not finish")
	}
}

func awaitShutdownWork(t *testing.T, s *Service, status string) *ShutdownWork {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		work := s.Snapshot().Shutdown
		if work != nil && !operationActive(work.Status) {
			if work.Status != status {
				t.Fatal("shutdown outcome", work)
			}
			return work
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("shutdown did not finish")
	return nil
}

func TestShutdownImmediateAdmissionOwnershipAndFailureAcknowledgement(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	directory := recoverySetupDirectory(t)
	configureRecovery(t, s, directory, false)
	store := configuredStore(t, s)
	// A nonempty directory at this owned lease path makes cleanup fail without
	// changing any explicit saved project or another session's data.
	if err := os.Remove(store.leasePath()); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(store.leasePath(), 0700); err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(store.leasePath(), "do-not-remove")
	if err := os.WriteFile(protected, []byte("owned fault fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	refuse(t, s, "app.close.start", ReplaceParams{}, "unsaved-changes")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("refused close changed state")
	}
	admitted := call(t, s, "app.close.start", ReplaceParams{DiscardChanges: true}, "start").Result.(Snapshot)
	if !admitted.Closed || admitted.Shutdown == nil || admitted.Shutdown.Cancelable || admitted.Shutdown.Actor != "test" || admitted.Shutdown.InputRevision != before.Revision || admitted.Shutdown.CommandRevision != admitted.CommandRevision {
		t.Fatal("close admission lost input/actor identity")
	}
	work := awaitShutdownWork(t, s, "failed")
	if work.RecoveryFinalized || !work.WorkersJoined || work.ExitReady || work.Error == nil || work.Error.Code != "shutdown-failed" || work.RecoveryError == "" {
		t.Fatal("failed cleanup was silently acknowledged", work)
	}
	view := call(t, s, "app.close.inspect", IDParams{ID: work.ID}, "inspect").Result.(*ShutdownWork)
	view.Error.Message = "corrupted"
	if s.Snapshot().Shutdown.Error.Message == "corrupted" {
		t.Fatal("shutdown metadata aliases consumer")
	}
	refuse(t, s, "app.close.acknowledge", IDParams{ID: "wrong"}, "not-found")
	refuse(t, s, "view.pan", map[string]float64{"dx": 1}, "closed")
	ack := call(t, s, "app.close.acknowledge", IDParams{ID: work.ID}, "ack").Result.(Snapshot)
	if !ack.ReadyToExit() || ack.Shutdown.Status != "failed" || ack.Shutdown.Error == nil || before.Document != ack.Document || !reflect.DeepEqual(before.Components, ack.Components) || !reflect.DeepEqual(before.Transient, ack.Transient) {
		t.Fatal("acknowledgement hid failure or changed circuit")
	}
	if _, err := os.Stat(protected); err != nil {
		t.Fatal("acknowledgement retried destructive cleanup", err)
	}
	retry := call(t, s, "app.close.start", ReplaceParams{DiscardChanges: true}, "start")
	if !retry.Duplicate || retry.Result.(Snapshot).Shutdown.Status != "queued" {
		t.Fatal("start retry changed original admission")
	}
}

func TestShutdownWaitingDisconnectRetryAndActualWorkerJoining(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	s := newService(func(ctx context.Context, circuit Circuit) (Result, error) {
		close(entered)
		<-release // Deliberately uncooperative until the test releases it.
		return SolveLED(ctx, circuit)
	})
	defer func() { unblock(); s.Close() }()
	<-entered
	rev := s.Snapshot().CommandRevision
	raw, _ := json.Marshal(ReplaceParams{DiscardChanges: true})
	request := Request{Version: 1, ID: "waiting", Command: "app.close", Key: "waiting", ExpectedCommandRevision: &rev, Params: raw}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(ctx, request, "closer") }()
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Shutdown == nil {
		if time.Now().After(deadline) {
			t.Fatal("waiting close never admitted")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case r := <-done:
		if r.Error == nil || r.Error.Code != "canceled" {
			t.Fatal("disconnected wait lost accepted close")
		}
	case <-time.After(time.Second):
		t.Fatal("caller cancellation blocked behind worker")
	}
	state := s.Snapshot()
	if state.ReadyToExit() || state.Shutdown.WorkersJoined || state.Shutdown.Actor != "closer" {
		t.Fatal("unreturned DC solver was not joined")
	}
	before := s.Snapshot()
	refuse(t, s, "app.close.acknowledge", IDParams{ID: state.Shutdown.ID}, "unavailable")
	if before.CommandRevision != s.Snapshot().CommandRevision {
		t.Fatal("active-shutdown acknowledgement advanced command revision")
	}
	go func() { done <- s.Dispatch(context.Background(), request, "closer") }()
	unblock()
	select {
	case r := <-done:
		if !r.Duplicate || r.Error != nil || !r.Result.(Snapshot).ReadyToExit() || !r.Result.(Snapshot).Shutdown.WorkersJoined {
			t.Fatal("waiting retry did not join original close", r.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("retry did not finish after worker returned")
	}
}
