package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
	"reflect"
	"strings"
	"testing"
)

func TestMeasurementChooserCoversTargetsAndMatchesAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	for _, target := range []workbench.MeasurementSelectParams{{Kind: "object", ID: "R1"}, {Kind: "terminal", ID: "C1.1"}, {Kind: "net", ID: "net:C1.1"}, {Kind: "probe", ID: "i:R1"}, {}} {
		before := s.service.Snapshot()
		selected := "__none"
		if target.Kind != "" {
			selected = target.Kind + "|" + target.ID
		}
		uiAction(t, s, "measurement-source", &ui.ChangeEvent{SelectedID: selected})
		fromUI := s.service.Snapshot()
		agentAction(t, s.service, "measurement.select", target)
		if !reflect.DeepEqual(fromUI.MeasurementInspection, s.service.Snapshot().MeasurementInspection) {
			t.Fatal("chooser differs from command", target)
		}
		assertCursorMeasurementPreservesPhysics(t, before, fromUI)
	}
}
func TestContextReadingsFollowTargetWithoutInventedLEDValues(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{})
	root := s.view()
	if _, ok := findNode(root, "current-title"); ok {
		t.Fatal("blank circuit shows LED metrics")
	}
	if _, ok := findNode(root, "measurement-context-empty"); !ok {
		t.Fatal("blank lacks useful prompt")
	}
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "measurement.select", workbench.MeasurementSelectParams{Kind: "terminal", ID: "C1.1"})
	before := s.service.Snapshot()
	root = s.view()
	n, ok := findNode(root, "context-reading-v:C1.1-value")
	if !ok || !strings.Contains(n.Text, "V") {
		t.Fatal("terminal has no contextual voltage", n.Text)
	}
	footer, _ := findNode(root, "readings")
	if _, ok := findNode(footer, "current-title"); ok {
		t.Fatal("unrelated summary shown beside target")
	}
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("presentation changed experiment")
	}
}
func TestSimulationStatusDoesNotHideInvalidityWhilePaused(t *testing.T) {
	for _, status := range []string{"error", "unsupported", "incomplete"} {
		got := simulationStatus(workbench.Snapshot{Status: status, Paused: true})
		if strings.Contains(got, "Live") || got == "Paused" {
			t.Fatal("misleading paused status", status, got)
		}
	}
}
