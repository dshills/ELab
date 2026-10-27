package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestDiscreteClassABUserAndAgentLoadEditStep(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "examples-menu", nil)
	uiAction(t, user, "example-search", &ui.ChangeEvent{Text: "all-discrete"})
	grid := browserGridNode(t, user)
	user.app.Do(func() {
		grid.SelectionChange(ui.ListSelectionEvent{TargetID: "discrete-class-ab", IDs: []string{"discrete-class-ab"}})
	})
	uiAction(t, user, ui.SheetConfirmID(exampleBrowserID), nil)
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "discrete-class-ab"})
	for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, user, cmd, nil)
		agentAction(t, agent, cmd, struct{}{})
	}
	left, right := awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.Scope.Selected, right.Scope.Selected) || len(left.Mixed.Transistors) != 8 || len(left.Mixed.OpAmps) != 0 || left.Document.Dirty {
		t.Fatal("shared example differs", left.Diagnostics, right.Diagnostics)
	}
	for range 50 {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .0005})
	}
	left, right = user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || math.Abs(left.Mixed.Time-.025) > 1e-10 {
		t.Fatal("shared live step differs")
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "RF"})
	uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "2k"})
	uiAction(t, user, "apply-value", nil)
	agentAction(t, agent, "component.set", workbench.ValueParams{ID: "RF", Value: "2k"})
	for _, cmd := range []string{"edit.undo", "simulation.restart"} {
		uiAction(t, user, cmd, nil)
		agentAction(t, agent, cmd, struct{}{})
	}
	left, right = awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
	if !reflect.DeepEqual(left.Components, right.Components) || !reflect.DeepEqual(left.Mixed, right.Mixed) {
		t.Fatal("shared edit/undo/restart differs")
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "Q6"})
	for _, probe := range []string{"v:Q6", "v:Q6.base-emitter", "i:Q6.collector"} {
		if _, ok := findNode(user.view(), "plot-"+probe); !ok {
			t.Fatal("missing transistor plot", probe)
		}
	}
	if err := ui.Validate(user.view()); err != nil {
		t.Fatal(err)
	}
	if mesh, err := user.mesh(*user.drawing.View); err != nil || mesh == nil {
		t.Fatal("amplifier mesh missing", err)
	}
}
