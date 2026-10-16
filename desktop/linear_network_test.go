package main

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestNative24VLinearConstructionMatchesAgent(t *testing.T) {
	g := workbench.SplitSupplyExampleGraph()
	g.Components[0].Value, g.Components[1].Value = 12, 12
	g.Wires[4].From = "V2.negative"
	user, agent, _ := buildNetworkFixtureUIAndAgent(t, g)
	left, right := user.service.Snapshot(), agent.Snapshot()
	if left.Mixed == nil || right.Mixed == nil || !reflect.DeepEqual(left.Mixed.Resistors, right.Mixed.Resistors) || math.Abs(left.Mixed.Resistors[0].Voltage-24) > 1e-12 || math.Abs(left.Mixed.Resistors[0].Current-.024) > 1e-12 {
		t.Fatal("native high-voltage linear network differs", left.Diagnostics, right.Diagnostics)
	}
	root := user.view()
	for id, text := range map[string]string{"voltage-value": "24 V", "current-value": "24 mA"} {
		node, ok := findNode(root, id)
		if !ok || !strings.Contains(node.Text, text) {
			t.Fatal("native high-voltage result not visible", id, node.Text)
		}
	}
}

func TestNative24VLEDConstructionMatchesAgent(t *testing.T) {
	g := workbench.SplitSupplyExampleGraph()
	g.Components[0].Value, g.Components[1].Value = 12, 12
	g.Wires[4].From = "V2.negative"
	led, err := workbench.MakeComponent("D1", "led", workbench.Point{X: 350, Y: 100}, "")
	if err != nil {
		t.Fatal(err)
	}
	g.Components = append(g.Components, led)
	g.Wires[1].From = "D1.cathode"
	g.Wires = append(g.Wires, workbench.Wire{ID: "W6", From: "R1.2", To: "D1.anode", Connected: true})
	user, agent, _ := buildNetworkFixtureUIAndAgent(t, g)
	left, right := user.service.Snapshot(), agent.Snapshot()
	if left.Mixed == nil || right.Mixed == nil || !reflect.DeepEqual(left.Mixed.LEDs, right.Mixed.LEDs) || left.Mixed.LEDs[0].Current < .021 || left.Mixed.LEDs[0].Current > .023 || left.Mixed.KCLResidual > 1e-10 || math.Abs(left.Mixed.PowerResidual) > 1e-8 {
		t.Fatal("native nonlinear high-rail construction differs", left.Diagnostics, right.Diagnostics)
	}
}
