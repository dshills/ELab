package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func capturedComparisonScreen(t *testing.T) (*screen, string) {
	t.Helper()
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	ids := []string{}
	for _, duration := range []float64{.02, .02, .01} {
		agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1", "i:C1"}})
		id := s.service.Snapshot().LastOperation
		completedVerification(t, s.service, id)
		ids = append(ids, id)
	}
	agentAction(t, s.service, "operation.select", workbench.IDParams{ID: ids[1]})
	agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	agentAction(t, s.service, "recording.compare.set", workbench.RecordingCompareParams{ID: ids[1], ReferenceID: ids[0], ShowStatistics: true, ShowOverlay: true})
	return s, ids[2]
}

func TestCapturedComparisonLateParserRefusalCannotInstallText(t *testing.T) {
	for _, change := range []string{"agent-draft", "flag", "time", "reference", "selection", "clear", "view", "document"} {
		t.Run(change, func(t *testing.T) {
			s, nextReference := capturedComparisonScreen(t)
			original := verificationControl(t, s, "recorded-compare-time")
			params := workbench.RecordingCompareParams(s.service.Snapshot().Recording.Comparison.Settings)
			switch change {
			case "agent-draft":
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "agent draft"})
			case "flag":
				params.ShowStatistics = false
				agentAction(t, s.service, "recording.compare.set", params)
			case "time":
				params.Time = .005
				agentAction(t, s.service, "recording.compare.set", params)
			case "reference":
				params.ReferenceID = nextReference
				agentAction(t, s.service, "recording.compare.set", params)
			case "selection":
				params.ID, params.ReferenceID = params.ReferenceID, params.ID
				agentAction(t, s.service, "operation.select", workbench.IDParams{ID: params.ID})
				agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
				agentAction(t, s.service, "recording.compare.set", params)
			case "clear":
				params.ReferenceID = ""
				agentAction(t, s.service, "recording.compare.set", params)
			case "view":
				agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "live"})
			case "document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			before := s.service.Snapshot()
			s.app.Do(func() { original.Change(ui.ChangeEvent{Text: "oops"}) })
			if s.recordedComparisonRejected != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("late comparison parser failure installed text or replaced accepted state")
			}
			if change != "clear" && change != "view" && change != "document" {
				fresh := verificationControl(t, s, "recorded-compare-time")
				s.app.Do(func() { fresh.Change(ui.ChangeEvent{Text: "2 ms"}) })
				after := s.service.Snapshot()
				cfg := after.Recording.Comparison.Settings
				previous := before.Recording.Comparison.Settings
				if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || cfg.Time != .002 || cfg.ID != previous.ID || cfg.ReferenceID != previous.ReferenceID || cfg.ShowOverlay != previous.ShowOverlay || cfg.ShowStatistics != previous.ShowStatistics || !reflect.DeepEqual(before.Drawing, after.Drawing) {
					t.Fatal("fresh comparison lost agent work or independent preferences", s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			}
		})
	}
}

func TestCapturedComparisonCurrentRefusalSurvivesPreferenceChanges(t *testing.T) {
	s, _ := capturedComparisonScreen(t)
	field := verificationControl(t, s, "recorded-compare-time")
	before := s.service.Snapshot()
	s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
	if s.recordedComparisonRejected == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("current comparison refusal was not retained purely")
	}
	params := workbench.RecordingCompareParams(before.Recording.Comparison.Settings)
	params.ShowOverlay, params.ShowStatistics = false, false
	agentAction(t, s.service, "recording.compare.set", params)
	agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "agent draft"})
	current := verificationControl(t, s, "recorded-compare-time")
	var pending ui.Node
	s.app.Do(func() { pending, _ = findNode(s.view(), "recorded-compare-pending") })
	if current.Text != "oops" || !strings.Contains(pending.Text, "last accepted comparison") {
		t.Fatal("preference edit erased the retained time or hid cached-result identity")
	}
	before = s.service.Snapshot()
	s.app.Do(func() { current.Change(ui.ChangeEvent{Text: "0 s"}) })
	after := s.service.Snapshot()
	if s.errorMessage != "" || s.recordedComparisonRejected != nil || after.CommandRevision != before.CommandRevision+1 || !reflect.DeepEqual(before.Recording.Comparison, after.Recording.Comparison) || !reflect.DeepEqual(before.Drawing, after.Drawing) {
		t.Fatal("fresh correction lost current comparison, preferences or agent draft", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestCapturedComparisonRejectedEntryIsPrunedWithItsOwner(t *testing.T) {
	for _, change := range []string{"time", "reference", "reference-reset", "off", "selected-capture", "view", "document"} {
		t.Run(change, func(t *testing.T) {
			s, nextReference := capturedComparisonScreen(t)
			field := verificationControl(t, s, "recorded-compare-time")
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
			if s.recordedComparisonRejected == nil || s.errorMessage == "" {
				t.Fatal("fixture requires a current retained comparison refusal")
			}
			params := workbench.RecordingCompareParams(s.service.Snapshot().Recording.Comparison.Settings)
			switch change {
			case "time":
				params.Time = .005
				agentAction(t, s.service, "recording.compare.set", params)
			case "reference":
				params.ReferenceID = nextReference
				agentAction(t, s.service, "recording.compare.set", params)
			case "reference-reset":
				uiAction(t, s, "recorded-compare-reference", &ui.ChangeEvent{SelectedID: params.ReferenceID})
			case "off":
				uiAction(t, s, "recorded-compare-reference", &ui.ChangeEvent{SelectedID: "__none"})
			case "selected-capture":
				agentAction(t, s.service, "operation.select", workbench.IDParams{ID: nextReference})
			case "view":
				agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "live"})
			case "document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			before := s.service.Snapshot()
			s.app.Do(func() { s.view() })
			if s.recordedComparisonRejected != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("comparison refusal followed another owner or pruning changed state")
			}
		})
	}
}
