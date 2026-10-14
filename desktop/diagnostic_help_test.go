package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeDiagnosticDisclosureMatchesAgentAndKeepsWireGesture(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	agentAction(t, s.service, "object.delete", workbench.IDParams{ID: "W1"})
	uiAction(t, s, "tool-wire", nil)
	s.app.Do(func() { s.canvasClick(workbench.Point{X: -200, Y: -34}) })
	before := s.service.Snapshot()
	uiAction(t, s, "diagnostic-details", &ui.ChangeEvent{Checked: true})
	after := s.service.Snapshot()
	if !after.DiagnosticPresentation.DetailsVisible || after.Drawing != before.Drawing || after.RunID != before.RunID || after.Document != before.Document || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) {
		t.Fatal("technical disclosure interrupts a wire gesture or changes physical state")
	}
	agentAction(t, s.service, "diagnostics.details.set", workbench.DiagnosticDetailsParams{Visible: false})
	if s.service.Snapshot().DiagnosticPresentation.DetailsVisible {
		t.Fatal("agent disclosure disagrees with UI")
	}
}

func TestNativeFailedValueKeepsSharedRequestHelpThroughDisclosure(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	before := s.service.Snapshot()
	s.app.Do(func() {
		r := s.invoke("component.set", workbench.ValueParams{ID: "R1", Value: "nonsense"})
		if r.Error == nil || r.Help == nil {
			t.Fatal("missing shared failed-value help")
		}
	})
	if s.requestHelp == nil || s.requestHelp.Objects[0] != "R1" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("failed value help mutates service state")
	}
	s.app.Do(func() {
		root := s.view()
		node, ok := findNode(root, "diagnostic-details")
		if !ok {
			t.Fatal("missing request disclosure")
		}
		node.Change(ui.ChangeEvent{Checked: true})
	})
	if s.requestHelp == nil || s.errorMessage == "" || !s.service.Snapshot().DiagnosticPresentation.DetailsVisible {
		t.Fatal("opening details erases the failed request explanation")
	}
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	s.app.Do(func() { s.view() })
	if s.requestHelp != nil || s.errorMessage != "" {
		t.Fatal("successful correction retains a ghost request marker")
	}
}
