package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func configureRecovery(t *testing.T, s *Service, dir string, enabled bool) {
	t.Helper()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: dir, Enabled: enabled, Interval: 1}, "configure")
}
func recoveryChoice(t *testing.T, s *Service) RecoveryChoiceParams {
	t.Helper()
	candidates := s.Snapshot().Recovery.Candidates
	if len(candidates) != 1 {
		t.Fatalf("want one candidate, got %+v", candidates)
	}
	return RecoveryChoiceParams{ID: candidates[0].ID, SHA256: candidates[0].SHA256}
}
func TestRecoveryRestoresCopyWithoutOverwritingSavedFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.elab")
	s := NewExample()
	configureRecovery(t, s, dir, false)
	call(t, s, "project.save", SaveParams{Path: source}, "save")
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	before := settled(t, s)
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || before.RunID != after.RunID || before.Document != after.Document {
		t.Fatal("autosave changed circuit/save point")
	}
	s.Close()
	fresh := New()
	defer fresh.Close()
	configureRecovery(t, fresh, dir, false)
	candidate := fresh.Snapshot().Recovery.Candidates[0]
	if candidate.ActiveSession || candidate.SourcePath != source {
		t.Fatalf("closed source was not recoverable: %+v", candidate)
	}
	call(t, fresh, "recovery.inspect", choice, "inspect")
	call(t, fresh, "recovery.restore", choice, "restore")
	restored := settled(t, fresh)
	if !restored.Document.Dirty || restored.Document.Path != "" || restored.Document.RecoveredFrom != source || !reflect.DeepEqual(before.Components, restored.Components) {
		t.Fatalf("bad recovered copy: %+v", restored.Document)
	}
	refuse(t, fresh, "project.save", SaveParams{}, "unavailable")
	copyPath := filepath.Join(t.TempDir(), "copy.elab")
	call(t, fresh, "project.save", SaveParams{Path: copyPath}, "save-copy")
	current, _ := os.ReadFile(source)
	if !bytes.Equal(original, current) {
		t.Fatal("recovery overwrote source")
	}
	call(t, fresh, "recovery.dismiss", choice, "dismiss")
	if len(fresh.Snapshot().Recovery.Candidates) != 0 {
		t.Fatal("dismiss did not remove chosen snapshot")
	}
}
func TestRecoveryCorruptionFallbackAndStaleChoice(t *testing.T) {
	s := NewExample()
	dir := t.TempDir()
	configureRecovery(t, s, dir, false)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit-one")
	call(t, s, "recovery.checkpoint", struct{}{}, "one")
	older := recoveryChoice(t, s)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "3k"}, "edit-two")
	call(t, s, "recovery.checkpoint", struct{}{}, "two")
	newest := recoveryChoice(t, s)
	path := s.recoveryStore.candidatePath(newest.ID)
	data, _ := os.ReadFile(path)
	var capsule recoveryCapsule
	if err := json.Unmarshal(data, &capsule); err != nil {
		t.Fatal(err)
	}
	capsule.DocumentSHA256 = "damaged"
	damaged, _ := json.Marshal(capsule)
	if err := os.WriteFile(path, damaged, 0600); err != nil {
		t.Fatal(err)
	}
	call(t, s, "recovery.scan", struct{}{}, "scan")
	state := s.Snapshot().Recovery
	if len(state.Problems) != 1 || recoveryChoice(t, s).ID != older.ID {
		t.Fatalf("missing valid fallback: %+v", state)
	}
	newest.DiscardChanges = true
	refuse(t, s, "recovery.restore", newest, "invalid-recovery")
	refuse(t, s, "recovery.dismiss", newest, "file-conflict")
	damagedChoice := RecoveryChoiceParams{ID: state.Problems[0].ID, SHA256: state.Problems[0].SHA256}
	call(t, s, "recovery.dismiss", damagedChoice, "dismiss-bad")
	s.Close()
}
func TestRecoveryLiveOwnershipGenerationAndDirtyGuards(t *testing.T) {
	dir := t.TempDir()
	s := NewExample()
	defer s.Close()
	configureRecovery(t, s, dir, false)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	other := New()
	defer other.Close()
	configureRecovery(t, other, dir, false)
	choice := recoveryChoice(t, other)
	if !other.Snapshot().Recovery.Candidates[0].ActiveSession {
		t.Fatal("live owner not detected")
	}
	refuse(t, other, "recovery.dismiss", choice, "unavailable")
	call(t, other, "component.place", PlaceParams{Kind: "resistor", Position: Point{0, 0}, Value: "1k"}, "dirty")
	refuse(t, other, "recovery.restore", choice, "unsaved-changes")
	generation := other.Snapshot().Document.Generation + 1
	choice.DiscardChanges = true
	choice.ExpectedGeneration = &generation
	refuse(t, other, "recovery.restore", choice, "conflict")
	choice.ExpectedGeneration = nil
	choice.SHA256 = "old"
	refuse(t, other, "recovery.restore", choice, "file-conflict")
}
func TestRecoveryCloseAndSaveRejectLateWorkerCaptures(t *testing.T) {
	for _, action := range []string{"save", "discard", "close"} {
		t.Run(action, func(t *testing.T) {
			s := NewExample()
			configureRecovery(t, s, t.TempDir(), true)
			call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
			s.mu.Lock()
			capture := s.recoveryCaptureLocked()
			store := s.recoveryStore
			s.mu.Unlock()
			call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
			switch action {
			case "save":
				call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "saved.elab")}, "save")
			case "discard":
				call(t, s, "app.close", ReplaceParams{DiscardChanges: true}, "discard")
			case "close":
				s.Close()
			}
			if _, _, err := store.checkpoint(capture); err != nil {
				t.Fatal(err)
			}
			candidates, _, err := store.scan()
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if action == "close" {
				want = 1
			}
			if len(candidates) != want {
				t.Fatalf("late worker recreated or lost data: %d want %d", len(candidates), want)
			}
			s.Close()
		})
	}
}
func TestRecoveryAutomaticCheckpointAndRetention(t *testing.T) {
	s := NewExample()
	defer s.Close()
	configureRecovery(t, s, t.TempDir(), true)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	deadline := time.Now().Add(4 * time.Second)
	for len(s.Snapshot().Recovery.Candidates) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	choice := recoveryChoice(t, s)
	if choice.ID == "" {
		t.Fatal("automatic snapshot missing")
	}
	for i, value := range []string{"3k", "4k", "5k"} {
		call(t, s, "component.set", ValueParams{ID: "R1", Value: value}, value)
		call(t, s, "recovery.checkpoint", struct{}{}, value+"-checkpoint")
		if i == 2 {
			entries, _ := filepath.Glob(filepath.Join(s.Snapshot().Recovery.Directory, "*.elab-recovery"))
			if len(entries) != 2 {
				t.Fatalf("retention: %d", len(entries))
			}
		}
	}
	state := s.Snapshot()
	state.Recovery.Candidates[0].SHA256 = "changed"
	if s.Snapshot().Recovery.Candidates[0].SHA256 == "changed" {
		t.Fatal("snapshot aliases recovery metadata")
	}
}
func TestRecoveryFailuresPreserveSession(t *testing.T) {
	s := New()
	defer s.Close()
	refuse(t, s, "recovery.checkpoint", struct{}{}, "unavailable")
	dir := t.TempDir()
	configureRecovery(t, s, dir, false)
	refuse(t, s, "recovery.configure", RecoveryConfigureParams{Directory: dir, Enabled: true, Interval: 0}, "invalid-params")
	for i := 0; i < MaxRecoveryEntries; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("entry-%03d", i)), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	refuse(t, s, "recovery.configure", RecoveryConfigureParams{Directory: dir, Enabled: true, Interval: 2}, "invalid-params")
}

func TestRecoveryPrivateDirectoryAndFailedWriteGuards(t *testing.T) {
	s := NewExample()
	defer s.Close()
	insecure := t.TempDir()
	if err := os.Chmod(insecure, 0755); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "recovery.configure", RecoveryConfigureParams{Directory: insecure, Enabled: true, Interval: 2}, "file-error")
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "recovery.configure", RecoveryConfigureParams{Directory: link, Enabled: true, Interval: 2}, "file-error")
	directory := filepath.Join(t.TempDir(), "recovery")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	configureRecovery(t, s, directory, false)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	moved := directory + "-preserved"
	if err := os.Rename(directory, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(directory, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "recovery.checkpoint", struct{}{}, "file-error")
	if err := os.Remove(directory); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(moved, directory); err != nil {
		t.Fatal(err)
	}
	call(t, s, "recovery.checkpoint", struct{}{}, "retry-checkpoint")
	call(t, s, "recovery.select", IDParams{ID: ""}, "clear-selection")
	call(t, s, "recovery.scan", struct{}{}, "refresh")
	if s.Snapshot().Recovery.Selected != "" {
		t.Fatal("refresh replaced explicit empty selection")
	}
}
