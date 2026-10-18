package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func recordedCursorRC(t *testing.T) (*screen, func()) {
	t.Helper()
	s, cleanup := testScreen(t)
	recordedCursorSource(t, s.service)
	return s, cleanup
}

func recordedCursorSource(t *testing.T, service *workbench.Service) {
	t.Helper()
	agentAction(t, service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, service, "simulation.pause", struct{}{})
	duration := .02
	agentAction(t, service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
	completedVerification(t, service, service.Snapshot().LastOperation)
	agentAction(t, service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
}

func TestRecordedCursorActionWaitsForBothFieldsAndRunsOnce(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	t.Cleanup(cleanup)
	before := s.service.Snapshot()
	finish := map[string]func(error){}
	s.settleField = func(id string, done func(error)) error { finish[id] = done; return nil }
	var original ui.Node
	s.app.Do(func() { original, _ = findNode(s.view(), "recorded-cursor-select-b") })
	uiAction(t, s, "recorded-cursor-select-b", nil)
	if finish["recorded-cursor-a"] == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("selection did not wait for recorded field A")
	}
	s.app.Do(original.Activate) // duplicate admission must not replace the pending action
	uiAction(t, s, "recorded-cursor-a", &ui.ChangeEvent{Text: "4 ms"})
	s.app.Do(func() { finish["recorded-cursor-a"](nil) })
	if finish["recorded-cursor-b"] == nil || s.service.Snapshot().Recording.Scope.ActiveCursor != "a" {
		t.Fatal("selection did not wait for recorded field B")
	}
	uiAction(t, s, "recorded-cursor-b", &ui.ChangeEvent{Text: "16 ms"})
	s.app.Do(func() {
		finish["recorded-cursor-a"](nil)
		finish["recorded-cursor-b"](nil)
		finish["recorded-cursor-b"](nil)
	})
	after := s.service.Snapshot()
	scope := after.Recording.Scope
	if s.pendingCursorAction != nil || s.errorMessage != "" || scope.A == nil || scope.B == nil || scope.A.Requested != .004 || scope.B.Requested != .016 || scope.ActiveCursor != "b" || after.CommandRevision != before.CommandRevision+3 {
		t.Fatal("deferred recorded fields did not select once", scope, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestRecordedCursorActionRefusesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-cursor", "other-recording", "live-field", "document-replaced", "invalid-time"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			first := s.service.Snapshot().Recording.ID
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
			uiAction(t, s, "recorded-cursor-select-b", nil)
			uiAction(t, s, "recorded-cursor-a", &ui.ChangeEvent{Text: "4 ms"})
			switch scenario {
			case "agent-cursor":
				agentAction(t, s.service, "recording.cursor.set", workbench.RecordingCursorParams{OperationID: first, Cursor: "b", Time: .012})
			case "other-recording":
				duration := .01
				agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
				completedVerification(t, s.service, s.service.Snapshot().LastOperation)
				if s.service.Snapshot().Recording.ID == first {
					t.Fatal("fixture did not switch recording")
				}
			case "live-field":
				agentAction(t, s.service, "scope.cursor.select", workbench.CursorSelectParams{Cursor: "b"})
			case "document-replaced":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "invalid-time":
				s.app.Do(func() {
					field, _ := findNode(s.view(), "recorded-cursor-b")
					field.Change(ui.ChangeEvent{Text: "400 ms"})
				})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				finish(nil)
				if s.pendingCursorAction != nil {
					finish(nil)
				}
			})
			if s.pendingCursorAction != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("pending selection changed intervening state", scenario, s.errorMessage)
			}
		})
	}
}

func TestRecordedCursorActionSettlementFailurePreservesState(t *testing.T) {
	for _, admission := range []bool{true, false} {
		t.Run(map[bool]string{true: "admission", false: "completion"}[admission], func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error {
				finish = done
				if admission {
					return errors.New("recorded field retired")
				}
				return nil
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				button, _ := findNode(s.view(), "recorded-cursor-select-b")
				button.Activate()
				if !admission {
					finish(errors.New("composition incomplete"))
				}
			})
			if s.pendingCursorAction != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("failed settlement selected a recorded cursor", s.errorMessage)
			}
		})
	}
}
