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

func verificationControl(t *testing.T, s *screen, id string) ui.Node {
	t.Helper()
	if _, ok := findNode(s.view(), id); !ok {
		uiAction(t, s, "inspector-section-simulation", nil)
		for _, option := range simulationTools() {
			if _, ok := findNode(s.view(), id); ok {
				break
			}
			uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: option.ID})
		}
	}
	var node ui.Node
	s.app.Do(func() {
		var ok bool
		node, ok = findNode(s.view(), id)
		if !ok {
			t.Fatal("missing production verification control", id)
		}
	})
	return node
}

func TestVerificationDeferredFieldsCompleteOnlyOnce(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	agentAction(t, s.service, "simulation.pause", struct{}{})
	type settlement struct {
		id   string
		done func(error)
	}
	var work []settlement
	settle := func(id string, done func(error)) error { work = append(work, settlement{id, done}); return nil }
	s.settleField, s.settleNumber = settle, settle
	before := s.service.Snapshot()
	uiAction(t, s, "verify-start", nil)
	for i, id := range []string{"verify-duration", "verify-step", "verify-initial", "verify-cap-C1"} {
		if len(work) != i+1 || work[i].id != id {
			t.Fatal("action did not await next form field", work)
		}
		if len(s.service.Snapshot().Operations) != 0 {
			t.Fatal("capture started before settlement")
		}
		if i == 0 || i == 1 {
			field := verificationControl(t, s, id)
			text := "20 ms"
			if i == 1 {
				text = "1 ms"
			}
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: text}) })
		}
		s.app.Do(func() { work[i].done(nil); work[i].done(nil) })
	}
	state := s.service.Snapshot()
	if s.errorMessage != "" || len(state.Operations) != 1 || state.VerificationSettings.Duration != .02 || state.VerificationSettings.Step != .001 || before.RunID != state.RunID || !reflect.DeepEqual(before.Transient, state.Transient) {
		t.Fatal("deferred fields did not capture exactly once", s.errorMessage, state.VerificationSettings)
	}
}

func TestVerificationSettlementPreservesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-before-field", "agent-after-field", "selection", "document", "other-user-field", "other-form-field", "rejected-field", "settlement-error", "admission-error"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.rc", nil)
			agentAction(t, s.service, "simulation.pause", struct{}{})
			field := verificationControl(t, s, "verify-duration")
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error {
				finish = done
				if scenario == "admission-error" {
					return errors.New("field retired")
				}
				return nil
			}
			button := verificationControl(t, s, "verify-start")
			s.app.Do(button.Activate)
			if finish == nil {
				t.Fatal("Verify did not request field settlement")
			}
			if scenario == "agent-before-field" {
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			}
			if scenario == "rejected-field" {
				s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "nope"}) })
			} else if scenario != "admission-error" {
				s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "20 ms"}) })
			}
			switch scenario {
			case "agent-after-field":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			case "selection":
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "C1"})
			case "document":
				agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
			case "other-user-field":
				uiAction(t, s, "display-voltage-range", &ui.ChangeEvent{Number: 10})
			case "other-form-field":
				other := verificationControl(t, s, "verify-mode")
				s.app.Do(func() { other.Change(ui.ChangeEvent{SelectedID: "discharge"}) })
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				if scenario == "settlement-error" {
					finish(errors.New("composition incomplete"))
				} else {
					finish(nil)
				}
			})
			// A failed first field must stop before asking for the next one. An
			// intervening command can instead be detected at the final guarded dispatch.
			for i := 0; i < 4 && s.pendingVerificationAction != nil; i++ {
				s.app.Do(func() { finish(nil) })
			}
			if s.errorMessage == "" || s.pendingVerificationAction != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("Verify changed intervening work or ignored failure", scenario, s.errorMessage)
			}
		})
	}
}

func TestVerificationOwnViewportContinuationDoesNotAdoptOtherWork(t *testing.T) {
	for _, gap := range []string{"none", "agent", "user", "document"} {
		t.Run(gap, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.rc", nil)
			agentAction(t, s.service, "simulation.pause", struct{}{})
			button := verificationControl(t, s, "verify-start")
			field := verificationControl(t, s, "verify-duration")
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "20 ms"}) })
			switch gap {
			case "agent":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			case "user":
				uiAction(t, s, "display-voltage-range", &ui.ChangeEvent{Number: 10})
			case "document":
				agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
			}
			s.drawing.widget = &core.WidgetBase{}
			beforeViewport := s.service.Snapshot()
			s.app.Do(func() { s.observeSchematicViewport(canvas.Viewport{Width: 520, Height: 138, Zoom: 1}, beforeViewport) })
			drainViewport(t, s, func() bool { return s.service.Snapshot().Viewport.Registered })
			before := s.service.Snapshot()
			s.app.Do(button.Activate)
			after := s.service.Snapshot()
			if gap == "none" {
				if s.errorMessage != "" || len(after.Operations) != 1 || after.Operations[0].RequestedDuration != .02 {
					t.Fatal("own viewport lost field handoff", s.errorMessage)
				}
			} else if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
				t.Fatal("viewport hid intervening actor/document", gap, s.errorMessage)
			}
		})
	}
}
