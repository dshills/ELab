package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestGeometryWaitsForOwnDeferredValueDraft(t *testing.T) {
	for _, control := range []string{"rotate-part", "duplicate-part"} {
		t.Run(control, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			field := prepareValuePreset(t, s)
			before := s.service.Snapshot()
			var finish func(error)
			s.settleField = func(id string, done func(error)) error {
				if id != "value-entry" {
					t.Fatal("settled unrelated field", id)
				}
				finish = done
				return nil
			}
			uiAction(t, s, control, nil)
			if finish == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("geometry did not wait for the private field")
			}
			s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "9k"}); finish(nil); finish(nil) })
			after := s.service.Snapshot()
			if s.errorMessage != "" || s.pendingValueApply != nil || after.CommandRevision != before.CommandRevision+2 {
				t.Fatal("deferred action did not complete exactly once", s.errorMessage)
			}
			for _, p := range after.Components {
				if p.Kind == "resistor" && p.Value != 3000 {
					t.Fatal("geometry implicitly applied a private value", p)
				}
			}
			uiAction(t, s, "edit.undo", nil)
			if !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) {
				t.Fatal("one Undo did not restore the graph")
			}
		})
	}
}

func TestGeometrySettlementPreservesInterveningWork(t *testing.T) {
	for _, control := range []string{"rotate-part", "duplicate-part"} {
		for _, scenario := range []string{"agent-before-draft", "agent-after-draft", "selection", "document", "other-user-field", "rejected-field", "settlement-error", "admission-error"} {
			t.Run(control+"/"+scenario, func(t *testing.T) {
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
					button, _ := findNode(s.view(), control)
					button.Activate()
				})
				if finish == nil {
					t.Fatal("geometry did not request settlement")
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
					t.Fatal("geometry ignored intervening work or settlement failure", scenario, s.errorMessage)
				}
			})
		}
	}
}

func TestGeometryContinuesOnlyOwnDraftBeforeActivation(t *testing.T) {
	for _, control := range []string{"rotate-part", "duplicate-part"} {
		for _, intervene := range []bool{false, true} {
			t.Run(control+map[bool]string{false: "/own", true: "/agent"}[intervene], func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				field := prepareValuePreset(t, s)
				var button ui.Node
				s.app.Do(func() { button, _ = findNode(s.view(), control); field.Change(ui.ChangeEvent{Text: "9k"}) })
				if intervene {
					agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "4k"})
				}
				before := s.service.Snapshot()
				s.app.Do(button.Activate)
				after := s.service.Snapshot()
				if intervene {
					if s.errorMessage == "" || !reflect.DeepEqual(before, after) {
						t.Fatal("borrowed agent revision")
					}
				} else if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 {
					t.Fatal("own blur continuation failed", s.errorMessage)
				}
			})
		}
	}
}
