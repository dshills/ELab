//go:build offscreen

package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
)

func TestMountedScopeCursorRefusalRetainsTextAndRecovers(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name := "live"
		if recorded {
			name = "recorded"
		}
		t.Run(name, func(t *testing.T) {
			var s *screen
			var cleanup func()
			var m *mountedControlWindow
			id, command := "cursor-time-a", "scope.cursor.set"
			if recorded {
				s, cleanup = recordedCursorRC(t)
				t.Cleanup(cleanup)
				id, command = "recorded-cursor-a", "recording.cursor.set"
				m = mountedRecordedCursorControls(t, s)
			} else {
				s, cleanup = testScreen(t)
				t.Cleanup(cleanup)
				uiAction(t, s, "example.rc", nil)
				uiAction(t, s, "simulation.pause", nil)
				uiAction(t, s, "simulation.restart", nil)
				agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
				m = mountedCursorControls(t, s)
			}
			for _, bad := range []string{"oops", "10 Hz", "-1 ms", "30 ms"} {
				before := s.service.Snapshot()
				params := map[string]any{"cursor": "a", "time_s": bad}
				if recorded {
					params["operation_id"] = before.Recording.ID
				}
				raw, _ := json.Marshal(params)
				revision := before.CommandRevision
				response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: bad, Key: bad, Command: command, Params: raw, ExpectedCommandRevision: &revision}, "agent.refusal")
				if response.Error == nil || response.Error.Code != "invalid-params" || response.Help == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("fixture needs a refused agent time and unchanged state", response)
				}
				typeMountedCursor(t, m, id, bad)
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				m.frame()
				if field := mountedControlElement(t, m, id); field.Value != bad {
					t.Fatalf("rejected %q disappeared before it could be corrected: field=%q error=%q", bad, field.Value, s.errorMessage)
				}
				if !reflect.DeepEqual(before, s.service.Snapshot()) || !reflect.DeepEqual(response.Help, s.requestHelp) || !strings.Contains(s.requestHelp.Action, "0.02 s") || !strings.Contains(s.requestHelp.Action, "ms") {
					t.Fatal("native refusal changes state or hides shared bounds/units", s.requestHelp)
				}
				selectB := "cursor-select-b"
				if recorded {
					selectB = "recorded-cursor-select-b"
				}
				clickMountedControl(t, m, selectB)
				if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" {
					t.Fatal("selection silently bypassed a previously rejected cursor field")
				}
				typeMountedCursor(t, m, id, "4 ms")
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				after := s.service.Snapshot()
				cursor := after.Scope.A
				if recorded {
					cursor = after.Recording.Scope.A
				}
				if cursor == nil || !cursor.Valid || cursor.Requested != .004 || s.errorMessage != "" || s.requestHelp != nil {
					t.Fatal("first corrected commit did not recover", cursor, s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
				clickMountedControl(t, m, selectB)
				if s.errorMessage != "" {
					t.Fatal("corrected field still blocks cursor selection", s.errorMessage)
				}
			}
		})
	}
}

func TestMountedScopeCursorRefusalStaysWithItsOwner(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		for _, change := range []string{"repaint", "cursor-replaced", "new-run", "new-capture"} {
			if !recorded && change == "new-capture" {
				continue
			}
			name := "live/" + change
			if recorded {
				name = "recorded/" + change
			}
			t.Run(name, func(t *testing.T) {
				var s *screen
				var m *mountedControlWindow
				id, command := "cursor-time-a", "scope.cursor.set"
				if recorded {
					var cleanup func()
					s, cleanup = recordedCursorRC(t)
					t.Cleanup(cleanup)
					m = mountedRecordedCursorControls(t, s)
					id, command = "recorded-cursor-a", "recording.cursor.set"
				} else {
					var cleanup func()
					s, cleanup = testScreen(t)
					t.Cleanup(cleanup)
					uiAction(t, s, "example.rc", nil)
					uiAction(t, s, "simulation.pause", nil)
					uiAction(t, s, "simulation.restart", nil)
					agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
					m = mountedCursorControls(t, s)
				}
				typeMountedCursor(t, m, id, "oops")
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				m.frame()
				switch change {
				case "cursor-replaced":
					agentAction(t, s.service, command, workbench.CursorParams{Cursor: "a", Time: .004})
				case "new-run":
					agentAction(t, s.service, "simulation.restart", struct{}{})
				case "new-capture":
					duration := .02
					agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
					completedVerification(t, s.service, s.service.Snapshot().LastOperation)
				}
				m.frame()
				shouldRetain := change == "repaint" || recorded && change == "new-run"
				if field := mountedControlElement(t, m, id); (field.Value == "oops") != shouldRetain {
					t.Fatalf("rejected draft followed the wrong owner: retain=%t field=%q", shouldRetain, field.Value)
				}
			})
		}
	}
}

func TestMountedScopeCursorRefusalRetriesAfterHistoryAdvances(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "simulation.restart", nil)
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
	m := mountedCursorControls(t, s)
	typeMountedCursor(t, m, "cursor-time-a", "30 ms")
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	m.frame()
	if s.errorMessage == "" || s.service.Snapshot().Scope.A != nil {
		t.Fatal("fixture requires a rejected future time")
	}
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
	m.frame()
	if field := mountedControlElement(t, m, "cursor-time-a"); field.Value != "30 ms" {
		t.Fatal("same-run progress lost the pending time", field.Value)
	}
	before := s.service.Snapshot()
	// Selecting another cursor already settles pending native time fields. A
	// previously rejected, now valid quantity must use that same guarded path.
	clickMountedControl(t, m, "cursor-select-b")
	after := s.service.Snapshot()
	if after.Scope.A == nil || !after.Scope.A.Valid || after.Scope.A.Requested != .03 || after.Scope.ActiveCursor != "b" || after.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" {
		t.Fatal("retained time could not be retried after history grew", after.Scope.A, after.Scope.ActiveCursor, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}
