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

func blockInputSolver(t *testing.T, s *Service, fail bool) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s.mu.Lock()
	s.inputSolve = func(ctx context.Context, state *mixedState, trace *Transient, graph Graph, mode, event string) (*mixedState, *Transient, error) {
		close(entered)
		<-release
		if fail {
			return nil, nil, fmt.Errorf("controlled algebraic input failure")
		}
		return solveInputWork(ctx, state, trace, graph, mode, event)
	}
	s.mu.Unlock()
	return entered, func() { once.Do(func() { close(release) }) }
}
func awaitInputStatus(t *testing.T, s *Service, status string) Snapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		state := s.Snapshot()
		if state.InputWork != nil && state.InputWork.Status == status {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatalf("input did not reach %s: %+v", status, state.InputWork)
		}
		time.Sleep(time.Millisecond)
	}
}
func TestBlockedWiperWorkAllowsNavigationSaveAndAtomicCancel(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.step", StepParams{Duration: .01}, "step")
	before := s.Snapshot()
	entered, release := blockInputSolver(t, s, false)
	defer release()
	admitted := promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .9}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	if admitted.InputWork == nil || admitted.InputWork.Actor != "test" || admitted.InputWork.Wiper.Position != .9 || admitted.Mixed.Potentiometers[0].WiperPosition != .5 || admitted.Document != before.Document || !reflect.DeepEqual(admitted.UndoEntry, before.UndoEntry) {
		t.Fatal("input admission publishes staged graph/history/physics")
	}
	promptLiveCommand(t, s, "component.move", MoveParams{ID: "P1", Position: Point{X: 300, Y: -200}})
	promptLiveCommand(t, s, "display.preferences.set", DefaultDisplayPreferences())
	path := filepath.Join(t.TempDir(), "committed.elab")
	promptLiveCommand(t, s, "project.save", SaveParams{Path: path})
	promptLiveCommand(t, s, "simulation.work.cancel", IDParams{ID: admitted.InputWork.ID})
	release()
	state := awaitInputStatus(t, s, "canceled")
	if !reflect.DeepEqual(state.Transient, before.Transient) || state.Mixed.Potentiometers[0].WiperPosition != .5 || state.RunID != before.RunID || state.Document.Dirty {
		t.Fatal("canceled input partially changes committed state")
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	saved := settled(t, fresh)
	if saved.Mixed.Potentiometers[0].WiperPosition != .5 || saved.Components[1].Position != (Point{X: 300, Y: -200}) {
		t.Fatal("save includes staged wiper or loses independent geometry")
	}
}
func TestWiperCompletionMergesPresentationAndMakesOneActorHistoryEntry(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.step", StepParams{Duration: .01}, "step")
	before := s.Snapshot()
	entered, release := blockInputSolver(t, s, false)
	defer release()
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "component.move", MoveParams{ID: "P1", Position: Point{X: 300, Y: -200}})
	release()
	state := awaitInputStatus(t, s, "completed")
	part, _ := s.graph.Component("P1")
	if part.Position != (Point{X: 300, Y: -200}) || *part.WiperPosition != .8 || state.Mixed.Potentiometers[0].WiperPosition != .8 || state.Transient.Sample.Time != before.Transient.Sample.Time || state.Transient.Sample.Voltage != before.Transient.Sample.Voltage || state.RunID != before.RunID || state.UndoEntry.Actor != "test" || state.UndoName != "Adjust wiper" {
		t.Fatal("input commit loses geometry, charge/time or attribution")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	part, _ = s.graph.Component("P1")
	if *part.WiperPosition != .5 || part.Position != (Point{X: 300, Y: -200}) {
		t.Fatal("one input undo reverses independent geometry")
	}
}
func TestLateInputFailureIsDiscardedAfterElectricalEdit(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockInputSolver(t, s, true)
	defer release()
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "P1", Value: "20k"})
	before := s.Snapshot()
	release()
	after := awaitInputStatus(t, s, "superseded")
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || after.Status == "error" || after.InputWork.Error.Code != "superseded" {
		t.Fatal("obsolete input failure overwrites newer circuit")
	}
}
func TestCurrentInputFailurePreservesGraphTimeChargeAndHistory(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	entered, release := blockInputSolver(t, s, true)
	defer release()
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8})
	awaitLiveSampleBlock(t, entered)
	release()
	after := awaitInputStatus(t, s, "failed")
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || after.Document != before.Document || after.InputWork.Error.Code != "solve-failed" {
		t.Fatal("failed current input changes committed state")
	}
}
func TestRunningInputIsNonblockingAndKeepsReadingsFrozenUntilAtomicApply(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	settled(t, s)
	entered, release := blockInputSolver(t, s, false)
	defer release()
	admitted := promptLiveCommand(t, s, "simulation.input.set", InputParams{Mode: "low"}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	before := s.Snapshot()
	time.Sleep(60 * time.Millisecond)
	during := s.Snapshot()
	if before.Transient.Sample.Time != during.Transient.Sample.Time || during.Transient.Mode == "low" || during.Display.Animate {
		t.Fatal("live clock publishes during staged input")
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 10, "dy": 20})
	release()
	after := awaitInputStatus(t, s, "completed")
	if after.RunID != admitted.RunID || after.Transient.Mode != "low" || after.InputWork.Actor != "test" {
		t.Fatal("running network override does not atomically complete")
	}
}
func TestInputAdmissionRetryOwnsSnapshotAndDoesNotLaunchAgain(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockInputSolver(t, s, false)
	defer release()
	raw, _ := json.Marshal(WiperParams{ID: "P1", Position: .8})
	rev := s.Snapshot().CommandRevision
	r := Request{Version: 1, ID: "first", Command: "potentiometer.wiper.set", Params: raw, ExpectedCommandRevision: &rev, Key: "input-key"}
	first := s.Dispatch(context.Background(), r, "actor")
	if first.Error != nil {
		t.Fatal(first.Error)
	}
	awaitLiveSampleBlock(t, entered)
	state := first.Result.(Snapshot)
	refuseStepWork(t, s, "edit.redo", HistoryParams{}, "unavailable")
	if s.Snapshot().InputWork.Status != "running" {
		t.Fatal("rejected history request cancels unrelated input work")
	}
	*state.InputWork.Wiper = WiperParams{ID: "changed", Position: .1}
	r.ID = "retry"
	second := s.Dispatch(context.Background(), r, "actor")
	if !second.Duplicate || second.Result.(Snapshot).InputWork.Wiper.ID != "P1" || second.Result.(Snapshot).InputWork.ID != s.Snapshot().InputWork.ID {
		t.Fatal("retry exposes mutable admission or launches a new input worker")
	}
	release()
	awaitInputStatus(t, s, "completed")
}

func TestVerificationCapturedDuringWiperWorkPredatesPublication(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockInputSolver(t, s, false)
	defer release()
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .8})
	awaitLiveSampleBlock(t, entered)
	duration := .001
	_, record := verify(t, s, VerifyParams{Duration: &duration}, "verify-pending")
	if !record.Operation.SourceUnchanged {
		t.Fatal("verification of committed pending-input state is already mismatched")
	}
	release()
	after := awaitInputStatus(t, s, "completed")
	if inspectOperation(t, s, record.Operation.ID).Operation.SourceUnchanged || after.ActuationRevision != record.Metadata.ActuationRevision+1 {
		t.Fatal("publication keeps old verification associated with changed input")
	}
}

func TestQueuedNetworkInputResumeUsesWorkerAndRetainsPendingOnCancel(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "queue")
	before := s.Snapshot()
	entered, release := blockInputSolver(t, s, false)
	defer release()
	admitted := promptLiveCommand(t, s, "simulation.resume", struct{}{}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	if admitted.InputWork == nil || admitted.InputWork.Mode != "low" || admitted.InputWork.Actor != "test" || admitted.Paused {
		t.Fatal("resume did not admit shared input worker")
	}
	refuseStepWork(t, s, "simulation.step.start", StepParams{Duration: .001}, "busy")
	promptLiveCommand(t, s, "simulation.pause", struct{}{})
	release()
	after := awaitInputStatus(t, s, "canceled")
	if !reflect.DeepEqual(before.Transient, after.Transient) || !after.Paused || after.Transient.PendingMode != "low" {
		t.Fatal("canceled resume partially applied queued input")
	}
}
