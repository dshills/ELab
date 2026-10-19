//go:build darwin && cgo

package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecordedTimingDirectionWaitsForFieldsAndRunsOnce(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	before := s.service.Snapshot()
	finish := map[string]func(error){}
	s.settleField = func(id string, done func(error)) error { finish[id] = done; return nil }
	node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
	s.app.Do(func() {
		s.acceptMenuChoice(node, choices, index, serial, revision)
		s.acceptMenuChoice(node, choices, index, serial, revision)
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || finish["recorded-timing-level"] == nil {
		t.Fatal("direction did not await private fields")
	}
	for _, field := range []struct{ id, text string }{
		{"recorded-timing-level", "3 V"}, {"recorded-timing-second-level", "4 V"},
		{"recorded-timing-start", "1 ms"}, {"recorded-timing-end", "450 ms"},
	} {
		if finish[field.id] == nil {
			t.Fatal("direction skipped a related field", field.id)
		}
		uiAction(t, s, field.id, &ui.ChangeEvent{Text: field.text})
		s.app.Do(func() { finish[field.id](nil); finish[field.id](nil) })
	}
	after := s.service.Snapshot()
	cfg := after.Recording.Timing.Settings
	if s.errorMessage != "" || s.pendingRecordedTimingPage != nil || s.fieldContext != nil || s.menuCommandRevision != nil || cfg.Level != "3 V" || cfg.SecondLevel != "4 V" || cfg.Start != .001 || cfg.End != .45 || cfg.Direction != "rising" || cfg.Offset != 0 || after.CommandRevision != before.CommandRevision+5 {
		t.Fatal("direction lost settled settings, duplicated or leaked context", cfg, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestNativeRecordedTimingDirectionRefusesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent", "other-user-field", "other-recording", "new-document", "menu-reopened", "file-busy", "pending-file"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareRecordedTiming(t, s.service)
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
			node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
			s.app.Do(func() { s.acceptMenuChoice(node, choices, index, serial, revision) })
			uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "3 V"})
			switch scenario {
			case "agent":
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
			case "menu-reopened":
				s.app.Do(func() { s.menuSerial++ })
			case "file-busy":
				s.app.Do(func() { s.fileBusy = true })
			case "pending-file":
				s.app.Do(func() { s.pendingCommand = "project.clear" })
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				for range 5 {
					if s.pendingRecordedTimingPage != nil {
						finish(nil)
					}
				}
			})
			if s.errorMessage == "" || s.pendingRecordedTimingPage != nil || s.fieldContext != nil || s.menuCommandRevision != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("direction adopted intervening work or leaked context", scenario, s.errorMessage)
			}
		})
	}
}

func TestNativeRecordedTimingDirectionSettlementFailurePreservesState(t *testing.T) {
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
			node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
			s.app.Do(func() { s.acceptMenuChoice(node, choices, index, serial, revision) })
			if !admission {
				s.app.Do(func() { finish(errors.New("composition incomplete")) })
			}
			if s.errorMessage == "" || s.pendingRecordedTimingPage != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("failed settlement changed direction", s.errorMessage)
			}
		})
	}
}
