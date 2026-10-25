package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestTransformerNativeTurnsDotsAndCouplingMatchAgent(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example.coupled-windings"})
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "coupled-windings"})
	uiAction(t, u, "simulation.pause", nil)
	agentAction(t, a, "simulation.pause", struct{}{})
	awaitInductorCircuit(t, u.service)
	awaitInductorCircuit(t, a)
	var cfg workbench.TransformerConfig
	for _, p := range a.Snapshot().Components {
		if p.ID == "T1" {
			cfg = *p.Power.Transformer
		}
	}
	uiAction(t, u, "transformer-ratio", &ui.ChangeEvent{Number: .75})
	cfg.Ratio = .75
	agentAction(t, a, "transformer.configure", workbench.TransformerParams{ID: "T1", Parameters: cfg})
	uiAction(t, u, "transformer-reverse", &ui.ChangeEvent{Checked: true})
	cfg.Reversed = true
	agentAction(t, a, "transformer.configure", workbench.TransformerParams{ID: "T1", Parameters: cfg})
	uiAction(t, u, "transformer-coupling", &ui.ChangeEvent{Number: .9999})
	cfg.Coupling = .9999
	agentAction(t, a, "transformer.configure", workbench.TransformerParams{ID: "T1", Parameters: cfg})
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
		agentAction(t, s, "simulation.step", workbench.StepParams{Duration: .002})
		s.Wait(t.Context())
	}
	if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) || !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("native transformer parameters differ from shared commands")
	}
	for _, id := range []string{"transformer-derived", "transformer-voltages", "transformer-currents", "transformer-flux", "transformer-energy", "transformer-power", "plot-v:T1.secondary", "plot-i:T1.primary1", "plot-i:T1.secondary1"} {
		if _, ok := findNode(u.view(), id); !ok {
			t.Fatal("missing winding control/readout", id)
		}
	}
}
