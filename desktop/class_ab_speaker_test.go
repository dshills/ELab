package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestClassABSpeakerBrowserStepAndAgentParity(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "examples-menu", nil)
	uiAction(t, user, "example-search", &ui.ChangeEvent{Text: "discrete loudspeaker"})
	grid := browserGridNode(t, user)
	user.app.Do(func() {
		grid.SelectionChange(ui.ListSelectionEvent{TargetID: "class-ab-speaker", IDs: []string{"class-ab-speaker"}})
	})
	uiAction(t, user, ui.SheetConfirmID(exampleBrowserID), nil)
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "class-ab-speaker"})
	for _, s := range []*workbench.Service{user.service, agent} {
		agentAction(t, s, "simulation.pause", struct{}{})
		agentAction(t, s, "simulation.restart", struct{}{})
		awaitInductorCircuit(t, s)
	}
	uiAction(t, user, "transient-step", nil)
	agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .0001})
	a, b := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(a.Components, b.Components) || !reflect.DeepEqual(a.Mixed, b.Mixed) || a.Mixed.Time != .0001 || len(a.Mixed.Transistors) != 8 || len(a.Mixed.Speakers) != 1 {
		t.Fatal("user/agent amplifier-speaker step differs")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, user.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(user.view()); err != nil {
			t.Fatal(theme, err)
		}
	}
}
