package main

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
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

func TestNativeRecordedTimeRefusalUsesSharedActionableHelpAndCorrection(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	defer cleanup()
	index := 4
	agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
	before := s.service.Snapshot()
	raw, err := json.Marshal(map[string]any{"time_s": "oops"})
	if err != nil {
		t.Fatal(err)
	}
	revision := before.CommandRevision
	response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "bad-position", Key: "bad-position", Command: "recording.seek", Params: raw, ExpectedCommandRevision: &revision}, "agent.test")
	if response.Error == nil || response.Help == nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("agent refusal changed the recording")
	}
	field := recordedPlayControl(t, s, "recorded-time")
	s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "oops"}) })
	if !reflect.DeepEqual(s.requestHelp, response.Help) || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("native invalid time differs from shared agent guidance")
	}
	s.app.Do(func() {
		root := s.view()
		summary, ok := findNode(root, "diagnostic-0-summary")
		if !ok || summary.Text != response.Help.Summary || !strings.Contains(summary.Text, "recording") {
			t.Fatal("visible summary hides the invalid recording position")
		}
		action, ok := findNode(root, "diagnostic-0-action")
		if !ok || action.Text != response.Help.Action || !strings.Contains(action.Text, "0.02 s") || !strings.Contains(action.Text, "ms") {
			t.Fatal("visible action hides the valid range or time units")
		}
	})
	uiAction(t, s, "recorded-time", &ui.ChangeEvent{Text: "3 ms"})
	uiAction(t, s, "recorded-previous", nil)
	after := s.service.Snapshot()
	if after.Recording.Index != 2 || s.requestHelp != nil || s.errorMessage != "" {
		t.Fatal("correction did not recover and clear shared guidance", after.Recording, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
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
