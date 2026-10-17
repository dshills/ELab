package workbench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDisableExistingRecoveryDoesNotRequireAvailableStorage(t *testing.T) {
	for _, command := range []string{"recovery.configure", "recovery.configure.start"} {
		t.Run(command, func(t *testing.T) {
			s := NewExample()
			defer s.Close()
			directory := recoverySetupDirectory(t)
			source := filepath.Join(t.TempDir(), "manual.elab")
			call(t, s, "project.save", SaveParams{Path: source}, "manual")
			manual, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "dirty")
			call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 60}, "configure")
			call(t, s, "recovery.checkpoint", struct{}{}, "last-good")
			choice := recoveryChoice(t, s)
			capsuleName := choice.ID + ".elab-recovery"
			good, err := os.ReadFile(filepath.Join(directory, capsuleName))
			if err != nil {
				t.Fatal(err)
			}
			store, capture := captureAutomaticRecovery(s)
			moved := directory + "-preserved"
			if err := os.Rename(directory, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(directory, []byte("owned unavailable storage"), 0600); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Remove(directory); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(moved, directory); err != nil {
					t.Fatal(err)
				}
			}()
			result, failure := prepareAutomaticRecoveryResult(store, capture)
			if failure == nil {
				t.Fatal("owned storage interruption did not fail")
			}
			publishAutomaticRecovery(s, store, capture, result, failure)
			before := settled(t, s)
			call(t, s, command, RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 3.5}, "disable")
			after := awaitProjectStatus(t, s, "completed")
			if after.Recovery.Enabled || after.Recovery.Interval != 3.5 || !*after.ProjectWork.RecoveryConfigured || after.ProjectWork.CatalogApplied == nil || *after.ProjectWork.CatalogApplied {
				t.Fatal("disable lost settings/outcome or claimed a filesystem refresh", after.ProjectWork)
			}
			expected := before.Recovery
			expected.Enabled, expected.Interval = false, 3.5
			if !reflect.DeepEqual(expected, after.Recovery) || configuredStore(t, s) != store {
				t.Fatal("disable replaced the last-known error/catalog/lease")
			}
			if before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || before.RunID != after.RunID {
				t.Fatal("disable changed the open circuit/history/save point")
			}
			current, err := os.ReadFile(filepath.Join(moved, capsuleName))
			if err != nil || !bytes.Equal(current, good) {
				t.Fatal("disable changed the last complete capsule", err)
			}
			current, err = os.ReadFile(source)
			if err != nil || !bytes.Equal(current, manual) {
				t.Fatal("disable changed the manual source", err)
			}
		})
	}
}

func TestDisableRecoveryDoesNotWaitForCapturedWriteOrPublishItsLateStatus(t *testing.T) {
	s := NewExample()
	defer s.Close()
	directory := recoverySetupDirectory(t)
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 60}, "setup")
	store, capture := captureAutomaticRecovery(s)
	store.mu.Lock()
	locked := true
	release := func() {
		if locked {
			store.mu.Unlock()
			locked = false
		}
	}
	defer release()
	started, done := make(chan struct{}), make(chan error, 1)
	go func() {
		close(started)
		result, err := prepareAutomaticRecoveryResult(store, capture)
		publishAutomaticRecovery(s, store, capture, result, err)
		done <- err
	}()
	<-started
	promptLiveCommand(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 2})
	before := awaitProjectStatus(t, s, "completed")
	if before.Recovery.Enabled || *before.ProjectWork.CatalogApplied {
		t.Fatal("disable waited for the store or claimed a scan")
	}
	release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("captured checkpoint did not finish")
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("captured write replaced disabled scheduling/catalog status")
	}
	paths, err := filepath.Glob(filepath.Join(directory, "*.elab-recovery"))
	if err != nil || len(paths) != 1 {
		t.Fatal("disabled scheduling discarded an already captured write", paths, err)
	}
}

func TestDisableRecoveryKeepsAdmissionValidationAndCanBeCanceled(t *testing.T) {
	s := NewExample()
	defer s.Close()
	directory := recoverySetupDirectory(t)
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 60}, "setup")
	before := s.Snapshot().Recovery
	refuse(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 0}, "invalid-params")
	other := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(other, []byte("owned fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "recovery.configure", RecoveryConfigureParams{Directory: other, Enabled: false, Interval: 2}, "file-error")
	entered, release := blockRecoveryConfiguration(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "recovery.configure.start", RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 2}).Result.(Snapshot)
	prepared := <-entered
	if prepared.store != configuredStore(t, s) {
		t.Fatal("same-store disable staged a new lease")
	}
	promptLiveCommand(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID})
	release()
	after := awaitProjectStatus(t, s, "canceled")
	if !reflect.DeepEqual(before, after.Recovery) || *after.ProjectWork.RecoveryConfigured {
		t.Fatal("canceled disable changed scheduling/catalog")
	}
}

func TestDisableRecoverySemanticsAreDiscoverableForBothAliases(t *testing.T) {
	for _, command := range []string{"recovery.configure", "recovery.configure.start"} {
		caps, err := detailedCapabilities(command)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		semantics, ok := schema["x-existing-store-disable"].(map[string]any)
		if !ok || semantics["folder_access"] != false || semantics["recovery_catalog_applied"] != false || semantics["preserves_last_known_catalog_and_error"] != true || semantics["in_flight_checkpoint_may_finish"] != true {
			t.Fatal("same-store disable behavior not discoverable", semantics)
		}
	}
}
