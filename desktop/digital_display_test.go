package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestDigitalDisplayNativeLoadAndAgentParity(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example.display-counter"})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "display-counter"})
	for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, user, cmd, nil)
		agentAction(t, agent, cmd, struct{}{})
	}
	a, b := awaitInductorCircuit(t, user.service), awaitInductorCircuit(t, agent)
	if !reflect.DeepEqual(a.Mixed, b.Mixed) || a.Document.Dirty || a.DisplayModel == nil {
		t.Fatal("display native defaults", a.Diagnostics)
	}
	for range 4 {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: a.TransientSettings.StepSeconds})
	}
	if !reflect.DeepEqual(user.service.Snapshot().Mixed, agent.Snapshot().Mixed) {
		t.Fatal("display step parity")
	}
	if reading, ok := findNode(user.view(), "display-selected-reading"); !ok || !strings.Contains(reading.Text, "Segments") {
		t.Fatal("missing display inspector")
	}
	for _, id := range []string{"plot-v:DS1.a", "plot-i:DS1.a"} {
		if _, ok := findNode(user.view(), id); !ok {
			t.Fatal("missing display plot", id)
		}
	}
	for _, kind := range []string{"bcd-decoder", "seven-segment"} {
		p, _ := workbench.MakeComponent("U1", kind, workbench.Point{}, "")
		for _, rotation := range []int{0, 90, 180, 270} {
			p.Rotation = rotation
			point := workbench.RotateLocal(workbench.Point{X: 80, Y: 180}, rotation)
			if hitComponent(workbench.Snapshot{Components: []workbench.Component{p}}, point) != "U1" {
				t.Fatal("display body hit", kind, rotation)
			}
		}
	}
}
