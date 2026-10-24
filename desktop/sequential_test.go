package main

import (
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestSequentialBlankDrawingAndAgentParity(t *testing.T) {
	user, agent, ids := buildNetworkFixtureUIAndAgent(t, workbench.TFlipFlopExampleGraph())
	for range 30 {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: user.service.Snapshot().TransientSettings.StepSeconds})
	}
	left, right := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.SequentialModel, right.SequentialModel) {
		t.Fatal("sequential drawing/step differs", left.Diagnostics, right.Diagnostics)
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["U1"]})
	if reading, ok := findNode(user.view(), "sequential-selected-reading"); !ok || !strings.Contains(reading.Text, "Clock") {
		t.Fatal("sequential inspector absent")
	}
	for _, probe := range []string{"v:" + ids["U1"], "v:" + ids["U1"] + ".clk", "v:" + ids["U1"] + ".q", "v:" + ids["U1"] + ".nq"} {
		if _, ok := findNode(user.view(), "plot-"+probe); !ok {
			t.Fatal("missing sequential plot", probe)
		}
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["SRESET"]})
	uiAction(t, user, "switch-actuate", nil)
	agentAction(t, agent, "switch.set", workbench.SwitchParams{ID: ids["SRESET"], Closed: true})
	uiAction(t, user, "transient-step", nil)
	agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: left.TransientSettings.StepSeconds})
	if a, b := user.service.Snapshot(), agent.Snapshot(); !reflect.DeepEqual(a.Mixed, b.Mixed) || a.Mixed.Sequential[0].Memory.Value != 0 {
		t.Fatal("native/agent asynchronous reset")
	}
	if err := ui.Validate(user.view()); err != nil {
		t.Fatal(err)
	}
}
func TestSequentialNativeExamplesAndProtection(t *testing.T) {
	for _, id := range []string{"d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter"} {
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
			if !reflect.DeepEqual(a.Mixed, b.Mixed) || !reflect.DeepEqual(a.Scope.Selected, b.Scope.Selected) || a.Document.Dirty {
				t.Fatal("sequential example defaults")
			}
			if title, ok := findNode(user.view(), "scope-title"); !ok || title.Text != "Live sequential scope" {
				t.Fatal("wrong sequential scope title", title.Text)
			}
			for range 25 {
				uiAction(t, user, "transient-step", nil)
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: a.TransientSettings.StepSeconds})
			}
			if !reflect.DeepEqual(user.service.Snapshot().Mixed, agent.Snapshot().Mixed) {
				t.Fatal("sequential example step divergence")
			}
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "SRESET"})
			uiAction(t, user, "switch-actuate", nil)
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + id})
			if user.pendingCommand != "example.load" {
				t.Fatal("sequential example bypassed unsaved protection")
			}
			uiAction(t, user, "discard-continue", nil)
		})
	}
}
func TestSequentialSymbolsAndPinTextRotate(t *testing.T) {
	ink := color.NRGBA{A: 255}
	for _, kind := range []string{"d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter"} {
		p, _ := workbench.MakeComponent("U1", kind, workbench.Point{}, "")
		base := &meshBuilder{}
		drawComponentSymbol(base, workbench.Snapshot{}, p, ink, ink, ink, ink, true)
		for _, angle := range []int{90, 180, 270} {
			p.Rotation = angle
			rotated := &meshBuilder{}
			drawComponentSymbol(rotated, workbench.Snapshot{}, p, ink, ink, ink, ink, true)
			if len(base.vertices) != len(rotated.vertices) {
				t.Fatal("sequential rotation changed shape")
			}
			for i, v := range base.vertices {
				want := workbench.RotateLocal(workbench.Point{X: v.Position.X, Y: v.Position.Y}, angle)
				got := rotated.vertices[i].Position
				if math.Hypot(want.X-got.X, want.Y-got.Y) > 1e-9 {
					t.Fatal("sequential pin text rotation", kind, angle)
				}
			}
			edge := workbench.RotateLocal(workbench.Point{X: 60, Y: 80}, angle)
			if hitComponent(workbench.Snapshot{Components: []workbench.Component{p}}, edge) != "U1" {
				t.Fatal("sequential body hit")
			}
		}
	}
}
