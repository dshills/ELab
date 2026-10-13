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

func TestNPNConstructionAndDriveUIAgentParity(t *testing.T) {
	user, close := testScreen(t)
	defer close()
	agent := workbench.New()
	defer agent.Close()
	fixture := workbench.NPNExampleGraph()
	ids := map[string]string{}
	for _, part := range fixture.Components {
		uiAction(t, user, "part-"+part.Kind, nil)
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
	uiAction(t, user, "tool-wire", nil)
	for _, wire := range fixture.Wires {
		terminal := func(ref string) string {
			for old, new := range ids {
				if len(ref) > len(old) && ref[:len(old)+1] == old+"." {
					return new + ref[len(old):]
				}
			}
			return ref
		}
		from, to := terminal(wire.From), terminal(wire.To)
		uiAction(t, user, "wire-from", &ui.ChangeEvent{SelectedID: from})
		uiAction(t, user, "wire-to", &ui.ChangeEvent{SelectedID: to})
		uiAction(t, user, "connect-terminals", nil)
		agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: from, To: to})
	}
	wait := func(s *workbench.Service) workbench.Snapshot {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		state, err := s.Wait(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return state
	}
	u, a := wait(user.service), wait(agent)
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || u.NPN == nil || a.NPN == nil || u.NPN.CollectorCurrent != a.NPN.CollectorCurrent {
		t.Fatal("NPN drawing diverged")
	}
	for _, drive := range []struct{ control, value, state string }{{"npn-off", "0 V", "cutoff"}, {"npn-active", "0.8 V", "active"}, {"npn-on", "5 V", "saturation"}} {
		uiAction(t, user, drive.control, nil)
		agentAction(t, agent, "component.set", workbench.ValueParams{ID: ids["VIN"], Value: drive.value})
		u, a = wait(user.service), wait(agent)
		if u.NPN.State != drive.state || u.NPN.State != a.NPN.State || math.Abs(u.NPN.CollectorCurrent-a.NPN.CollectorCurrent) > 1e-14 {
			t.Fatalf("drive %s differs", drive.value)
		}
	}
	uiAction(t, user, "simulation.pause", nil)
	uiAction(t, user, "npn-off", nil)
	if state := user.service.Snapshot(); !state.NPN.Stale || !state.Paused {
		t.Fatal("paused drive should be stale")
	}
	uiAction(t, user, "simulation.resume", nil)
	if wait(user.service).NPN.State != "cutoff" {
		t.Fatal("resume failed")
	}
	uiAction(t, user, "example.npn", nil)
	if user.pendingCommand != "example.npn" {
		t.Fatal("NPN example bypassed unsaved-work protection")
	}
	uiAction(t, user, "discard-continue", nil)
	if wait(user.service).NPN.State != "saturation" {
		t.Fatal("NPN example failed")
	}
}
