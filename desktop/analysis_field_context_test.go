package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

var analysisEntryCases = []struct{ id, correction string }{
	{"verify-duration", "20 ms"}, {"verify-step", "500 us"},
	{"sweep-range-start", "1 V"}, {"sweep-range-stop", "4 V"}, {"sweep-points", "7"},
	{"recorded-timing-level", "3 V"}, {"recorded-timing-second-level", "4 V"},
	{"recorded-timing-start", "1 ms"}, {"recorded-timing-end", "450 ms"},
}

func analysisEntryScreen(t *testing.T, id string) *screen {
	t.Helper()
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	if strings.HasPrefix(id, "recorded-timing-") {
		prepareRecordedTiming(t, s.service)
	} else {
		uiAction(t, s, "example.rc", nil)
		agentAction(t, s.service, "simulation.pause", struct{}{})
	}
	return s
}

func analysisEntryRejections(s *screen) int {
	return len(s.verificationRejected) + len(s.sweepFieldRejected) + len(s.recordedTimingRejected)
}

func TestAnalysisParserRefusalCannotInstallAnOldDraft(t *testing.T) {
	for _, entry := range analysisEntryCases {
		t.Run(entry.id, func(t *testing.T) {
			s := analysisEntryScreen(t, entry.id)
			field := verificationControl(t, s, entry.id)
			agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "new agent draft"})
			before := s.service.Snapshot()
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
			current := verificationControl(t, s, entry.id)
			if analysisEntryRejections(s) != 0 || current.Text == "oops" {
				t.Fatal("old parser failure installed a refusal in the newer form", current.Text)
			}
			if entry.id == "sweep-points" && s.errorMessage != "Sweep settings changed. Review them and try again." {
				t.Fatal("stale point parser blamed the newer form's accepted count", s.errorMessage)
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("stale entry replaced the current field or changed accepted state")
			}
			s.app.Do(func() { current.Change(ui.ChangeEvent{Text: entry.correction}) })
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || analysisEntryRejections(s) != 0 {
				t.Fatal("fresh correction did not succeed exactly once", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
			if !reflect.DeepEqual(before.Drawing, after.Drawing) {
				t.Fatal("fresh correction replaced the agent's editor draft")
			}
		})
	}
}

func TestAnalysisCurrentRefusalRemainsCorrectableAfterOtherWork(t *testing.T) {
	for _, entry := range analysisEntryCases {
		t.Run(entry.id, func(t *testing.T) {
			s := analysisEntryScreen(t, entry.id)
			field := verificationControl(t, s, entry.id)
			before := s.service.Snapshot()
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
			if analysisEntryRejections(s) != 1 || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("current refusal lost its entry or changed accepted state")
			}
			agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "new agent draft"})
			before = s.service.Snapshot()
			current := verificationControl(t, s, entry.id)
			if current.Text != "oops" {
				t.Fatal("unrelated later work erased an already owned refusal")
			}
			s.app.Do(func() { current.Change(ui.ChangeEvent{Text: entry.correction}) })
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || analysisEntryRejections(s) != 0 {
				t.Fatal("current correction failed or retained stale refusal state", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
			if !reflect.DeepEqual(before.Drawing, after.Drawing) {
				t.Fatal("correcting an owned refusal replaced the agent's editor draft")
			}
		})
	}
}
