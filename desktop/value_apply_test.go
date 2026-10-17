package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestValueApplyWaitsForNativeFieldAndUsesSharedCommands(t *testing.T) {
	for _, kind := range []string{"resistor", "net-label"} {
		t.Run(kind, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "X1", Kind: kind})
			uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "X1"})
			var finish func(error)
			s.settleField = func(id string, done func(error)) error {
				if id != "value-entry" {
					t.Fatal(id)
				}
				finish = done
				return nil
			}
			before := s.service.Snapshot()
			uiAction(t, s, "apply-value", nil)
			if finish == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("Apply did not wait for native text")
			}
			value := "10k"
			if kind == "net-label" {
				value = "OUTPUT"
			}
			uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: value})
			s.app.Do(func() { finish(nil); finish(nil) })
			after := s.service.Snapshot()
			if after.CommandRevision != before.CommandRevision+2 || s.pendingValueApply != nil || s.errorMessage != "" {
				t.Fatal("expected exactly draft + component commands", s.errorMessage)
			}
			if kind == "resistor" && after.Components[0].Value != 10000 || kind == "net-label" && after.Components[0].Label != value {
				t.Fatal("settled text not applied", after.Components[0])
			}
		})
	}
}

func TestValueApplySettlementPreservesInterveningAgentWork(t *testing.T) {
	for _, when := range []string{"before-change", "after-change", "document-replaced"} {
		t.Run(when, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
			var field ui.Node
			s.app.Do(func() { field, _ = findNode(s.view(), "value-entry") })
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
			uiAction(t, s, "apply-value", nil)
			if when == "after-change" {
				s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "10k"}) })
			}
			if when == "document-replaced" {
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			} else {
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				if when != "after-change" {
					field.Change(ui.ChangeEvent{Text: "10k"})
				}
				finish(nil)
			})
			if s.errorMessage == "" || s.pendingValueApply != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("settlement overwrote intervening work", s.errorMessage)
			}
		})
	}
}

func TestValueApplySettlementFailureDoesNotApplyOldText(t *testing.T) {
	for _, mode := range []string{"admission", "completion", "invalid-draft"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
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
				apply, _ := findNode(s.view(), "apply-value")
				apply.Activate()
				if mode == "invalid-draft" {
					field, _ := findNode(s.view(), "value-entry")
					field.Change(ui.ChangeEvent{Text: strings.Repeat("x", 129)})
					finish(nil)
				} else if mode == "completion" {
					finish(errors.New("composition unfinished"))
				}
			})
			if s.errorMessage == "" || s.pendingValueApply != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("failed settlement applied stale value", s.errorMessage)
			}
		})
	}
}

func TestValueApplyDoesNotAdoptARefreshedFieldAfterAgentEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	var finish func(error)
	s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
	uiAction(t, s, "apply-value", nil)
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	s.app.Do(func() {
		field, _ := findNode(s.view(), "value-entry")
		field.Change(ui.ChangeEvent{Text: "10k"})
		finish(nil)
	})
	state := s.service.Snapshot()
	if s.errorMessage == "" || s.pendingValueApply != nil || state.Circuit.Resistance != 2000 || state.Drawing.Value != "10k" {
		t.Fatal("pending Apply adopted a later field context", s.errorMessage)
	}
}
