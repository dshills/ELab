package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestBlocksInspectorUserAgentParityAndVisibleAcquisition(t *testing.T) {
	u, done := testScreen(t)
	defer done()
	a := workbench.New()
	defer a.Close()
	for _, s := range []*workbench.Service{u.service, a} {
		agentAction(t, s, "example.load", workbench.ExampleLoadParams{ID: "rc-block"})
		agentAction(t, s, "simulation.pause", struct{}{})
		awaitBlockRC(t, s)
	}
	state := u.service.Snapshot()
	g := workbench.Graph{Components: state.Components, Wires: state.Wires}
	r, _ := g.Component("R1")
	if point, ok := workbench.TerminalPosition(r, "1"); ok && terminalHit(state, point, 1) == "R1.1" {
		t.Fatal("hidden terminal acquired")
	}
	if hitComponent(state, r.Position) == r.ID {
		t.Fatal("collapsed member hit")
	}
	uiAction(t, u, "block-value-resistance", &ui.ChangeEvent{Text: "2k"})
	agentAction(t, a, "block.parameter.set", workbench.BlockValueParams{ID: "FILTER", Parameter: "resistance", Value: "2k"})
	if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("parameter parity")
	}
	uiAction(t, u, "block-expand", nil)
	agentAction(t, a, "block.view.set", workbench.BlockViewParams{ID: "FILTER", Expanded: true})
	if hitComponent(u.service.Snapshot(), r.Position) != "R1" {
		t.Fatal("expanded member unavailable")
	}
	if !reflect.DeepEqual(u.service.Snapshot().Components, a.Snapshot().Components) {
		t.Fatal("view parity")
	}
	held, ok := findNode(u.view(), "block-value-resistance")
	if !ok {
		t.Fatal("field missing")
	}
	agentAction(t, u.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	before := u.service.Snapshot().Components
	u.app.Do(func() { held.Change(ui.ChangeEvent{Text: "4k"}) })
	if !reflect.DeepEqual(before, u.service.Snapshot().Components) {
		t.Fatal("stale block input")
	}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, u.service, "appearance.theme.set", map[string]string{"theme": theme})
		if err := ui.Validate(u.view()); err != nil {
			t.Fatal(theme, err)
		}
	}
}
func TestBlocksCreateFormUsesSharedCommand(t *testing.T) {
	u, done := testScreen(t)
	defer done()
	agentAction(t, u.service, "example.load", workbench.ExampleLoadParams{ID: "step-response"})
	agentAction(t, u.service, "simulation.pause", struct{}{})
	awaitBlockRC(t, u.service)
	agentAction(t, u.service, "workspace.inspector.set", workbench.InspectorParams{Section: "document"})
	for id, text := range map[string]string{"block-form-0": "FILTER", "block-form-1": "RC block", "block-form-2": "R1,C1", "block-form-3": "in=R1.1,out=C1.1,ground=C1.2", "block-form-4": "resistance=R1,capacitance=C1"} {
		uiAction(t, u, id, &ui.ChangeEvent{Text: text})
	}
	uiAction(t, u, "block-create", nil)
	state := u.service.Snapshot()
	p, ok := (workbench.Graph{Components: state.Components, Wires: state.Wires}).Component("FILTER")
	if !ok || p.Block == nil || len(p.Block.Ports) != 3 {
		t.Fatal("shared create failed", state.Diagnostics)
	}
}

func awaitBlockRC(t *testing.T, s *workbench.Service) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := s.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	state := s.Snapshot()
	if len(state.Diagnostics) != 0 || state.Transient == nil {
		t.Fatal("RC did not initialize", state.Diagnostics)
	}
}

func TestBlockHitExtentsUseHalfSizesLikeSelectionOutline(t *testing.T) {
	g := workbench.RCBlockExampleGraph()
	block, _ := g.Component("FILTER")
	w, h := workbench.BlockBodySize(block)
	w += 30
	h += 8
	state := workbench.Snapshot{Components: g.Components, Wires: g.Wires}
	for _, point := range []workbench.Point{{X: block.Position.X + w + 1, Y: block.Position.Y}, {X: block.Position.X, Y: block.Position.Y + h + 1}} {
		if hitComponent(state, point) == block.ID {
			t.Fatal("block captured outside the drawn half-extent outline", point)
		}
	}
	if hitComponent(state, block.Position) != block.ID {
		t.Fatal("body did not acquire")
	}
}
