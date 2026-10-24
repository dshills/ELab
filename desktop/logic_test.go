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

func TestLogicBlankDrawingAndAgentParity(t *testing.T) {
	// DC input preserves a deterministic from-blank fixture (waveform editing is
	// independently covered by the example menus and existing source controls).
	fixture := workbench.LogicSwitchesExampleGraph()
	user, agent, ids := buildNetworkFixtureUIAndAgent(t, fixture)
	for _, name := range []string{"SA", "SB"} {
		uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids[name]})
		uiAction(t, user, "switch-actuate", nil)
		agentAction(t, agent, "switch.set", workbench.SwitchParams{ID: ids[name], Closed: true})
	}
	uiAction(t, user, "simulation.resume", nil)
	agentAction(t, agent, "simulation.resume", struct{}{})
	awaitInductorCircuit(t, user.service)
	awaitInductorCircuit(t, agent)
	left, right := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.LogicModel, right.LogicModel) || left.Mixed.LogicGates[0].State != "high" {
		t.Fatal("logic drawing/actuation parity", left.Diagnostics, right.Diagnostics)
	}
	id := ids["U1"]
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: id})
	if reading, ok := findNode(user.view(), "logic-selected-reading"); !ok || !strings.Contains(reading.Text, "output high") {
		t.Fatal("logic inspector absent")
	}
	for _, probe := range []string{"v:" + id, "i:" + id, "v:" + id + ".a", "v:" + id + ".b"} {
		if _, ok := findNode(user.view(), "plot-"+probe); !ok {
			t.Fatal("logic plot absent", probe)
		}
	}
	if err := ui.Validate(user.view()); err != nil {
		t.Fatal(err)
	}
}
func TestLogicNativeExamplesAndAgentParity(t *testing.T) {
	for _, id := range []string{"logic-inverter", "logic-switches", "logic-half-adder"} {
		t.Run(id, func(t *testing.T) {
			user, cleanup := testScreen(t)
			defer cleanup()
			agent := workbench.New()
			defer agent.Close()
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + id})
			agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: id})
			for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
				if id == "logic-switches" && cmd == "simulation.restart" {
					continue
				}
				uiAction(t, user, cmd, nil)
				agentAction(t, agent, cmd, struct{}{})
			}
			left, right := awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
			if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.Scope.Selected, right.Scope.Selected) || left.Document.Dirty {
				t.Fatal("logic example defaults")
			}
			if title, ok := findNode(user.view(), "scope-title"); id != "logic-switches" && (!ok || title.Text != "Live logic scope") {
				t.Fatal("logic caption", title.Text)
			}
			if id != "logic-switches" {
				for range 30 {
					uiAction(t, user, "transient-step", nil)
					agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: left.TransientSettings.StepSeconds})
				}
			}
			if !reflect.DeepEqual(user.service.Snapshot().Mixed, agent.Snapshot().Mixed) {
				t.Fatal("logic step divergence")
			}
		})
	}
}
func TestLogicSymbolRotationAndPins(t *testing.T) {
	ink := color.NRGBA{A: 255}
	for _, kind := range []string{"logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor"} {
		p, _ := workbench.MakeComponent("U1", kind, workbench.Point{}, "")
		base := &meshBuilder{}
		drawComponentSymbol(base, workbench.Snapshot{}, p, ink, ink, ink, ink, true)
		for _, angle := range []int{90, 180, 270} {
			p.Rotation = angle
			rotated := &meshBuilder{}
			drawComponentSymbol(rotated, workbench.Snapshot{}, p, ink, ink, ink, ink, true)
			if len(base.vertices) != len(rotated.vertices) {
				t.Fatal("logic rotation shape")
			}
			for i, v := range base.vertices {
				want := workbench.RotateLocal(workbench.Point{X: v.Position.X, Y: v.Position.Y}, angle)
				got := rotated.vertices[i].Position
				if math.Hypot(want.X-got.X, want.Y-got.Y) > 1e-9 {
					t.Fatal("logic pin text rotation", kind, angle)
				}
			}
			at := workbench.RotateLocal(workbench.Point{X: 38, Y: 38}, angle)
			if hitComponent(workbench.Snapshot{Components: []workbench.Component{p}}, at) != "U1" {
				t.Fatal("logic body selection")
			}
		}
	}
}
