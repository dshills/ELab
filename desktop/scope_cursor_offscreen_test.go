//go:build offscreen

package main

import (
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

func mountedCursorControls(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		controls := s.cursorControls(state)
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "cursor-time-a", Sequence: 1}}
		if state.Transient == nil || state.Transient.Stale {
			root.Focus.ID = "cursor-select-a"
		}
		for i := 0; i < 6; i += 3 {
			row := ui.Node{ID: controls.Children[i].ID + "-row", Kind: ui.Row}
			for _, n := range controls.Children[i : i+3] {
				if n.Kind == ui.TextField {
					n.MinWidth, n.MinHeight = 190, 40
				}
				row.Children = append(row.Children, n)
			}
			root.Children = append(root.Children, row)
		}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	s.settleField = func(id string, done func(error)) error {
		return cogent.SettleTextField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func clickMountedControl(t *testing.T, m *mountedControlWindow, id string) {
	t.Helper()
	element := mountedControlElement(t, m, id)
	at := element.Bounds.Min.Add(element.Bounds.Size().Div(2))
	for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
		m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
	}
}

func typeMountedCursor(t *testing.T, m *mountedControlWindow, id, text string) {
	t.Helper()
	clickMountedControl(t, m, id)
	if m.snapshot().FocusedID != id {
		t.Fatal("cursor field did not receive pointer focus", id)
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
	for _, r := range text {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	if field := mountedControlElement(t, m, id); field.Value != text {
		t.Fatal("native field did not retain typed quantity", field.Value, text)
	}
}

func assertCursorMeasurementPreservesPhysics(t *testing.T, before, after workbench.Snapshot) {
	t.Helper()
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Document != after.Document ||
		!reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) ||
		!reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) ||
		!reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.RedoEntry, after.RedoEntry) {
		t.Fatal("cursor measurement changed physics, document or circuit history")
	}
}

func TestMountedScopeCursorEnterSelectClearMatchesPublicAgent(t *testing.T) {
	for _, family := range []struct {
		name, a, b string
		steps      int
		step, dt   float64
	}{{"rc", "4 ms", "16 ms", 1, .02, .012}, {"flasher", "100 ms", "260 ms", 70, .005, .16}} {
		t.Run(family.name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agent := workbench.New()
			t.Cleanup(agent.Close)
			for _, service := range []*workbench.Service{s.service, agent} {
				agentAction(t, service, "example."+family.name, workbench.ReplaceParams{})
				agentAction(t, service, "simulation.pause", struct{}{})
				agentAction(t, service, "simulation.restart", struct{}{})
				for range family.steps {
					agentAction(t, service, "simulation.step", workbench.StepParams{Duration: family.step})
				}
			}
			m := mountedCursorControls(t, s)
			before := s.service.Snapshot()
			for _, edit := range []struct{ cursor, text string }{{"a", family.a}, {"b", family.b}} {
				if edit.cursor == "b" {
					clickMountedControl(t, m, "cursor-select-b")
					agentAction(t, agent, "scope.cursor.select", workbench.CursorSelectParams{Cursor: "b"})
				}
				start := s.service.Snapshot()
				typeMountedCursor(t, m, "cursor-time-"+edit.cursor, edit.text)
				if !reflect.DeepEqual(start, s.service.Snapshot()) {
					t.Fatal("private cursor typing changed public state")
				}
				m.frame() // ordinary repaint must preserve unfinished quantity text
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				agentAction(t, agent, "scope.cursor.set", map[string]any{"cursor": edit.cursor, "time_s": edit.text})
				if state := s.service.Snapshot(); state.CommandRevision != start.CommandRevision+1 || !reflect.DeepEqual(state.Scope, agent.Snapshot().Scope) {
					t.Fatal("native time commit differs from public agent", s.errorMessage, state.Scope.A, state.Scope.B)
				}
				m.frame()
			}
			state := s.service.Snapshot()
			if state.Scope.DeltaSeconds == nil || math.Abs(*state.Scope.DeltaSeconds-family.dt) > 1e-12 || state.Scope.Frequency == nil || math.Abs(*state.Scope.Frequency-1/family.dt) > 1e-9 {
				t.Fatal("cursor timing differs from known interval", family.dt)
			}
			clickMountedControl(t, m, "cursor-clear-a")
			agentAction(t, agent, "scope.cursor.clear", workbench.CursorSelectParams{Cursor: "a"})
			after := s.service.Snapshot()
			if after.Scope.A != nil || after.Scope.DeltaSeconds != nil || !reflect.DeepEqual(after.Scope, agent.Snapshot().Scope) {
				t.Fatal("native clear differs from agent or kept obsolete timing", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestMountedScopeCursorFocusLossThenClearUsesOneClick(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "scope.cursor.set", workbench.CursorParams{Cursor: "a", Time: .004})
	m := mountedCursorControls(t, s)
	before := s.service.Snapshot()
	typeMountedCursor(t, m, "cursor-time-a", "8 ms")
	clickMountedControl(t, m, "cursor-clear-a")
	after := s.service.Snapshot()
	if after.Scope.A != nil || after.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" {
		t.Fatal("one click did not settle and clear the cursor once", after.Scope.A, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestMountedScopeCursorHeldSelectionRefusesAgentAfterOwnCommit(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	t.Cleanup(cleanup)
	m := mountedCursorControls(t, s)
	typeMountedCursor(t, m, "cursor-time-a", "4 ms")
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	m.frame()
	button := mountedControlElement(t, m, "cursor-select-b")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "scope.cursor.set", workbench.CursorParams{Cursor: "b", Time: .012})
	before := s.service.Snapshot()
	m.frame()
	if mountedControlElement(t, m, "cursor-select-b").Incarnation != button.Incarnation {
		t.Fatal("fixture did not retain the held selection button")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || !strings.Contains(s.errorMessage, "expected command revision") {
		t.Fatal("held selection borrowed intervening agent context", s.errorMessage)
	}
	clickMountedControl(t, m, "cursor-select-b")
	after := s.service.Snapshot()
	if after.Scope.ActiveCursor != "b" || after.Scope.B == nil || after.Scope.B.Requested != .012 || after.CommandRevision != before.CommandRevision+1 || s.errorMessage != "" {
		t.Fatal("fresh selection failed or overwrote agent time", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestMountedScopeCursorFocusLossThenSelectUsesOneClick(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	t.Cleanup(cleanup)
	m := mountedCursorControls(t, s)
	before := s.service.Snapshot()
	typeMountedCursor(t, m, "cursor-time-a", "4 ms")
	clickMountedControl(t, m, "cursor-select-b")
	after := s.service.Snapshot()
	if after.Scope.A == nil || after.Scope.A.Requested != .004 || after.Scope.ActiveCursor != "b" || after.CommandRevision != before.CommandRevision+2 || s.errorMessage != "" {
		t.Fatalf("one click failed to commit A and select B: A=%+v active=%s revision=%d→%d error=%s", after.Scope.A, after.Scope.ActiveCursor, before.CommandRevision, after.CommandRevision, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestMountedScopeCursorStaleSelectPreservesPhysics(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "scope.cursor.set", workbench.CursorParams{Cursor: "a", Time: .004})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
	before := s.service.Snapshot()
	if before.Transient == nil || !before.Transient.Stale || before.Scope.A != nil {
		t.Fatal("graph edit did not stale measurements and clear cursor requests")
	}
	m := mountedCursorControls(t, s)
	clickMountedControl(t, m, "cursor-select-b")
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.Scope.A != nil || after.Scope.ActiveCursor != "b" || after.CommandRevision != before.CommandRevision+1 {
		t.Fatal("disabled time fields blocked stale Select", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}
