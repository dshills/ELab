//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedPlacementControls(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		production := ui.Node{ID: "placement", Kind: ui.Column, Children: s.placementControls(state)}
		root := ui.Node{ID: "root", Kind: ui.Column}
		if state.Drawing.Tool == "place" {
			root.Focus = ui.FocusRequest{ID: "position-x", Sequence: 1}
			position := ui.Node{ID: "fixture-position", Kind: ui.Row}
			for _, id := range []string{"place-label", "position-x", "position-y", "place-at-position"} {
				node, ok := findNode(production, id)
				if !ok {
					if id == "place-label" {
						continue
					}
					t.Fatalf("missing placement control %s", id)
				}
				node.MinWidth, node.MinHeight = 150, 40
				if id == "position-x" || id == "position-y" {
					position.Children = append(position.Children, node)
				} else {
					if id == "place-label" {
						root.Focus.ID = id
					} else {
						root.Children = append(root.Children, position)
					}
					root.Children = append(root.Children, node)
				}
			}
		}

		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	s.settleNumber = func(id string, done func(error)) error {
		return cogent.SettleNumberField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func TestMountedPlacementOneClickUsesTypedCoordinate(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	m := mountedPlacementControls(t, s)
	before := s.service.Snapshot()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, r := range "120" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("coordinate typing prematurely changed public state")
	}
	clickMountedControl(t, m, "place-at-position")
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Components) != 1 || after.Components[0].Position != (workbench.Point{X: 120}) {
		t.Fatalf("one-click placement did not use typed X: position=%+v components=%+v error=%s", after.Drawing.Position, after.Components, s.errorMessage)
	}
}

func TestMountedPlacementSequentialCoordinates(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	agentAction(t, s.service, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: -200}})
	m := mountedPlacementControls(t, s)
	for _, field := range []struct{ id, text string }{{"position-x", "0"}, {"position-y", "-110"}} {
		clickMountedControl(t, m, field.id)
		m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
		for _, r := range field.text {
			m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
		}
		m.frame()
	}
	// The Cocoa accessibility press invokes the descriptor action without
	// transferring pointer focus from the numeric field first.
	uiAction(t, s, "place-at-position", nil)
	m.settle()
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Components) != 1 || after.Components[0].Position != (workbench.Point{Y: -110}) {
		t.Fatalf("sequential coordinate placement: components=%+v drawing=%+v error=%s", after.Components, after.Drawing, s.errorMessage)
	}
}

func TestMountedPlacementLabelAndBatchedCoordinatesMatchAgent(t *testing.T) {
	for _, kind := range []string{"resistor", "net-label"} {
		t.Run(kind, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agent := workbench.New()
			t.Cleanup(agent.Close)
			for _, svc := range []*workbench.Service{s.service, agent} {
				agentAction(t, svc, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: kind})
			}
			m := mountedPlacementControls(t, s)
			id, text := "position-y", "-94"
			if kind == "net-label" {
				id, text = "place-label", "SUPPLY"
			}
			clickMountedControl(t, m, id)
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, r := range text {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
			}
			m.frame()
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("private placement entry changed public state")
			}
			button := mountedControlElement(t, m, "place-at-position")
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
				if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: kind, Button: events.Left, Position: at}); err != nil {
					t.Fatal(err)
				}
			}
			m.settle()
			params := workbench.PlaceParams{Kind: kind, Position: before.Drawing.Position}
			if kind == "net-label" {
				params.Value = text
				agentAction(t, agent, "editor.value.set", workbench.DraftParams{Value: text})
			} else {
				agentAction(t, agent, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: before.Drawing.Position.X, Y: -94}})
				params.Position = agent.Snapshot().Drawing.Position
			}
			agentAction(t, agent, "component.place", params)
			after := s.service.Snapshot()
			expected := agent.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || !reflect.DeepEqual(after.Components, expected.Components) || !reflect.DeepEqual(after.Wires, expected.Wires) || after.Selected != expected.Selected {
				t.Fatal("batched focus-loss Place differs from public agent", after.Components, s.errorMessage, "revision", before.CommandRevision, after.CommandRevision, "pending", s.pendingPlacement)
			}
			uiAction(t, s, "edit.undo", nil)
			agentAction(t, agent, "edit.undo", struct{}{})
			if !reflect.DeepEqual(s.service.Snapshot().Components, agent.Snapshot().Components) || len(s.service.Snapshot().Components) != 0 {
				t.Fatal("one Undo did not remove the placed component")
			}
		})
	}
}

func TestMountedPlacementHeldClickRefusesAgentAfterOwnCommit(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	m := mountedPlacementControls(t, s)
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, r := range "120" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	m.frame()
	button := mountedControlElement(t, m, "place-at-position")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: 200, Y: 100}})
	before := s.service.Snapshot()
	focus := s.keyboardFocus
	m.frame()
	if mountedControlElement(t, m, "place-at-position").Incarnation != button.Incarnation {
		t.Fatal("fixture did not retain held button")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || focus != s.keyboardFocus {
		t.Fatal("held Place borrowed an agent coordinate", s.errorMessage)
	}
	clickMountedControl(t, m, "place-at-position")
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Components) != 1 || after.Components[0].Position != before.Drawing.Position {
		t.Fatal("fresh Place did not preserve agent coordinates", s.errorMessage)
	}
}
