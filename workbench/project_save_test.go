package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func blockSavePreparation(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.projectSave = func(ctx context.Context, capture saveCapture, path string, progress func(string, int)) (*preparedSave, error) {
		prepared, err := prepareProjectSave(ctx, capture, path, progress)
		close(entered)
		<-release
		return prepared, err
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func blockSavePublication(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.projectPublish = func(prepared *preparedSave) error { close(entered); <-release; return publishProjectSave(prepared) }
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func savedLED(t *testing.T) (*Service, string) {
	t.Helper()
	s := NewExample()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	path := filepath.Join(t.TempDir(), "LED.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "baseline")
	return s, path
}
func inspectProjectWork(t *testing.T, s *Service, id string) *ProjectWork {
	t.Helper()
	raw, _ := json.Marshal(IDParams{ID: id})
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "inspect", Command: "project.work.inspect", Params: raw}, "test")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(*ProjectWork)
}
func TestSaveCapturePreservesLaterEditsAndNavigationWithoutChangingPhysicsHistory(t *testing.T) {
	s, path := savedLED(t)
	defer s.Close()
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "value")
	entered, release := blockSavePreparation(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "project.save.start", SaveParams{}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "3k"})
	prefs := DefaultDisplayPreferences()
	prefs.VoltageColor = true
	promptLiveCommand(t, s, "display.preferences.set", prefs)
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.Document.Path != path || !after.Document.SchematicDirty || !after.Document.PreferencesDirty || !after.Document.Dirty || after.ProjectWork.Associated == nil || !*after.ProjectWork.Associated || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) || !reflect.DeepEqual(after.Result, before.Result) || after.RunID != before.RunID || after.View != before.View {
		t.Fatal("save loses later edits/physics/history or marks newer data clean")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	part, _ := d.Graph.Component("R1")
	digest := sha256.Sum256(data)
	if part.Value != 2000 || d.DisplayPreferences != nil || after.ProjectWork.ArtifactSHA256 != hex.EncodeToString(digest[:]) || after.ProjectWork.ID != admitted.ProjectWork.ID {
		t.Fatal("file includes later edits or lacks artifact identity")
	}
}
func TestPreparedSaveCancelKeepsOriginalDestinationAndRemovesTemporary(t *testing.T) {
	s, path := savedLED(t)
	defer s.Close()
	original, _ := os.ReadFile(path)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "value")
	before := s.Snapshot()
	entered, release := blockSavePreparation(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "project.save.start", SaveParams{}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	refuseStepWork(t, s, "project.save.start", SaveParams{}, "busy")
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: "old"}, "operation-unavailable")
	promptLiveCommand(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID})
	release()
	after := awaitProjectStatus(t, s, "canceled")
	data, _ := os.ReadFile(path)
	temps, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".elab-save-*"))
	if string(data) != string(original) || len(temps) != 0 || after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) || after.ProjectWork.ArtifactSHA256 != "" {
		t.Fatal("canceled save replaces old file/save point or leaks temporary")
	}
}
func TestSaveChecksExternalDestinationAgainBeforeAtomicPublication(t *testing.T) {
	s, path := savedLED(t)
	defer s.Close()
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "value")
	before := s.Snapshot()
	entered, release := blockSavePreparation(t, s)
	defer release()
	promptLiveCommand(t, s, "project.save.start", SaveParams{Overwrite: true})
	awaitLiveSampleBlock(t, entered)
	outside := []byte("new outside change after preparation\n")
	if err := os.WriteFile(path, outside, 0600); err != nil {
		t.Fatal(err)
	}
	release()
	after := awaitProjectStatus(t, s, "failed")
	data, _ := os.ReadFile(path)
	if after.ProjectWork.Error.Code != "file-conflict" || string(data) != string(outside) || after.Document != before.Document {
		t.Fatal("save overwrites a change made after destination approval")
	}
}
func TestReplacementBeforeSavePublicationDiscardsPreparedFile(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	path := filepath.Join(t.TempDir(), "old-document.elab")
	entered, release := blockSavePreparation(t, s)
	defer release()
	promptLiveCommand(t, s, "project.save.start", SaveParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true})
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "superseded")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("superseded prepared save published a file")
	}
	if after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) {
		t.Fatal("superseded save associates a replacement document")
	}
}
func TestPublishingSaveRefusesCancelAndNeverAssociatesReplacement(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	path := filepath.Join(t.TempDir(), "captured.elab")
	entered, release := blockSavePublication(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "project.save.start", SaveParams{Path: path}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	publishing := inspectProjectWork(t, s, admitted.ProjectWork.ID)
	if publishing.Phase != "publishing" || publishing.Cancelable {
		t.Fatal("publication falsely offers cancellation")
	}
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: publishing.ID}, "operation-committing")
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true})
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	if after.ProjectWork.Associated == nil || *after.ProjectWork.Associated || after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || len(d.Graph.Components) != 4 {
		t.Fatal("published prior snapshot overwrites/associates the replacement")
	}
	*inspectProjectWork(t, s, publishing.ID).Associated = true
	if *inspectProjectWork(t, s, publishing.ID).Associated {
		t.Fatal("project inspection aliases completion")
	}
}
func TestLegacySaveWaitDisconnectAndRetryDoesNotRewriteFile(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	path := filepath.Join(t.TempDir(), "saved.elab")
	entered, release := blockSavePreparation(t, s)
	defer release()
	raw, _ := json.Marshal(SaveParams{Path: path})
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "save", Command: "project.save", Params: raw, ExpectedCommandRevision: &rev, Key: "original"}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(ctx, request, "saver") }()
	awaitLiveSampleBlock(t, entered)
	cancel()
	select {
	case r := <-done:
		if r.Error == nil || r.Error.Code != "canceled" {
			t.Fatal("wait did not report cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("disconnected request remains blocked")
	}
	release()
	after := awaitProjectStatus(t, s, "completed")
	info, _ := os.Stat(path)
	retry := s.Dispatch(context.Background(), request, "saver")
	again, _ := os.Stat(path)
	if !retry.Duplicate || retry.Error != nil || retry.Result.(Snapshot).ProjectWork.ID != after.ProjectWork.ID || !os.SameFile(info, again) {
		t.Fatal("save retry rewrites file or loses original result")
	}
}
func TestRecoveryOlderCleanCaptureCannotEraseWiperCommitWithSameCommandRevision(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "pot.elab")}, "save")
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: filepath.Join(t.TempDir(), "recovery"), Enabled: false, Interval: 1}, "configure")
	entered, release := blockInputSolver(t, s, false)
	defer release()
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8})
	awaitLiveSampleBlock(t, entered)
	s.mu.Lock()
	clean := s.recoveryCaptureLocked()
	store := s.recoveryStore
	s.mu.Unlock()
	release()
	awaitInputStatus(t, s, "completed")
	s.mu.Lock()
	dirty := s.recoveryCaptureLocked()
	s.mu.Unlock()
	if clean.dirty || !dirty.dirty || clean.commandRevision != dirty.commandRevision || clean.stateRevision >= dirty.stateRevision {
		t.Fatal("fixture does not exercise same-command background publication")
	}
	id, _, err := store.checkpoint(dirty)
	if err != nil || id == "" {
		t.Fatal("new dirty snapshot missing", err)
	}
	if _, _, err := store.checkpoint(clean); err != nil {
		t.Fatal(err)
	}
	candidates, _, err := store.scan()
	if err != nil || len(candidates) != 1 || candidates[0].ID != id {
		t.Fatal("late clean cleanup erases newer wiper data")
	}
}

func TestProjectOperationRetentionKeepsStableIdentityAndExpiresOldCompletedWork(t *testing.T) {
	s, path := savedLED(t)
	defer s.Close()
	first := s.Snapshot().ProjectWork.ID
	for i := 0; i < MaxProjectOperations; i++ {
		call(t, s, "project.save", SaveParams{Path: path}, "save-"+string(rune('a'+i)))
	}
	latest := s.Snapshot().ProjectWork.ID
	if inspectProjectWork(t, s, latest).Status != "completed" {
		t.Fatal("latest operation is not inspectable")
	}
	raw, _ := json.Marshal(IDParams{ID: first})
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "old", Command: "project.work.inspect", Params: raw}, "test")
	if r.Error == nil || r.Error.Code != "operation-unavailable" {
		t.Fatal("completed operation retention is unbounded")
	}
	s.mu.Lock()
	count := len(s.projectJobs)
	s.mu.Unlock()
	if count != MaxProjectOperations {
		t.Fatal("unexpected retained operation count", count)
	}
}

func TestReplacementDuringSaveRecoveryCleanupHasNoCurrentAssociation(t *testing.T) {
	s, path := savedLED(t)
	defer s.Close()
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: filepath.Join(t.TempDir(), "recovery"), Enabled: false, Interval: 1}, "configure")
	s.mu.Lock()
	store := s.recoveryStore
	s.mu.Unlock()
	store.mu.Lock()
	var once sync.Once
	release := func() { once.Do(func() { store.mu.Unlock() }) }
	defer release()
	admitted := promptLiveCommand(t, s, "project.save.start", SaveParams{}).Result.(Snapshot)
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().ProjectWork.Phase != "recovery-cleanup" {
		if time.Now().After(deadline) {
			t.Fatal("save did not reach unlocked recovery cleanup")
		}
		time.Sleep(time.Millisecond)
	}
	promptLiveCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true})
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.ID != admitted.ProjectWork.ID || after.ProjectWork.Associated == nil || *after.ProjectWork.Associated || after.Document != before.Document {
		t.Fatal("late cleanup completion falsely associates replacement")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("already published captured file lost", err)
	}
}
