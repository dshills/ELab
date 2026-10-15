package main

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeJunctionAndNetLabelControlsMatchAgentCommands(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "example.reset", nil)
	agentAction(t, agent, "example.reset", workbench.ReplaceParams{})
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "W1"})
	agentAction(t, agent, "component.select", workbench.IDParams{ID: "W1"})
	uiAction(t, user, "wire-insert-junction", nil)
	agentAction(t, agent, "wire.junction.insert", workbench.JunctionInsertParams{WireIDs: []string{"W1"}})
	if !reflect.DeepEqual(user.service.Snapshot().Components, agent.Snapshot().Components) || !reflect.DeepEqual(user.service.Snapshot().Wires, agent.Snapshot().Wires) {
		t.Fatal("native/agent junction insertion differs")
	}
	uiAction(t, user, "edit.undo", nil)
	agentAction(t, agent, "edit.undo", struct{}{})
	// Named remote endpoints replace the supply wire through ordinary user edits.
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "W1"})
	uiAction(t, user, "delete-object", nil)
	agentAction(t, agent, "object.delete", workbench.IDParams{ID: "W1"})
	for i, position := range []workbench.Point{{X: -300, Y: -100}, {X: 300, Y: -100}} {
		uiAction(t, user, "part-list", &ui.ChangeEvent{SelectedID: "net-label"})
		uiAction(t, user, "place-label", &ui.ChangeEvent{Text: "VCC"})
		uiAction(t, user, "position-x", &ui.ChangeEvent{Number: float32(position.X)})
		uiAction(t, user, "position-y", &ui.ChangeEvent{Number: float32(position.Y)})
		uiAction(t, user, "place-at-position", nil)
		agentAction(t, agent, "component.place", workbench.PlaceParams{Kind: "net-label", Position: position, Value: "VCC"})
		id := user.service.Snapshot().Selected
		pair := workbench.ConnectParams{From: "V1.positive", To: id + ".node"}
		if i == 1 {
			pair = workbench.ConnectParams{From: id + ".node", To: "R1.1"}
		}
		uiAction(t, user, "tool-wire", nil)
		uiAction(t, user, "wire-from", &ui.ChangeEvent{SelectedID: pair.From})
		uiAction(t, user, "wire-to", &ui.ChangeEvent{SelectedID: pair.To})
		uiAction(t, user, "connect-terminals", nil)
		agentAction(t, agent, "wire.connect", pair)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	u, err := user.service.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, err := agent.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || !reflect.DeepEqual(u.Nets, a.Nets) {
		t.Fatal("native and agent named connectivity differs")
	}
	if u.Result == nil || u.Result.Stale || math.Abs(u.Result.Current-.00291679) > 2e-8 {
		t.Fatal("named supply did not resume LED simulation")
	}
	uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "SUPPLY"})
	uiAction(t, user, "apply-value", nil)
	agentAction(t, agent, "component.set", workbench.ValueParams{ID: a.Selected, Value: "SUPPLY"})
	if !reflect.DeepEqual(user.service.Snapshot().Components, agent.Snapshot().Components) {
		t.Fatal("native label edit differs")
	}
}

func TestNativeCrossingPlacementRequiresExplicitReviewedJoin(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	for _, p := range []workbench.PlaceParams{{ID: "Left", Kind: "junction", Position: workbench.Point{X: -100}}, {ID: "Right", Kind: "junction", Position: workbench.Point{X: 100}}, {ID: "Top", Kind: "junction", Position: workbench.Point{Y: -100}}, {ID: "Bottom", Kind: "junction", Position: workbench.Point{Y: 100}}} {
		agentAction(t, user.service, "component.place", p)
	}
	agentAction(t, user.service, "wire.connect", workbench.ConnectParams{ID: "Horizontal", From: "Left.node", To: "Right.node"})
	agentAction(t, user.service, "wire.connect", workbench.ConnectParams{ID: "Vertical", From: "Top.node", To: "Bottom.node"})
	uiAction(t, user, "part-list", &ui.ChangeEvent{SelectedID: "junction"})
	user.app.Do(func() { user.canvasClick(workbench.Point{}) })
	state := user.service.Snapshot()
	if len(state.Components) != 4 || len(state.Wires) != 2 || state.Drawing.Position != (workbench.Point{}) || user.errorMessage == "" {
		t.Fatal("ambiguous pointer silently joined wires or omitted review position")
	}
	user.errorMessage = ""
	node, ok := findNode(user.view(), "junction-targets")
	if !ok || node.Text != "Explicit junction targets: Horizontal, Vertical" {
		t.Fatal("crossing review omitted targeted wire identities")
	}
	uiAction(t, user, "junction-join-routes", nil)
	if len(user.service.Snapshot().Wires) != 4 || len(user.service.Snapshot().Components) != 5 {
		t.Fatal("reviewed crossing join failed")
	}
}
