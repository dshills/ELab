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

func TestNMOSBlankConstructionUIAgentParity(t *testing.T) {
	testMOSFETBlankConstruction(t, workbench.NMOSExampleGraph())
}

func buildNetworkFixtureUIAndAgent(t *testing.T, fixture workbench.Graph) (*screen, *workbench.Service, map[string]string) {
	t.Helper()
	user, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
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
		if part.Unit != "" {
			value := workbench.FormatValue(part.Value, part.Unit)
			uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: value})
			uiAction(t, user, "apply-value", nil)
			agentAction(t, agent, "component.set", workbench.ValueParams{ID: id, Value: value})
		}
	}
	uiAction(t, user, "tool-wire", nil)
	for _, wire := range fixture.Wires {
		terminal := func(ref string) string { a, b, _ := strings.Cut(ref, "."); return ids[a] + "." + b }
		from, to := terminal(wire.From), terminal(wire.To)
		uiAction(t, user, "wire-from", &ui.ChangeEvent{SelectedID: from})
		uiAction(t, user, "wire-to", &ui.ChangeEvent{SelectedID: to})
		uiAction(t, user, "connect-terminals", nil)
		agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: from, To: to})
	}
	ctxReady, cancelReady := context.WithTimeout(context.Background(), time.Second)
	defer cancelReady()
	if _, err := user.service.Wait(ctxReady); err != nil {
		t.Fatal(err)
	}
	if _, err := agent.Wait(ctxReady); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
		if cmd == "simulation.restart" && user.service.Snapshot().Transient == nil {
			continue // Static DC construction is already fresh after Wait.
		}
		uiAction(t, user, cmd, nil)
		agentAction(t, agent, cmd, struct{}{})
	}
	left, right := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(left.Components, right.Components) || !reflect.DeepEqual(left.Wires, right.Wires) || left.Mixed == nil || right.Mixed == nil {
		t.Fatal("blank network construction diverged", left.Diagnostics, right.Diagnostics)
	}
	return user, agent, ids
}

func testMOSFETBlankConstruction(t *testing.T, fixture workbench.Graph) {
	t.Helper()
	user, agent, ids := buildNetworkFixtureUIAndAgent(t, fixture)
	if !reflect.DeepEqual(user.service.Snapshot().Mixed.MOSFETs, agent.Snapshot().Mixed.MOSFETs) {
		t.Fatal("MOSFET construction results diverged")
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: ids["M1"]})
	for _, probe := range []string{"v:" + ids["M1"], "v:" + ids["M1"] + ".gate-source", "i:" + ids["M1"] + ".drain"} {
		if _, ok := findNode(user.view(), "plot-"+probe); !ok {
			t.Fatal("MOSFET plot action missing", probe)
		}
	}
}

func TestNMOSExampleSharedLoadPulseAndProtection(t *testing.T) {
	testMOSFETExample(t, "nmos", "cutoff")
}

func testMOSFETExample(t *testing.T, example, edgeRegion string) {
	t.Helper()
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + example})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: example})
	ctxReady, cancelReady := context.WithTimeout(context.Background(), time.Second)
	defer cancelReady()
	if _, err := user.service.Wait(ctxReady); err != nil {
		t.Fatal(err)
	}
	if _, err := agent.Wait(ctxReady); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, user, cmd, nil)
		agentAction(t, agent, cmd, struct{}{})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := user.service.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := agent.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	}
	u, a := user.service.Snapshot(), agent.Snapshot()
	if u.Example == nil || u.Example.Info.ID != example || u.Mixed == nil || u.Mixed.Stale || u.Mixed.MOSFETs[0].State != edgeRegion || u.Mixed.MOSFETs[0].GateCurrent != 0 || !reflect.DeepEqual(u.Mixed.MOSFETs, a.Mixed.MOSFETs) || math.Abs(u.Transient.Sample.Time-.2) > 1e-12 {
		t.Fatal("MOSFET example pulse parity", u.Diagnostics, a.Diagnostics)
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "RL"})
	uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "2k"})
	uiAction(t, user, "apply-value", nil)
	if !user.service.Snapshot().Mixed.Stale {
		t.Fatal("paused edit failed to mark stale")
	}
	uiAction(t, user, "examples-menu", &ui.ChangeEvent{SelectedID: "example." + example})
	if user.pendingCommand != "example.load" {
		t.Fatal("MOSFET load bypassed replacement protection")
	}
	uiAction(t, user, "discard-continue", nil)
	if user.service.Snapshot().Example.Modified {
		t.Fatal("protected MOSFET load failed")
	}
}
