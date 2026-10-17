package workbench

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func captureAutomaticRecovery(s *Service) (*recoveryStore, automaticRecoveryCapture) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recoveryStore, automaticRecoveryCapture{s.recoveryCaptureLocked(), s.recoveryCatalogRevision}
}

func prepareAutomaticRecoveryResult(store *recoveryStore, capture automaticRecoveryCapture) (recoveryWorkResult, error) {
	id, savedAt, err := store.checkpoint(capture.recoveryCapture)
	candidates, problems, scanErr := store.scan()
	if err == nil {
		err = scanErr
	}
	return recoveryWorkResult{id: id, savedAt: savedAt, candidates: candidates, problems: problems, scanError: scanErr}, err
}

func publishAutomaticRecovery(s *Service, store *recoveryStore, capture automaticRecoveryCapture, result recoveryWorkResult, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publishAutomaticRecoveryLocked(store, capture, result, err)
}

func TestAutomaticRecoveryFailureAndRetrySurviveInterveningEdits(t *testing.T) {
	s := NewExample()
	defer s.Close()
	base := t.TempDir()
	directory := filepath.Join(base, "recovery")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	configureRecovery(t, s, directory, false)
	source := filepath.Join(base, "manual.elab")
	call(t, s, "project.save", SaveParams{Path: source}, "manual")
	manual, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "first-edit")
	store, capture := captureAutomaticRecovery(s)
	moved := directory + "-preserved"
	if err := os.Rename(directory, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(directory, []byte("owned unavailable storage"), 0600); err != nil {
		t.Fatal(err)
	}
	restored := false
	restore := func() {
		if !restored {
			if err := os.Remove(directory); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(moved, directory); err != nil {
				t.Fatal(err)
			}
			restored = true
		}
	}
	defer restore()
	result, failure := prepareAutomaticRecoveryResult(store, capture)
	if failure == nil {
		t.Fatal("owned filesystem fault did not fail")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "3k"}, "intervening-edit")
	before := settled(t, s)
	publishAutomaticRecovery(s, store, capture, result, failure)
	after := s.Snapshot()
	if after.Recovery.Error == "" {
		t.Fatal("automatic storage failure was hidden by an intervening edit")
	}
	assertAutomaticRecoveryPreservesEditor(t, before, after)
	restore()
	store, capture = captureAutomaticRecovery(s)
	result, err = prepareAutomaticRecoveryResult(store, capture)
	if err != nil || result.id == "" {
		t.Fatal("storage retry did not write a complete capsule", err)
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "4k"}, "later-edit")
	before = settled(t, s)
	publishAutomaticRecovery(s, store, capture, result, nil)
	after = s.Snapshot()
	if after.Recovery.Error != "" || after.Recovery.LastSnapshot != result.id || len(after.Recovery.Candidates) != 1 {
		t.Fatal("successful retry hidden by a later edit", after.Recovery)
	}
	assertAutomaticRecoveryPreservesEditor(t, before, after)
	choice := RecoveryChoiceParams{ID: result.id, SHA256: after.Recovery.Candidates[0].SHA256}
	inspection := call(t, s, "recovery.inspect", choice, "inspect-retry").Result.(RecoveryInspection)
	part, _ := inspection.Document.Graph.Component("R1")
	if part.Value != 3000 || !after.Document.Dirty {
		t.Fatal("retry metadata pretended to save the later editor state")
	}
	current, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(current, manual) {
		t.Fatal("automatic fault/retry changed the explicit save", err)
	}
}

func assertAutomaticRecoveryPreservesEditor(t *testing.T, before, after Snapshot) {
	t.Helper()
	if before.CommandRevision != after.CommandRevision || before.Document != after.Document || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) {
		t.Fatal("automatic recovery publication changed editor/physics/history")
	}
}

func TestAutomaticRecoveryCannotReplaceNewerCatalogOrStore(t *testing.T) {
	for _, action := range []string{"scan", "save", "reconfigure", "close"} {
		t.Run(action, func(t *testing.T) {
			s := NewExample()
			defer s.Close()
			configureRecovery(t, s, t.TempDir(), false)
			call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "dirty")
			store, capture := captureAutomaticRecovery(s)
			result, err := prepareAutomaticRecoveryResult(store, capture)
			if err != nil {
				t.Fatal(err)
			}
			switch action {
			case "scan":
				call(t, s, "recovery.scan", struct{}{}, "newer-scan")
				call(t, s, "recovery.select", IDParams{ID: ""}, "clear-selection")
			case "save":
				call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "manual.elab")}, "newer-save")
			case "reconfigure":
				directory := t.TempDir()
				if err := os.Chmod(directory, 0700); err != nil {
					t.Fatal(err)
				}
				call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 1}, "newer-configure")
			case "close":
				s.Close()
			}
			before := s.Snapshot()
			for _, failure := range []error{nil, os.ErrPermission} {
				publishAutomaticRecovery(s, store, capture, result, failure)
				after := s.Snapshot()
				if !reflect.DeepEqual(before, after) {
					t.Fatal("late automatic catalog/error replaced newer publication")
				}
			}
		})
	}
}

func TestAutomaticRecoveryKeepsExplicitlyClearedSelection(t *testing.T) {
	s := NewExample()
	defer s.Close()
	configureRecovery(t, s, t.TempDir(), false)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "dirty")
	store, capture := captureAutomaticRecovery(s)
	result, err := prepareAutomaticRecoveryResult(store, capture)
	if err != nil {
		t.Fatal(err)
	}
	call(t, s, "recovery.select", IDParams{ID: ""}, "clear-selection")
	publishAutomaticRecovery(s, store, capture, result, nil)
	if s.Snapshot().Recovery.Selected != "" || len(s.Snapshot().Recovery.Candidates) != 1 {
		t.Fatal("automatic publication lost explicit empty selection/catalog")
	}
}

func TestAutomaticRecoveryCannotResurrectRetiredCapsuleDuringEditedSaveCleanup(t *testing.T) {
	s := NewExample()
	defer s.Close()
	configureRecovery(t, s, t.TempDir(), false)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "dirty")
	store, capture := captureAutomaticRecovery(s)
	result, err := prepareAutomaticRecoveryResult(store, capture)
	if err != nil {
		t.Fatal(err)
	}
	path := store.candidatePath(result.id)
	store.mu.Lock()
	locked := true
	release := func() {
		if locked {
			store.mu.Unlock()
			locked = false
		}
	}
	defer release()
	promptLiveCommand(t, s, "project.save.start", SaveParams{Path: filepath.Join(t.TempDir(), "manual.elab")})
	deadline := time.Now().Add(2 * time.Second)
	for s.Snapshot().ProjectWork.Phase != "recovery-cleanup" {
		if time.Now().After(deadline) {
			t.Fatal("save did not enter recovery cleanup")
		}
		time.Sleep(time.Millisecond)
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "3k"}, "edit-during-cleanup")
	release()
	awaitProjectStatus(t, s, "completed")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("saved baseline did not retire old capsule", err)
	}
	before := settled(t, s)
	publishAutomaticRecovery(s, store, capture, result, nil)
	after := s.Snapshot()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("late automatic result resurrected a capsule retired by an edited save cleanup")
	}
}
