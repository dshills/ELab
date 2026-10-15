package main

import (
	"reflect"
	"slices"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestComponentListFocusOrderAndNoMatches(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "part-search", &ui.ChangeEvent{Text: "battery"})
	root := s.view()
	list, ok := findNode(root, "part-list")
	if !ok || list.Kind != ui.ListBox || len(list.Options) != 1 || list.Options[0].ID != "dc-source" || root.Focus.ID != "part-list" {
		t.Fatal("alias filter does not feed managed list")
	}
	uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "dc-source"})
	root = s.view()
	order := ui.FocusOrder(root)
	previous := -1
	for _, id := range []string{"part-search", "part-list", "position-x", "position-y", "place-at-position", "wire-by-name"} {
		index := slices.Index(order, id)
		if id == "wire-by-name" {
			continue
		} // Empty drawings cannot connect yet.
		if index <= previous || previous >= 0 && index != previous+1 {
			t.Fatal("placement is not in consecutive keyboard order", order)
		}
		previous = index
	}
	row, _ := findNode(root, "position-row")
	if row.Kind != ui.Column {
		t.Fatal("narrow rail coordinates can clip side-by-side")
	}
	uiAction(t, s, "place-at-position", nil)
	if s.view().Focus.ID != "part-search" {
		t.Fatal("removed placement action lost focus continuation")
	}
	before := s.service.Snapshot()
	uiAction(t, s, "part-search", &ui.ChangeEvent{Text: "no-matching-device"})
	list, _ = findNode(s.view(), "part-list")
	if !list.Disabled || len(list.Options) != 0 || list.SelectedID != "" || s.view().Focus.ID != "part-search" {
		t.Fatal("empty filter leaves stale selection")
	}
	if !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) {
		t.Fatal("filtering changed drawing")
	}
}

func TestComponentListStaleBindingAndFailedPlacementPreserveFocus(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	var old ui.Node
	s.app.Do(func() { old, _ = findNode(s.view(), "part-list") })
	agentAction(t, s.service, "parts.search.set", workbench.SearchParams{Query: "capacitor"})
	before := s.service.Snapshot()
	s.app.Do(func() { old.Change(ui.ChangeEvent{SelectedID: "resistor"}) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old list choice overwrote newer agent context")
	}
	uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "capacitor"})
	uiAction(t, s, "place-at-position", nil)
	uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "capacitor"})
	focus := s.keyboardFocus
	before = s.service.Snapshot()
	s.app.Do(func() { place, _ := findAction(s.view(), "place-at-position"); place.Activate() })
	if s.errorMessage == "" || s.keyboardFocus != focus || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("occupied placement moved focus or circuit")
	}
}

func TestNamedWireWorkflowIsNearbyAndUsesSharedCommands(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "dc-source"})
	uiAction(t, s, "place-at-position", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "wire-by-name", nil)
	if s.view().Focus.ID != "wire-from" {
		t.Fatal("named-wire entry did not focus first terminal")
	}
	order := ui.FocusOrder(s.view())
	a, b, c := slices.Index(order, "wire-from"), slices.Index(order, "wire-to"), slices.Index(order, "connect-terminals")
	if a < 0 || b != a+1 || c != -1 {
		t.Fatal("terminal selectors not adjacent or premature connect enabled", order)
	}
	uiAction(t, s, "wire-from", &ui.ChangeEvent{SelectedID: "V1.positive"})
	uiAction(t, s, "wire-to", &ui.ChangeEvent{SelectedID: "V1.negative"})
	uiAction(t, s, "connect-terminals", nil)
	state := s.service.Snapshot()
	if len(state.Wires) != 1 || state.Wires[0].From != "V1.positive" || state.Wires[0].To != "V1.negative" || state.Drawing.From != "" || s.view().Focus.ID != "wire-from" {
		t.Fatal("named wire did not use ordinary guarded command")
	}
	// This deliberately incomplete drawing is allowed; wire edit adds one history item.
	if !reflect.DeepEqual(before.Components, state.Components) || state.UndoName != "Connect wire" {
		t.Fatal("wire workflow changed component data or lost history")
	}
	uiAction(t, s, "drawing.cancel", nil)
	if s.view().Focus.ID != "part-search" || s.service.Snapshot().Drawing.Tool != "select" {
		t.Fatal("Escape did not return to part search")
	}
}
