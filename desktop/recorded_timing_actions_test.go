package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func prepareRecordedTiming(t *testing.T, service *workbench.Service) {
	t.Helper()
	agentAction(t, service, "example.npn-pulse", workbench.ReplaceParams{})
	agentAction(t, service, "simulation.pause", struct{}{})
	duration := .5
	agentAction(t, service, "simulation.verify", workbench.VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}})
	completedVerification(t, service, service.Snapshot().LastOperation)
	agentAction(t, service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	agentAction(t, service, "workspace.inspector.set", workbench.InspectorParams{Section: "simulation"})
	state := service.Snapshot()
	agentAction(t, service, "recording.timing.set", workbench.RecordingTimingParams{ID: state.Recording.ID, ProbeID: "v:VIN.positive", Level: "2.5 V", Direction: "both", End: .5})
}

func TestRecordedTimingPageWaitsForAllFieldsAndRunsOnce(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	before := s.service.Snapshot()
	finish := map[string]func(error){}
	s.settleField = func(id string, done func(error)) error { finish[id] = done; return nil }
	var original ui.Node
	s.app.Do(func() { original, _ = findNode(s.view(), "recorded-timing-next") })
	uiAction(t, s, "recorded-timing-next", nil)
	if !reflect.DeepEqual(before, s.service.Snapshot()) || finish["recorded-timing-level"] == nil {
		t.Fatal("page did not await threshold settlement")
	}
	s.app.Do(original.Activate)
	for _, field := range []struct{ id, text string }{
		{"recorded-timing-level", "3 V"}, {"recorded-timing-second-level", "4 V"},
		{"recorded-timing-start", "1 ms"}, {"recorded-timing-end", "450 ms"},
	} {
		if finish[field.id] == nil {
			t.Fatal("page did not await field", field.id)
		}
		uiAction(t, s, field.id, &ui.ChangeEvent{Text: field.text})
		s.app.Do(func() { finish[field.id](nil); finish[field.id](nil) })
	}
	after := s.service.Snapshot()
	cfg := after.Recording.Timing.Settings
	if s.pendingRecordedTimingPage != nil || s.errorMessage != "" || cfg.Level != "3 V" || cfg.SecondLevel != "4 V" || cfg.Start != .001 || cfg.End != .45 || cfg.Offset != 8 || after.CommandRevision != before.CommandRevision+5 {
		t.Fatal("page lost accepted settings or ran more than once", cfg, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestRecordedTimingPageRefusesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-before", "agent-after", "other-user-field", "other-recording", "new-document", "invalid-field"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareRecordedTiming(t, s.service)
			var original ui.Node
			s.app.Do(func() { original, _ = findNode(s.view(), "recorded-timing-next") })
			var finish func(error)
			settlements := 0
			s.settleField = func(_ string, done func(error)) error { settlements++; finish = done; return nil }
			if scenario == "agent-before" {
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "agent draft"})
				before := s.service.Snapshot()
				s.app.Do(original.Activate)
				if settlements != 0 || s.pendingRecordedTimingPage != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("stale page settled fields or altered agent work")
				}
				return
			}
			s.app.Do(original.Activate)
			uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "3 V"})
			switch scenario {
			case "agent-after":
				cfg := workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings)
				cfg.Level = "2 V"
				agentAction(t, s.service, "recording.timing.set", cfg)
			case "other-user-field":
				uiAction(t, s, "recorded-time", &ui.ChangeEvent{Text: "2 ms"})
			case "other-recording":
				duration := .01
				agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
				completedVerification(t, s.service, s.service.Snapshot().LastOperation)
			case "new-document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "invalid-field":
				s.app.Do(func() {
					field, _ := findNode(s.view(), "recorded-timing-end")
					field.Change(ui.ChangeEvent{Text: "900 ms"})
				})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				for range 5 {
					if s.pendingRecordedTimingPage != nil {
						finish(nil)
					}
				}
			})
			if s.pendingRecordedTimingPage != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("pending page adopted intervening work", scenario, s.errorMessage)
			}
		})
	}
}

func TestRecordedTimingPageSettlementFailurePreservesState(t *testing.T) {
	for _, admission := range []bool{true, false} {
		t.Run(map[bool]string{true: "admission", false: "completion"}[admission], func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareRecordedTiming(t, s.service)
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error {
				finish = done
				if admission {
					return errors.New("field retired")
				}
				return nil
			}
			before := s.service.Snapshot()
			s.app.Do(func() { button, _ := findNode(s.view(), "recorded-timing-next"); button.Activate() })
			if !admission {
				s.app.Do(func() { finish(errors.New("composition incomplete")) })
			}
			if s.pendingRecordedTimingPage != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("failed settlement changed timing", s.errorMessage)
			}
		})
	}
}

func TestRecordedTimingPagePreviousUsesAcceptedFieldSettings(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	cfg := workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings)
	cfg.Offset = 8
	agentAction(t, s.service, "recording.timing.set", cfg)
	before := s.service.Snapshot()
	var original ui.Node
	s.app.Do(func() { original, _ = findNode(s.view(), "recorded-timing-previous") })
	uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "3 V"})
	s.app.Do(original.Activate)
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.Recording.Timing.Settings.Level != "3 V" || after.Recording.Timing.Settings.Offset != 0 || after.CommandRevision != before.CommandRevision+2 {
		t.Fatal("previous page lost accepted field settings", after.Recording.Timing.Settings, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestRecordedTimingPageStopsWhenEditedThresholdHasNoMoreArrivals(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	var original ui.Node
	s.app.Do(func() { original, _ = findNode(s.view(), "recorded-timing-next") })
	uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "6 V"})
	before := s.service.Snapshot()
	if before.Recording.Timing.Total != 0 {
		t.Fatal("fixture still has arrivals above the 5 V pulse")
	}
	s.app.Do(original.Activate)
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingRecordedTimingPage != nil {
		t.Fatal("page advanced beyond the edited arrival list", s.errorMessage)
	}
}
