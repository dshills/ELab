package main

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestFlasherDrawingInitialConditionsAndAgentParity(t *testing.T) {
	user, close := testScreen(t)
	defer close()
	agent := workbench.New()
	defer agent.Close()
	fixture := workbench.FlasherExampleGraph()
	ids := map[string]string{}
	uiAction(t, user, "simulation.pause", nil)
	agentAction(t, agent, "simulation.pause", struct{}{})
	for _, p := range fixture.Components {
		uiAction(t, user, "part-"+p.Kind, nil)
		user.app.Do(func() { user.canvasClick(p.Position) })
		id := user.service.Snapshot().Selected
		ids[p.ID] = id
		agentAction(t, agent, "component.place", workbench.PlaceParams{Kind: p.Kind, Position: p.Position})
		if p.Unit != "" {
			value := fmt.Sprintf("%g %s", p.Value, p.Unit)
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: value})
			uiAction(t, user, "apply-value", nil)
			agentAction(t, agent, "component.set", workbench.ValueParams{ID: id, Value: value})
		}
		for turn := 0; turn < p.Rotation; turn += 90 {
			uiAction(t, user, "rotate-part", nil)
			agentAction(t, agent, "component.rotate", workbench.IDParams{ID: id})
		}
		if p.InitialVoltage != nil {
			uiAction(t, user, "cap-initial", &ui.ChangeEvent{Number: float32(*p.InitialVoltage)})
			agentAction(t, agent, "capacitor.initial.set", workbench.ValueParams{ID: id, Value: fmt.Sprintf("%.9g V", float32(*p.InitialVoltage))})
		}
	}
	ref := func(pin string) string {
		for old, id := range ids {
			if len(pin) > len(old) && pin[:len(old)+1] == old+"." {
				return id + pin[len(old):]
			}
		}
		return pin
	}
	uiAction(t, user, "tool-wire", nil)
	for _, w := range fixture.Wires {
		from, to := ref(w.From), ref(w.To)
		uiAction(t, user, "wire-from", &ui.ChangeEvent{SelectedID: from})
		uiAction(t, user, "wire-to", &ui.ChangeEvent{SelectedID: to})
		uiAction(t, user, "connect-terminals", nil)
		agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: from, To: to})
	}
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	cfg := agent.Snapshot().TransientSettings
	cfg.StepSeconds = .005
	uiAction(t, user, "setting-step", &ui.ChangeEvent{Text: "5 ms"})
	agentAction(t, agent, "simulation.transient.set", cfg)
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	for i := 0; i < 65; i++ {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .005})
	}
	u, a := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || u.Mixed == nil || a.Mixed == nil || u.Transient.Sample != a.Transient.Sample {
		t.Fatal("mixed construction/stepping diverged")
	}
	for i, c := range u.Mixed.Capacitors {
		if math.Abs(c.Voltage-a.Mixed.Capacitors[i].Voltage) > 1e-9 {
			t.Fatal("capacitor memory differs")
		}
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["C1"]})
	uiAction(t, user, "cap-initial", &ui.ChangeEvent{Number: -.2})
	if !user.service.Snapshot().Mixed.Stale {
		t.Fatal("initial voltage edit did not stale result")
	}
	uiAction(t, user, "edit.undo", nil)
	uiAction(t, user, "example.flasher", nil)
	if user.pendingCommand != "example.flasher" {
		t.Fatal("flasher bypasses dirty protection")
	}
	uiAction(t, user, "discard-continue", nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s, err := user.service.Wait(ctx)
	if err != nil || s.Mixed == nil || s.Analysis != workbench.MixedAnalysis {
		t.Fatal("flasher example failed")
	}
}
func TestFlasherRoutesAvoidSymbolBodies(t *testing.T) {
	g := workbench.FlasherExampleGraph()
	for _, w := range g.Wires {
		path := wirePath(g, w)
		a, _ := g.TerminalPoint(w.From)
		b, _ := g.TerminalPoint(w.To)
		if len(path) < 2 || path[0].X != a.X || path[0].Y != a.Y || path[len(path)-1].X != b.X || path[len(path)-1].Y != b.Y {
			t.Fatal("route lost terminals")
		}
		for _, p := range g.Components {
			r := routeBody(p)
			for i := 2; i < len(path)-2; i++ {
				u, v := path[i-1], path[i]
				if u.X == v.X && u.X > r.left && u.X < r.right && math.Max(u.Y, v.Y) > r.top && math.Min(u.Y, v.Y) < r.bottom || u.Y == v.Y && u.Y > r.top && u.Y < r.bottom && math.Max(u.X, v.X) > r.left && math.Min(u.X, v.X) < r.right {
					t.Fatalf("wire %s crosses %s body", w.ID, p.ID)
				}
			}
		}
	}
}
