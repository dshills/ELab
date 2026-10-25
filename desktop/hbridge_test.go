package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestHBridgeNativeExampleSwitchesPlotsAndAgentParity(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example.hbridge-motor"})
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "hbridge-motor"})
	for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, u, cmd, nil)
		agentAction(t, a, cmd, struct{}{})
	}
	left, right := awaitInductorCircuit(t, u.service), awaitInductorCircuit(t, a)
	if !reflect.DeepEqual(left.Mixed, right.Mixed) || left.Document.Dirty || left.HBridgeModel == nil {
		t.Fatal("native example differs")
	}
	for _, id := range []string{"hbridge-state", "hbridge-inputs", "hbridge-voltage", "hbridge-current", "hbridge-loss", "hbridge-clamps", "hbridge-truth-table", "plot-v:U1", "plot-i:U1.vm", "plot-i:U1.out1", "plot-v:U1.enable"} {
		if _, ok := findNode(u.view(), id); !ok {
			t.Fatal("missing driver control/readout", id)
		}
	}
	for _, theme := range []string{"light", "dark", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(theme, err)
		}
	}
	// Existing switch UI routes are the driver API; no private mode override.
	for _, id := range []string{"S1", "S2", "SEN"} {
		uiAction(t, u, "selection", &ui.ChangeEvent{SelectedID: id})
		uiAction(t, u, "switch-actuate", nil)
		p, _ := workbench.HBridgeExampleGraph().Component(id)
		agentAction(t, a, "switch.set", workbench.SwitchParams{ID: id, Closed: !*p.SwitchClosed})
	}
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "simulation.step", workbench.StepParams{Duration: .00005})
		s.Wait(t.Context())
	}
	if !reflect.DeepEqual(u.service.Snapshot().Mixed, a.Snapshot().Mixed) || !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("native mode switches diverge from agent")
	}
	uiAction(t, u, "selection", &ui.ChangeEvent{SelectedID: "U1"})
	if node, ok := findNode(u.view(), "hbridge-state"); !ok || !strings.Contains(node.Text, "coast") {
		t.Fatal("driver mode missing")
	}
	uiAction(t, u, "plot-i:U1.out1", nil)
	selected := u.service.Snapshot().Scope.Selected
	found := false
	for _, id := range selected {
		found = found || id == "i:U1.out1"
	}
	if !found {
		t.Fatal("output plot not dispatched")
	}
}
func TestHBridgeSymbolAllRotationsAndBrowser(t *testing.T) {
	for _, rotation := range []int{0, 90, 180, 270} {
		p, err := workbench.MakeComponent("U1", "hbridge", workbench.Point{X: 100, Y: 100}, "")
		if err != nil {
			t.Fatal(err)
		}
		p.Rotation = rotation
		at := workbench.RotateLocal(workbench.Point{X: 64, Y: 74}, rotation)
		at.X += p.Position.X
		at.Y += p.Position.Y
		if hitComponent(workbench.Snapshot{Components: []workbench.Component{p}}, at) != "U1" {
			t.Fatal("driver body not selectable", rotation)
		}
		seen := map[workbench.Point]bool{}
		for _, pin := range p.Terminals {
			at, ok := workbench.TerminalPosition(p, pin)
			if !ok || seen[at] {
				t.Fatal(rotation, pin, at)
			}
			seen[at] = true
		}
		if len(workbench.HBridgeSymbolPaths()) != 8 || len(workbench.HBridgeSymbolLabels(p)) != 7 {
			t.Fatal("missing bridge pins/labels")
		}
	}
	u, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, u, "part-category", &ui.ChangeEvent{SelectedID: "motors"})
	uiAction(t, u, "part-search", &ui.ChangeEvent{Text: "H bridge"})
	if state := u.service.Snapshot(); len(state.MatchingParts) != 1 || state.MatchingParts[0].Kind != "hbridge" {
		t.Fatal("driver search")
	}
}
