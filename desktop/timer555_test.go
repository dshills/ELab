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

func TestTimer555BlankDrawingAndAgentParity(t *testing.T) {
	fixture := workbench.Timer555FlasherExampleGraph()
	// From-blank circuits use discharged capacitors on both paths. The shipped
	// bypass startup voltage is covered by example loading below.
	for i := range fixture.Components {
		fixture.Components[i].InitialVoltage = nil
	}
	user, agent, ids := buildNetworkFixtureUIAndAgent(t, fixture)
	for range 20 {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	left, right := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || !reflect.DeepEqual(left.Timer555Model, right.Timer555Model) || !left.Mixed.Timers555[0].High {
		t.Fatal("555 native/agent drawing differs", left.Diagnostics, right.Diagnostics)
	}
	id := ids["U1"]
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: id})
	reading, ok := findNode(user.view(), "timer555-selected-reading")
	if !ok || !strings.Contains(reading.Text, "output high") || !strings.Contains(reading.Text, "Trigger level") {
		t.Fatal("555 lacks state and threshold readings")
	}
	for _, probe := range []string{"v:" + id, "i:" + id, "v:" + id + ".trigger", "v:" + id + ".threshold", "v:" + id + ".control"} {
		if _, ok := findNode(user.view(), "plot-"+probe); !ok {
			t.Fatal("missing 555 plot", probe)
		}
	}
	part := left.Components[2]
	if hitComponent(left, workbench.Point{X: part.Position.X + 60, Y: part.Position.Y + 80}) != id {
		t.Fatal("555 body edge cannot be selected")
	}
	if err := ui.Validate(user.view()); err != nil {
		t.Fatal(err)
	}
	if mesh, err := user.mesh(*user.drawing.View); err != nil || mesh == nil {
		t.Fatal("555 symbol", err)
	}
}
func TestTimer555NativeExamplesSharedControlsAndProtection(t *testing.T) {
	for _, id := range []string{"555-flasher", "555-oscillator", "555-one-shot"} {
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
				t.Fatal("555 example defaults differ")
			}
			for range 30 {
				uiAction(t, user, "transient-step", nil)
				agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: left.TransientSettings.StepSeconds})
			}
			left, right = user.service.Snapshot(), agent.Snapshot()
			if math.Abs(left.Transient.Sample.Time-30*left.TransientSettings.StepSeconds) > 1e-12 || !reflect.DeepEqual(left.Mixed, right.Mixed) {
				t.Fatal("555 steps diverged")
			}
			uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "C1"})
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "22u"})
			uiAction(t, user, "apply-value", nil)
			uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + id})
			if user.pendingCommand != "example.load" {
				t.Fatal("555 bypassed unsaved protection")
			}
			uiAction(t, user, "discard-continue", nil)
			if user.service.Snapshot().Example.Modified {
				t.Fatal("555 example did not reset")
			}
		})
	}
}

func TestTimer555SymbolAndPinTextRotateTogether(t *testing.T) {
	ink := color.NRGBA{A: 255}
	part, _ := workbench.MakeComponent("U1", "timer555", workbench.Point{}, "")
	original := &meshBuilder{}
	drawComponentSymbol(original, workbench.Snapshot{}, part, ink, ink, ink, ink, true)
	for _, rotation := range []int{90, 180, 270} {
		part.Rotation = rotation
		rotated := &meshBuilder{}
		drawComponentSymbol(rotated, workbench.Snapshot{}, part, ink, ink, ink, ink, true)
		if len(original.vertices) != len(rotated.vertices) {
			t.Fatal("555 rotation changed geometry")
		}
		for i, v := range original.vertices {
			expected := workbench.RotateLocal(workbench.Point{X: v.Position.X, Y: v.Position.Y}, rotation)
			got := rotated.vertices[i].Position
			if math.Hypot(got.X-expected.X, got.Y-expected.Y) > 1e-9 {
				t.Fatal("pin text did not rotate with symbol", rotation, i, expected, got)
			}
		}
		edge := workbench.RotateLocal(workbench.Point{X: 60, Y: 80}, rotation)
		if hitComponent(workbench.Snapshot{Components: []workbench.Component{part}}, edge) != "U1" {
			t.Fatal("rotated 555 body edge is not selectable")
		}
	}
}
