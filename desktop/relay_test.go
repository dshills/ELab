package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRelayNativeExamplesAndSharedValueEdit(t *testing.T) {
	for _, id := range []string{"relay-switch", "relay-flasher"} {
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
			awaitInductorCircuit(t, user.service)
			awaitInductorCircuit(t, agent)
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "150 mH"})
			uiAction(t, user, "apply-value", nil)
			agentAction(t, agent, "component.set", workbench.ValueParams{ID: "K1", Value: "150 mH"})
			for _, service := range []*workbench.Service{user.service, agent} {
				agentAction(t, service, "simulation.restart", struct{}{})
				awaitInductorCircuit(t, service)
			}
			a, b := user.service.Snapshot(), agent.Snapshot()
			if !reflect.DeepEqual(a.Mixed, b.Mixed) {
				t.Fatal("relay value parity")
			}
			if _, ok := findNode(user.view(), "relay-selected-reading"); !ok {
				t.Fatal("missing relay inspector")
			}
			for _, probe := range []string{"plot-v:K1", "plot-i:K1", "plot-v:K1.coil1"} {
				if _, ok := findNode(user.view(), probe); !ok {
					t.Fatal("missing relay plot", probe)
				}
			}
		})
	}
}
