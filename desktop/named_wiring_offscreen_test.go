//go:build offscreen

package main

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// These are the production selectors before macOS menu adaptation. Native
// adapter callbacks have separate tests; neither fixture drives an OS menu.
func mountedNamedWireControls(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	s.app.Do(func() { s.requestKeyboardFocus("wire-from") })
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		production := ui.Node{ID: "named-wires", Kind: ui.Column, Children: s.namedWireControls(state)}
		focus := s.keyboardFocus
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: focus}
		for _, id := range []string{"wire-from", "wire-to"} {
			node, ok := findNode(production, id)
			if !ok {
				t.Fatalf("missing wire selector %s", id)
			}
			node.MinWidth, node.MinHeight = 300, 40
			root.Children = append(root.Children, node)
		}
		connect, ok := findNode(production, "connect-terminals")
		if !ok {
			t.Fatal("missing Connect")
		}
		connect.MinWidth, connect.MinHeight = 150, 40
		root.Children = append(root.Children, ui.Node{ID: "actions", Kind: ui.Row, Children: []ui.Node{connect, s.button("edit.undo", "Undo")}})
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func chooseMountedWireTerminal(t *testing.T, m *mountedControlWindow, id, target string) {
	t.Helper()
	for range 4 {
		if m.snapshot().FocusedID == id {
			break
		}
		m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeTab})
	}
	if m.snapshot().FocusedID != id {
		t.Fatalf("could not focus %s using Tab", id)
	}
	value := func() string {
		state := m.screen.service.Snapshot()
		if id == "wire-from" {
			return state.Drawing.From
		}
		return state.Drawing.To
	}
	for range 16 {
		if value() == target {
			return
		}
		m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeDownArrow})
		if m.screen.errorMessage != "" {
			t.Fatal("selector rejected keyboard input", m.screen.errorMessage)
		}
	}
	t.Fatalf("keyboard selector %s did not choose %s; current=%s", id, target, value())
}

func TestMountedNamedWiringLEDMatchesAgentAndUndo(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	for _, service := range []*workbench.Service{s.service, agent} {
		prepareNamedWiringLED(t, service)
	}
	m := mountedNamedWireControls(t, s)
	initial := s.service.Snapshot()
	if mountedControlElement(t, m, "connect-terminals").Enabled {
		t.Fatal("Connect enabled without endpoints")
	}
	for i, pair := range namedLEDConnections() {
		chooseMountedWireTerminal(t, m, "wire-from", pair.From)
		if mountedControlElement(t, m, "connect-terminals").Enabled {
			t.Fatal("Connect enabled without destination")
		}
		chooseMountedWireTerminal(t, m, "wire-to", pair.To)
		before := s.service.Snapshot()
		clickMountedControl(t, m, "connect-terminals")
		agentAction(t, agent, "wire.terminals.set", workbench.TerminalsParams{From: pair.From, To: pair.To})
		agentAction(t, agent, "wire.connect", pair)
		actual, err := s.service.Wait(m.ctx)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := agent.Wait(m.ctx)
		if err != nil {
			t.Fatal(err)
		}
		if s.errorMessage != "" || len(actual.Wires) != i+1 || actual.CommandRevision != before.CommandRevision+1 || actual.UndoName != "Connect wire" || !reflect.DeepEqual(actual.Components, initial.Components) {
			t.Fatal("wire did not commit exactly once", s.errorMessage)
		}
		assertNamedWiringParity(t, actual, expected)
		if m.snapshot().FocusedID != "wire-from" || mountedControlElement(t, m, "connect-terminals").Enabled {
			t.Fatal("successful connection did not reset endpoints/focus")
		}
	}
	complete := s.service.Snapshot()
	recordNamedWiringObservation(t, "mounted-complete", complete)
	if complete.Result == nil || complete.Result.Stale || math.Abs(complete.Result.Current-.002916796577441) > 5e-12 {
		t.Fatal("LED loop did not produce reference current", complete.Result)
	}
	clickMountedControl(t, m, "edit.undo")
	agentAction(t, agent, "edit.undo", struct{}{})
	undone, err := s.service.Wait(m.ctx)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := agent.Wait(m.ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertNamedWiringParity(t, undone, expected)
	recordNamedWiringObservation(t, "mounted-undo", undone)
	if len(undone.Wires) != 3 || undone.Result != nil && !undone.Result.Stale {
		t.Fatal("Undo kept a current closed-loop reading")
	}
}

func TestMountedNamedWiringDuplicateRefusesWithoutHistoryOrFocusChange(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	pair := namedLEDConnections()[0]
	for _, service := range []*workbench.Service{s.service, agent} {
		prepareNamedWiringLED(t, service)
		agentAction(t, service, "wire.connect", pair)
	}
	m := mountedNamedWireControls(t, s)
	chooseMountedWireTerminal(t, m, "wire-from", pair.From)
	chooseMountedWireTerminal(t, m, "wire-to", pair.To)
	before := s.service.Snapshot()
	focus := s.keyboardFocus
	clickMountedControl(t, m, "connect-terminals")
	if s.errorMessage == "" || !strings.Contains(s.errorMessage, "already connected") || !reflect.DeepEqual(before, s.service.Snapshot()) || focus != s.keyboardFocus {
		t.Fatal("duplicate changed graph/history/focus", s.errorMessage)
	}
	agentAction(t, agent, "wire.terminals.set", workbench.TerminalsParams{From: pair.From, To: pair.To})
	params, _ := json.Marshal(pair)
	revision := agent.Snapshot().CommandRevision
	response := agent.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "duplicate", Command: "wire.connect", Params: params, ExpectedCommandRevision: &revision, Key: "duplicate-existing-wire"}, "agent")
	if response.Error == nil || response.Error.Message != s.errorMessage {
		t.Fatal("UI/agent duplicate refusal differs", s.errorMessage, response.Error)
	}
	assertNamedWiringParity(t, s.service.Snapshot(), agent.Snapshot())
}

func TestMountedNamedWiringHeldConnectRefusesAgentEndpointChange(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareNamedWiringLED(t, s.service)
	pair := namedLEDConnections()[0]
	agentAction(t, s.service, "wire.terminals.set", workbench.TerminalsParams{From: pair.From, To: pair.To})
	m := mountedNamedWireControls(t, s)
	button := mountedControlElement(t, m, "connect-terminals")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	next := namedLEDConnections()[1]
	agentAction(t, s.service, "wire.terminals.set", workbench.TerminalsParams{From: next.From, To: next.To})
	before := s.service.Snapshot()
	focus := s.keyboardFocus
	m.frame()
	if mountedControlElement(t, m, "connect-terminals").Incarnation != button.Incarnation {
		t.Fatal("fixture replaced held Connect")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || focus != s.keyboardFocus || !strings.Contains(s.errorMessage, "expected command revision") {
		t.Fatal("held Connect adopted new endpoints", s.errorMessage)
	}
	clickMountedControl(t, m, "connect-terminals")
	actual := s.service.Snapshot()
	if s.errorMessage != "" || len(actual.Wires) != 1 || actual.Wires[0].From != next.From || actual.Wires[0].To != next.To {
		t.Fatal("fresh Connect did not use current endpoints", s.errorMessage)
	}
}
