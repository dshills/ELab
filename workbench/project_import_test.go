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

func importFixture(t *testing.T) string {
	t.Helper()
	s := pausedRC(t)
	defer s.Close()
	path, _ := exportedRecording(t, s, .01, []string{"v:C1", "i:C1"})
	return path
}
func blockRecordingImport(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.projectImport = func(ctx context.Context, path string, progress func(string, int)) (*verificationRecord, [32]byte, error) {
		progress("reading", 0)
		close(entered)
		<-release
		return prepareRecordingImport(ctx, path, progress)
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func TestBlockedImportAllowsEditingAndSteppingThenAddsOnlyCompleteRecord(t *testing.T) {
	path := importFixture(t)
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	entered, release := blockRecordingImport(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	if admitted.Document != before.Document || len(admitted.Operations) != len(before.Operations) || admitted.Recording != nil {
		t.Fatal("import admission leaks staged record")
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	call(t, s, "simulation.step", StepParams{Duration: .005}, "step")
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	edited := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.ResultOperationID == "" || after.ProjectWork.SourceSHA256 == "" || after.ProjectWork.ViewerSelected == nil || !*after.ProjectWork.ViewerSelected || after.Recording.ID != after.ProjectWork.ResultOperationID || !after.Recording.Imported || after.Document.RecordingCount != 1 || !after.Document.RecordingsDirty {
		t.Fatal("import lacks complete asset/provenance/kept viewer")
	}
	if !reflect.DeepEqual(after.Components, edited.Components) || !reflect.DeepEqual(after.Transient, edited.Transient) || !reflect.DeepEqual(after.UndoEntry, edited.UndoEntry) || after.RunID != edited.RunID || after.ElectricalRevision != edited.ElectricalRevision || after.View != edited.View {
		t.Fatal("import changes newer live editing/physics/history")
	}
	operation := inspectOperation(t, s, after.ProjectWork.ResultOperationID)
	if operation.Operation.Actor != "test" || operation.Operation.SourceSHA256 != after.ProjectWork.SourceSHA256 || operation.Operation.SourceUnchanged {
		t.Fatal("import falsely associates live source or loses admitting actor")
	}
}
func TestImportCancellationAndReplacementNeverAttachStagedRecord(t *testing.T) {
	for _, command := range []string{"project.work.cancel", "project.clear", "app.close"} {
		t.Run(command, func(t *testing.T) {
			path := importFixture(t)
			s := pausedRC(t)
			defer s.Close()
			entered, release := blockRecordingImport(t, s)
			defer release()
			admitted := promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path}).Result.(Snapshot)
			awaitLiveSampleBlock(t, entered)
			var p any = ReplaceParams{DiscardChanges: true}
			status := "superseded"
			if command == "project.work.cancel" {
				p = IDParams{ID: admitted.ProjectWork.ID}
				status = "canceled"
			}
			if command == "app.close" {
				status = "canceled"
			}
			promptLiveCommand(t, s, command, p)
			before := s.Snapshot()
			release()
			after := awaitProjectStatus(t, s, status)
			if after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Transient, before.Transient) || len(after.Operations) != len(before.Operations) || after.ProjectWork.ResultOperationID != "" {
				t.Fatal("discarded import changes replacement/closed project")
			}
		})
	}
}
func TestImportDoesNotOverrideLaterViewerChoice(t *testing.T) {
	path := importFixture(t)
	s := pausedRC(t)
	defer s.Close()
	duration := .01
	_, existing := verify(t, s, VerifyParams{Duration: &duration}, "existing")
	entered, release := blockRecordingImport(t, s)
	defer release()
	promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "recording.view.set", RecordingViewParams{Mode: "live"})
	promptLiveCommand(t, s, "recording.cursor.set", CursorParams{Cursor: "a", Time: .005})
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.ViewerSelected == nil || *after.ProjectWork.ViewerSelected || after.SelectedOperation != existing.Operation.ID || !reflect.DeepEqual(after.Recording, before.Recording) || after.Document.RecordingCount != 2 {
		t.Fatal("late import overrides live/viewer choice or loses kept asset")
	}
	*inspectProjectWork(t, s, after.ProjectWork.ID).ViewerSelected = true
	if *inspectProjectWork(t, s, after.ProjectWork.ID).ViewerSelected {
		t.Fatal("import inspection exposes mutable viewer outcome")
	}
}
func TestImportRechecksRecordingCapacityAfterValidation(t *testing.T) {
	path := importFixture(t)
	s := pausedRC(t)
	defer s.Close()
	duration := .001
	for i := 0; i < MaxVerificationOperations-1; i++ {
		verify(t, s, VerifyParams{Duration: &duration}, "existing-"+string(rune('a'+i)))
	}
	entered, release := blockRecordingImport(t, s)
	defer release()
	promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	verify(t, s, VerifyParams{Duration: &duration}, "last-slot")
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "failed")
	if after.ProjectWork.Error.Code != "record-limit" || !reflect.DeepEqual(after.Operations, before.Operations) || !reflect.DeepEqual(after.Recording, before.Recording) || after.Document != before.Document {
		t.Fatal("late import evicts kept data after capacity is consumed")
	}
}
func TestImportWaitingDisconnectRetryAndSaveRoundTrip(t *testing.T) {
	path := importFixture(t)
	s := pausedRC(t)
	defer s.Close()
	entered, release := blockRecordingImport(t, s)
	defer release()
	raw, _ := json.Marshal(RecordingImportParams{Path: path})
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "import", Command: "verification.import", Params: raw, ExpectedCommandRevision: &rev, Key: "original"}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(ctx, request, "importer") }()
	awaitLiveSampleBlock(t, entered)
	cancel()
	select {
	case r := <-done:
		if r.Error == nil || r.Error.Code != "canceled" {
			t.Fatal("request wait lacks cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("request wait remains blocked")
	}
	release()
	after := awaitProjectStatus(t, s, "completed")
	retry := s.Dispatch(context.Background(), request, "importer")
	if !retry.Duplicate || retry.Error != nil || retry.Result.(Snapshot).ProjectWork.ResultOperationID != after.ProjectWork.ResultOperationID || after.Document.RecordingCount != 1 {
		t.Fatal("import retry duplicates asset or loses terminal outcome")
	}
	saved := filepath.Join(t.TempDir(), "imported.elab")
	call(t, s, "project.save", SaveParams{Path: saved}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: saved}, "load")
	restored := settled(t, fresh)
	if restored.Document.RecordingCount != 1 || !restored.Recording.Viewing || restored.Document.Dirty || restored.Transient.Sample.Time != 0 {
		t.Fatal("imported kept asset does not round trip with fresh live startup")
	}
}
