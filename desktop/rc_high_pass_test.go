package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRCHighPassBlankConstructionMatchesAgent(t *testing.T) {
	user, agent, _ := buildNetworkFixtureUIAndAgent(t, workbench.RCHighPassExampleGraph())
	for i := 0; i < 60; i++ {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	left, right := user.service.Snapshot(), agent.Snapshot()
	want := (5*math.Exp(-5) - 5) * math.Exp(-1)
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.Transient.Samples, right.Transient.Samples) || math.Abs(left.Mixed.Resistors[0].Voltage-want) > .003 {
		t.Fatal("blank high-pass construction/negative pulse differs", left.Mixed)
	}
}

func TestRCHighPassNativeMenuProbeDefaultsAndProtectedReplacement(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.rc-high-pass"})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "rc-high-pass"})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, s, command, nil)
		agentAction(t, agent, command, struct{}{})
	}
	for i := 0; i < 60; i++ {
		uiAction(t, s, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	left, right := s.service.Snapshot(), agent.Snapshot()
	if left.Example.Info.ID != "rc-high-pass" || left.Selected != "R1" || !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.Scope.Selected, []string{"v:V1.positive", "v:R1", "v:C1"}) || !reflect.DeepEqual(left.Scope.Traces, right.Scope.Traces) || left.Mixed.Resistors[0].Voltage > -1.8 {
		t.Fatal("native high-pass template/default traces differ from generic agent load")
	}
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "20k"})
	uiAction(t, s, "apply-value", nil)
	if !s.service.Snapshot().Mixed.Stale {
		t.Fatal("paused resistance edit changed accepted physics")
	}
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.rc-high-pass"})
	if s.pendingCommand != "example.load" || !s.service.Snapshot().Example.Modified {
		t.Fatal("high-pass menu bypasses unsaved protection")
	}
	uiAction(t, s, "discard-continue", nil)
	if s.service.Snapshot().Example.Modified || s.service.Snapshot().Selected != "R1" {
		t.Fatal("protected generic load lost high-pass template")
	}
}
