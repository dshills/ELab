package main

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeLiveInputControlsUsePublishedInventory(t *testing.T) {
	for _, example := range []string{"example.rc", "example.npn-pulse", "example.flasher", "example.rectifier", "example.reset"} {
		t.Run(example, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, example, workbench.ReplaceParams{DiscardChanges: true})
			uiAction(t, s, "simulation.pause", nil)
			if len(s.service.Snapshot().LiveInputModes) > 0 {
				uiAction(t, s, "simulation.restart", nil)
			} else {
				agentAction(t, s.service, "simulation.restart", struct{}{})
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if _, err := s.service.Wait(ctx); err != nil {
				t.Fatal(err)
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				root := s.view()
				for _, mode := range []string{"auto", "charge", "discharge", "high", "low"} {
					_, found := findNode(root, "input-"+mode)
					want := false
					for _, allowed := range before.LiveInputModes {
						if allowed == mode {
							want = true
						}
					}
					if found != want {
						t.Fatal("UI/inventory disagreement", before.Analysis, mode, found, want)
					}
				}
			})
			for _, mode := range before.LiveInputModes {
				uiAction(t, s, "input-"+mode, nil)
				after := s.service.Snapshot()
				if after.Transient.PendingMode != mode || !reflect.DeepEqual(before.Transient.Sample, after.Transient.Sample) {
					t.Fatal("native input did not queue", mode)
				}
			}
		})
	}
}

func TestNativeMicrosecondSampleAtMinimumPlayback(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "setting-speed", &ui.ChangeEvent{Number: .001})
	uiAction(t, s, "setting-step", &ui.ChangeEvent{Text: "1 us"})
	uiAction(t, s, "simulation.restart", nil)
	state := s.service.Snapshot()
	if state.TransientSettings.PlaybackSpeed != .001 || state.TransientSettings.StepSeconds != 1e-6 || state.Transient.Stale {
		t.Fatal("native range endpoint lost decimal intent", state.TransientSettings)
	}
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .001})
	state = s.service.Snapshot()
	if state.StepWork.CompletedSamples != 1000 || math.Abs(state.Transient.Sample.Time-.001) > 1e-15 {
		t.Fatal("native configured 1000-step boundary refused", state.StepWork)
	}
}

func TestNativeLiveControlsSmokeFromDirtyPausedCircuit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- nativeLiveControlsSmoke(ctx, s.app, s) }()
	for {
		s.app.Drain()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			part, ok := (workbench.Graph{Components: s.service.Snapshot().Components}).Component("R1")
			if !ok || part.Value != 10000 {
				t.Fatal("smoke did not replace its owned dirty fixture")
			}
			return
		case <-ctx.Done():
			t.Fatal("live-control smoke owner workflow did not finish")
		case <-time.After(time.Millisecond):
		}
	}
}
