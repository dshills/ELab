package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func recoveryDismissFixture(t *testing.T) (string, RecoveryChoiceParams) {
	t.Helper()
	source := NewExample()
	defer source.Close()
	settled(t, source)
	directory := recoverySetupDirectory(t)
	configureRecovery(t, source, directory, false)
	call(t, source, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, source)
	source.Close()
	return directory, choice
}
func recoveryDismissService(t *testing.T) (*Service, RecoveryChoiceParams, string) {
	t.Helper()
	directory, choice := recoveryDismissFixture(t)
	s := pausedRC(t)
	configureRecovery(t, s, directory, false)
	return s, choice, filepath.Join(directory, choice.ID+".elab-recovery")
}
func blockRecoveryDismissPreparation(t *testing.T, s *Service) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.recoveryDismissPrepare = func(ctx context.Context, capture recoveryDismissCapture, progress func(string, int)) (*preparedRecoveryDismiss, error) {
		prepared, err := prepareRecoveryDismiss(ctx, capture, progress)
		close(entered)
		<-release
		return prepared, err
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func TestRecoveryDismissAllowsEditingThenRemovesOnlyCapturedSnapshot(t *testing.T) {
	s, choice, path := recoveryDismissService(t)
	defer s.Close()
	entered, release := blockRecoveryDismissPreparation(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "recovery.dismiss.start", choice).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	if *admitted.ProjectWork.RecoveryDismissed || admitted.ProjectWork.RecoveryID != choice.ID {
		t.Fatal("admission claims deletion")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("preparation removed snapshot", err)
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 3, "dy": 4})
	call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	before := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	if !*after.ProjectWork.RecoveryDismissed || after.ProjectWork.SourceSHA256 != choice.SHA256 || after.ProjectWork.Actor != "test" || len(after.Recovery.Candidates) != 0 {
		t.Fatal("dismiss lacks exact result/provenance")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("confirmed snapshot not removed", err)
	}
	if !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Transient, before.Transient) || after.Document != before.Document || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) || after.View != before.View {
		t.Fatal("dismiss changes later circuit/physics/history")
	}
	*inspectProjectWork(t, s, after.ProjectWork.ID).RecoveryDismissed = false
	if !*inspectProjectWork(t, s, after.ProjectWork.ID).RecoveryDismissed {
		t.Fatal("inspection aliases deletion outcome")
	}
}
func TestRecoveryDismissCancelAndClosePreserveFile(t *testing.T) {
	for _, closeService := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "close"}[closeService], func(t *testing.T) {
			s, choice, path := recoveryDismissService(t)
			defer s.Close()
			data, _ := os.ReadFile(path)
			entered, release := blockRecoveryDismissPreparation(t, s)
			defer release()
			admitted := promptLiveCommand(t, s, "recovery.dismiss.start", choice).Result.(Snapshot)
			awaitLiveSampleBlock(t, entered)
			var closed chan struct{}
			if closeService {
				closed = make(chan struct{})
				go func() { s.Close(); close(closed) }()
				deadline := time.Now().Add(time.Second)
				for s.Snapshot().Status != "closed" {
					if time.Now().After(deadline) {
						t.Fatal("close did not cancel dismissal")
					}
					time.Sleep(time.Millisecond)
				}
			} else {
				promptLiveCommand(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID})
			}
			release()
			after := awaitProjectStatus(t, s, "canceled")
			if closed != nil {
				select {
				case <-closed:
				case <-time.After(time.Second):
					t.Fatal("close did not join dismissal")
				}
			}
			current, err := os.ReadFile(path)
			if err != nil || !reflect.DeepEqual(current, data) || *after.ProjectWork.RecoveryDismissed {
				t.Fatal("canceled dismissal changed snapshot", err)
			}
		})
	}
}
func TestRecoveryDismissRechecksContentIdentityAndOwnership(t *testing.T) {
	for _, change := range []string{"content", "identity", "owner", "symlink"} {
		t.Run(change, func(t *testing.T) {
			s, choice, path := recoveryDismissService(t)
			defer s.Close()
			data, _ := os.ReadFile(path)
			entered, release := blockRecoveryDismissPreparation(t, s)
			defer release()
			promptLiveCommand(t, s, "recovery.dismiss.start", choice)
			awaitLiveSampleBlock(t, entered)
			code := "file-conflict"
			switch change {
			case "content":
				if err := os.WriteFile(path, append(data, ' '), 0600); err != nil {
					t.Fatal(err)
				}
			case "identity":
				other := path + ".replacement"
				if err := os.WriteFile(other, data, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(other, path); err != nil {
					t.Fatal(err)
				}
			case "owner":
				var capsule recoveryCapsule
				_ = json.Unmarshal(data, &capsule)
				lease, _ := json.Marshal(recoveryLease{Session: capsule.Session, PID: os.Getpid()})
				if err := os.WriteFile(filepath.Join(filepath.Dir(path), "session-"+capsule.Session+".json"), lease, 0600); err != nil {
					t.Fatal(err)
				}
				code = "unavailable"
			case "symlink":
				if err := os.Rename(path, path+".preserved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".preserved", path); err != nil {
					t.Fatal(err)
				}
				code = "file-error"
			}
			release()
			after := awaitProjectStatus(t, s, "failed")
			if after.ProjectWork.Error.Code != code || *after.ProjectWork.RecoveryDismissed {
				t.Fatal("changed file guard failed", after.ProjectWork)
			}
			if _, err := os.Lstat(path); err != nil {
				t.Fatal("changed file was removed", err)
			}
		})
	}
}
func TestRecoveryDismissCommitAndScanStayOutsideEditorLock(t *testing.T) {
	s, choice, path := recoveryDismissService(t)
	defer s.Close()
	store := configuredStore(t, s)
	entered, release := blockRecoveryDismissPreparation(t, s)
	defer release()
	admitted := promptLiveCommand(t, s, "recovery.dismiss.start", choice).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	store.mu.Lock()
	var once sync.Once
	unlock := func() { once.Do(store.mu.Unlock) }
	defer unlock()
	release()
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().ProjectWork.Phase != "dismissing-recovery" {
		if time.Now().After(deadline) {
			t.Fatal("dismiss did not reach commit")
		}
		time.Sleep(time.Millisecond)
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 1, "dy": 2})
	call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
	refuseStepWork(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID}, "operation-committing")
	unlock()
	after := awaitProjectStatus(t, s, "completed")
	if !*after.ProjectWork.RecoveryDismissed {
		t.Fatal("commit not reported")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
func TestRecoveryDismissPreservesNewerCatalogAndReportsFollowupFailure(t *testing.T) {
	for _, scanFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "newer-catalog", true: "scan-failure"}[scanFailure], func(t *testing.T) {
			s, choice, _ := recoveryDismissService(t)
			defer s.Close()
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			s.mu.Lock()
			s.recoveryDismissExecute = func(capture recoveryDismissCapture, prepared *preparedRecoveryDismiss) (recoveryDismissResult, error) {
				close(entered)
				<-release
				result, err := executeRecoveryDismiss(capture, prepared)
				if scanFailure {
					result.scanError = errors.New("controlled follow-up scan failure")
				}
				return result, err
			}
			s.mu.Unlock()
			promptLiveCommand(t, s, "recovery.dismiss.start", choice)
			awaitLiveSampleBlock(t, entered)
			s.mu.Lock()
			s.recovery.Problems = append(s.recovery.Problems, RecoveryProblem{ID: "newer-problem", Message: "preserve"})
			s.recoveryCatalogRevision++
			s.mu.Unlock()
			promptLiveCommand(t, s, "recovery.select", IDParams{ID: ""})
			unblock()
			after := awaitProjectStatus(t, s, "completed")
			if !*after.ProjectWork.RecoveryDismissed || *after.ProjectWork.CatalogApplied || len(after.Recovery.Candidates) != 0 || after.Recovery.Selected != "" || len(after.Recovery.Problems) != 1 || after.Recovery.Problems[0].ID != "newer-problem" {
				t.Fatal("dismiss overwrites newer catalog/selection")
			}
			if scanFailure && (after.ProjectWork.CleanupWarning == "" || after.Recovery.Error == "") {
				t.Fatal("successful deletion hides follow-up failure")
			}
		})
	}
}
func TestRecoveryDismissWaitingRetryAndDocumentReplacement(t *testing.T) {
	s, choice, _ := recoveryDismissService(t)
	defer s.Close()
	entered, release := blockRecoveryDismissPreparation(t, s)
	defer release()
	raw, _ := json.Marshal(choice)
	rev := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "dismiss", Command: "recovery.dismiss", Key: "dismiss", Params: raw, ExpectedCommandRevision: &rev}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Response, 1)
	go func() { done <- s.Dispatch(ctx, request, "dismisser") }()
	awaitLiveSampleBlock(t, entered)
	cancel()
	select {
	case response := <-done:
		if response.Error == nil || response.Error.Code != "canceled" {
			t.Fatal("wait did not cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("request wait blocked")
	}
	promptLiveCommand(t, s, "project.clear", ReplaceParams{DiscardChanges: true})
	generation := s.Snapshot().Document.Generation
	release()
	after := awaitProjectStatus(t, s, "completed")
	if len(after.Components) != 0 || after.Document.Generation != generation || !*after.ProjectWork.RecoveryDismissed {
		t.Fatal("directory-scoped dismissal alters replacement")
	}
	retry := s.Dispatch(context.Background(), request, "dismisser")
	if !retry.Duplicate || retry.Error != nil || retry.Result.(Snapshot).ProjectWork.ID != after.ProjectWork.ID || after.ProjectWork.Actor != "dismisser" {
		t.Fatal("retry duplicates removal or loses actor")
	}
}

func TestRecoveryDismissRejectsMissingPreparationOrRemovalOutcome(t *testing.T) {
	for _, stage := range []string{"prepare", "execute"} {
		t.Run(stage, func(t *testing.T) {
			s, choice, path := recoveryDismissService(t)
			defer s.Close()
			s.mu.Lock()
			if stage == "prepare" {
				s.recoveryDismissPrepare = func(context.Context, recoveryDismissCapture, func(string, int)) (*preparedRecoveryDismiss, error) {
					return nil, nil
				}
			} else {
				s.recoveryDismissExecute = func(recoveryDismissCapture, *preparedRecoveryDismiss) (recoveryDismissResult, error) {
					return recoveryDismissResult{}, nil
				}
			}
			s.mu.Unlock()
			refuse(t, s, "recovery.dismiss", choice, "file-error")
			if *s.Snapshot().ProjectWork.RecoveryDismissed {
				t.Fatal("missing outcome claims removal")
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatal("missing outcome removed fixture", err)
			}
		})
	}
}
