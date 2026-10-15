package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func loadFixture(t *testing.T) string {
	t.Helper()
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	path := filepath.Join(t.TempDir(), "LED.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	return path
}
func blockProjectLoader(t *testing.T, s *Service, fail bool) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.projectLoad = func(ctx context.Context, path string, progress func(string, int)) (Document, [32]byte, error) {
		progress("reading", 0)
		close(entered)
		<-release
		if fail {
			return Document{}, [32]byte{}, fmt.Errorf("controlled late read failure")
		}
		return prepareProjectLoad(ctx, path, progress)
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func awaitProjectStatus(t *testing.T, s *Service, status string) Snapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		state := s.Snapshot()
		if state.ProjectWork != nil && state.ProjectWork.Status == status {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatalf("open did not reach %s: %+v", status, state.ProjectWork)
		}
		time.Sleep(time.Millisecond)
	}
}
func TestBlockedProjectLoadNavigationAndCancelPreserveExistingDocument(t *testing.T) {
	path := loadFixture(t)
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	entered, release := blockProjectLoader(t, s, false)
	defer release()
	admitted := promptLiveCommand(t, s, "project.load.start", LoadParams{Path: path, DiscardChanges: true}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	if admitted.ProjectWork == nil || admitted.ProjectWork.Status != "queued" || admitted.ProjectWork.Actor != "test" || admitted.Document != before.Document || !reflect.DeepEqual(admitted.Components, before.Components) {
		t.Fatal("open admission partially replaced document")
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "drawing.cancel", struct{}{})
	refuseStepWork(t, s, "project.load.start", LoadParams{Path: path, DiscardChanges: true}, "busy")
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: "old"}, "operation-unavailable")
	promptLiveCommand(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID})
	release()
	after := awaitProjectStatus(t, s, "canceled")
	if after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) || after.View == before.View {
		t.Fatal("canceled open replaced current document or lost navigation")
	}
}
func TestProjectLoadSuccessfulCommitAllowsNavigationAndAddsOriginalActorUndo(t *testing.T) {
	path := loadFixture(t)
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	entered, release := blockProjectLoader(t, s, false)
	defer release()
	admitted := promptLiveCommand(t, s, "project.load.start", LoadParams{Path: path, DiscardChanges: true}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	release()
	after := awaitProjectStatus(t, s, "completed")
	settled(t, s)
	if after.Document.Path != path || after.Document.Dirty || after.Document.Generation != before.Document.Generation+1 || !after.Paused || after.UndoEntry.Actor != "test" || after.UndoEntry.CommandRevision != admitted.ProjectWork.CommandRevision || after.ProjectWork.ResultRun != after.RunID {
		t.Fatal("open commit lacks fresh saved state/original attribution")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if !reflect.DeepEqual(s.Snapshot().Components, before.Components) || s.Snapshot().Document.Path != path || !s.Snapshot().Document.Dirty {
		t.Fatal("load undo fails to restore previous drawing with loaded path still current")
	}
}
func TestProjectLoadSupersededByEditsReplacementSaveAndNewRecording(t *testing.T) {
	for _, command := range []string{"component.move", "component.set", "display.preferences.set", "example.reset", "project.save", "simulation.verify"} {
		t.Run(command, func(t *testing.T) {
			path := loadFixture(t)
			s := pausedRC(t)
			defer s.Close()
			entered, release := blockProjectLoader(t, s, true)
			defer release()
			promptLiveCommand(t, s, "project.load.start", LoadParams{Path: path, DiscardChanges: true})
			awaitLiveSampleBlock(t, entered)
			s.mu.Lock()
			job := s.projectJob
			s.mu.Unlock()
			var p any
			switch command {
			case "component.move":
				p = MoveParams{ID: "R1", Position: Point{300, -200}}
			case "component.set":
				p = ValueParams{ID: "R1", Value: "20k"}
			case "display.preferences.set":
				v := DefaultDisplayPreferences()
				v.VoltageColor = true
				p = v
			case "example.reset":
				p = ReplaceParams{DiscardChanges: true}
			case "project.save":
				p = SaveParams{Path: filepath.Join(t.TempDir(), "edited.elab")}
			case "simulation.verify":
				d := .001
				p = VerifyParams{Duration: &d}
			}
			promptLiveCommand(t, s, command, p)
			before := s.Snapshot()
			release()
			select {
			case <-job.done:
			case <-time.After(5 * time.Second):
				t.Fatal("old open did not settle")
			}
			s.mu.Lock()
			oldWork := cloneProjectWork(&job.work)
			s.mu.Unlock()
			after := s.Snapshot()
			if after.Document.Generation != before.Document.Generation || !reflect.DeepEqual(after.Components, before.Components) || after.Document.Path != before.Document.Path || oldWork.Status != "superseded" || oldWork.Error.Code != "superseded" {
				t.Fatal("late open/failure overwrites intervening work")
			}
		})
	}
}
func TestLegacyProjectLoadWaitDisconnectAndRetryRetainOneOutcome(t *testing.T) {
	path := loadFixture(t)
	s := pausedRC(t)
	defer s.Close()
	entered, release := blockProjectLoader(t, s, false)
	defer release()
	raw, _ := json.Marshal(LoadParams{Path: path, DiscardChanges: true})
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "load", Command: "project.load", Params: raw, ExpectedCommandRevision: &rev, Key: "original"}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(ctx, request, "loader") }()
	awaitLiveSampleBlock(t, entered)
	cancel()
	select {
	case r := <-done:
		if r.Error == nil || r.Error.Code != "canceled" {
			t.Fatal("disconnected wait lacks honest response")
		}
	case <-time.After(time.Second):
		t.Fatal("request wait did not cancel")
	}
	if !operationActive(s.Snapshot().ProjectWork.Status) {
		t.Fatal("disconnect canceled admitted open")
	}
	release()
	after := awaitProjectStatus(t, s, "completed")
	retry := s.Dispatch(context.Background(), request, "loader")
	if !retry.Duplicate || retry.Error != nil || retry.Result.(Snapshot).ProjectWork.ID != after.ProjectWork.ID {
		t.Fatal("retry did not return original terminal outcome")
	}
	retry.Result.(Snapshot).ProjectWork.Status = "changed"
	if s.Snapshot().ProjectWork.Status != "completed" {
		t.Fatal("retry aliases work")
	}
}

func TestImmediateProjectLoadRetryAndClosePreserveAdmissionOwnership(t *testing.T) {
	path := loadFixture(t)
	s := pausedRC(t)
	defer s.Close()
	entered, release := blockProjectLoader(t, s, false)
	defer release()
	raw, _ := json.Marshal(LoadParams{Path: path, DiscardChanges: true})
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "start", Command: "project.load.start", Params: raw, ExpectedCommandRevision: &rev, Key: "start-original"}
	admitted := s.Dispatch(context.Background(), request, "test")
	if admitted.Error != nil {
		t.Fatal(admitted.Error)
	}
	awaitLiveSampleBlock(t, entered)
	work := admitted.Result.(Snapshot).ProjectWork
	work.Path = "mutated"
	work.Status = "mutated"
	request.ID = "retry"
	retry := s.Dispatch(context.Background(), request, "test")
	if !retry.Duplicate || retry.Error != nil || retry.Result.(Snapshot).ProjectWork.Status != "queued" || retry.Result.(Snapshot).ProjectWork.Path != path {
		t.Fatal("immediate retry corrupts admission or starts new work")
	}
	before := s.Snapshot()
	closed := promptLiveCommand(t, s, "app.close.start", ReplaceParams{DiscardChanges: true}).Result.(Snapshot)
	if !closed.Closed || closed.ProjectWork.Status != "canceling" {
		t.Fatal("close does not cancel pending file preparation")
	}
	release()
	after := awaitProjectStatus(t, s, "canceled")
	if !after.Closed || after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("late file read installs into closed session")
	}
}
