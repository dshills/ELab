package workbench

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"
)

func TestLiveNetworkClockCrossesPulseEdgesWithoutAccumulatedTimeDrift(t *testing.T) {
	g := RCExampleGraph()
	load, err := MakeComponent("Load", "resistor", Point{200, 100}, "1k")
	if err != nil {
		t.Fatal(err)
	}
	g.Components = append(g.Components, load)
	g.Wires = append(g.Wires, Wire{ID: "WL1", From: "C1.1", To: "Load.1", Connected: true}, Wire{ID: "WL2", From: "Load.2", To: "GND.reference", Connected: true})
	s := newGraphService(SolveLED, g)
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	cfg := TransientSettings{StepSeconds: .001, HighSeconds: .1, LowSeconds: .1, PlaybackSpeed: 1}
	call(t, s, "simulation.transient.set", cfg, "settings")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	if !IsNetworkAnalysis(before.Analysis) || before.Transient.Stale {
		t.Fatal("fixture must exercise the real adaptive capacitor network")
	}
	call(t, s, "simulation.resume", struct{}{}, "resume")
	deadline := time.Now().Add(15 * time.Second)
	var state Snapshot
	for {
		state = s.Snapshot()
		if state.Status == "error" || state.Transient.Stale {
			t.Fatalf("valid live grid stopped at %.17g: %+v", state.Transient.Sample.Time, state.Diagnostics)
		}
		if state.Transient.Sample.Time >= 1.2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("live output did not reach the checked pulse boundary")
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Check the live-produced samples against an independent loaded-RC law.
	// Each half period is over 100 time constants, so residual from the previous
	// half period is negligible compared with the numerical voltage tolerance.
	steady, tau := 5./11, (10000.*1000/(10000+1000))*1e-6
	for _, sample := range state.Transient.Samples {
		index := math.Round(sample.Time / cfg.StepSeconds)
		at := index * cfg.StepSeconds
		if !sameSampleTime(sample.Time, at) {
			t.Fatalf("output drifted off the requested grid: %.17g vs %.17g", sample.Time, at)
		}
		phase := int(math.Round(index)) % 200
		if phase == 0 || phase == 100 {
			continue // The retained left/right algebraic edge frames share time.
		}
		elapsed := float64(phase%100) * cfg.StepSeconds
		want := steady * math.Exp(-elapsed/tau)
		if phase < 100 {
			want = steady * (1 - math.Exp(-elapsed/tau))
		}
		// Clock correctness must retain the circuit law without demanding that
		// accumulated adaptive integration error equal a single local error.
		if math.Abs(sample.Voltage-want) > 1e-5 {
			t.Fatalf("loaded RC at %g: got %g want %g", sample.Time, sample.Voltage, want)
		}
	}
}

func TestLiveNetworkClockResumeKeepsCommittedTimeOrigin(t *testing.T) {
	s := newGraphService(SolveLED, switchRCGraph(t, true))
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.step", StepParams{Duration: .031}, "manual-origin")
	before := s.Snapshot()
	first := make(chan float64, 1)
	var once sync.Once
	s.mu.Lock()
	s.networkAdvance = func(ctx context.Context, state *mixedState, trace *Transient, end float64, graph Graph) (*mixedState, *Transient, error) {
		once.Do(func() { first <- end })
		return advanceLiveNetwork(ctx, state, trace, end, graph)
	}
	s.mu.Unlock()
	call(t, s, "simulation.resume", struct{}{}, "resume")
	select {
	case end := <-first:
		want := before.Transient.Sample.Time + before.TransientSettings.StepSeconds
		if end != want || end <= before.Transient.Sample.Time {
			t.Fatalf("resumed output requested %g after committed %g; want %g", end, before.Transient.Sample.Time, want)
		}
	case <-time.After(time.Second):
		t.Fatal("resumed real clock did not request an output")
	}
}
