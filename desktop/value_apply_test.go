package main

import (
	"errors"
	"fmt"
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

func TestValueApplyContinuesOwnDraftCommittedBeforeClick(t *testing.T) {
	for _, tc := range []struct{ changes, captureAfter int }{{1, 0}, {2, 0}, {2, 1}} {
		changes := tc.changes
		t.Run(fmt.Sprintf("%d-drafts/capture-%d", changes, tc.captureAfter), func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
			var field, apply ui.Node
			s.app.Do(func() {
				root := s.view()
				field, _ = findNode(root, "value-entry")
				apply, _ = findNode(root, "apply-value")
			})
			before := s.service.Snapshot()
			values := []string{"2.2k", "3.3k"}
			s.app.Do(func() {
				for i, text := range values[:changes] {
					field.Change(ui.ChangeEvent{Text: text})
					if tc.captureAfter == i+1 {
						apply, _ = findNode(s.view(), "apply-value")
					}
				}
				apply.Activate()
			})
			after := s.service.Snapshot()
			want := 2200.
			if changes == 2 {
				want = 3300
			}
			if s.errorMessage != "" || after.Circuit.Resistance != want || after.CommandRevision != before.CommandRevision+uint64(changes)+1 {
				t.Fatal("own accepted value drafts did not continue into one Apply", s.errorMessage, after.Circuit.Resistance)
			}
		})
	}
}

func TestValueApplyOwnDraftContinuationRefusesInterveningEdits(t *testing.T) {
	for _, when := range []string{"agent-before-draft", "agent-after-draft", "other-user-field", "new-document"} {
		t.Run(when, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
			var apply ui.Node
			s.app.Do(func() { apply, _ = findNode(s.view(), "apply-value") })
			if when == "agent-before-draft" {
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			}
			uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "3.3k"})
			switch when {
			case "agent-after-draft":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			case "other-user-field":
				uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 11})
			case "new-document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			before := s.service.Snapshot()
			s.app.Do(func() { apply.Activate() })
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingValueApply != nil {
				t.Fatal("own-draft continuation borrowed an intervening actor/field/document context")
			}
		})
	}
}
