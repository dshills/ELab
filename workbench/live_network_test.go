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

// Block at the numerical boundary, then run the real kernel (or return an
// obsolete failure). This proves service availability independently of how
// fast this host happens to solve the small fixture.
func blockFirstLiveSample(t *testing.T, s *Service, obsoleteError bool) (<-chan struct{}, func()) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	var first, finish sync.Once
	s.mu.Lock()
	s.networkAdvance = func(state *mixedState, trace *Transient, end float64, graph Graph) (*mixedState, *Transient, error) {
		blocked := false
		first.Do(func() { blocked = true; close(entered) })
		if blocked {
			<-release
			if obsoleteError {
				return nil, nil, fmt.Errorf("obsolete sample failure")
			}
		}
		return advanceLiveNetwork(state, trace, end, graph)
	}
	s.mu.Unlock()
	return entered, func() { finish.Do(func() { close(release) }) }
}

func promptLiveCommand(t *testing.T, s *Service, command string, p any) Response {
	t.Helper()
	result := make(chan Response, 1)
	started := time.Now()
	go func() {
		raw, _ := json.Marshal(p)
		rev := s.Snapshot().Revision
		r := Request{Version: 1, ID: command, Command: command, Params: raw, ExpectedRevision: &rev, Key: command + fmt.Sprint(started.UnixNano())}
		result <- s.Dispatch(context.Background(), r, "test")
	}()
	select {
	case r := <-result:
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		t.Logf("%s during blocked kernel: %s", command, time.Since(started))
		return r
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("%s blocked behind numerical work", command)
		return Response{}
	}
}

func awaitLiveSampleBlock(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("live sample did not enter numerical boundary")
	}
}

func TestBlockedLiveNetworkAllowsNavigationSaveUndoAndPause(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.move", MoveParams{ID: "R1", Position: Point{100, 100}}, "move")
	entered, release := blockFirstLiveSample(t, s, false)
	defer release()
	call(t, s, "simulation.resume", struct{}{}, "resume")
	awaitLiveSampleBlock(t, entered)
	before := s.Snapshot()
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "drawing.cancel", struct{}{})
	promptLiveCommand(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "working.elab")})
	promptLiveCommand(t, s, "edit.undo", HistoryParams{ExpectedEntryRevision: &before.UndoEntry.CommandRevision})
	promptLiveCommand(t, s, "simulation.pause", struct{}{})
	paused := s.Snapshot()
	if paused.RunID != before.RunID || paused.Transient.Sample.Time != before.Transient.Sample.Time || !paused.Paused || paused.View == before.View {
		t.Fatal("responsive presentation commands changed physics or failed to apply")
	}
	release()
	s.liveWorkers.Wait()
	after := s.Snapshot()
	if !reflect.DeepEqual(after.Transient, paused.Transient) || !reflect.DeepEqual(after.Mixed, paused.Mixed) || after.Revision != paused.Revision {
		t.Fatal("sample published after pause")
	}
}

func TestLiveNetworkDiscardsObsoleteFailureAfterQueuedSwitch(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockFirstLiveSample(t, s, true)
	defer release()
	call(t, s, "simulation.resume", struct{}{}, "resume")
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "switch.set", SwitchParams{ID: "S1", Closed: false})
	queued := s.Snapshot()
	release()
	deadline := time.Now().Add(time.Second)
	for {
		state := s.Snapshot()
		if state.Status == "error" || state.Transient.Stale {
			t.Fatal("obsolete failure invalidated current inputs")
		}
		if !state.SwitchStates[0].Pending {
			if *state.SwitchStates[0].AppliedClosed || state.RunID != queued.RunID || state.Transient.Events[len(state.Transient.Events)-1].Time != .001 {
				t.Fatal("final desired input did not apply at first accepted sample")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("new input never accepted")
		}
		time.Sleep(time.Millisecond)
	}
	call(t, s, "simulation.pause", struct{}{}, "finish")
}

func TestLiveNetworkDiscardsWorkAfterRestartAndClose(t *testing.T) {
	for _, command := range []string{"simulation.restart", "app.close.start"} {
		t.Run(command, func(t *testing.T) {
			s := newGraphService(SolveLED, switchRCGraph(t, true))
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			entered, release := blockFirstLiveSample(t, s, true)
			defer release()
			call(t, s, "simulation.resume", struct{}{}, "resume")
			awaitLiveSampleBlock(t, entered)
			var p any = struct{}{}
			if command == "app.close.start" {
				p = ReplaceParams{DiscardChanges: true}
			}
			promptLiveCommand(t, s, command, p)
			if command == "simulation.restart" {
				promptLiveCommand(t, s, "simulation.pause", struct{}{})
			}
			before := s.Snapshot()
			release()
			s.liveWorkers.Wait()
			if command == "app.close.start" {
				s.Close()
			}
			after := s.Snapshot()
			if (command != "app.close.start" && after.Revision != before.Revision) || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Diagnostics, before.Diagnostics) {
				t.Fatal("obsolete worker changed restarted/closed session")
			}
		})
	}
}

func TestLiveNetworkAcceptsSampleAcrossPresentationEdits(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockFirstLiveSample(t, s, false)
	defer release()
	call(t, s, "simulation.resume", struct{}{}, "resume")
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	promptLiveCommand(t, s, "component.move", MoveParams{ID: "R1", Position: Point{100, 100}})
	before := s.Snapshot()
	release()
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Transient.Sample.Time == 0 {
		if time.Now().After(deadline) {
			t.Fatal("presentation edit prevented live publication")
		}
		time.Sleep(time.Millisecond)
	}
	call(t, s, "simulation.pause", struct{}{}, "finish")
	s.liveWorkers.Wait()
	after := s.Snapshot()
	if before.View != after.View || !reflect.DeepEqual(before.Components, after.Components) || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID || after.Mixed.Stale || after.Revision <= before.Revision {
		t.Fatal("live sample overwrote presentation work or lost physical identity")
	}
}

func TestRepeatedResumeKeepsOneLiveClock(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	entered, release := blockFirstLiveSample(t, s, false)
	defer release()
	call(t, s, "simulation.resume", struct{}{}, "resume")
	awaitLiveSampleBlock(t, entered)
	before := s.Snapshot()
	promptLiveCommand(t, s, "simulation.resume", struct{}{})
	time.Sleep(120 * time.Millisecond) // span two clock ticks while the first sample is blocked
	after := s.Snapshot()
	if after.Transient.Sample.Time != before.Transient.Sample.Time || after.RunID != before.RunID || !reflect.DeepEqual(after.Mixed, before.Mixed) {
		t.Fatal("repeated Resume started an additional clock")
	}
	call(t, s, "simulation.pause", struct{}{}, "finish")
	release()
	s.liveWorkers.Wait()
}
