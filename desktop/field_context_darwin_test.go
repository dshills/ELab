//go:build darwin && cgo

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeChoiceRetainsFieldDocumentContext(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	old := s.service.Snapshot()
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
	before := s.service.Snapshot()
	s.app.Do(func() {
		node := ui.Node{ID: "old-choice", Kind: ui.Selector, Options: []ui.Option{{ID: "R1", Label: "R1"}}, Change: func(e ui.ChangeEvent) { s.invoke("component.select", workbench.IDParams{ID: e.SelectedID}) }}
		// Isolate document identity even with a current command revision.
		s.bindFieldContext(&node, actionContext{before.CommandRevision, old.Document.Generation})
		native := s.nativeChoiceButton(node, true, before.CommandRevision)
		choices := native.Custom.(menuChoices)
		s.acceptMenuChoice(native, choices, 0, s.menuSerial, before.CommandRevision)
	})
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.fieldContext != nil || s.menuCommandRevision != nil {
		t.Fatal("native adaptation discarded document context or leaked owner guards")
	}
}
