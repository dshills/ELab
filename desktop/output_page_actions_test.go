package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func captureOutputAction(t *testing.T, s *screen, id string) ui.Node {
	t.Helper()
	var node ui.Node
	s.app.Do(func() {
		state := s.service.Snapshot()
		if id == "pdf-fit" {
			node = s.outputFitAction(state)
		} else {
			node = s.outputPreviewAction(state, id, "Preview")
		}
	})
	return node
}

func TestOutputPageFitContinuesAcceptedSettingsFromOriginalAction(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	action := captureOutputAction(t, s, "pdf-fit")
	for _, field := range []struct {
		id    string
		value ui.ChangeEvent
	}{
		{"pdf-paper", ui.ChangeEvent{SelectedID: "a4"}},
		{"pdf-orientation", ui.ChangeEvent{SelectedID: "portrait"}},
		{"pdf-margin", ui.ChangeEvent{Number: 17}},
		{"pdf-color", ui.ChangeEvent{Checked: true}},
		{"pdf-measurements", ui.ChangeEvent{Checked: true}},
		{"pdf-scale", ui.ChangeEvent{Number: 150}},
		{"pdf-tile", ui.ChangeEvent{Checked: true}},
		{"pdf-overlap", ui.ChangeEvent{Number: 7}},
	} {
		uiAction(t, s, field.id, &field.value)
	}
	before := s.service.Snapshot()
	expected := before.SchematicPage
	expected.Scale = nil
	expected.Tile = false
	expected.OverlapMM = nil
	s.app.Do(action.Activate)
	after := s.service.Snapshot()
	if s.errorMessage != "" || !reflect.DeepEqual(after.SchematicPage, expected) || after.CommandRevision != before.CommandRevision+1 {
		t.Fatal("Fit used stale rather than accepted page settings", after.SchematicPage, s.errorMessage)
	}
	assertOutputCircuitPreserved(t, before, after)
}

func TestOutputPageContinuationRefusesOtherActorsFieldsAndDocuments(t *testing.T) {
	for _, control := range []string{"print-preview", "preview-refresh", "pdf-fit"} {
		t.Run(control, func(t *testing.T) {
			for _, scenario := range []string{"agent-before-field", "agent-after-field", "other-user-field", "new-document", "failed-field"} {
				t.Run(scenario, func(t *testing.T) {
					s, cleanup := testScreen(t)
					t.Cleanup(cleanup)
					uiAction(t, s, "example.reset", nil)
					agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
					action := captureOutputAction(t, s, control)
					agentPage := s.service.Snapshot().SchematicPage
					margin := 7.
					agentPage.MarginMM = &margin
					if scenario == "agent-before-field" {
						agentAction(t, s.service, "export.page.set", agentPage)
					}
					uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 17})
					switch scenario {
					case "agent-after-field":
						agentAction(t, s.service, "export.page.set", agentPage)
					case "other-user-field":
						uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "2k"})
					case "new-document":
						agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
					case "failed-field":
						s.app.Do(func() { field, _ := findNode(s.view(), "pdf-margin"); field.Change(ui.ChangeEvent{Number: 99}) })
					}
					before := s.service.Snapshot()
					s.app.Do(action.Activate)
					if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
						t.Fatal("output action adopted unrelated context", control, scenario, s.errorMessage)
					}
				})
			}
		})
	}
}

func assertOutputCircuitPreserved(t *testing.T, before, after workbench.Snapshot) {
	t.Helper()
	if before.Document != after.Document || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.RedoEntry, after.RedoEntry) || !reflect.DeepEqual(before.Scope, after.Scope) {
		t.Fatal("output controls changed circuit, physics, document, scope or history")
	}
}
