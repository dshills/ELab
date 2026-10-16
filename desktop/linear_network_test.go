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
