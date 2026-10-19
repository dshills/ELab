package main

import (
	"errors"
	"reflect"
	"testing"

	"cogentcore.org/core/core"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func TestDCSweepDeferredSettingsCaptureOnlyOnce(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	type settlement struct {
		id   string
		done func(error)
	}
	var settlements []settlement
	s.settleField = func(id string, done func(error)) error {
		settlements = append(settlements, settlement{id, done})
		return nil
	}
	uiAction(t, s, "sweep-start", nil)
	for i, text := range []string{"1 V", "4 V", "7"} {
		if len(settlements) != i+1 || settlements[i].id != sweepStartFields[i] || len(s.service.Snapshot().DCSweep.Operations) != 0 {
			t.Fatal("sweep did not wait for its settings", settlements)
		}
		field := verificationControl(t, s, sweepStartFields[i])
		s.app.Do(func() {
			field.Change(ui.ChangeEvent{Text: text})
			settlements[i].done(nil)
			settlements[i].done(nil)
		})
	}
	state := s.service.Snapshot()
	if s.errorMessage != "" || s.pendingSweepStart != nil || len(state.DCSweep.Operations) != 1 || state.DCSweep.Settings.Start != "1 V" || state.DCSweep.Settings.Stop != "4 V" || state.DCSweep.Settings.Points != 7 {
		t.Fatal("deferred fields lost settings or started twice", s.errorMessage, state.DCSweep.Settings)
	}
}

func TestDCSweepSettlementPreservesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-before-field", "agent-after-field", "selection", "document", "other-user-field", "other-sweep-field", "wrong-pending-field", "rejected-field", "settlement-error", "admission-error"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "simulation.pause", nil)
			field := verificationControl(t, s, "sweep-range-start")
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error {
				finish = done
				if scenario == "admission-error" {
					return errors.New("field retired")
				}
				return nil
			}
			button := verificationControl(t, s, "sweep-start")
			s.app.Do(button.Activate)
			if finish == nil {
				t.Fatal("Run did not request settlement")
			}
			if scenario == "agent-before-field" {
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			}
			if scenario == "wrong-pending-field" {
				field = verificationControl(t, s, "sweep-points")
			}
			if scenario != "admission-error" {
				text := "1 V"
				if scenario == "rejected-field" {
					text = "nope"
				}
				s.app.Do(func() { field.Change(ui.ChangeEvent{Text: text}) })
			}
			switch scenario {
			case "agent-after-field":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			case "selection":
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "D1"})
			case "document":
				agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
			case "other-user-field":
				uiAction(t, s, "display-voltage-range", &ui.ChangeEvent{Number: 10})
			case "other-sweep-field":
				uiAction(t, s, "sweep-probe", &ui.ChangeEvent{SelectedID: "i:D1"})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				if scenario == "settlement-error" {
					finish(errors.New("composition incomplete"))
				} else {
					finish(nil)
				}
			})
			for i := 0; i < 3 && s.pendingSweepStart != nil; i++ {
				s.app.Do(func() { finish(nil) })
			}
			if s.errorMessage == "" || s.pendingSweepStart != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("Run ignored interruption or captured intervening work", scenario, s.errorMessage)
			}
		})
	}
}

func TestDCSweepInvalidFieldCannotFallBackAfterAnotherSuccess(t *testing.T) {
	for _, edit := range []struct{ id, bad, good string }{{"sweep-range-start", "nope", "1 V"}, {"sweep-range-stop", "13 V", "4 V"}, {"sweep-points", "seven", "7"}} {
		t.Run(edit.id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			field := verificationControl(t, s, edit.id)
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: edit.bad}) })
			if s.errorMessage == "" {
				t.Fatal("invalid field accepted")
			}
			other := "sweep-points"
			text := "11"
			if edit.id == other {
				other, text = "sweep-range-stop", "4 V"
			}
			uiAction(t, s, other, &ui.ChangeEvent{Text: text})
			before := s.service.Snapshot()
			button := verificationControl(t, s, "sweep-start")
			s.app.Do(button.Activate)
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("other field hid invalid text and started with old settings")
			}
			uiAction(t, s, edit.id, &ui.ChangeEvent{Text: edit.good})
			uiAction(t, s, "sweep-start", nil)
			if s.errorMessage != "" || len(s.service.Snapshot().DCSweep.Operations) != 1 {
				t.Fatal("fresh correction cannot Run", s.errorMessage)
			}
		})
	}
}

func TestDCSweepOwnViewportContinuationDoesNotAdoptOtherWork(t *testing.T) {
	for _, gap := range []string{"none", "agent", "user", "document"} {
		t.Run(gap, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "simulation.pause", nil)
			button := verificationControl(t, s, "sweep-start")
			uiAction(t, s, "sweep-points", &ui.ChangeEvent{Text: "7"})
			switch gap {
			case "agent":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			case "user":
				uiAction(t, s, "display-voltage-range", &ui.ChangeEvent{Number: 10})
			case "document":
				agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
			}
			s.drawing.widget = &core.WidgetBase{}
			beforeViewport := s.service.Snapshot()
			s.app.Do(func() { s.observeSchematicViewport(canvas.Viewport{Width: 520, Height: 138, Zoom: 1}, beforeViewport) })
			drainViewport(t, s, func() bool { return s.service.Snapshot().Viewport.Registered })
			before := s.service.Snapshot()
			s.app.Do(button.Activate)
			after := s.service.Snapshot()
			if gap == "none" {
				if s.errorMessage != "" || len(after.DCSweep.Operations) != 1 || after.DCSweep.Settings.Points != 7 {
					t.Fatal("own viewport lost settings handoff", s.errorMessage)
				}
			} else if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
				t.Fatal("viewport hid unrelated work", gap, s.errorMessage)
			}
		})
	}
}
