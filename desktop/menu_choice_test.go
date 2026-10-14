package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeMenuRejectsStaleDisabledAndUnlistedChoices(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	calls := 0
	choices := menuChoices{Kind: ui.MenuButton, Options: []ui.Option{{ID: "valid", Checkable: true}, {ID: "disabled", Disabled: true}, {ID: "separator", Separator: true}}, Change: func(e ui.ChangeEvent) {
		if e.SelectedID != "valid" || e.Kind != ui.MenuButton || !e.Checked {
			t.Fatal("wrong stable option ID", e.SelectedID)
		}
		calls++
	}}
	node := ui.Node{ID: "menu", Kind: ui.Selector}
	s.app.Do(func() {
		revision := s.service.Snapshot().CommandRevision
		s.acceptMenuChoice(node, choices, -1, s.menuSerial, revision)
		s.acceptMenuChoice(node, choices, 3, s.menuSerial, revision)
		s.acceptMenuChoice(node, choices, 2, s.menuSerial, revision)
		s.acceptMenuChoice(node, choices, 1, s.menuSerial, revision)
		s.acceptMenuChoice(node, choices, 0, s.menuSerial+1, revision)
		s.fileBusy = true
		s.acceptMenuChoice(node, choices, 0, s.menuSerial, revision)
		s.fileBusy = false
		if calls != 0 {
			t.Fatal("invalid choice was accepted")
		}
		s.acceptMenuChoice(node, choices, 0, s.menuSerial, revision)
		if calls != 1 {
			t.Fatal("fresh choice was refused")
		}
		s.invoke("component.place", workbench.PlaceParams{Kind: "resistor", Position: workbench.Point{X: 20}})
		s.acceptMenuChoice(node, choices, 0, s.menuSerial, revision)
		if calls != 1 {
			t.Fatal("stale menu changed agent-edited state")
		}
	})
}

func TestNativeMenuChoiceGuardCoversCommandAdmission(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	s.app.Do(func() {
		revision := s.service.Snapshot().CommandRevision
		choices := menuChoices{Options: []ui.Option{{ID: "__none"}}, Change: func(_ ui.ChangeEvent) {
			// Interleave an agent mutation after the popup predicate but before
			// its original callback dispatches the user's captured choice.
			raw, err := json.Marshal(workbench.PlaceParams{Kind: "resistor", Position: workbench.Point{X: 20}})
			if err != nil {
				t.Fatal(err)
			}
			response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "interleaved", Key: "interleaved", Command: "component.place", Params: raw, ExpectedCommandRevision: &revision}, "agent.interleaved")
			if response.Error != nil {
				t.Fatal(response.Error)
			}
			response = s.invoke("component.select", workbench.IDParams{})
			if response.Error == nil || response.Error.Code != "conflict" {
				t.Fatal("stale choice passed command admission", response.Error)
			}
		}}
		s.acceptMenuChoice(ui.Node{ID: "selection"}, choices, 0, s.menuSerial, revision)
		state := s.service.Snapshot()
		if state.Selected != "R1" || len(state.Components) != 1 || s.menuCommandRevision != nil {
			t.Fatal("stale menu lost agent state or leaked its guard")
		}
		if s.invoke("component.select", workbench.IDParams{}).Error != nil || s.service.Snapshot().Selected != "" {
			t.Fatal("fresh command did not recover after stale refusal")
		}
	})
}
