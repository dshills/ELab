package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestProbeControlsAndCursorPointerAgentParity(t *testing.T) {
	user, close := testScreen(t)
	defer close()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "example.flasher", nil)
	agentAction(t, agent, "example.flasher", workbench.ReplaceParams{})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, user, command, nil)
		agentAction(t, agent, command, struct{}{})
	}
	for i := 0; i < 70; i++ {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .005})
	}
	before := user.service.Snapshot()
	// Replace one channel with a capacitor-current probe, after samples exist.
	uiAction(t, user, "probe-5", &ui.ChangeEvent{SelectedID: "i:C2"})
	ids := append([]string{}, agent.Snapshot().Scope.Selected...)
	ids[5] = "i:C2"
	agentAction(t, agent, "scope.probes.set", workbench.ProbeParams{IDs: ids})
	uiAction(t, user, "cursor-time-a", &ui.ChangeEvent{Number: 100})
	agentAction(t, agent, "scope.cursor.set", workbench.CursorParams{Cursor: "a", Time: .1})
	uiAction(t, user, "cursor-select-b", nil)
	agentAction(t, agent, "scope.cursor.select", workbench.CursorSelectParams{Cursor: "b"})
	user.app.Do(func() { user.scope.placeCursor(58 + (800-78-58)*.75) })
	agentAction(t, agent, "scope.cursor.set", workbench.CursorParams{Cursor: "b", Time: agent.Snapshot().Transient.Sample.Time * .75})
	u, a := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Scope, a.Scope) {
		t.Fatal("native callbacks/pointer and agent scope state differ")
	}
	if u.RunID != before.RunID || u.ElectricalRevision != before.ElectricalRevision || u.Transient.Sample != before.Transient.Sample {
		t.Fatal("probe/cursors reset circuit")
	}
	if u.Scope.DeltaSeconds == nil || math.Abs(*u.Scope.DeltaSeconds-.16) > .0051 {
		t.Fatal("cursor did not snap to recorded grid")
	}
	uiAction(t, user, "cursor-clear-a", nil)
	agentAction(t, agent, "scope.cursor.clear", workbench.CursorSelectParams{Cursor: "a"})
	if !reflect.DeepEqual(user.service.Snapshot().Scope, agent.Snapshot().Scope) || user.service.Snapshot().Scope.DeltaSeconds != nil {
		t.Fatal("clear control differs")
	}
	mesh, err := user.scopeMesh(*user.scope.View)
	if err != nil || len(mesh.Vertices()) == 0 {
		t.Fatal("scope plot missing")
	}
	root := user.view()
	if err := ui.Validate(root); err != nil {
		t.Fatal(err)
	}
}
