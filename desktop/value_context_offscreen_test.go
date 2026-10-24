//go:build offscreen

package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedValueDraftDoesNotLeakAcrossRetainedDocumentReplacement(t *testing.T) {
	for _, scenario := range []string{"unchanged-repaint", "document-replacement", "selected-object"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			if scenario == "selected-object" {
				agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "R2", Kind: "resistor", Value: "1k"})
			}
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
			path := filepath.Join(t.TempDir(), "retained-value.elab")
			agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
			view := func() ui.Node {
				state := s.service.Snapshot()
				var field ui.Node
				for _, p := range state.Components {
					if p.ID == state.Selected {
						for _, node := range s.componentInspector(state, workbench.Graph{Components: state.Components, Wires: state.Wires}, p) {
							if node.ID == "value-entry" {
								field = node
							}
						}
					}
				}
				field.MinWidth, field.MinHeight = 360, 40
				root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "value-entry", Sequence: 1}, Children: []ui.Node{field}}
				s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
				return root
			}
			m := newMountedControlWindow(t, s, view)
			field := func() cogent.ManualElement {
				t.Helper()
				for _, element := range m.snapshot().Elements {
					if element.ID == "value-entry" {
						return element
					}
				}
				t.Fatal("value field missing")
				return cogent.ManualElement{}
			}
			before := s.service.Snapshot()
			original := field()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, ch := range []struct {
				rune rune
				code key.Codes
			}{{'2', key.Code2}, {'.', key.CodeFullStop}, {'2', key.Code2}, {'k', key.CodeK}} {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: ch.rune, Code: ch.code})
			}
			if field().Value != "2.2k" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatalf("fixture must hold private uncommitted text: field=%+v", field())
			}
			// Complete both public commands before one frame. The same field ID and
			// controlled text return, so the native renderer can retain its binding.
			if scenario == "document-replacement" {
				agentAction(t, s.service, "project.load", workbench.LoadParams{Path: path, DiscardChanges: true})
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
			} else if scenario == "selected-object" {
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R2"})
			}
			replacement := s.service.Snapshot()
			if scenario == "document-replacement" && replacement.Document.Generation == before.Document.Generation {
				t.Fatal("fixture did not replace document")
			}
			m.frame()
			current := field()
			t.Logf("native incarnation %d→%d; text after redraw %v; controlled %q", original.Incarnation, current.Incarnation, current.Value, replacement.Drawing.Value)
			if scenario == "unchanged-repaint" {
				if current.Incarnation != original.Incarnation || current.Value != "2.2k" {
					t.Fatal("ordinary repaint discarded an unfinished draft")
				}
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				accepted := s.service.Snapshot()
				if accepted.Drawing.Value != "2.2k" || accepted.CommandRevision != before.CommandRevision+1 || !reflect.DeepEqual(accepted.Components, before.Components) {
					t.Fatal("ordinary retained field did not commit exactly its draft")
				}
				return
			}
			if current.Incarnation == original.Incarnation || current.Value != replacement.Drawing.Value {
				t.Fatal("logical owner change retained private field state")
			}
			// Focus the new widget explicitly: rejection must not be a side effect
			// of Enter falling on no focused control after remounting.
			for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
				m.input(cogent.ManualInput{Type: kind, Button: events.Left, Target: "value-entry", TargetIncarnation: current.Incarnation})
			}
			if m.snapshot().FocusedID != "value-entry" {
				t.Fatal("fresh field did not receive actual pointer focus")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			if !reflect.DeepEqual(replacement, s.service.Snapshot()) {
				t.Fatalf("old private text acted on new owner: draft %q→%q", replacement.Drawing.Value, s.service.Snapshot().Drawing.Value)
			}
			// Enter completes editing and can move focus. Start the fresh edit
			// through the actual pointer router rather than relying on old focus.
			for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
				m.input(cogent.ManualInput{Type: kind, Button: events.Left, Target: "value-entry", TargetIncarnation: current.Incarnation})
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, ch := range []struct {
				rune rune
				code key.Codes
			}{{'3', key.Code3}, {'.', key.CodeFullStop}, {'3', key.Code3}, {'k', key.CodeK}} {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: ch.rune, Code: ch.code})
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			accepted := s.service.Snapshot()
			if accepted.Drawing.Value != "3.3k" || accepted.CommandRevision != replacement.CommandRevision+1 || !reflect.DeepEqual(accepted.Components, replacement.Components) || !reflect.DeepEqual(accepted.Wires, replacement.Wires) {
				t.Fatalf("fresh field failed: draft=%q revision=%d want=%d field=%+v focus=%s error=%s", accepted.Drawing.Value, accepted.CommandRevision, replacement.CommandRevision+1, field(), m.snapshot().FocusedID, s.errorMessage)
			}
		})
	}
}
