package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func capturedStatisticsScreen(t *testing.T, imported bool) *screen {
	t.Helper()
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	if imported {
		path, err := filepath.Abs("../workbench/testdata/recorded-statistics-short-tail.json")
		if err != nil {
			t.Fatal(err)
		}
		agentAction(t, s.service, "verification.import", workbench.RecordingImportParams{Path: path})
		agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	} else {
		recordedCursorSource(t, s.service)
	}
	r := s.service.Snapshot().Recording
	agentAction(t, s.service, "recording.statistics.set", workbench.RecordingStatisticsParams{ID: r.ID, ProbeID: r.Scope.Available[0].ID, End: r.LastSampleTime})
	return s
}

func TestCapturedStatisticsLateParserRefusalCannotInstallText(t *testing.T) {
	for _, change := range []string{"agent-draft", "peer-endpoint", "probe", "clear", "document"} {
		t.Run(change, func(t *testing.T) {
			s := capturedStatisticsScreen(t, false)
			original := verificationControl(t, s, "recorded-statistics-end")
			params := workbench.RecordingStatisticsParams(s.service.Snapshot().Recording.Statistics.Settings)
			switch change {
			case "agent-draft":
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "agent draft"})
			case "peer-endpoint":
				params.Start = .002
				agentAction(t, s.service, "recording.statistics.set", params)
			case "probe":
				params.ProbeID = s.service.Snapshot().Recording.Scope.Available[1].ID
				agentAction(t, s.service, "recording.statistics.set", params)
			case "clear":
				agentAction(t, s.service, "recording.statistics.set", workbench.RecordingStatisticsParams{ID: params.ID})
			case "document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				original.Change(ui.ChangeEvent{Text: "oops"})
				if len(s.recordedStatisticsRejected) != 0 {
					t.Fatal("old parser failure installed rejected text in the new measurement")
				}
			})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("stale measurement callback changed accepted state")
			}
			if change != "clear" && change != "document" {
				fresh := verificationControl(t, s, "recorded-statistics-end")
				s.app.Do(func() { fresh.Change(ui.ChangeEvent{Text: "18 ms"}) })
				after := s.service.Snapshot()
				if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || after.Recording.Statistics.Settings.End != .018 || !reflect.DeepEqual(before.Drawing, after.Drawing) {
					t.Fatal("fresh correction failed or replaced agent work", s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			}
		})
	}
}

func TestCapturedStatisticsCorrectionPreservesRejectedPeer(t *testing.T) {
	s := capturedStatisticsScreen(t, false)
	for _, side := range []string{"start", "end"} {
		field := verificationControl(t, s, "recorded-statistics-"+side)
		s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
	}
	if len(s.recordedStatisticsRejected) != 2 {
		t.Fatal("fixture requires two separately retained entries")
	}
	var pending ui.Node
	s.app.Do(func() { pending, _ = findNode(s.view(), "recorded-statistics-pending") })
	if !strings.Contains(pending.Text, "last accepted measurement") {
		t.Fatal("cached readout is not distinguished from the rejected entry")
	}
	agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "agent draft"})
	before := s.service.Snapshot()
	uiAction(t, s, "recorded-statistics-start", &ui.ChangeEvent{Text: "0 s"})
	after := s.service.Snapshot()
	peer := verificationControl(t, s, "recorded-statistics-end")
	if s.errorMessage != "" || len(s.recordedStatisticsRejected) != 1 || peer.Text != "oops" || after.CommandRevision != before.CommandRevision+1 || !reflect.DeepEqual(before.Recording.Statistics, after.Recording.Statistics) || !reflect.DeepEqual(before.Drawing, after.Drawing) {
		t.Fatal("one correction lost its peer, agent draft or cached measurement", s.errorMessage, peer.Text)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestCapturedStatisticsRejectedEntryIsPrunedWithItsOwner(t *testing.T) {
	for _, change := range []string{"endpoint", "probe", "off", "view", "capture", "document"} {
		t.Run(change, func(t *testing.T) {
			s := capturedStatisticsScreen(t, false)
			field := verificationControl(t, s, "recorded-statistics-end")
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
			if s.errorMessage == "" || len(s.recordedStatisticsRejected) != 1 {
				t.Fatal("fixture requires a retained measurement refusal")
			}
			params := workbench.RecordingStatisticsParams(s.service.Snapshot().Recording.Statistics.Settings)
			switch change {
			case "endpoint":
				params.End = .018
				agentAction(t, s.service, "recording.statistics.set", params)
			case "probe":
				params.ProbeID = s.service.Snapshot().Recording.Scope.Available[1].ID
				agentAction(t, s.service, "recording.statistics.set", params)
			case "off":
				uiAction(t, s, "recorded-statistics-probe", &ui.ChangeEvent{SelectedID: "__none"})
			case "view":
				agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "live"})
			case "capture":
				duration := .01
				agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
				completedVerification(t, s.service, s.service.Snapshot().LastOperation)
			case "document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			before := s.service.Snapshot()
			s.app.Do(func() { s.view() })
			if len(s.recordedStatisticsRejected) != 0 || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("old measurement entry followed a replacement owner or pruning changed state")
			}
		})
	}
}
