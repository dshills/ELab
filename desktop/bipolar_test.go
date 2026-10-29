package main

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestBipolarBlankDrawingUsesSameCommandsAndElectricalResults(t *testing.T) {
	for _, example := range []string{"pnp", "emitter-follower", "bipolar-push-pull"} {
		t.Run(example, func(t *testing.T) {
			fixture := workbench.PNPExampleGraph()
			if example == "emitter-follower" {
				fixture = workbench.EmitterFollowerExampleGraph()
			}
			if example == "bipolar-push-pull" {
				fixture = workbench.ComplementaryBipolarExampleGraph()
			}
			for i, p := range fixture.Components {
				if p.ID == "VIN" {
					fixture.Components[i], _ = workbench.MakeComponent("VIN", "dc-source", p.Position, "2.5 V")
				}
			}
			user, agent, ids := buildNetworkFixtureUIAndAgent(t, fixture)
			left, right := user.service.Snapshot(), agent.Snapshot()
			if !reflect.DeepEqual(left.Mixed, right.Mixed) {
				t.Fatal("blank-drawn bipolar circuit differs from external semantic commands")
			}
			id := ids["Q1"]
			if example == "bipolar-push-pull" {
				id = ids["Q2"]
			}
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: id})
			reading, ok := findNode(user.view(), "bipolar-selected-reading")
			if !ok || !strings.Contains(reading.Text, "VBE") || !strings.Contains(reading.Text, "IC") {
				t.Fatal("selected transistor lacks signed junction/current readings")
			}
			for probe, label := range map[string]string{"v:" + id: "Plot VCE", "v:" + id + ".base-emitter": "Plot VBE", "v:" + id + ".collector": "Plot collector voltage", "i:" + id + ".collector": "Plot collector current"} {
				if node, ok := findNode(user.view(), "plot-"+probe); !ok || node.Text != label {
					t.Fatal("transistor plot action missing or ambiguous", probe, node.Text)
				}
			}
			selectedState := user.service.Snapshot()
			selectedGraph := workbench.Graph{Components: selectedState.Components, Wires: selectedState.Wires}
			selectedPart, _ := selectedGraph.Component(id)
			panel := user.componentInspector(selectedState, selectedGraph, selectedPart)
			readingIndex, modelIndex := -1, -1
			details, ok := findNode(ui.Node{Children: panel}, "workspace-details-component")
			if !ok {
				t.Fatal("missing model Details")
			}
			for i, node := range details.Children {
				if node.ID == "bipolar-selected-reading" {
					readingIndex = i
				}
				if node.ID == "npn-model" {
					modelIndex = i
				}
			}
			if readingIndex < 0 || modelIndex < readingIndex {
				t.Fatal("transistor readings should precede model notes")
			}
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["RL"]})
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "2k"})
			uiAction(t, user, "apply-value", nil)
			agentAction(t, agent, "component.set", workbench.ValueParams{ID: ids["RL"], Value: "2k"})
			if !user.service.Snapshot().Mixed.Stale {
				t.Fatal("paused edit replaced accepted transistor result")
			}
			uiAction(t, user, "simulation.resume", nil)
			agentAction(t, agent, "simulation.resume", struct{}{})
			left, right = awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
			if !reflect.DeepEqual(left.Mixed, right.Mixed) {
				t.Fatal("edited/restarted bipolar results diverge")
			}
		})
	}
}

func TestBipolarNativeExamplesShareProtectedLoadAndLiveStep(t *testing.T) {
	for _, id := range []string{"pnp", "emitter-follower", "bipolar-push-pull"} {
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
			if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.Scope.Selected, right.Scope.Selected) || left.Example.Info.ID != id || left.Document.Dirty {
				t.Fatal("native and agent template defaults differ", left.Diagnostics, right.Diagnostics)
			}
			for range 60 {
				uiAction(t, user, "transient-step", nil)
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: left.TransientSettings.StepSeconds})
			}
			left, right = user.service.Snapshot(), agent.Snapshot()
			if !reflect.DeepEqual(left.Mixed, right.Mixed) || math.Abs(left.Transient.Sample.Time-.06) > 1e-12 {
				t.Fatal("bipolar live stepping diverged", left.Diagnostics, right.Diagnostics)
			}
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "RL"})
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "2k"})
			uiAction(t, user, "apply-value", nil)
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + id})
			if user.pendingCommand != "example.load" {
				t.Fatal("new example bypassed unsaved-work protection")
			}
			uiAction(t, user, "discard-continue", nil)
			if user.service.Snapshot().Example.Modified {
				t.Fatal("reset did not restore template")
			}
		})
	}
}
