package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func prepareValuePreset(t *testing.T, s *screen) ui.Node {
	t.Helper()
	uiAction(t, s, "example.reset", nil)
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	uiAction(t, s, "workspace-details-component", &ui.ChangeEvent{Checked: true})
	var field ui.Node
	s.app.Do(func() { field, _ = findNode(s.view(), "value-entry") })
	return field
}

func TestValuePresetDeferredDraftKeepsExplicitValue(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	field := prepareValuePreset(t, s)
	before := s.service.Snapshot()
	var finish func(error)
	s.settleField = func(id string, done func(error)) error {
		if id != "value-entry" {
			t.Fatal("settled an unrelated field", id)
		}
		finish = done
		return nil
	}
	uiAction(t, s, "r-2k", nil)
	if finish == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("preset must wait for the native field")
	}
	s.app.Do(func() {
		field.Change(ui.ChangeEvent{Text: "9k"})
		finish(nil)
		finish(nil)
	})
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || s.pendingValueApply != nil {
		t.Fatal("deferred preset did not complete once", s.errorMessage)
	}
	for _, part := range after.Components {
		if part.ID == "R1" && part.Value != 2000 {
			t.Fatal("preset adopted the draft instead of its explicit value", part.Value)
		}
	}
	uiAction(t, s, "edit.undo", nil)
	if !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) {
		t.Fatal("preset created more than one circuit edit")
	}
}

func TestValuePresetSettlementPreservesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-before-draft", "agent-after-draft", "selection", "document", "other-user-field", "rejected-field", "settlement-error", "admission-error"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			field := prepareValuePreset(t, s)
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error {
				finish = done
				if scenario == "admission-error" {
					return errors.New("field retired")
				}
				return nil
			}
			before := s.service.Snapshot()
			s.app.Do(func() {
				preset, _ := findNode(s.view(), "r-2k")
				preset.Activate()
			})
			if finish == nil {
				t.Fatal("preset did not request field settlement")
			}
			if scenario == "agent-before-draft" {
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "4k"})
				before = s.service.Snapshot()
				s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "9k"}) })
			} else if scenario == "rejected-field" {
				s.app.Do(func() { field.Change(ui.ChangeEvent{Text: strings.Repeat("x", 129)}) })
			} else if scenario != "admission-error" {
				s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "9k"}) })
				switch scenario {
				case "agent-after-draft":
					agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "4k"})
				case "selection":
					agentAction(t, s.service, "component.select", workbench.IDParams{ID: "D1"})
				case "document":
					agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
				case "other-user-field":
					uiAction(t, s, "display-voltage-range", &ui.ChangeEvent{Number: 10})
				}
				before = s.service.Snapshot()
			}
			s.app.Do(func() {
				if scenario == "settlement-error" {
					finish(errors.New("composition incomplete"))
				} else {
					finish(nil)
				}
			})
			if s.errorMessage == "" || s.pendingValueApply != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("preset changed intervening work or ignored a settlement failure", scenario, s.errorMessage)
			}
		})
	}
}
