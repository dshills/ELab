package main

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeTransitionThresholdsMatchAgentAndSurviveTimeFields(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned transition timing", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	duration := .04
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "recorded-timing-probe", &ui.ChangeEvent{SelectedID: "v:C1"})
	uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "500 mV"})
	uiAction(t, s, "recorded-timing-second-level", &ui.ChangeEvent{Text: "4.5 V"})
	uiAction(t, s, "recorded-timing-direction", &ui.ChangeEvent{SelectedID: "falling"})
	uiAction(t, s, "recorded-timing-start", &ui.ChangeEvent{Text: "1 ms"})
	uiAction(t, s, "recorded-timing-end", &ui.ChangeEvent{Text: "40 ms"})
	state := s.service.Snapshot()
	result := state.Recording.Timing
	if result.Settings.SecondLevel != "4.5 V" || result.Transitions == nil || result.Transitions.Rise == nil || result.Transitions.Fall != nil {
		t.Fatal("native transition missing or time field dropped threshold", result)
	}
	expected := .01 * math.Log(9)
	if result.Transitions.Rise.Minimum > expected || result.Transitions.Rise.Maximum < expected {
		t.Fatal("analytic RC rise outside native bounds")
	}
	raw, _ := json.Marshal(workbench.RecordingTimingParams(result.Settings))
	response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "transition-query", Command: "recording.timing", Params: raw}, "agent")
	if response.Error != nil || !reflect.DeepEqual(response.Result, result) {
		t.Fatal("native/agent mismatch", response.Error)
	}
	if before.Document != state.Document || before.RunID != state.RunID || !reflect.DeepEqual(before.Transient, state.Transient) || !reflect.DeepEqual(before.UndoEntry, state.UndoEntry) {
		t.Fatal("native timing changed experiment/history")
	}
	s.app.Do(func() {
		root := s.view()
		for _, id := range []string{"recorded-transition-levels", "recorded-transition-rise", "recorded-transition-fall"} {
			if _, ok := findNode(root, id); !ok {
				t.Fatal("missing transition readout", id)
			}
		}
	})
	uiAction(t, s, "recorded-timing-second-level", &ui.ChangeEvent{Text: ""})
	if s.service.Snapshot().Recording.Timing.Transitions != nil {
		t.Fatal("blank other threshold did not disable transitions")
	}
	uiAction(t, s, "recorded-timing-probe", &ui.ChangeEvent{SelectedID: "__none"})
	if s.service.Snapshot().Recording.Timing != nil {
		t.Fatal("Timing off retained measurement")
	}
}
