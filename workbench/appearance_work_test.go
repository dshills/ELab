package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestAppearanceAdmissionPublishesStableWorkIdentity(t *testing.T) {
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		close(entered)
		<-release
		return readAppearance(ctx, path)
	}
	r := call(t, s, "appearance.configure", AppearanceConfigureParams{Path: filepath.Join(t.TempDir(), "theme.json")}, "configure-with-job")
	<-entered
	data, err := json.Marshal(r.Result)
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Work struct {
			ID string `json:"id"`
		} `json:"appearance_work"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	if state.Work.ID == "" {
		t.Fatal("appearance configuration admitted background IO without an observable operation ID")
	}
	done()
	s.Close() // Join file IO before TempDir's later-registered cleanup runs.
}

// Command-revision guards allow independent background progress without
// mistaking a progress notification for a competing human/agent edit.
func appearanceCommand(t *testing.T, s *Service, command string, p any, key string) Response {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	rev := s.Snapshot().CommandRevision
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: key, Command: command, Params: raw, ExpectedCommandRevision: &rev, Key: key}, "appearance-test")
	if r.Error != nil {
		t.Fatalf("%s: %+v", command, r.Error)
	}
	return r
}

func inspectAppearanceWork(t *testing.T, s *Service, id string) *AppearanceWork {
	t.Helper()
	raw, _ := json.Marshal(IDParams{ID: id})
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "inspect", Command: "appearance.work.inspect", Params: raw}, "reader")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(*AppearanceWork)
}

func awaitAppearanceWork(t *testing.T, s *Service, id, status string) *AppearanceWork {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for {
		s.mu.Lock()
		changed := s.changed
		s.mu.Unlock()
		work := inspectAppearanceWork(t, s, id)
		if work.Status == status {
			return work
		}
		if !operationActive(work.Status) {
			t.Fatalf("work ended unexpectedly: %+v", work)
		}
		select {
		case <-changed:
		case <-deadline.C:
			t.Fatalf("work did not become %s: %+v", status, work)
		}
	}
}

func TestAppearanceWorkCapturesInputsArtifactAndOwnedRetry(t *testing.T) {
	s := New()
	defer s.Close()
	before := settled(t, s)
	path := filepath.Join(t.TempDir(), "appearance.json")
	raw, _ := json.Marshal(AppearanceConfigureParams{Path: path})
	req := Request{Version: 1, ID: "configure", Command: "appearance.configure", Params: raw, Key: "original", ExpectedCommandRevision: &before.CommandRevision}
	r := s.Dispatch(t.Context(), req, "owner")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	admitted := r.Result.(Snapshot).AppearanceWork
	if admitted == nil || admitted.ID == "" || admitted.Actor != "owner" || admitted.Path != path || admitted.Command != req.Command || admitted.InputRevision != before.Revision || admitted.CommandRevision != before.CommandRevision+1 || admitted.DocumentGeneration != before.Document.Generation || admitted.ElectricalRevision != before.ElectricalRevision || admitted.RunID != before.RunID || admitted.InputTheme != before.Theme || admitted.RequestedTheme != "" {
		t.Fatalf("missing captured input: %+v", admitted)
	}
	completed := awaitAppearanceWork(t, s, admitted.ID, "completed")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !completed.Published || !completed.CurrentAtCompletion || completed.Cancelable || completed.Bytes != len(data) || completed.ArtifactSHA256 != fmt.Sprintf("%x", sha256.Sum256(data)) || completed.ResultTheme != before.Theme || completed.SourceSHA256 != "" {
		t.Fatalf("missing publication outcome: %+v", completed)
	}
	admitted.Path = "poisoned snapshot"
	completed.ArtifactSHA256 = "poisoned query"
	duplicate := s.Dispatch(t.Context(), req, "owner")
	if duplicate.Error != nil || !duplicate.Duplicate || duplicate.Result.(Snapshot).AppearanceWork.Path != path || duplicate.Result.(Snapshot).AppearanceWork.Status != "queued" {
		t.Fatalf("retry lost admission: %+v", duplicate)
	}
	if inspectAppearanceWork(t, s, completed.ID).ArtifactSHA256 == completed.ArtifactSHA256 {
		t.Fatal("query shares service metadata")
	}
	after := s.Snapshot()
	if !reflect.DeepEqual(after.Document, before.Document) || after.ElectricalRevision != before.ElectricalRevision || after.RunID != before.RunID || after.CanUndo != before.CanUndo || after.CanRedo != before.CanRedo {
		t.Fatal("appearance work changed circuit/history")
	}
	noop := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": before.Theme}, "same-theme").Result.(Snapshot).AppearanceWork
	work := awaitAppearanceWork(t, s, noop.ID, "completed")
	if noop.ID == completed.ID || work.Published || work.SourceSHA256 != work.ArtifactSHA256 || work.ArtifactSHA256 == "" || work.RequestedTheme != before.Theme {
		t.Fatalf("no-op save outcome: %+v", work)
	}
	appearanceCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "replace-document")
	preserved := inspectAppearanceWork(t, s, completed.ID)
	if preserved.DocumentGeneration != before.Document.Generation || preserved.Actor != "owner" || preserved.CommandRevision != before.CommandRevision+1 {
		t.Fatal("document replacement rewrote captured appearance input")
	}
	pure := s.Snapshot()
	inspectAppearanceWork(t, s, completed.ID)
	if !reflect.DeepEqual(pure, s.Snapshot()) {
		t.Fatal("inspection mutated the service")
	}
}

func TestAppearanceWorkSupersededReadAndExactCancellation(t *testing.T) {
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		close(entered)
		<-release
		return readAppearance(ctx, path)
	}
	path := appearancePath(t, "dark")
	first := appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "first").Result.(Snapshot).AppearanceWork
	<-entered
	second := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "system"}, "second").Result.(Snapshot).AppearanceWork
	if old := inspectAppearanceWork(t, s, first.ID); old.Status != "canceling" || !old.Superseded || old.Cancelable {
		t.Fatalf("blocked superseded read: %+v", old)
	}
	before := s.Snapshot()
	refuseAppearanceWork(t, s, first.ID, "operation-unavailable")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("stale ID cancellation changed current work")
	}
	done()
	old := awaitAppearanceWork(t, s, first.ID, "superseded")
	latest := awaitAppearanceWork(t, s, second.ID, "completed")
	if old.Published || old.Error == nil || old.Error.Code != "superseded" || latest.ResultTheme != "system" || !latest.Published || savedAppearance(t, path) != "system" || s.Snapshot().Theme != "system" {
		t.Fatalf("coalesced load/save: old=%+v, latest=%+v", old, latest)
	}
}

func refuseAppearanceWork(t *testing.T, s *Service, id, code string) {
	t.Helper()
	raw, _ := json.Marshal(IDParams{ID: id})
	rev := s.Snapshot().CommandRevision
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "refuse", Command: "appearance.work.cancel", Params: raw, ExpectedCommandRevision: &rev, Key: "refuse-" + id}, "appearance-test")
	if r.Error == nil || r.Error.Code != code {
		t.Fatalf("expected %s: %+v", code, r)
	}
}

func TestAppearanceWorkCancelReadPreservesFileAndDoesNotCancelReplacement(t *testing.T) {
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	var reads atomic.Int32
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		if reads.Add(1) == 1 {
			close(entered)
			<-release
		}
		return readAppearance(ctx, path)
	}
	path := appearancePath(t, "dark")
	first := appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "first").Result.(Snapshot).AppearanceWork
	<-entered
	raw, _ := json.Marshal(IDParams{ID: first.ID})
	rev := s.Snapshot().CommandRevision
	req := Request{Version: 1, ID: "cancel", Command: "appearance.work.cancel", Params: raw, ExpectedCommandRevision: &rev, Key: "cancel-once"}
	r := s.Dispatch(t.Context(), req, "owner")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if work := inspectAppearanceWork(t, s, first.ID); work.Status != "canceling" || work.Cancelable || work.Superseded || s.Snapshot().Appearance.Status != "disabled" || s.Snapshot().Theme != "light" || savedAppearance(t, path) != "dark" {
		t.Fatalf("incorrect cancellation: %+v", work)
	}
	newPath := appearancePath(t, "system")
	second := appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: newPath}, "replacement").Result.(Snapshot).AppearanceWork
	awaitAppearanceWork(t, s, second.ID, "completed")
	if retry := s.Dispatch(t.Context(), req, "owner"); retry.Error != nil || !retry.Duplicate {
		t.Fatalf("cancel retry: %+v", retry)
	}
	if s.Snapshot().Appearance.Path != newPath || s.Snapshot().Appearance.Status != "ready" {
		t.Fatal("old retry canceled replacement")
	}
	done()
	if old := awaitAppearanceWork(t, s, first.ID, "canceled"); old.Published || old.Error == nil || old.Error.Code != "canceled" {
		t.Fatalf("canceled read outcome: %+v", old)
	}
}

func TestAppearanceWorkPublishingRetainsActualOutcomeAndCoalesces(t *testing.T) {
	s := New()
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	var publishes atomic.Int32
	s.appearancePublish = func(p *preparedFile) error {
		if publishes.Add(1) == 1 {
			close(entered)
			<-release
		}
		return publishPreparedFile(p)
	}
	path := appearancePath(t, "light")
	appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
	appearanceReady(t, s)
	first := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "dark").Result.(Snapshot).AppearanceWork
	<-entered
	before := s.Snapshot()
	refuseAppearanceWork(t, s, first.ID, "operation-committing")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("publication cancellation refusal changed state")
	}
	middle := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "light"}, "middle").Result.(Snapshot).AppearanceWork
	last := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "system"}, "last").Result.(Snapshot).AppearanceWork
	if work := inspectAppearanceWork(t, s, middle.ID); work.Status != "superseded" || work.Published || work.Cancelable {
		t.Fatalf("coalesced queued work: %+v", work)
	}
	if work := inspectAppearanceWork(t, s, first.ID); work.Status != "running" || work.Phase != "publishing" || work.Cancelable || !work.Superseded {
		t.Fatalf("publisher ended early: %+v", work)
	}
	done()
	old := awaitAppearanceWork(t, s, first.ID, "completed")
	latest := awaitAppearanceWork(t, s, last.ID, "completed")
	if !old.Published || old.CurrentAtCompletion || old.ResultTheme != "dark" || !latest.Published || !latest.CurrentAtCompletion || publishes.Load() != 2 || savedAppearance(t, path) != "system" {
		t.Fatalf("publication outcomes: old=%+v, latest=%+v", old, latest)
	}
}

func TestAppearanceWorkDetachedPublisherKeepsOutcomeWithoutChangingReplacement(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			s := New()
			entered, release := make(chan struct{}), make(chan struct{})
			done := appearanceRelease(t, s, release)
			var publishes atomic.Int32
			s.appearancePublish = func(p *preparedFile) error {
				if publishes.Add(1) == 1 {
					close(entered)
					<-release
					if fail {
						return fmt.Errorf("original publisher failed")
					}
				}
				return publishPreparedFile(p)
			}
			path := appearancePath(t, "light")
			appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "configure")
			appearanceReady(t, s)
			first := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "dark").Result.(Snapshot).AppearanceWork
			<-entered
			newPath := appearancePath(t, "system")
			second := appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: newPath}, "replace").Result.(Snapshot).AppearanceWork
			awaitAppearanceWork(t, s, second.ID, "completed")
			before := s.Snapshot()
			done()
			status := "completed"
			if fail {
				status = "failed"
			}
			old := awaitAppearanceWork(t, s, first.ID, status)
			after := s.Snapshot()
			if old.Published == fail || old.CurrentAtCompletion || !old.Superseded || after.Appearance != before.Appearance || after.Theme != "system" || after.AppearanceWork.ID != second.ID {
				t.Fatalf("detached publisher contaminated replacement: %+v", old)
			}
			if fail && (old.Error == nil || old.Error.Message != "original publisher failed") {
				t.Fatalf("detached failure lost: %+v", old)
			}
		})
	}
}

func TestAppearanceWorkFailureRetentionAndSessionOnlyChoices(t *testing.T) {
	s := New()
	defer s.Close()
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	first := appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: path}, "corrupt").Result.(Snapshot).AppearanceWork
	failed := awaitAppearanceWork(t, s, first.ID, "failed")
	if failed.Error == nil || failed.Published || failed.Cancelable || s.Snapshot().Appearance.Status != "error" {
		t.Fatalf("failure lost: %+v", failed)
	}
	failed.Error.Message = "poison"
	if inspectAppearanceWork(t, s, first.ID).Error.Message == "poison" {
		t.Fatal("failure pointer leaked")
	}
	choice := appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, "session-only").Result.(Snapshot)
	if choice.AppearanceWork.ID != first.ID || choice.AppearanceWork.CommandRevision == choice.CommandRevision || choice.Theme != "dark" {
		t.Fatal("failed store admitted a new write")
	}
	// A single active read is protected while terminal jobs turn over.
	entered, release := make(chan struct{}), make(chan struct{})
	done := appearanceRelease(t, s, release)
	s.appearanceRead = func(ctx context.Context, path string) (appearanceLoaded, error) {
		close(entered)
		<-release
		return readAppearance(ctx, path)
	}
	active := appearanceCommand(t, s, "appearance.configure", AppearanceConfigureParams{Path: appearancePath(t, "light")}, "held-read").Result.(Snapshot).AppearanceWork
	<-entered
	var latest *AppearanceWork
	for i := 0; i < MaxAppearanceOperations+4; i++ {
		latest = appearanceCommand(t, s, "appearance.theme.set", map[string]string{"theme": "dark"}, fmt.Sprint("choice-", i)).Result.(Snapshot).AppearanceWork
	}
	if old := inspectAppearanceWork(t, s, active.ID); old.Status != "canceling" {
		t.Fatal("active canceled reader evicted")
	}
	s.mu.Lock()
	retained, evicted := len(s.appearanceJobs), s.appearanceJobs[first.ID] == nil
	s.mu.Unlock()
	if retained != MaxAppearanceOperations || !evicted {
		t.Fatalf("unbounded retention: %d, evicted=%v", retained, evicted)
	}
	done()
	awaitAppearanceWork(t, s, latest.ID, "completed")
	s.Close()
	if work := inspectAppearanceWork(t, s, latest.ID); work.Status != "completed" || !s.Snapshot().Shutdown.WorkersJoined {
		t.Fatalf("close lost work: %+v", work)
	}
}

func TestAppearanceWorkDiscoveryAndStrictInspection(t *testing.T) {
	s := New()
	defer s.Close()
	for _, command := range []string{"appearance.work.inspect", "appearance.work.cancel"} {
		schema := navigationSchema(t, command)
		if schema["x-command-contract"] == nil || schema["x-appearance-work"] == nil {
			t.Fatal("missing discovery contract", command)
		}
		for i, body := range []string{`{}`, `{"id":null}`, `{"id":1}`, `{"id":"unknown","extra":true}`} {
			rev := s.Snapshot().CommandRevision
			r := s.Dispatch(t.Context(), Request{Version: 1, ID: "bad", Command: command, Params: json.RawMessage(body), Key: fmt.Sprint(command, i), ExpectedCommandRevision: &rev}, "test")
			if r.Error == nil || r.Error.Code != "invalid-params" {
				t.Fatalf("%s %s: %+v", command, body, r)
			}
		}
		for _, id := range []string{"", "unknown"} {
			raw, _ := json.Marshal(IDParams{ID: id})
			rev := s.Snapshot().CommandRevision
			r := s.Dispatch(t.Context(), Request{Version: 1, ID: "missing", Command: command, Params: raw, Key: command + id, ExpectedCommandRevision: &rev}, "test")
			if r.Error == nil || r.Error.Code != "operation-unavailable" {
				t.Fatalf("%s missing ID: %+v", command, r)
			}
		}
	}
}
