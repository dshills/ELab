package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestAsyncStepAllowsEditingSaveAndCancellationWithoutPartialPhysics(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.move", MoveParams{ID: "R1", Position: Point{100, 100}}, "move")
	call(t, s, "simulation.input.set", InputParams{Mode: "low"}, "pending")
	before := s.Snapshot()
	entered, release := blockFirstLiveSample(t, s, false)
	defer release()
	accepted := promptLiveCommand(t, s, "simulation.step.start", StepParams{Duration: .01}).Result.(Snapshot)
	if accepted.StepWork == nil || accepted.StepWork.Status != "queued" || accepted.StepWork.Actor != "test" {
		t.Fatal("step admission not independently inspectable")
	}
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "progress.elab")})
	promptLiveCommand(t, s, "edit.undo", HistoryParams{ExpectedEntryRevision: &before.UndoEntry.CommandRevision})
	refuseStepWork(t, s, "simulation.step.start", StepParams{Duration: .001}, "busy")
	refuseStepWork(t, s, "simulation.work.cancel", IDParams{ID: "obsolete"}, "operation-unavailable")
	promptLiveCommand(t, s, "simulation.work.cancel", IDParams{ID: accepted.StepWork.ID})
	release()
	s.liveWorkers.Wait()
	after := s.Snapshot()
	if after.StepWork.Status != "canceled" || math.Abs(after.StepWork.Progress-float64(after.StepWork.CompletedSamples)/float64(after.StepWork.TotalSamples)) > 1e-12 || !reflect.DeepEqual(after.Mixed, before.Mixed) || !reflect.DeepEqual(after.Transient, before.Transient) || after.Transient.PendingMode != "low" || after.View == before.View {
		t.Fatal("canceled step leaked staged physics or lost editing")
	}
	accepted.StepWork.Status = "corrupted"
	if s.Snapshot().StepWork.Status != "canceled" {
		t.Fatal("step snapshot aliases service")
	}
}

func TestStepSupersededByRunInputOrProjectReplacement(t *testing.T) {
	for _, command := range []string{"simulation.restart", "simulation.resume", "simulation.input.set", "switch.set", "component.set", "example.rc", "app.close.start"} {
		t.Run(command, func(t *testing.T) {
			s := newGraphService(SolveLED, switchRCGraph(t, true))
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			entered, release := blockFirstLiveSample(t, s, true)
			defer release()
			call(t, s, "simulation.step.start", StepParams{Duration: .01}, "start")
			awaitLiveSampleBlock(t, entered)
			var p any = struct{}{}
			switch command {
			case "simulation.input.set":
				p = InputParams{Mode: "low"}
			case "switch.set":
				p = SwitchParams{ID: "S1", Closed: false}
			case "component.set":
				p = ValueParams{ID: "R1", Value: "20k"}
			case "example.rc", "app.close.start":
				p = ReplaceParams{DiscardChanges: true}
			}
			promptLiveCommand(t, s, command, p)
			if command == "simulation.resume" || command == "example.rc" {
				promptLiveCommand(t, s, "simulation.pause", struct{}{})
			}
			before := awaitTestInitialization(t, s)
			release()
			s.liveWorkers.Wait()
			after := s.Snapshot()
			if after.StepWork.Status != "superseded" && after.StepWork.Status != "canceled" || !reflect.DeepEqual(after.Mixed, before.Mixed) || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Components, before.Components) {
				t.Fatal("obsolete step overwrote current physics/project")
			}
			if after.StepWork.Error == nil || after.StepWork.Error.Message == "obsolete sample failure" {
				t.Fatal("obsolete numerical error won over cancellation")
			}
		})
	}
}

func TestLegacyStepLateFailureIsAtomicAndRetryReturnsItsOriginalOutcome(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	calls := 0
	s.mu.Lock()
	s.networkAdvance = func(state *mixedState, trace *Transient, end float64, g Graph) (*mixedState, *Transient, error) {
		calls++
		if calls == 2 {
			close(entered)
			<-release
			return nil, nil, fmt.Errorf("injected failure after staged sample")
		}
		return advanceLiveNetwork(state, trace, end, g)
	}
	s.mu.Unlock()
	rev := before.Revision
	request := Request{Version: 1, ID: "legacy", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":0.01}`), ExpectedRevision: &rev, Key: "legacy"}
	result := make(chan Response, 1)
	go func() { result <- s.Dispatch(context.Background(), request, "test") }()
	awaitLiveSampleBlock(t, entered)
	progress := s.Snapshot()
	if progress.StepWork.CompletedSamples != 1 || !reflect.DeepEqual(progress.Mixed, before.Mixed) || !reflect.DeepEqual(progress.Transient, before.Transient) {
		t.Fatal("partial step became committed physical state")
	}
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	// Release once, preserving a deferred cleanup if an assertion above fails.
	unblock()
	out := <-result
	if out.Error == nil || out.Error.Code != "solve-failed" || out.Result.(Snapshot).StepWork.Status != "failed" || !reflect.DeepEqual(s.Snapshot().Mixed, before.Mixed) || !reflect.DeepEqual(s.Snapshot().Transient, before.Transient) {
		t.Fatal("late failure partially advanced state")
	}
	originalFailure := *out.Error
	out.Error.Message = "caller-corrupted"
	request.ID = "retry"
	duplicate := s.Dispatch(context.Background(), request, "test")
	if !duplicate.Duplicate || !reflect.DeepEqual(duplicate.Result, out.Result) || !reflect.DeepEqual(duplicate.Error, &originalFailure) || calls != 2 {
		t.Fatal("legacy retry recomputed or lost terminal outcome")
	}
}

func refuseStepWork(t *testing.T, s *Service, command string, p any, code string) {
	t.Helper()
	raw, _ := json.Marshal(p)
	rev := s.Snapshot().CommandRevision
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: command, Command: command, Params: raw, ExpectedCommandRevision: &rev, Key: command + code}, "test")
	if r.Error == nil || r.Error.Code != code {
		t.Fatalf("%s expected %s, got %+v", command, code, r.Error)
	}
}

func TestLegacyWaitCancellationLeavesAdmittedStepAndRetryIntact(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockFirstLiveSample(t, s, false)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rev := s.Snapshot().CommandRevision
	r := Request{Version: 1, ID: "waiting", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":0.001}`), ExpectedCommandRevision: &rev, Key: "waiting"}
	result := make(chan Response, 1)
	go func() { result <- s.Dispatch(ctx, r, "test") }()
	awaitLiveSampleBlock(t, entered)
	cancel()
	response := <-result
	if response.Error == nil || response.Error.Code != "canceled" || s.Snapshot().StepWork.Status != "running" {
		t.Fatal("request wait canceled admitted work")
	}
	release()
	s.liveWorkers.Wait()
	done := s.Snapshot()
	if done.StepWork.Status != "completed" || done.Transient.Sample.Time != .001 {
		t.Fatal("admitted worker did not finish")
	}
	retry := s.Dispatch(context.Background(), r, "test")
	if retry.Error != nil || !retry.Duplicate || retry.Result.(Snapshot).StepWork.ID != done.StepWork.ID || retry.Result.(Snapshot).Transient.Sample.Time != .001 {
		t.Fatal("reconnected legacy retry lost completed outcome")
	}
}

func TestWiperEventSupersedesStagedStepWithoutBeingOverwritten(t *testing.T) {
	s := newGraphService(SolveLED, potGraph(t, "capacitor"))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockFirstLiveSample(t, s, true)
	defer release()
	call(t, s, "simulation.step.start", StepParams{Duration: .01}, "start")
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: .75})
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().InputWork.Status != "completed" {
		if time.Now().After(deadline) {
			t.Fatal("staged wiper did not complete while the obsolete manual step remained blocked")
		}
		time.Sleep(time.Millisecond)
	}
	before := s.Snapshot()
	release()
	s.liveWorkers.Wait()
	after := s.Snapshot()
	if after.StepWork.Status != "superseded" || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.Transient, after.Transient) || after.Mixed.Potentiometers[0].WiperPosition != .75 {
		t.Fatal("obsolete step overwrote wiper input")
	}
}
