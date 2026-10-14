package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeCurrentExampleResetIsProtectedAndShared(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	original := s.service.Snapshot()
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "20k"})
	uiAction(t, s, "apply-value", nil)
	uiAction(t, s, "example.current.reset", nil)
	if s.pendingCommand != "example.current.reset" || s.service.Snapshot().Example.Info.ID != "rc" || !s.service.Snapshot().Example.Modified {
		t.Fatal("native reset bypasses unsaved protection or selects LED")
	}
	uiAction(t, s, "cancel-file-action", nil)
	if !s.service.Snapshot().Example.Modified {
		t.Fatal("cancelled reset changes edited circuit")
	}
	uiAction(t, s, "example.current.reset", nil)
	uiAction(t, s, "discard-continue", nil)
	state := s.service.Snapshot()
	if state.Example.Info.ID != "rc" || state.Example.Modified || !state.Paused || !reflect.DeepEqual(state.Components, original.Components) || state.Transient.Sample.Time != 0 {
		t.Fatal("native reset differs from protected current-template command")
	}
	s.app.Do(func() {
		root := s.view()
		for _, id := range []string{"example-question", "example-expected", "example-try", "example-limitation"} {
			node, ok := findNode(root, id)
			if !ok || node.Text == "" {
				t.Fatal("missing guided example field", id)
			}
		}
	})
}

func TestNativeStaleExampleResetButtonRefusesReplacement(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	var captured ui.Node
	s.app.Do(func() {
		node, ok := findNode(s.view(), "example.current.reset")
		if !ok {
			t.Fatal("reset missing")
		}
		captured = node
	})
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "npn", DiscardChanges: true})
	before := s.service.Snapshot()
	s.app.Do(func() { captured.Activate() })
	if s.errorMessage == "" || s.pendingCommand != "" || s.service.Snapshot().Example.Info.ID != "npn" || !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) {
		t.Fatal("old reset button changes a replacement example")
	}
}

func TestManualTransistorExampleMenuAndSwitchMatchAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.switch-npn"})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "switch-npn"})
	if s.service.Snapshot().Selected != "S1" {
		t.Fatal("manual example does not expose its switch immediately")
	}
	for _, closed := range []bool{true, false} {
		uiAction(t, s, "switch-actuate", nil)
		agentAction(t, agent, "switch.set", workbench.SwitchParams{ID: "S1", Closed: closed})
		u, a := s.service.Snapshot(), agent.Snapshot()
		if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || u.Mixed.LEDs[0].Current != a.Mixed.LEDs[0].Current || u.SwitchStates[0].DesiredClosed != closed {
			t.Fatal("manual transistor user/agent behavior differs")
		}
	}
	uiAction(t, s, "switch-actuate", nil)
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.rc"})
	if s.pendingCommand != "example.rc" || s.service.Snapshot().Example.Info.ID != "switch-npn" {
		t.Fatal("replacement loses unsaved manual-switch experiment")
	}
	uiAction(t, s, "cancel-file-action", nil)
	if s.service.Snapshot().Mixed.LEDs[0].Current < .002 {
		t.Fatal("cancel loses lit circuit")
	}
}

func TestRectifierExampleMenuValueAndScopeMatchAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.rectifier"})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "rectifier"})
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, agent, "simulation.pause", struct{}{})
	uiAction(t, s, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	if state := s.service.Snapshot(); state.Selected != "C1" || state.Example.Info.ID != "rectifier" || len(state.Scope.Traces) != 3 {
		t.Fatal("rectifier does not expose capacitor/input/output")
	}
	for i := 0; i < 20; i++ {
		uiAction(t, s, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	u, a := s.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || !reflect.DeepEqual(u.Mixed, a.Mixed) || !reflect.DeepEqual(u.Scope.Traces, a.Scope.Traces) || u.Mixed.Capacitors[0].Voltage < 3 {
		t.Fatal("rectifier user/agent transient behavior differs")
	}
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "47u"})
	uiAction(t, s, "apply-value", nil)
	agentAction(t, agent, "component.set", workbench.ValueParams{ID: "C1", Value: "47u"})
	if !s.service.Snapshot().Transient.Stale {
		t.Fatal("paused capacitor edit mislabeled fresh")
	}
	uiAction(t, s, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	u, a = s.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Mixed, a.Mixed) || u.Transient.Sample.Time != 0 || !u.Example.Modified {
		t.Fatal("rectifier capacitance edit/restart differs")
	}
}
