package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestCursorActionWaitsForDeferredTimeFields(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	t.Cleanup(cleanup)
	finish := map[string]func(error){}
	requests := 0
	s.settleField = func(id string, done func(error)) error {
		requests++
		finish[id] = done
		return nil
	}
	var original ui.Node
	s.app.Do(func() { original, _ = findNode(s.view(), "cursor-select-b") })
	before := s.service.Snapshot()
	uiAction(t, s, "cursor-select-b", nil)
	if finish["cursor-time-a"] == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("selection did not wait for native cursor field completion")
	}
	s.app.Do(original.Activate)
	var pendingButton ui.Node
	s.app.Do(func() { pendingButton, _ = findNode(s.view(), "cursor-select-b") })
	if requests != 1 || !pendingButton.Disabled {
		t.Fatal("pending click was admitted twice or remained enabled")
	}
	uiAction(t, s, "cursor-time-a", &ui.ChangeEvent{Text: "4 ms"})
	s.app.Do(func() { finish["cursor-time-a"](nil) })
	if finish["cursor-time-b"] == nil || s.service.Snapshot().Scope.ActiveCursor != "a" {
		t.Fatal("selection ran before the second field settled")
	}
	uiAction(t, s, "cursor-time-b", &ui.ChangeEvent{Text: "16 ms"})
	s.app.Do(func() {
		finish["cursor-time-a"](nil)
		finish["cursor-time-b"](nil)
		finish["cursor-time-b"](nil)
	})
	after := s.service.Snapshot()
	if s.errorMessage != "" || s.pendingCursorAction != nil || after.Scope.ActiveCursor != "b" || after.Scope.A == nil || after.Scope.B == nil || after.Scope.A.Requested != .004 || after.Scope.B.Requested != .016 || after.CommandRevision != before.CommandRevision+3 {
		t.Fatal("deferred cursor commits/selection did not run once", s.errorMessage, after.Scope)
	}
}

func TestCursorActionDeferredCommitDoesNotAdoptInterveningAgentContext(t *testing.T) {
	for _, scenario := range []string{"agent-before-field", "agent-after-field", "other-user-field", "document-replaced"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			t.Cleanup(cleanup)
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
			var field ui.Node
			s.app.Do(func() { field, _ = findNode(s.view(), "cursor-time-a") })
			uiAction(t, s, "cursor-select-b", nil)
			if finish == nil {
				t.Fatal("selection did not wait for native field")
			}
			if scenario == "agent-after-field" {
				uiAction(t, s, "cursor-time-a", &ui.ChangeEvent{Text: "4 ms"})
			}
			switch scenario {
			case "document-replaced":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "other-user-field":
				uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 11})
			default:
				agentAction(t, s.service, "scope.cursor.set", workbench.CursorParams{Cursor: "a", Time: .012})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				if scenario != "agent-after-field" {
					// A repaint can rebind the same field to a newer context.
					// The original pending click must refuse before dispatch.
					if scenario != "document-replaced" {
						field, _ = findNode(s.view(), "cursor-time-a")
					}
					field.Change(ui.ChangeEvent{Text: "4 ms"})
				}
				finish(nil)
				if s.pendingCursorAction != nil {
					finish(nil) // second field completion after an intervening actor
				}
			})
			if s.errorMessage == "" || s.pendingCursorAction != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("deferred cursor edit or selection overwrote intervening work", s.errorMessage)
			}
		})
	}
}

func TestCursorActionSettlementFailurePreservesCursorState(t *testing.T) {
	for _, mode := range []string{"admission", "completion", "invalid-time"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			t.Cleanup(cleanup)
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error {
				finish = done
				if mode == "admission" {
					return errors.New("field retired")
				}
				return nil
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				button, _ := findNode(s.view(), "cursor-select-b")
				button.Activate()
				if finish == nil {
					t.Fatal("missing native field settlement")
				}
				if mode == "completion" {
					finish(errors.New("composition unfinished"))
				} else if mode == "invalid-time" {
					field, _ := findNode(s.view(), "cursor-time-a")
					field.Change(ui.ChangeEvent{Text: "400 ms"})
					finish(nil)
				}
			})
			if s.errorMessage == "" || s.pendingCursorAction != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("failed native cursor commit still selected B", s.errorMessage)
			}
		})
	}
}

func TestCursorActionContinuesOnlyConsecutiveOwnTimeFields(t *testing.T) {
	for _, intermediate := range []bool{false, true} {
		name := "original-action"
		if intermediate {
			name = "intermediate-action"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			t.Cleanup(cleanup)
			var button ui.Node
			s.app.Do(func() { button, _ = findNode(s.view(), "cursor-select-b") })
			before := s.service.Snapshot()
			uiAction(t, s, "cursor-time-a", &ui.ChangeEvent{Text: "4 ms"})
			if intermediate {
				s.app.Do(func() { button, _ = findNode(s.view(), "cursor-select-b") })
			}
			uiAction(t, s, "cursor-time-b", &ui.ChangeEvent{Text: "16 ms"})
			s.app.Do(button.Activate)
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.Scope.ActiveCursor != "b" || after.CommandRevision != before.CommandRevision+3 || after.Scope.A == nil || after.Scope.B == nil || after.Scope.A.Requested != .004 || after.Scope.B.Requested != .016 {
				t.Fatal("own cursor field responses failed to continue into Select", s.errorMessage, after.Scope)
			}
		})
	}
}

func TestCursorActionContinuationRefusesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-before-field", "agent-after-field", "other-user-field", "document-replaced", "failed-field"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			t.Cleanup(cleanup)
			var button ui.Node
			s.app.Do(func() { button, _ = findNode(s.view(), "cursor-select-b") })
			if scenario == "agent-before-field" {
				agentAction(t, s.service, "scope.cursor.set", workbench.CursorParams{Cursor: "b", Time: .012})
			}
			uiAction(t, s, "cursor-time-a", &ui.ChangeEvent{Text: "4 ms"})
			switch scenario {
			case "agent-after-field":
				agentAction(t, s.service, "scope.cursor.set", workbench.CursorParams{Cursor: "b", Time: .012})
			case "other-user-field":
				uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 11})
			case "document-replaced":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "failed-field":
				s.app.Do(func() {
					field, _ := findNode(s.view(), "cursor-time-a")
					field.Change(ui.ChangeEvent{Text: "400 ms"})
				})
				if s.errorMessage == "" {
					t.Fatal("out-of-history cursor time was accepted")
				}
			}
			before := s.service.Snapshot()
			s.app.Do(button.Activate)
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("cursor continuation borrowed an intervening/failed context", s.errorMessage)
			}
		})
	}
}
