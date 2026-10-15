package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func recoverySetupDirectory(t *testing.T) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "recovery")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	return directory
}

func blockRecoveryConfiguration(t *testing.T, s *Service) (<-chan recoveryConfigurationResult, func()) {
	t.Helper()
	entered := make(chan recoveryConfigurationResult, 1)
	release := make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.recoveryConfigure = func(ctx context.Context, capture recoveryConfigurationCapture, progress func(string, int)) (recoveryConfigurationResult, error) {
		result, err := prepareRecoveryConfiguration(ctx, capture, progress)
		if err != nil {
			t.Errorf("setup prepare %s: %v", capture.params.Directory, err)
		}
		entered <- result
		<-release
		return result, err
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func configuredStore(t *testing.T, s *Service) *recoveryStore {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recoveryStore
}
func TestRecoveryConfigurationStagingPreservesLiveEditingAndPreviousStore(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	configureRecovery(t, s, t.TempDir(), false)
	old := configuredStore(t, s)
	before := s.Snapshot()
	entered, release := blockRecoveryConfiguration(t, s)
	defer release()
	directory := filepath.Join(t.TempDir(), "new-recovery")
	admitted := promptLiveCommand(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 60}).Result.(Snapshot)
	prepared := <-entered
	if prepared.store == nil || configuredStore(t, s) != old || !reflect.DeepEqual(before.Recovery, s.Snapshot().Recovery) || *admitted.ProjectWork.RecoveryConfigured {
		t.Fatal("admission exposes staged setup")
	}
	if _, err := os.Stat(prepared.store.leasePath()); err != nil {
		t.Fatal(err)
	}
	refuseStepWork(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 2}, "busy")
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	call(t, s, "simulation.step", StepParams{Duration: .005}, "step")
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	edited := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.Recovery.Directory != directory || !after.Recovery.Enabled || after.Recovery.Interval != 60 || !*after.ProjectWork.RecoveryConfigured || after.ProjectWork.Actor != "test" {
		t.Fatal("setup publication missing", after.ProjectWork)
	}
	if !reflect.DeepEqual(after.Components, edited.Components) || !reflect.DeepEqual(after.Transient, edited.Transient) || !reflect.DeepEqual(after.UndoEntry, edited.UndoEntry) || after.Document != edited.Document || after.View != edited.View {
		t.Fatal("setup changes later editing/physics/history")
	}
	if _, err := os.Stat(old.leasePath()); !os.IsNotExist(err) {
		t.Fatal("previous lease not released", err)
	}
	*inspectProjectWork(t, s, after.ProjectWork.ID).RecoveryConfigured = false
	if !*inspectProjectWork(t, s, after.ProjectWork.ID).RecoveryConfigured {
		t.Fatal("inspection aliases configuration outcome")
	}
}
func TestRecoveryConfigurationCancelAndCloseCleanStagedLease(t *testing.T) {
	for _, closeApp := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "close"}[closeApp], func(t *testing.T) {
			s := NewExample()
			defer s.Close()
			settled(t, s)
			configureRecovery(t, s, t.TempDir(), false)
			old := configuredStore(t, s)
			before := s.Snapshot().Recovery
			entered, release := blockRecoveryConfiguration(t, s)
			defer release()
			admitted := promptLiveCommand(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: recoverySetupDirectory(t), Enabled: true, Interval: 60}).Result.(Snapshot)
			staged := <-entered
			if staged.store == nil {
				t.Fatal("test setup produced no staged lease")
			}
			if closeApp {
				promptLiveCommand(t, s, "app.close", ReplaceParams{DiscardChanges: true})
			} else {
				promptLiveCommand(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID})
			}
			release()
			after := awaitProjectStatus(t, s, "canceled")
			if *after.ProjectWork.RecoveryConfigured || configuredStore(t, s) != old || !reflect.DeepEqual(before, after.Recovery) {
				t.Fatal("canceled setup changes previous configuration")
			}
			if _, err := os.Stat(staged.store.leasePath()); !os.IsNotExist(err) {
				t.Fatal("staged lease leaked", err)
			}
			if !closeApp {
				if _, err := os.Stat(old.leasePath()); err != nil {
					t.Fatal("cancellation closed previous lease", err)
				}
			}
		})
	}
}
func TestRecoveryConfigurationLeaseCleanupDoesNotHoldEditorLock(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	configureRecovery(t, s, t.TempDir(), false)
	old := configuredStore(t, s)
	old.mu.Lock()
	var once sync.Once
	unlock := func() { once.Do(old.mu.Unlock) }
	defer unlock()
	admitted := promptLiveCommand(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: recoverySetupDirectory(t), Enabled: false, Interval: 2}).Result.(Snapshot)
	deadline := time.Now().Add(time.Second)
	for {
		state := s.Snapshot()
		if state.ProjectWork.Phase == "closing-previous-recovery" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("setup did not reach lease cleanup")
		}
		time.Sleep(time.Millisecond)
	}
	if configuredStore(t, s) == old || !*s.Snapshot().ProjectWork.RecoveryConfigured || s.Snapshot().ProjectWork.Cancelable {
		t.Fatal("cleanup phase lost atomic install")
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 4, "dy": 8})
	call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID}, "operation-committing")
	unlock()
	awaitProjectStatus(t, s, "completed")
}
func TestRecoveryConfigurationFailureKeepsPreviousSettingsAndCleansLease(t *testing.T) {
	for _, mode := range []string{"scan", "timeout", "nil"} {
		t.Run(mode, func(t *testing.T) {
			s := New()
			defer s.Close()
			configureRecovery(t, s, t.TempDir(), false)
			old := configuredStore(t, s)
			before := s.Snapshot().Recovery
			directory := recoverySetupDirectory(t)
			if mode == "scan" {
				for i := 0; i < MaxRecoveryEntries; i++ {
					if err := os.WriteFile(filepath.Join(directory, fmt.Sprintf("entry-%03d", i)), []byte("preserve"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				s.mu.Lock()
				s.recoveryConfigure = func(ctx context.Context, capture recoveryConfigurationCapture, progress func(string, int)) (recoveryConfigurationResult, error) {
					if mode == "nil" {
						return recoveryConfigurationResult{}, nil
					}
					result, err := prepareRecoveryConfiguration(ctx, capture, progress)
					if err != nil {
						return result, err
					}
					return result, context.DeadlineExceeded
				}
				s.mu.Unlock()
			}
			code := "invalid-params"
			if mode == "timeout" {
				code = "timeout"
			}
			if mode == "nil" {
				code = "file-error"
			}
			refuse(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 2}, code)
			if configuredStore(t, s) != old || !reflect.DeepEqual(before, s.Snapshot().Recovery) || *s.Snapshot().ProjectWork.RecoveryConfigured {
				t.Fatal("failed setup changes previous settings")
			}
			leases, err := filepath.Glob(filepath.Join(directory, "session-*.json"))
			if err != nil || len(leases) != 0 {
				t.Fatal("failed setup leaked lease", leases, err)
			}
			if mode == "scan" {
				entries, _ := os.ReadDir(directory)
				if len(entries) != MaxRecoveryEntries {
					t.Fatal("failed scan removed preexisting files")
				}
			}
		})
	}
}
func TestRecoveryConfigurationSameStorePreservesNewerCatalogAndSelection(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	configureRecovery(t, s, t.TempDir(), false)
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	old := configuredStore(t, s)
	entered, release := blockRecoveryConfiguration(t, s)
	defer release()
	promptLiveCommand(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: old.directory, Enabled: true, Interval: 60})
	<-entered
	promptLiveCommand(t, s, "recovery.select", IDParams{ID: ""})
	s.mu.Lock()
	s.recovery.Problems = []RecoveryProblem{{ID: "newer", Message: "newer catalog"}}
	s.recoveryCatalogRevision++
	s.mu.Unlock()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if configuredStore(t, s) != old || after.Recovery.Selected != "" || len(after.Recovery.Problems) != 1 || after.Recovery.Problems[0].ID != "newer" || *after.ProjectWork.CatalogApplied || !after.Recovery.Enabled {
		t.Fatal("same-store setup overwrites newer catalog/choice")
	}
}
func TestRecoveryConfigurationWaitingDisconnectRetryAndDocumentReplacement(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	entered, release := blockRecoveryConfiguration(t, s)
	defer release()
	raw, _ := json.Marshal(RecoveryConfigureParams{Directory: recoverySetupDirectory(t), Enabled: false, Interval: 2})
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "configure", Command: "recovery.configure", Key: "configure", Params: raw, ExpectedCommandRevision: &rev}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(ctx, request, "configurer") }()
	<-entered
	cancel()
	select {
	case response := <-done:
		if response.Error == nil || response.Error.Code != "canceled" {
			t.Fatal("request wait not canceled")
		}
	case <-time.After(time.Second):
		t.Fatal("request wait stuck")
	}
	promptLiveCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true})
	generation := s.Snapshot().Document.Generation
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.Document.Generation != generation || len(after.Components) != 0 || !*after.ProjectWork.RecoveryConfigured {
		t.Fatal("session setup undoes replacement or is superseded by it")
	}
	retry := s.Dispatch(context.Background(), request, "configurer")
	if !retry.Duplicate || retry.Error != nil || retry.Result.(Snapshot).ProjectWork.ID != after.ProjectWork.ID || after.ProjectWork.Actor != "configurer" {
		t.Fatal("retry duplicates setup or loses actor")
	}
}

func TestRecoveryConfigurationAliasesKeepUnitGrammarAndNumericOutputs(t *testing.T) {
	s := New()
	defer s.Close()
	directory := recoverySetupDirectory(t)
	for _, id := range []string{"recovery.configure", "recovery.configure.start"} {
		caps, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		interval := inputProperty(schema, "interval_s")
		if interval["$ref"] != TimeInputSchemaID || interval["minimum"] != float64(1) || interval["maximum"] != float64(60) {
			t.Fatal("setup alias lost time grammar/bounds", id)
		}
		call(t, s, id, map[string]any{"directory": directory, "enabled": false, "interval_s": "1500 ms"}, id)
		state := awaitProjectStatus(t, s, "completed")
		if state.Recovery.Interval != 1.5 {
			t.Fatal("unit interval not canonical seconds")
		}
	}
	var output map[string]any
	if err := json.Unmarshal(SchemaRegistry()[SnapshotSchemaID], &output); err != nil {
		t.Fatal(err)
	}
	if inputProperty(output, "recovery.interval_s")["type"] != "number" {
		t.Fatal("input time grammar widened snapshot output")
	}
}

func TestRecoveryConfigurationCloseJoinsStagedCleanup(t *testing.T) {
	s := New()
	defer s.Close()
	entered, release := blockRecoveryConfiguration(t, s)
	defer release()
	promptLiveCommand(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: recoverySetupDirectory(t), Enabled: false, Interval: 2})
	staged := <-entered
	if staged.store == nil {
		t.Fatal("no staged store")
	}
	done := make(chan struct{})
	go func() { s.Close(); close(done) }()
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Status != "closed" {
		if time.Now().After(deadline) {
			t.Fatal("close did not cancel setup")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-done:
		t.Fatal("close returned before setup worker drained")
	default:
	}
	release()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close did not join setup cleanup")
	}
	if _, err := os.Stat(staged.store.leasePath()); !os.IsNotExist(err) {
		t.Fatal("close left a staged lease", err)
	}
}
func TestRecoveryConfigurationCleanupWarningDoesNotUndoInstalledSettings(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	configureRecovery(t, s, t.TempDir(), false)
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	old := configuredStore(t, s)
	choice := recoveryChoice(t, s)
	path := old.candidatePath(choice.ID)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A nonempty directory at the old lease path forces a cleanup error while
	// leaving the recovery capsule untouched. Everything here is an owned fixture.
	if err := os.Rename(old.leasePath(), old.leasePath()+".preserved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(old.leasePath(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old.leasePath(), "owned-fixture"), []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	directory := recoverySetupDirectory(t)
	after := call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 2}, "new").Result.(Snapshot)
	if after.ProjectWork.Status != "completed" || !*after.ProjectWork.RecoveryConfigured || after.ProjectWork.CleanupWarning == "" || after.Recovery.Directory != directory {
		t.Fatal("cleanup error undoes or hides successful installation")
	}
	data, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(data, before) {
		t.Fatal("lease cleanup removed previous recovery data", err)
	}
}
