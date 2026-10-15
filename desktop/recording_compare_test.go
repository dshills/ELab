package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecordedComparisonMatchesQueryAndPreservesPhysics(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned captured comparison", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	duration := .02
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}})
	first := s.service.Snapshot().LastOperation
	completedVerification(t, s.service, first)
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}})
	second := s.service.Snapshot().LastOperation
	completedVerification(t, s.service, second)
	uiAction(t, s, "recording-show", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "recorded-compare-reference", &ui.ChangeEvent{SelectedID: first})
	uiAction(t, s, "recorded-compare-time", &ui.ChangeEvent{Text: "10 ms"})
	uiAction(t, s, "recorded-compare-statistics", &ui.ChangeEvent{Checked: true})
	uiAction(t, s, "recorded-compare-overlay", &ui.ChangeEvent{Checked: true})
	// Editing the numeric comparison time must preserve the independent plot flag.
	uiAction(t, s, "recorded-compare-time", &ui.ChangeEvent{Text: "10 ms"})
	state := s.service.Snapshot()
	comparison := state.Recording.Comparison
	if comparison == nil || !comparison.Available || !comparison.Settings.ShowStatistics || !comparison.Settings.ShowOverlay || comparison.A.ID != second || comparison.B.ID != first || comparison.Settings.Time != .01 || state.Recording.ReferenceOverlay == nil || !state.Recording.ReferenceOverlay.Scope.WindowValid {
		t.Fatal("native comparison not configured")
	}
	raw, _ := json.Marshal(workbench.RecordingCompareParams(comparison.Settings))
	response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "native-query", Command: "recording.compare", Params: raw}, "agent")
	if response.Error != nil || !reflect.DeepEqual(response.Result.(*workbench.RecordingComparison), comparison) {
		t.Fatal("native comparison differs from public query", response.Error)
	}
	if before.RunID != state.RunID || before.ElectricalRevision != state.ElectricalRevision || before.Transient.Sample != state.Transient.Sample || before.Document != state.Document || !reflect.DeepEqual(before.UndoEntry, state.UndoEntry) {
		t.Fatal("native comparison changed live work")
	}
	s.app.Do(func() {
		root := s.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		if _, ok := findNode(root, "recorded-compare-reading-v:C1"); !ok {
			t.Fatal("comparison readings absent")
		}
		if _, ok := findNode(root, "recorded-reference-identity"); !ok {
			t.Fatal("reference identity/style not shown")
		}
	})
	uiAction(t, s, "recorded-compare-reference", &ui.ChangeEvent{SelectedID: "__none"})
	if s.service.Snapshot().Recording.Comparison != nil || s.service.Snapshot().Recording.ReferenceOverlay != nil {
		t.Fatal("native clear comparison failed")
	}
}
