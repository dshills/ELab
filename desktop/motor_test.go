package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMotorNativeLoadAndLockedStartupMatchAgent(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example.motor-dc"})
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "motor-dc"})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, u, command, nil)
		agentAction(t, a, command, struct{}{})
	}
	awaitInductorCircuit(t, u.service)
	awaitInductorCircuit(t, a)
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.step", workbench.StepParams{Duration: .05})
		s.Wait(t.Context())
	}
	before := u.service.Snapshot().Mixed.Motors[0]
	uiAction(t, u, "motor-load-exact", &ui.ChangeEvent{Number: 100})
	agentAction(t, a, "motor.load.set", workbench.MotorLoadParams{ID: "MTR1", Load: .0001})
	for _, s := range []*workbench.Service{u.service, a} {
		s.Wait(t.Context())
	}
	if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
		t.Fatal("live motor load differs from agent")
	}
	after := u.service.Snapshot().Mixed.Motors[0]
	if after.Omega != before.Omega || after.Current != before.Current {
		t.Fatal("load jump changed stored speed/current")
	}
	var config workbench.MotorConfig
	for _, p := range a.Snapshot().Components {
		if p.Kind == "dc-motor" {
			config = *p.Power.Motor
		}
	}
	config.Locked = true
	uiAction(t, u, "motor-locked", &ui.ChangeEvent{Checked: true})
	agentAction(t, a, "motor.configure", workbench.MotorParams{ID: "MTR1", Parameters: config})
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
		t.Fatal("locked motor configuration differs")
	}
	for _, id := range []string{"motor-rpm", "motor-current", "motor-emf", "motor-energy", "motor-loss"} {
		if _, ok := findNode(u.view(), id); !ok {
			t.Fatal("missing motor reading", id)
		}
	}
}
