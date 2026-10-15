package main

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecordedPulseReadoutsUseSharedTimingAndSurviveFilters(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned pulse measurements", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.npn-pulse", nil)
	uiAction(t, s, "simulation.pause", nil)
	duration := .25
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "recorded-timing-probe", &ui.ChangeEvent{SelectedID: "v:VIN.positive"})
	uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "2.5 V"})
	first := s.service.Snapshot().Recording.Timing
	if first.Cycle == nil || first.Cycle.Duty == nil || math.Abs(first.Cycle.Duty.Minimum-.5) > 1e-12 || first.HighPulse == nil || first.LowPulse == nil {
		t.Fatal("native pulse summary absent")
	}
	s.app.Do(func() {
		root := s.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"recorded-pulse-high", "recorded-pulse-low", "recorded-pulse-period", "recorded-pulse-duty"} {
			if _, ok := findNode(root, id); !ok {
				t.Fatal("native pulse readout absent", id)
			}
		}
		node, _ := findNode(root, "recorded-pulse-duty")
		if !strings.Contains(node.Text, "50 → 50 % high") {
			t.Fatal("native duty percentage differs from shared fraction", node.Text)
		}
	})
	uiAction(t, s, "recorded-timing-direction", &ui.ChangeEvent{SelectedID: "falling"})
	filtered := s.service.Snapshot().Recording.Timing
	if !reflect.DeepEqual(first.Cycle, filtered.Cycle) || !reflect.DeepEqual(first.HighPulse, filtered.HighPulse) || !reflect.DeepEqual(first.LowPulse, filtered.LowPulse) {
		t.Fatal("native arrival filter changed complete interval pulse data")
	}
	raw, _ := json.Marshal(workbench.RecordingTimingParams(filtered.Settings))
	r := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "pulse-query", Command: "recording.timing", Params: raw}, "agent")
	if r.Error != nil || !reflect.DeepEqual(r.Result.(*workbench.RecordingTiming), filtered) {
		t.Fatal("native/public pulse results differ", r.Error)
	}
	after := s.service.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.NPN, after.NPN) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("native pulse measurement changed live circuit/history")
	}
	uiAction(t, s, "recorded-timing-end", &ui.ChangeEvent{Text: "120 ms"})
	if s.service.Snapshot().Recording.Timing.Cycle != nil {
		t.Fatal("short interval invents a complete cycle")
	}
	s.app.Do(func() {
		if _, ok := findNode(s.view(), "recorded-pulse-cycle-missing"); !ok {
			t.Fatal("missing cycle has no explanation")
		}
	})
	// Exercise the packaged helper's complete capture/owner workflow. Creating
	// a kept capture legitimately changes recording membership before measuring.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- nativeRecordingPulsesSmoke(ctx, s.app, s) }()
	for {
		s.app.Drain()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-ctx.Done():
			t.Fatal("pulse smoke owner workflow did not finish")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestNativeUnresolvedPulseDutyHasExplanationAndNoPercentage(t *testing.T) {
	r := &workbench.RecordingTiming{Cycle: &workbench.TimingCycle{Reason: "Saved brackets do not resolve a strictly positive cycle duration"}}
	root := ui.Node{ID: "root", Kind: ui.Column, Children: recordingPulseReadouts(r)}
	if err := ui.Validate(root); err != nil {
		t.Fatal(err)
	}
	if _, ok := findNode(root, "recorded-pulse-duty"); ok {
		t.Fatal("unresolved cycle invented a percentage")
	}
	if node, ok := findNode(root, "recorded-pulse-duty-unavailable"); !ok || node.Text != r.Cycle.Reason {
		t.Fatal("unresolved cycle lacks shared reason")
	}
}
