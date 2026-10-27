package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
	"reflect"
	"testing"
)

func TestTransistorProfileNativeControlsAndAgentParity(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", nil)
	uiAction(t, u, "example-search", &ui.ChangeEvent{Text: "Transistor gain"})
	grid := browserGridNode(t, u)
	u.app.Do(func() {
		grid.SelectionChange(ui.ListSelectionEvent{TargetID: "transistor-profile", IDs: []string{"transistor-profile"}})
	})
	uiAction(t, u, ui.SheetConfirmID(exampleBrowserID), nil)
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "transistor-profile"})
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.pause", struct{}{})
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	uiAction(t, u, "inspector-section-component", nil)
	node, ok := findNode(u.view(), "transistor-beta")
	if !ok {
		t.Fatal("missing profile controls")
	}
	uiAction(t, u, "transistor-beta", &ui.ChangeEvent{Number: 300})
	v := workbench.DefaultTransistorProfile()
	v.ForwardBeta = 300
	agentAction(t, a, "transistor.profile.set", workbench.TransistorProfileParams{ID: "Q1", Enabled: true, Parameters: v})
	if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("shared profile command differs")
	}
	agentAction(t, u.service, "example.load", workbench.ExampleLoadParams{ID: "transistor-profile", DiscardChanges: true})
	before := u.service.Snapshot()
	node.Change(ui.ChangeEvent{Number: 500})
	if !reflect.DeepEqual(before.Components, u.service.Snapshot().Components) {
		t.Fatal("stale profile callback changed replacement")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(theme, err)
		}
	}
}
