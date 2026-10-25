package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestOptocouplerNativeCTRAndSwitchMatchAgent(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example.opto-switch"})
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "opto-switch"})
	uiAction(t, u, "simulation.pause", nil)
	agentAction(t, a, "simulation.pause", struct{}{})
	awaitInductorCircuit(t, u.service)
	awaitInductorCircuit(t, a)
	uiAction(t, u, "opto-ctr", &ui.ChangeEvent{Number: 50})
	agentAction(t, a, "optocoupler.configure", workbench.OptocouplerParams{ID: "OC1", CTR: .5})
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "switch.set", workbench.SwitchParams{ID: "S1", Closed: true})
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) {
		t.Fatal("native CTR differs from agent")
	}
	for _, id := range []string{"opto-input", "opto-output", "opto-effective", "opto-state", "opto-ctr-slider", "plot-v:OC1.output", "plot-i:OC1.collector"} {
		if _, ok := findNode(u.view(), id); !ok {
			t.Fatal("missing optical control/readout", id)
		}
	}
	uiAction(t, u, "opto-ctr-slider", &ui.ChangeEvent{Number: 1})
	agentAction(t, a, "optocoupler.configure", workbench.OptocouplerParams{ID: "OC1", CTR: .01})
	if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("native minimum CTR loses exact bound")
	}
}
