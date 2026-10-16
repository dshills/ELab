package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestCMOSBlankDrawingMatchesAgentAtBothInputLevels(t *testing.T) {
	user, agent, _ := buildNetworkFixtureUIAndAgent(t, workbench.CMOSExampleGraph())
	for range 60 {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	left, right := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.Transient.Samples, right.Transient.Samples) || left.Mixed.MOSFETs[1].VDS < 4.99 {
		t.Fatal("blank drawn inverter differs from agent or fails to invert low input")
	}
}

func TestCMOSNativeMenuUsesProtectedGenericLoad(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.cmos"})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "cmos"})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, s, command, nil)
		agentAction(t, agent, command, struct{}{})
	}
	left, right := s.service.Snapshot(), agent.Snapshot()
	if left.Example.Info.ID != "cmos" || left.Selected != "MN" || len(left.Scope.Selected) != 4 || !reflect.DeepEqual(left.Scope.Selected, right.Scope.Selected) || !reflect.DeepEqual(left.Mixed, right.Mixed) {
		t.Fatal("native CMOS defaults differ from shared load")
	}
	root := s.view()
	region, _ := findNode(root, "power-title")
	caption, _ := findNode(root, "scope-caption")
	if strings.ToLower(region.Text) != "mp region" || !strings.Contains(caption.Text, "MP cutoff") {
		t.Fatal("region summary must identify MP while MN is selected", region.Text, caption.Text)
	}
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "VIN"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "2.5 V"})
	uiAction(t, s, "apply-value", nil)
	if !s.service.Snapshot().Mixed.Stale {
		t.Fatal("paused gate edit replaced accepted physics")
	}
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.cmos"})
	if s.pendingCommand != "example.load" || !s.service.Snapshot().Example.Modified {
		t.Fatal("CMOS menu bypasses protected replacement")
	}
	uiAction(t, s, "discard-continue", nil)
	if s.service.Snapshot().Example.Modified || s.service.Snapshot().Selected != "MN" {
		t.Fatal("protected CMOS replacement did not restore defaults")
	}
}
