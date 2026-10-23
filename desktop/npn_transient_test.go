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

func TestNPNPulseDrawingAndControlsMatchAgents(t *testing.T) {
	user, close := testScreen(t)
	defer close()
	agent := workbench.New()
	defer agent.Close()
	fixture := workbench.NPNPulseExampleGraph()
	ids := map[string]string{}
	for _, part := range fixture.Components {
		uiAction(t, user, "part-list", &ui.ChangeEvent{SelectedID: part.Kind})
		user.app.Do(func() { user.canvasClick(part.Position) })
		id := user.service.Snapshot().Selected
		ids[part.ID] = id
		agentAction(t, agent, "component.place", workbench.PlaceParams{Kind: part.Kind, Position: part.Position})
		for turn := 0; turn < part.Rotation; turn += 90 {
			uiAction(t, user, "rotate-part", nil)
			agentAction(t, agent, "component.rotate", workbench.IDParams{ID: id})
		}
		if part.ID == "RB" {
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "10k"})
			uiAction(t, user, "apply-value", nil)
			agentAction(t, agent, "component.set", workbench.ValueParams{ID: id, Value: "10k"})
		}
	}
	// Build while paused so both clients start from exactly the same time.
	uiAction(t, user, "simulation.pause", nil)
	agentAction(t, agent, "simulation.pause", struct{}{})
	uiAction(t, user, "tool-wire", nil)
	ref := func(pin string) string {
		for old, id := range ids {
			if len(pin) > len(old) && pin[:len(old)+1] == old+"." {
				return id + pin[len(old):]
			}
		}
		return pin
	}
	for _, w := range fixture.Wires {
		from, to := ref(w.From), ref(w.To)
		uiAction(t, user, "wire-from", &ui.ChangeEvent{SelectedID: from})
		uiAction(t, user, "wire-to", &ui.ChangeEvent{SelectedID: to})
		uiAction(t, user, "connect-terminals", nil)
		agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: from, To: to})
	}
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	// Changing durations uses the same saved/undoable settings command.
	uiAction(t, user, "setting-high", &ui.ChangeEvent{Text: "30 ms"})
	cfg := agent.Snapshot().TransientSettings
	cfg.HighSeconds = .03
	agentAction(t, agent, "simulation.transient.set", cfg)
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	for i := 0; i < 30; i++ {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	u, a := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || u.NPN.State != "cutoff" || u.Transient.Sample != a.Transient.Sample || u.NPN.CollectorCurrent != a.NPN.CollectorCurrent {
		t.Fatal("pulse drawing/stepping diverged")
	}
	uiAction(t, user, "input-high", nil)
	agentAction(t, agent, "simulation.input.set", workbench.InputParams{Mode: "high"})
	if user.service.Snapshot().Transient.PendingMode != "high" {
		t.Fatal("UI high override not queued")
	}
	uiAction(t, user, "transient-step", nil)
	agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	u, a = user.service.Snapshot(), agent.Snapshot()
	if u.Transient.Sample != a.Transient.Sample || u.NPN.State != "saturation" || math.Abs(u.Transient.Sample.Time-.031) > 1e-12 {
		t.Fatal("high override differs")
	}
	user.app.Do(func() {
		mesh, err := user.scopeMesh(*user.scope.View)
		if err != nil || mesh == nil {
			t.Fatal("NPN scope failed")
		}
		root := user.view()
		if _, ok := findNode(root, "setting-initial"); ok {
			t.Fatal("NPN experiment exposes irrelevant capacitor initial voltage")
		}
		n, ok := findNode(root, "scope-title")
		if !ok || n.Text != "Live NPN scope" {
			t.Fatal("missing NPN scope")
		}
	})
	uiAction(t, user, "example.npn-pulse", nil)
	if user.pendingCommand != "example.npn-pulse" {
		t.Fatal("blink example bypassed dirty protection")
	}
	uiAction(t, user, "discard-continue", nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	state, err := user.service.Wait(ctx)
	if err != nil || state.Analysis != workbench.NPNTransientAnalysis || state.Transient.Mode != "auto" {
		t.Fatal("blink example failed")
	}
}
