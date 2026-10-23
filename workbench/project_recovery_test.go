package workbench

import (
	"context"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func manualRecoveryService(t *testing.T) *Service {
	t.Helper()
	s := pausedRC(t)
	// Recovery fixtures exercise unsaved work, rather than an untouched template.
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "12k"}, "initial-edit")
	call(t, s, "simulation.restart", struct{}{}, "restart-edited")
	call(t, s, "recovery.configure", RecoveryConfigureParams{Directory: filepath.Join(t.TempDir(), "recovery"), Enabled: false, Interval: 1}, "configure")
	return s
}
func blockRecoveryPreparation(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.recoveryPrepare = func(ctx context.Context, capture manualRecoveryCapture, progress func(string, int)) ([]byte, error) {
		progress("serializing-recovery", 0)
		close(entered)
		<-release
		return prepareManualRecovery(ctx, capture, progress)
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func TestBlockedRecoveryCheckpointCapturesOriginalWhileEditingAndSteppingContinue(t *testing.T) {
	s := manualRecoveryService(t)
	defer s.Close()
	before := s.Snapshot()
	entered, release := blockRecoveryPreparation(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "recovery.work.start", RecoveryWorkParams{Kind: "checkpoint"}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	if len(admitted.Recovery.Candidates) != 0 || admitted.ProjectWork.Status != "queued" {
		t.Fatal("recovery admission publishes staged capsule")
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	call(t, s, "simulation.step", StepParams{Duration: .005}, "step")
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	edited := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.RecoveryID == "" || !reflect.DeepEqual(after.Components, edited.Components) || !reflect.DeepEqual(after.Transient, edited.Transient) || after.Document != edited.Document || !reflect.DeepEqual(after.UndoEntry, edited.UndoEntry) || after.View != edited.View {
		t.Fatal("checkpoint loses original capture identity or changes later editor state")
	}
	s.mu.Lock()
	store := s.recoveryStore
	s.mu.Unlock()
	_, d, _, err := store.read(after.ProjectWork.RecoveryID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Graph.Components, before.Components) || d.View != before.View {
		t.Fatal("capsule includes later graph/view instead of admitted capture")
	}
}
func TestRecoveryPreparationCancellationAndReplacementPreserveCapsules(t *testing.T) {
	for _, command := range []string{"project.work.cancel", "project.clear"} {
		t.Run(command, func(t *testing.T) {
			s := manualRecoveryService(t)
			defer s.Close()
			call(t, s, "recovery.checkpoint", struct{}{}, "baseline")
			beforeCandidates := s.Snapshot().Recovery.Candidates
			entered, release := blockRecoveryPreparation(t, s)
			defer release()
			admitted := promptLiveCommand(t, s, "recovery.work.start", RecoveryWorkParams{Kind: "checkpoint"}).Result.(Snapshot)
			awaitLiveSampleBlock(t, entered)
			var p any = ReplaceParams{DiscardChanges: true}
			status := "superseded"
			if command == "project.work.cancel" {
				p = IDParams{ID: admitted.ProjectWork.ID}
				status = "canceled"
			}
			promptLiveCommand(t, s, command, p)
			before := s.Snapshot()
			release()
			after := awaitProjectStatus(t, s, status)
			if after.Document != before.Document || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Recovery.Candidates, beforeCandidates) || after.ProjectWork.RecoveryID != "" {
				t.Fatal("canceled preparation writes/retires data or changes replacement")
			}
		})
	}
}
func TestBlockedRecoveryScanPreservesLaterSelection(t *testing.T) {
	s := manualRecoveryService(t)
	defer s.Close()
	call(t, s, "recovery.checkpoint", struct{}{}, "baseline")
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	s.mu.Lock()
	s.recoveryExecute = func(capture manualRecoveryCapture, data []byte) (recoveryWorkResult, error) {
		close(entered)
		<-release
		return executeManualRecovery(capture, data)
	}
	s.mu.Unlock()
	promptLiveCommand(t, s, "recovery.work.start", RecoveryWorkParams{Kind: "scan"})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "recovery.select", IDParams{ID: ""})
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	before := s.Snapshot()
	unblock()
	after := awaitProjectStatus(t, s, "completed")
	if after.Recovery.Selected != "" || after.Document != before.Document || after.View != before.View || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("scan overrides clear selection or alters project")
	}
}
func TestCheckpointPublicationFencesCancelAndPreservesReplacementMetadata(t *testing.T) {
	s := manualRecoveryService(t)
	defer s.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	s.mu.Lock()
	s.recoveryExecute = func(capture manualRecoveryCapture, data []byte) (recoveryWorkResult, error) {
		close(entered)
		<-release
		return executeManualRecovery(capture, data)
	}
	s.mu.Unlock()
	admitted := promptLiveCommand(t, s, "recovery.work.start", RecoveryWorkParams{Kind: "checkpoint"}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	if inspectProjectWork(t, s, admitted.ProjectWork.ID).Cancelable {
		t.Fatal("publishing checkpoint offers false cancellation")
	}
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID}, "operation-committing")
	promptLiveCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true})
	before := s.Snapshot()
	unblock()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.Associated == nil || *after.ProjectWork.Associated || after.Document != before.Document || !reflect.DeepEqual(after.Recovery, before.Recovery) || after.ProjectWork.RecoveryID == "" {
		t.Fatal("publishing older capture relabels/changes replacement")
	}
}
func TestLateManualScanDoesNotOverwriteNewerRecoveryCatalog(t *testing.T) {
	s := manualRecoveryService(t)
	defer s.Close()
	call(t, s, "recovery.checkpoint", struct{}{}, "baseline")
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	s.mu.Lock()
	s.recoveryExecute = func(capture manualRecoveryCapture, data []byte) (recoveryWorkResult, error) {
		result, err := executeManualRecovery(capture, data)
		close(entered)
		<-release
		return result, err
	}
	s.mu.Unlock()
	promptLiveCommand(t, s, "recovery.work.start", RecoveryWorkParams{Kind: "scan"})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	s.mu.Lock()
	capture := s.recoveryCaptureLocked()
	store := s.recoveryStore
	s.mu.Unlock()
	if _, _, err := store.checkpoint(capture); err != nil {
		t.Fatal(err)
	}
	// Simulate the real automatic publisher after its separate filesystem pass.
	candidates, problems, err := store.scan()
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.recovery.Candidates = candidates
	s.recovery.Problems = problems
	s.recoveryCatalogRevision++
	s.mu.Unlock()
	newer := s.Snapshot().Recovery
	unblock()
	after := awaitProjectStatus(t, s, "completed")
	if !reflect.DeepEqual(after.Recovery, newer) || after.ProjectWork.CatalogApplied == nil || *after.ProjectWork.CatalogApplied {
		t.Fatal("late scan replaces a newer published recovery list")
	}
}
