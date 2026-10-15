package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeLEDFadeTemplateUsesGenericSharedLoadAndProtectedReplacement(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.led-fade"})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "led-fade"})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, s, command, nil)
		agentAction(t, agent, command, struct{}{})
	}
	for i := 0; i < 205; i++ {
		uiAction(t, s, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	left, right := s.service.Snapshot(), agent.Snapshot()
	if left.Example == nil || left.Example.Info.ID != "led-fade" || left.Selected != "C1" || !reflect.DeepEqual(left.Components, right.Components) || !reflect.DeepEqual(left.Wires, right.Wires) || left.Mixed.Capacitors[0].Voltage != right.Mixed.Capacitors[0].Voltage || left.Mixed.LEDs[0].Current != right.Mixed.LEDs[0].Current || left.Transient.Sample.Input != 0 {
		t.Fatal("native template and generic agent load diverge")
	}
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "220uF"})
	uiAction(t, s, "apply-value", nil)
	if !s.service.Snapshot().Mixed.Stale {
		t.Fatal("paused capacitor edit silently advanced simulation")
	}
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.led-fade"})
	if s.pendingCommand != "example.load" || math.Abs(s.service.Snapshot().Components[len(left.Components)-1].Value-220e-6) > 1e-18 {
		t.Fatal("generic template load bypassed unsaved protection")
	}
	uiAction(t, s, "discard-continue", nil)
	if s.service.Snapshot().Example.Modified || s.pendingCommand != "" {
		t.Fatal("protected generic load lost selected template params")
	}
}
