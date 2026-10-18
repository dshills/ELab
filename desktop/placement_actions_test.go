package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestPlacementContinuesOnlyOwnAcceptedParameters(t *testing.T) {
	for _, kind := range []string{"resistor", "net-label"} {
		t.Run(kind, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: kind})
			var button ui.Node
			s.app.Do(func() { button, _ = findNode(s.view(), "place-at-position") })
			uiAction(t, s, "position-x", &ui.ChangeEvent{Number: 126})
			uiAction(t, s, "position-y", &ui.ChangeEvent{Number: -94})
			if kind == "net-label" {
				uiAction(t, s, "place-label", &ui.ChangeEvent{Text: "SUPPLY"})
			}
			before := s.service.Snapshot()
			s.app.Do(button.Activate)
			after := s.service.Snapshot()
			if s.errorMessage != "" || len(after.Components) != 1 || after.Components[0].Kind != kind || after.Components[0].Position != (workbench.Point{X: 130, Y: -90}) || after.CommandRevision != before.CommandRevision+1 || s.keyboardFocus.ID != "part-search" {
				t.Fatal("placement did not use its own accepted/snapped parameters", after.Components, s.errorMessage)
			}
			if kind == "net-label" && after.Components[0].Label != "SUPPLY" {
				t.Fatal("label lost accepted value")
			}
		})
	}
}

func TestPlacementContinuationRefusesInterveningWork(t *testing.T) {
	for _, scenario := range []string{"agent-before", "agent-after", "other-field", "new-document", "different-tool", "failed-field"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "resistor"})
			var button ui.Node
			s.app.Do(func() { button, _ = findNode(s.view(), "place-at-position") })
			if scenario == "agent-before" {
				agentAction(t, s.service, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: 200, Y: 100}})
			}
			uiAction(t, s, "position-x", &ui.ChangeEvent{Number: 120})
			switch scenario {
			case "agent-after":
				agentAction(t, s.service, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: 200, Y: 100}})
			case "other-field":
				uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 11})
			case "new-document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "different-tool":
				agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "capacitor"})
			case "failed-field":
				s.app.Do(func() { field, _ := findNode(s.view(), "position-y"); field.Change(ui.ChangeEvent{Number: 20000}) })
			}
			before := s.service.Snapshot()
			focus := s.keyboardFocus
			s.app.Do(button.Activate)
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.keyboardFocus != focus {
				t.Fatal("old placement borrowed intervening work or moved focus", s.errorMessage)
			}
		})
	}
}

func TestPlacementDeferredLabelSettlement(t *testing.T) {
	for _, scenario := range []string{"success", "agent-before-field", "agent-after-field", "other-field", "new-document", "admission-failure", "completion-failure", "invalid-label"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "net-label"})
			var finish func(error)
			requests := 0
			s.settleField = func(id string, done func(error)) error {
				if id != "place-label" {
					t.Fatal("settled an unrelated field", id)
				}
				finish = done
				requests++
				if scenario == "admission-failure" {
					return errors.New("retired label")
				}
				return nil
			}
			before := s.service.Snapshot()
			focus := s.keyboardFocus
			s.app.Do(func() { button, _ := findNode(s.view(), "place-at-position"); button.Activate() })
			if scenario == "admission-failure" {
				if s.pendingPlacement != nil || s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("admission failure placed component")
				}
				return
			}
			var held ui.Node
			s.app.Do(func() { held, _ = findNode(s.view(), "place-at-position"); held.Activate() })
			if !held.Disabled || requests != 1 || s.pendingPlacement == nil || s.keyboardFocus != focus {
				t.Fatal("pending placement repeated or moved focus")
			}
			if scenario == "agent-before-field" {
				agentAction(t, s.service, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: 200}})
			}
			if scenario != "completion-failure" {
				text := "SUPPLY"
				if scenario == "invalid-label" {
					text = "invalid label !"
				}
				s.app.Do(func() { field, _ := findNode(s.view(), "place-label"); field.Change(ui.ChangeEvent{Text: text}) })
			}
			switch scenario {
			case "agent-after-field":
				agentAction(t, s.service, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: 200}})
			case "other-field":
				uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 11})
			case "new-document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			current := s.service.Snapshot()
			s.app.Do(func() {
				var err error
				if scenario == "completion-failure" {
					err = errors.New("unfinished composition")
				}
				finish(err)
				finish(err)
			})
			after := s.service.Snapshot()
			if scenario == "success" {
				if s.errorMessage != "" || len(after.Components) != 1 || after.Components[0].Label != "SUPPLY" || after.CommandRevision != current.CommandRevision+1 || s.keyboardFocus.ID != "part-search" {
					t.Fatal("deferred label did not place once", s.errorMessage)
				}
			} else if s.errorMessage == "" || !reflect.DeepEqual(current, after) || s.keyboardFocus != focus {
				t.Fatal("failed/intervening label settlement placed component", s.errorMessage)
			}
			if s.pendingPlacement != nil {
				t.Fatal("placement pending after completion")
			}
		})
	}
}
