package main

import (
	"context"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestSplitSupplyBlankConstructionMatchesAgent(t *testing.T) {
	user, agent, ids := buildNetworkFixtureUIAndAgent(t, workbench.SplitSupplyExampleGraph())
	left, right := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Mixed.Resistors, right.Mixed.Resistors) || !reflect.DeepEqual(left.Mixed.Sources, right.Mixed.Sources) || !reflect.DeepEqual(left.Mixed.Nodes, right.Mixed.Nodes) || math.Abs(left.Mixed.Resistors[0].Current-.006) > 1e-12 {
		t.Fatal("split rail native/agent currents/voltages differ")
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["V2"]})
	if node, ok := findNode(user.view(), "measurement-value-v:"+ids["V2"]+".negative"); !ok || !strings.Contains(node.Text, "-3") || !strings.Contains(node.Text, "V") {
		t.Fatal("floating source has no signed negative-rail reading", node.Text)
	}
}

func TestSplitSupplyExampleLoadEditAndResetProtection(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.split-supply"})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, err := s.service.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	state := s.service.Snapshot()
	if state.Example == nil || state.Example.Info.ID != "split-supply" || state.Selected != "R1" || state.Mixed == nil || math.Abs(state.Mixed.Resistors[0].Voltage-6) > 1e-12 {
		t.Fatal("native split supply example failed", state.Diagnostics)
	}
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "2k"})
	uiAction(t, s, "apply-value", nil)
	uiAction(t, s, "example.current.reset", nil)
	if s.pendingCommand != "example.current.reset" {
		t.Fatal("split supply reset bypasses unsaved-work protection")
	}
	uiAction(t, s, "discard-continue", nil)
	if !reflect.DeepEqual(s.service.Snapshot().Components, workbench.SplitSupplyExampleGraph().Components) {
		t.Fatal("reset changed wrong example")
	}
}
