package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRegulatorNativeExamplesAndSharedValueEdit(t *testing.T) {
	for _, id := range []string{"regulator-supply", "regulator-dropout"} {
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
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "3.3 V"})
			uiAction(t, user, "apply-value", nil)
			agentAction(t, agent, "component.set", workbench.ValueParams{ID: "U1", Value: "3.3 V"})
			for _, service := range []*workbench.Service{user.service, agent} {
				agentAction(t, service, "simulation.restart", struct{}{})
				awaitInductorCircuit(t, service)
			}
			a, b := user.service.Snapshot(), agent.Snapshot()
			if !reflect.DeepEqual(a.Mixed, b.Mixed) {
				t.Fatal("regulator value parity")
			}
			if _, ok := findNode(user.view(), "regulator-selected-reading"); !ok {
				t.Fatal("missing regulator inspector")
			}
			if node, ok := findNode(user.view(), "plot-v:U1.in"); !ok || node.Text != "Plot in voltage" {
				t.Fatal("ambiguous terminal plot control")
			}
			for _, probe := range []string{"plot-v:U1", "plot-i:U1", "plot-v:U1.in"} {
				if _, ok := findNode(user.view(), probe); !ok {
					t.Fatal("missing regulator plot", probe)
				}
			}
		})
	}
}
