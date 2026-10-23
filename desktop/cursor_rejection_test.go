package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestCursorParserRefusalCannotRetainAnotherOwnersText(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name := "live"
		if recorded {
			name = "recorded"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			id := "cursor-time-a"
			if recorded {
				recordedCursorSource(t, s.service)
				id = "recorded-cursor-a"
			} else {
				uiAction(t, s, "example.rc", nil)
				uiAction(t, s, "simulation.pause", nil)
				uiAction(t, s, "simulation.restart", nil)
				agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
			}
			var old ui.Node
			s.app.Do(func() {
				var ok bool
				old, ok = findNode(s.view(), id)
				if !ok {
					t.Fatal("missing cursor field")
				}
			})
			if recorded {
				duration := .02
				agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
				completedVerification(t, s.service, s.service.Snapshot().LastOperation)
			} else {
				agentAction(t, s.service, "simulation.restart", struct{}{})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				old.Change(ui.ChangeEvent{Text: "oops"})
				field, _ := findNode(s.view(), id)
				if field.Text == "oops" || len(s.cursorTimeRejected) != 0 {
					t.Fatal("stale parser failure acquired the new owner's field")
				}
			})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("stale invalid text changed accepted state")
			}
		})
	}
}
