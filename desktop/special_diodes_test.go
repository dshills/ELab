package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestSpecialDiodeBlankConstructionAndValueUIAgentParity(t *testing.T) {
	for _, kind := range []string{"zener", "schottky"} {
		t.Run(kind, func(t *testing.T) {
			fixture := workbench.ZenerExampleGraph()
			fixture.Components[0], _ = workbench.MakeComponent("V1", "dc-source", fixture.Components[0].Position, "10 V")
			if kind == "schottky" {
				fixture.Components[2], _ = workbench.MakeComponent("D1", kind, fixture.Components[2].Position, "")
				fixture.Components[2].Rotation = 0
			}
			user, agent, ids := buildNetworkFixtureUIAndAgent(t, fixture)
			left, right := user.service.Snapshot(), agent.Snapshot()
			if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.ZenerModel, right.ZenerModel) || !reflect.DeepEqual(left.SchottkyModel, right.SchottkyModel) {
				t.Fatal("drawn diode results or models diverge")
			}
			id := ids["D1"]
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: id})
			if _, ok := findNode(user.view(), "mixed-selected-reading"); !ok {
				t.Fatal("signed diode readings absent")
			}
			for _, probe := range []string{"v:" + id, "i:" + id} {
				if _, ok := findNode(user.view(), "plot-"+probe); !ok {
					t.Fatal("diode probe action absent", probe)
				}
			}
			if kind == "zener" {
				uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "3.3 V"})
				uiAction(t, user, "apply-value", nil)
				agentAction(t, agent, "component.set", workbench.ValueParams{ID: id, Value: "3.3 V"})
				uiAction(t, user, "simulation.resume", nil)
				agentAction(t, agent, "simulation.resume", struct{}{})
				left, right = awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
				if !reflect.DeepEqual(left.Mixed, right.Mixed) || left.Mixed.Diodes[0].Voltage > -3.3 {
					t.Fatal("Zener UI/agent value edit differs")
				}
			}
			if err := ui.Validate(user.view()); err != nil {
				t.Fatal(err)
			}
			if mesh, err := user.mesh(*user.drawing.View); err != nil || mesh == nil {
				t.Fatal("diode symbol absent", err)
			}
		})
	}
}

func TestSpecialDiodeNativeExamplesUseSharedLoadStepAndProtection(t *testing.T) {
	for _, id := range []string{"zener", "schottky"} {
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
			left, right := awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
			if left.Example.Info.ID != id || left.Document.Dirty || !reflect.DeepEqual(left.Scope.Selected, right.Scope.Selected) || !reflect.DeepEqual(left.Mixed, right.Mixed) {
				t.Fatal("example defaults differ")
			}
			for range 30 {
				uiAction(t, user, "transient-step", nil)
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
			}
			left, right = user.service.Snapshot(), agent.Snapshot()
			if math.Abs(left.Transient.Sample.Time-.03) > 1e-12 || !reflect.DeepEqual(left.Mixed, right.Mixed) {
				t.Fatal("live diode stepping differs")
			}
			target := "D1"
			value := "3.3 V"
			if id == "schottky" {
				target = "R1"
				value = "220"
			}
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: target})
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: value})
			uiAction(t, user, "apply-value", nil)
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + id})
			if user.pendingCommand != "example.load" {
				t.Fatal("diode example bypassed unsaved protection")
			}
			uiAction(t, user, "discard-continue", nil)
			if user.service.Snapshot().Example.Modified {
				t.Fatal("example did not reset")
			}
		})
	}
}
