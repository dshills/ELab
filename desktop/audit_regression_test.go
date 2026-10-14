package main

import (
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeSearchModelAssumptionsAndUnavailableProbeRecovery(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "part-search", &ui.ChangeEvent{Text: "BATTERY"})
	if _, ok := findNode(s.view(), "part-dc-source"); !ok {
		t.Fatal("native hobbyist alias search failed")
	}
	uiAction(t, s, "part-search", &ui.ChangeEvent{Text: ""})
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	for _, id := range []string{"R1", "C1", "V1", "GND"} {
		uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: id})
		model, ok := findNode(s.view(), "part-model-id")
		if !ok || !strings.Contains(model.Text, "Model:") {
			t.Fatal("native model identity absent")
		}
		assumptions, ok := findNode(s.view(), "part-assumptions")
		if !ok || assumptions.Text == "" {
			t.Fatal("native model assumptions absent")
		}
	}
	agentAction(t, s.service, "scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1", "i:C1"}})
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "C1"})
	uiAction(t, s, "delete-object", nil)
	if _, ok := findNode(s.view(), "probe-unavailable-v:C1"); !ok {
		t.Fatal("native missing-source explanation absent")
	}
	uiAction(t, s, "remove-probe-v:C1", nil)
	if len(s.service.Snapshot().Scope.Unavailable) != 1 || s.service.Snapshot().Scope.Unavailable[0].ID != "i:C1" {
		t.Fatal("native removal discarded other missing trace")
	}
	if err := ui.Validate(s.view()); err != nil {
		t.Fatal(err)
	}
}
