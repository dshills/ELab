package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeIntervalMeasurementsUseSharedCommandsAndRejectStaleFields(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	duration := .02
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1", "i:R1"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "recorded-statistics-probe", &ui.ChangeEvent{SelectedID: "v:C1"})
	uiAction(t, s, "recorded-statistics-start", &ui.ChangeEvent{Text: "1.5 ms"})
	uiAction(t, s, "recorded-statistics-end", &ui.ChangeEvent{Text: "19.5 ms"})
	after := s.service.Snapshot()
	result := after.Recording.Statistics
	if result == nil || result.Settings.Start != .0015 || result.Settings.End != .0195 || result.ACRMS <= 0 || result.PeakToPeak <= 0 {
		t.Fatal("native interval measurement absent", result)
	}
	raw, _ := json.Marshal(workbench.RecordingStatisticsParams(result.Settings))
	r := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "native-statistics-query", Command: "recording.statistics", Params: raw}, "agent")
	if r.Error != nil || !reflect.DeepEqual(r.Result.(*workbench.RecordingStatistics), result) {
		t.Fatal("native/agent statistics differ", r.Error)
	}
	if before.Document != after.Document || before.RunID != after.RunID || before.Transient.Sample != after.Transient.Sample || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("native measurement altered live/document/history")
	}
	var stale ui.Node
	s.app.Do(func() {
		root := s.view()
		stale, _ = findNode(root, "recorded-statistics-start")
		if _, ok := findNode(root, "recorded-statistics-values"); !ok {
			t.Fatal("measurement values absent")
		}
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
	})
	p := workbench.RecordingStatisticsParams(result.Settings)
	p.Start = .003
	agentAction(t, s.service, "recording.statistics.set", p)
	current := s.service.Snapshot()
	s.app.Do(func() { stale.Change(ui.ChangeEvent{Text: "2 ms"}) })
	if s.errorMessage == "" || !reflect.DeepEqual(current, s.service.Snapshot()) {
		t.Fatal("stale native field replaced agent interval")
	}
	uiAction(t, s, "recorded-statistics-probe", &ui.ChangeEvent{SelectedID: "i:R1"})
	switched := s.service.Snapshot().Recording.Statistics
	if switched.Probe.Unit != "A" || switched.Settings.Start != .003 || switched.Settings.End != .0195 {
		t.Fatal("switching probe changed units or measurement interval")
	}
	uiAction(t, s, "recorded-statistics-probe", &ui.ChangeEvent{SelectedID: "__none"})
	if s.service.Snapshot().Recording.Statistics != nil {
		t.Fatal("native measurement clear failed")
	}
}
