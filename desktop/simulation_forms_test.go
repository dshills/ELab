package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
	"reflect"
	"testing"
)

func TestFocusedSimulationFormsRetainExistingDescriptors(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	state := s.service.Snapshot()
	for _, tool := range []string{"live", "recordings", "spectrum", "sweeps", "tolerances"} {
		t.Run(tool, func(t *testing.T) {
			original := []ui.Node{{ID: "advanced-field", Kind: ui.TextField, Text: "unchanged"}, {ID: "advanced-action", Kind: ui.Button, Text: "Do work"}}
			nodes := s.focusSimulationForm(state, tool, original)
			wrapper := ui.Node{Children: nodes}
			for _, want := range original {
				got, ok := findNode(wrapper, want.ID)
				if !ok || !reflect.DeepEqual(got, want) {
					t.Fatal("lost existing descriptor", want.ID)
				}
			}
			d, ok := findNode(wrapper, "workspace-details-"+tool)
			if !ok || d.Checked {
				t.Fatal("Advanced default", tool)
			}
			d.Change(ui.ChangeEvent{Checked: true})
			if !s.service.Snapshot().WorkspaceDetails.Expanded(tool) {
				t.Fatal("disclosure does not use shared state")
			}
			state = s.service.Snapshot()
		})
	}
}
func TestRecordedSpectrumExplainsMissingRecording(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "simulation"})
	agentAction(t, s.service, "workspace.simulation.set", workbench.SimulationToolParams{Tool: "spectrum"})
	root := s.view()
	for _, id := range []string{"spectrum-prerequisite", "spectrum-record-first", "spectrum-import-recording"} {
		if _, ok := findNode(root, id); !ok {
			t.Fatal("missing next step", id)
		}
	}
	if _, ok := findNode(root, "simulation-primary-actions"); ok {
		t.Fatal("empty spectrum has unusable primary action")
	}
	uiAction(t, s, "spectrum-record-first", nil)
	if s.service.Snapshot().SimulationTool != "recordings" {
		t.Fatal("record prerequisite does not navigate")
	}
}
