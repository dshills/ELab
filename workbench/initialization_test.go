package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func blockFirstInitialization(t *testing.T, s *Service, failure bool) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var once, finish sync.Once
	s.mu.Lock()
	s.initialSolver = func(ctx context.Context, p kernelPlan) (*simulationKernel, error) {
		blocked := false
		once.Do(func() { blocked = true; close(entered) })
		if blocked {
			<-release
			if failure {
				return nil, fmt.Errorf("obsolete initialization failure")
			}
		}
		return solveInitial(ctx, p)
	}
	s.mu.Unlock()
	return entered, func() { finish.Do(func() { close(release) }) }
}

func TestBlockedInitializationAllowsNavigationSaveAndPause(t *testing.T) {
	for _, example := range []string{"example.flasher", "example.npn-pulse"} {
		t.Run(example, func(t *testing.T) {
			s := New()
			defer s.Close()
			entered, release := blockFirstInitialization(t, s, false)
			defer release()
			accepted := promptLiveCommand(t, s, example, ReplaceParams{}).Result.(Snapshot)
			if accepted.Status != "updating" || accepted.Mixed != nil || accepted.NPN != nil {
				t.Fatal("initialization admission blocked or published partial data")
			}
			awaitLiveSampleBlock(t, entered)
			promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
			promptLiveCommand(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "initializing.elab")})
			promptLiveCommand(t, s, "simulation.pause", struct{}{})
			paused := s.Snapshot()
			release()
			s.liveWorkers.Wait()
			after := s.Snapshot()
			if after.Revision != paused.Revision || !reflect.DeepEqual(after.Mixed, paused.Mixed) || !reflect.DeepEqual(after.NPN, paused.NPN) || after.Status != "paused" {
				t.Fatal("initialization published after pause")
			}
			call(t, s, "simulation.restart", struct{}{}, "restart")
			ready := settled(t, s)
			if ready.Status != "paused" || ready.Transient == nil || ready.Transient.Stale || ready.Transient.Sample.Time != 0 || ready.View != after.View {
				t.Fatal("paused restart did not initialize fresh without losing view")
			}
		})
	}
}

func TestInitializationSupersededByUndoNewSourceAndClose(t *testing.T) {
	for _, command := range []string{"edit.undo", "component.set", "app.close.start"} {
		t.Run(command, func(t *testing.T) {
			s := New()
			defer s.Close()
			entered, release := blockFirstInitialization(t, s, true)
			defer release()
			promptLiveCommand(t, s, "example.flasher", ReplaceParams{})
			awaitLiveSampleBlock(t, entered)
			var p any = struct{}{}
			if command == "component.set" {
				p = ValueParams{ID: "V1", Value: "3V"}
			}
			if command == "app.close.start" {
				p = ReplaceParams{DiscardChanges: true}
			}
			promptLiveCommand(t, s, command, p)
			if command == "component.set" {
				awaitTestInitialization(t, s)
				promptLiveCommand(t, s, "simulation.pause", struct{}{})
			}
			before := s.Snapshot()
			release()
			s.liveWorkers.Wait()
			if command == "app.close.start" {
				s.Close()
			}
			after := s.Snapshot()
			if (command != "app.close.start" && before.Revision != after.Revision) || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Diagnostics, after.Diagnostics) || !reflect.DeepEqual(before.Mixed, after.Mixed) {
				t.Fatal("obsolete initializer replaced current run or failure")
			}
		})
	}
}

func TestInitializationPublishesAcrossPresentationEditsAndAvoidsRepeatedResume(t *testing.T) {
	s := New()
	defer s.Close()
	entered, release := blockFirstInitialization(t, s, false)
	defer release()
	promptLiveCommand(t, s, "example.flasher", ReplaceParams{})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "component.move", MoveParams{ID: "V1", Position: Point{100, 100}})
	before := s.Snapshot()
	promptLiveCommand(t, s, "simulation.resume", struct{}{})
	if s.Snapshot().RunID != before.RunID {
		t.Fatal("repeated resume restarted initialization")
	}
	release()
	ready := settled(t, s)
	if ready.Mixed == nil || ready.Mixed.Stale || !reflect.DeepEqual(ready.Components, before.Components) || ready.RunID != before.RunID {
		t.Fatal("initial publication lost presentation work")
	}
	call(t, s, "simulation.pause", struct{}{}, "finish")
}

func TestInitializationDeadlinePublishesErrorInsteadOfStayingUpdating(t *testing.T) {
	s := New()
	defer s.Close()
	s.mu.Lock()
	s.initialTimeout = 10 * time.Millisecond
	s.initialSolver = func(ctx context.Context, _ kernelPlan) (*simulationKernel, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	s.mu.Unlock()
	promptLiveCommand(t, s, "example.flasher", ReplaceParams{})
	state := settled(t, s)
	if state.Status != "error" || state.Mixed != nil || len(state.Diagnostics) != 1 || !strings.Contains(state.Diagnostics[0].Message, "deadline exceeded") {
		t.Fatal("expired startup remained updating or published partial data")
	}
}

func TestNetworkNewtonObservesCanceledContext(t *testing.T) {
	c, _, d := FlasherExampleGraph().Compile()
	if d != nil {
		t.Fatal(d)
	}
	n := newMixedNetwork(c)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state, err := n.solveContext(ctx, nil, make([]float64, len(n.caps)), 0, &Transient{Settings: DefaultTransientSettings(), Mode: "auto"}, 0)
	if state != nil || err != context.Canceled {
		t.Fatal("network initialization ignored cancellation")
	}
}

func TestPausedRestartRefreshesBothDCPaths(t *testing.T) {
	for _, example := range []string{"example.reset", "example.npn"} {
		t.Run(example, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, example, ReplaceParams{}, "example")
			settled(t, s)
			call(t, s, "simulation.pause", struct{}{}, "pause")
			id := s.Snapshot().Components[0].ID
			call(t, s, "component.set", ValueParams{ID: id, Value: "3V"}, "edit")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			ready := settled(t, s)
			if !ready.Paused || ready.Status != "paused" || ready.Result != nil && ready.Result.Stale || ready.NPN != nil && ready.NPN.Stale {
				t.Fatal("paused DC restart did not produce fresh readings")
			}
		})
	}
}

func TestMalformedLegacyStepDoesNotWaitForInitialization(t *testing.T) {
	s := New()
	defer s.Close()
	entered, release := blockFirstInitialization(t, s, false)
	defer release()
	promptLiveCommand(t, s, "example.flasher", ReplaceParams{})
	awaitLiveSampleBlock(t, entered)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	rev := s.Snapshot().CommandRevision
	r := s.Dispatch(ctx, Request{Version: 1, ID: "invalid", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":"bad"}`), ExpectedCommandRevision: &rev, Key: "invalid"}, "test")
	if r.Error == nil || r.Error.Code != "invalid-params" || ctx.Err() != nil {
		t.Fatal("malformed step waited for initialization")
	}
}

func TestCachedLegacyStepDoesNotWaitForUnrelatedInitialization(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	original := call(t, s, "simulation.step", StepParams{Duration: .001}, "cached")
	entered, release := blockFirstInitialization(t, s, false)
	defer release()
	promptLiveCommand(t, s, "example.flasher", ReplaceParams{DiscardChanges: true})
	awaitLiveSampleBlock(t, entered)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	rev := before.Revision
	r := s.Dispatch(ctx, Request{Version: 1, ID: "retry", Command: "simulation.step", Params: json.RawMessage(`{"duration_s":0.001}`), ExpectedRevision: &rev, Key: "cached"}, "test")
	if r.Error != nil || !r.Duplicate || ctx.Err() != nil || !reflect.DeepEqual(r.Result, original.Result) {
		t.Fatal("cached retry waited for unrelated initialization")
	}
}
