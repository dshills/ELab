package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestButtonNativeControlsAndAgentParity(t *testing.T) {
	for _, id := range []string{"button-counter", "button-debounce"} {
		t.Run(id, func(t *testing.T) {
			user, cleanup := testScreen(t)
			defer cleanup()
			agent := workbench.New()
			defer agent.Close()
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + id})
			agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: id})
			for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
				uiAction(t, user, cmd, nil)
				agentAction(t, agent, cmd, struct{}{})
			}
			a, b := awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
			if !reflect.DeepEqual(a.Mixed, b.Mixed) {
				t.Fatal("button example defaults")
			}
			uiAction(t, user, "button-click", nil)
			agentAction(t, agent, "button.click", workbench.IDParams{ID: "SCOUNT"})
			awaitInductorCircuit(t, agent)
			if _, err := agent.Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(user.service.Snapshot().Mixed, agent.Snapshot().Mixed) {
				t.Fatal("button click parity")
			}
			for range 12 {
				uiAction(t, user, "transient-step", nil)
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: a.TransientSettings.StepSeconds})
			}
			if !reflect.DeepEqual(user.service.Snapshot().Mixed, agent.Snapshot().Mixed) {
				t.Fatal("button bounce parity")
			}
			uiAction(t, user, "button-hold", nil)
			agentAction(t, agent, "button.set", workbench.ButtonParams{ID: "SCOUNT", Pressed: false})
			if _, err := agent.Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(user.service.Snapshot().Mixed, agent.Snapshot().Mixed) || user.service.Snapshot().Document.Dirty {
				t.Fatal("button release dirtied project or diverged")
			}
		})
	}
}
