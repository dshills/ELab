package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"
)

func blockRecoveryRestore(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.recoveryRestore = func(ctx context.Context, store *recoveryStore, p RecoveryChoiceParams, progress func(string, int)) (recoveryCapsule, Document, string, error) {
		capsule, document, hash, err := prepareRecoveryRestore(ctx, store, p, progress)
		close(entered)
		<-release
		return capsule, document, hash, err
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}

func TestRecoveryRestoreWaitingDisconnectRetryPreservesOneCommit(t *testing.T) {
	s := manualRecoveryService(t)
	defer s.Close()
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	choice.DiscardChanges = true
	entered, release := blockRecoveryRestore(t, s)
	defer release()
	raw, _ := json.Marshal(choice)
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "restore", Command: "recovery.restore", Params: raw, ExpectedCommandRevision: &rev, Key: "restore-original"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(ctx, request, "restorer") }()
	awaitLiveSampleBlock(t, entered)
	cancel()
	select {
	case response := <-done:
		if response.Error == nil || response.Error.Code != "canceled" {
			t.Fatal("disconnected restore wait lacks cancellation result")
		}
	case <-time.After(time.Second):
		t.Fatal("waiting request did not disconnect")
	}
	release()
	after := awaitProjectStatus(t, s, "completed")
	retry := s.Dispatch(context.Background(), request, "restorer")
	if !retry.Duplicate || retry.Error != nil || retry.Result.(Snapshot).ProjectWork.ID != after.ProjectWork.ID || s.Snapshot().Document.Generation != after.Document.Generation {
		t.Fatal("retry loses original result or repeats installation")
	}
	retry.Result.(Snapshot).ProjectWork.Status = "mutated"
	if s.Snapshot().ProjectWork.Status != "completed" {
		t.Fatal("retry aliases retained job")
	}
}

func TestBlockedRecoveryRestoreDiscardsAfterCancellationEditOrReplacement(t *testing.T) {
	for _, command := range []string{"project.work.cancel", "component.set", "project.clear"} {
		t.Run(command, func(t *testing.T) {
			s := manualRecoveryService(t)
			defer s.Close()
			call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
			choice := recoveryChoice(t, s)
			choice.DiscardChanges = true
			entered, release := blockRecoveryRestore(t, s)
			defer release()
			before := s.Snapshot()
			admitted := promptLiveCommand(t, s, "recovery.restore.start", choice).Result.(Snapshot)
			awaitLiveSampleBlock(t, entered)
			if admitted.Document != before.Document || admitted.RunID != before.RunID || !reflect.DeepEqual(admitted.UndoEntry, before.UndoEntry) {
				t.Fatal("restore admission installs staged data")
			}
			promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
			var p any
			status := "superseded"
			switch command {
			case "project.work.cancel":
				p, status = IDParams{ID: admitted.ProjectWork.ID}, "canceled"
			case "component.set":
				p = ValueParams{ID: "R1", Value: "30k"}
			case "project.clear":
				p = ReplaceParams{DiscardChanges: true}
			}
			promptLiveCommand(t, s, command, p)
			edited := s.Snapshot()
			release()
			after := awaitProjectStatus(t, s, status)
			if after.Document != edited.Document || after.RunID != edited.RunID || after.View != edited.View || !reflect.DeepEqual(after.Components, edited.Components) || !reflect.DeepEqual(after.UndoEntry, edited.UndoEntry) {
				t.Fatal("stale restore overrides newer project state")
			}
		})
	}
}

func TestBackgroundRecoveryRestoreCommitsOneNewUnassociatedCopy(t *testing.T) {
	s := manualRecoveryService(t)
	defer s.Close()
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	choice.DiscardChanges = true
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "30k"}, "edit")
	entered, release := blockRecoveryRestore(t, s)
	defer release()
	before := s.Snapshot()
	admitted := promptLiveCommand(t, s, "recovery.restore.start", choice).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.Document.Generation != before.Document.Generation+1 || after.Document.Path != "" || !after.Document.Dirty || after.Document.RecoveredFrom == "" || after.RunID != before.RunID+1 || !after.Paused || after.ProjectWork.SourceSHA256 != choice.SHA256 || after.ProjectWork.RecoveryID != choice.ID || after.ProjectWork.ResultGeneration != after.Document.Generation {
		t.Fatalf("restore result lacks fresh copy/provenance: %+v", after)
	}
	if after.UndoEntry == nil || after.UndoEntry.Actor != admitted.ProjectWork.Actor || after.UndoEntry.CommandRevision != admitted.ProjectWork.CommandRevision {
		t.Fatal("restore history loses original admitting actor/revision")
	}
	if inspectProjectWork(t, s, admitted.ProjectWork.ID).Status != "completed" {
		t.Fatal("stable job inspection loses completed restore")
	}
}
