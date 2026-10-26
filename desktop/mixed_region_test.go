package main

import (
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMixedNPNMOSCaptionNamesTheDeviceProvidingTraceState(t *testing.T) {
	graph := workbench.NPNPulseExampleGraph()
	mos, _ := workbench.MakeComponent("MX", "nmos", workbench.Point{X: 400, Y: 60}, "")
	load, _ := workbench.MakeComponent("RX", "resistor", workbench.Point{X: 400, Y: -100}, "1k")
	graph.Components = append(graph.Components, mos, load)
	graph.Wires = append(graph.Wires, []workbench.Wire{
		{ID: "WX1", From: "VCC.positive", To: "RX.1", Connected: true},
		{ID: "WX2", From: "RX.2", To: "MX.drain", Connected: true},
		{ID: "WX3", From: "MX.source", To: "GND1.reference", Connected: true},
		{ID: "WX4", From: "MX.gate", To: "GND1.reference", Connected: true},
	}...)
	user, agent, ids := buildNetworkFixtureUIAndAgent(t, graph)
	state := user.service.Snapshot()
	if state.Mixed.MOSFETs[0].State != "cutoff" || state.Mixed.Transistors[0].State != "saturation" || state.Transient.Sample.State != "saturation" || agent.Snapshot().Transient.Sample.State != state.Transient.Sample.State {
		t.Fatal("fixture must separate NPN trace state from cutoff MOSFET")
	}
	check := func() {
		t.Helper()
		state := user.service.Snapshot()
		region := state.Mixed.Transistors[0].State
		caption, _ := findNode(user.view(), "scope-caption-detail")
		if !strings.Contains(caption.Text, ids["Q1"]+" "+region) || strings.Contains(caption.Text, ids["MX"]+" "+region) {
			t.Fatal("scope associates the NPN state with the wrong device", caption.Text)
		}
		// The separate whole-network card intentionally reports its named
		// first MOSFET; selection must not silently change either source.
		title, _ := findNode(user.view(), "power-title")
		value, _ := findNode(user.view(), "power-value")
		if !strings.EqualFold(title.Text, ids["MX"]+" region") || value.Text != "cutoff" {
			t.Fatal("independent named MOSFET summary changed", title.Text, value.Text)
		}
	}
	check()
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["MX"]})
	check()
	for range 50 {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	if user.service.Snapshot().Transient.Sample.State != "cutoff" || agent.Snapshot().Transient.Sample.State != "cutoff" {
		t.Fatal("pulse edge must turn the NPN off")
	}
	check()
}
