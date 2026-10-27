//go:build offscreen

package main

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedComponentBrowserSearchAndKeyboardPlacement(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	s.requestKeyboardFocus("part-search")
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: s.keyboardFocus, Children: s.componentPalette(state)}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	// Native fields commit on Return. Typing the whole query must stay in the
	// field; focus only transfers to results after the accepted search commit.
	for _, r := range "MOSFET" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	if state := s.service.Snapshot(); state.PartSearch != "" || len(state.MatchingParts) != 6 {
		t.Fatal("typing prematurely committed or lost Common")
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	state := s.service.Snapshot()
	if state.PartSearch != "MOSFET" || len(state.MatchingParts) != 2 || s.keyboardFocus.ID != "part-list" {
		t.Fatalf("full search did not reach results: %s %+v %s", state.PartSearch, state.MatchingParts, s.errorMessage)
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	if state = s.service.Snapshot(); state.Drawing.Tool != "place" || state.Drawing.Kind != "pmos" {
		t.Fatal("Return did not choose first matching part", state.Drawing)
	}
	// A shared agent category edit immediately changes the native catalog and
	// clears the query without changing the selected placement kind.
	agentAction(t, s.service, "parts.category.set", workbench.PartCategoryParams{Category: "passives"})
	m.frame()
	if state = s.service.Snapshot(); state.PartSearch != "" || len(state.MatchingParts) != 10 || state.Drawing.Kind != "pmos" {
		t.Fatal("agent category did not update UI")
	}
	speakerIndex := -1
	for i, part := range state.MatchingParts {
		if part.Kind == "speaker" {
			speakerIndex = i
		}
	}
	if speakerIndex < 0 {
		t.Fatal("loudspeaker missing from Passives")
	}
	// The category replacement removes the formerly focused MOSFET row.
	// Independently focus the updated list before testing new user input.
	s.requestKeyboardFocus("part-list")
	m.frame()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeHome})
	for range speakerIndex {
		m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeDownArrow})
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	if state = s.service.Snapshot(); state.Drawing.Tool != "place" || state.Drawing.Kind != "speaker" {
		t.Fatal("mounted Passives list did not choose the loudspeaker", state.Drawing)
	}
}
