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

func mountedValuePreset(t *testing.T, s *screen, preset string) *mountedControlWindow {
	t.Helper()
	m := newMountedControlWindow(t, s, func() ui.Node {
		assembled := s.view()
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "value-entry", Sequence: 1}}
		for _, id := range []string{"value-entry", preset} {
			node, ok := findNode(assembled, id)
			if !ok {
				t.Fatalf("missing production value control %s", id)
			}
			root.Children = append(root.Children, node)
		}
		return root
	})
	s.settleField = func(id string, done func(error)) error {
		return cogent.SettleTextField(m.ctx, s.app, "main", id, done)
	}
	return m
}

func TestMountedButtonOwnerValuePresetsSettleOwnDraft(t *testing.T) {
	for _, fixture := range []struct {
		preset, example, part, initial, draft, target string
	}{
		{"r-1k", "example.reset", "R1", "3k", "2.2k", "1k"},
		{"r-2k", "example.reset", "R1", "3k", "2.2k", "2k"},
		{"c-1u", "example.rc", "C1", "47uF", "3.3uF", "1uF"},
		{"c-100u", "example.rc", "C1", "47uF", "3.3uF", "100uF"},
	} {
		t.Run(fixture.preset, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, fixture.example, nil)
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: fixture.part})
			agentAction(t, s.service, "component.set", workbench.ValueParams{ID: fixture.part, Value: fixture.initial})
			reference := workbench.New()
			t.Cleanup(reference.Close)
			agentAction(t, reference, fixture.example, workbench.ReplaceParams{})
			agentAction(t, reference, "simulation.pause", struct{}{})
			agentAction(t, reference, "component.select", workbench.IDParams{ID: fixture.part})
			agentAction(t, reference, "component.set", workbench.ValueParams{ID: fixture.part, Value: fixture.initial})
			agentAction(t, reference, "component.set", workbench.ValueParams{ID: fixture.part, Value: fixture.target})
			want := reference.Snapshot().Components
			m := mountedValuePreset(t, s, fixture.preset)
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, r := range fixture.draft {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
			}
			if field := mountedControlElement(t, m, "value-entry"); field.Value != fixture.draft || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("fixture must retain private text without changing the circuit", field.Value)
			}
			clickMountedControl(t, m, fixture.preset)
			after := s.service.Snapshot()
			if !reflect.DeepEqual(after.Components, want) {
				t.Fatalf("one preset click differs from the public agent: revision=%d→%d error=%s", before.CommandRevision, after.CommandRevision, s.errorMessage)
			}
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || !reflect.DeepEqual(before.Wires, after.Wires) {
				t.Fatal("preset must settle its own draft and set the component once", s.errorMessage)
			}
			if after.UndoEntry == nil || before.UndoEntry != nil && after.UndoEntry.CommandRevision == before.UndoEntry.CommandRevision {
				t.Fatal("preset did not create a circuit history entry")
			}
			uiAction(t, s, "edit.undo", nil)
			if !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) {
				t.Fatal("one shared Undo did not restore the pre-preset circuit")
			}
		})
	}
}

func TestMountedButtonOwnerValuePresetRefusesAgentAfterDraft(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareValuePreset(t, s)
	m := mountedValuePreset(t, s, "r-2k")
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, r := range "9k" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	button := mountedControlElement(t, m, "r-2k")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "4k"})
	before := s.service.Snapshot()
	m.frame()
	if mountedControlElement(t, m, "r-2k").Incarnation != button.Incarnation {
		t.Fatal("fixture must retain the held preset")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("held preset changed an agent's later edit", s.errorMessage)
	}
	clickMountedControl(t, m, "r-2k")
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 {
		t.Fatal("fresh preset did not apply once", s.errorMessage)
	}
	for _, part := range after.Components {
		if part.ID == "R1" && part.Value != 2000 {
			t.Fatal("fresh preset did not set the requested value", part.Value)
		}
	}
}
