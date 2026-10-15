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

func TestNativeRecordedTimingMatchesPublicQueryAndClears(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned threshold timing", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	duration := .02
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "recorded-timing-probe", &ui.ChangeEvent{SelectedID: "v:C1"})
	uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "2500 mV"})
	uiAction(t, s, "recorded-timing-direction", &ui.ChangeEvent{SelectedID: "rising"})
	uiAction(t, s, "recorded-timing-start", &ui.ChangeEvent{Text: "1 ms"})
	state := s.service.Snapshot()
	result := state.Recording.Timing
	if result == nil || result.Total != 1 || result.Level != 2.5 || result.Crossings[0].Before.Index != 6 || result.Crossings[0].After.Index != 7 {
		t.Fatal("native threshold bracket missing")
	}
	raw, _ := json.Marshal(workbench.RecordingTimingParams(result.Settings))
	response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "native-timing-query", Command: "recording.timing", Params: raw}, "agent")
	if response.Error != nil || !reflect.DeepEqual(response.Result.(*workbench.RecordingTiming), result) {
		t.Fatal("native and public timing differ", response.Error)
	}
	if before.Document != state.Document || before.RunID != state.RunID || before.ElectricalRevision != state.ElectricalRevision || before.Transient.Sample != state.Transient.Sample || !reflect.DeepEqual(before.UndoEntry, state.UndoEntry) {
		t.Fatal("native timing changed live work")
	}
	s.app.Do(func() {
		root := s.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		if _, ok := findNode(root, "recorded-timing-edge-0"); !ok {
			t.Fatal("bracket label absent")
		}
	})
	uiAction(t, s, "recorded-timing-probe", &ui.ChangeEvent{SelectedID: "__none"})
	if s.service.Snapshot().Recording.Timing != nil {
		t.Fatal("native timing off failed")
	}
}

func TestNativeRecordedTimingPagesAndDirectionReset(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned threshold pages", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.npn-pulse", nil)
	uiAction(t, s, "simulation.pause", nil)
	duration := .5
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	uiAction(t, s, "recorded-timing-probe", &ui.ChangeEvent{SelectedID: "v:VIN.positive"})
	uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "2.5 V"})
	first := s.service.Snapshot().Recording.Timing
	if first.Total <= 8 || len(first.Crossings) != 8 || first.NextOffset != 8 {
		t.Fatal("pulse fixture did not fill timing page", first)
	}
	uiAction(t, s, "recorded-timing-next", nil)
	last := s.service.Snapshot().Recording.Timing
	if last.Settings.Offset != 8 || last.NextOffset != last.Total || len(last.Crossings) != last.Total-8 || !reflect.DeepEqual(first.RisingSpacing, last.RisingSpacing) {
		t.Fatal("native next page or summary changed", last)
	}
	uiAction(t, s, "recorded-timing-previous", nil)
	if !reflect.DeepEqual(s.service.Snapshot().Recording.Timing, first) {
		t.Fatal("native previous page failed")
	}
	uiAction(t, s, "recorded-timing-next", nil)
	uiAction(t, s, "recorded-timing-direction", &ui.ChangeEvent{SelectedID: "rising"})
	filtered := s.service.Snapshot().Recording.Timing
	if filtered.Settings.Offset != 0 || filtered.Total >= first.Total {
		t.Fatal("direction change retained old page")
	}
}
