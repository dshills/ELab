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

func TestNativeValueRefusalSharesDomainHelpAndFirstApplyRecovery(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "0"})
	before := s.service.Snapshot()
	s.app.Do(func() {
		apply, ok := findAction(s.view(), "apply-value")
		if !ok {
			t.Fatal("missing value action")
		}
		apply.Activate() // This deliberately rejected input must keep its error.
	})
	if s.requestHelp == nil || !strings.Contains(s.requestHelp.Summary, "R1 needs a valid resistance") || !strings.Contains(s.requestHelp.Action, "10–100000 Ω") || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("native refusal lost domain guidance or changed accepted state", s.requestHelp)
	}
	agent := workbench.NewExample()
	defer agent.Close()
	agentAction(t, agent, "simulation.pause", struct{}{})
	raw, _ := json.Marshal(workbench.ValueParams{ID: "R1", Value: "0"})
	rev := agent.Snapshot().CommandRevision
	response := agent.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "agent-bad", Key: "agent-bad", Command: "component.set", Params: raw, ExpectedCommandRevision: &rev}, "agent.test")
	if !reflect.DeepEqual(s.requestHelp, response.Help) {
		t.Fatal("UI and agent do not receive identical value guidance", s.requestHelp, response.Help)
	}
	s.app.Do(func() {
		root := s.view()
		message, _ := findNode(root, "messages")
		action, _ := findNode(root, "diagnostic-0-action")
		if message.Text != response.Help.Summary || action.Text != response.Help.Action {
			t.Fatal("visible help does not use shared response", message.Text, action.Text)
		}
	})
	s.app.Do(func() {
		field, ok := findNode(s.view(), "value-entry")
		if !ok {
			t.Fatal("missing value field")
		}
		field.Change(ui.ChangeEvent{Text: "2k"})
	})
	uiAction(t, s, "apply-value", nil)
	after := s.service.Snapshot()
	part, _ := (workbench.Graph{Components: after.Components}).Component("R1")
	if part.Value != 2000 || s.requestHelp != nil || s.errorMessage != "" {
		t.Fatal("corrected value did not apply on first action", part, s.errorMessage)
	}
	uiAction(t, s, "edit.undo", nil)
	part, _ = (workbench.Graph{Components: s.service.Snapshot().Components}).Component("R1")
	if part.Value != 1000 {
		t.Fatal("correction is not one undoable edit", part)
	}
}
