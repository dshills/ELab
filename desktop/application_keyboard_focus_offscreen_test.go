//go:build offscreen

package main

import (
	"image/color"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedApplicationKeyboard(t *testing.T, fieldID string) (*screen, *mountedControlWindow, workbench.Point) {
	t.Helper()
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	part, _ := (workbench.Graph{Components: s.service.Snapshot().Components}).Component("R1")
	moved := workbench.Point{X: part.Position.X + 10, Y: part.Position.Y}
	agentAction(t, s.service, "component.move", workbench.MoveParams{ID: "R1", Position: moved})
	theme := ui.Theme{Default: ui.Style{Foreground: ui.Some(color.NRGBA{R: 24, G: 24, B: 24, A: 255}), Background: ui.Some(color.NRGBA{R: 248, G: 248, B: 248, A: 255}), Typography: ui.Some(ui.Typography{Family: "Noto Sans", Size: 16})}}
	m := newMountedControlWindowSize(t, s, func() ui.Node {
		full := s.view()
		field, ok := findNode(full, fieldID)
		if !ok || field.Disabled {
			t.Fatal("missing enabled production text field", fieldID)
		}
		field.MinWidth, field.MinHeight = 120, 40
		// A supported non-editor control tests the real scene's application
		// fallback without pretending to qualify a custom schematic extension.
		focus := ui.Node{ID: "document-focus", Kind: ui.Button, Text: "Document focus", Activate: func() {}}
		return ui.Node{ID: "root", Kind: ui.Row, Focus: ui.FocusRequest{ID: fieldID, Sequence: 1}, Children: []ui.Node{field, focus}}
	}, 720, 320, theme, func(config *cogent.ManualConfig) {
		config.Commands = s.commands
	})
	if m.snapshot().FocusedID != fieldID {
		t.Fatal("production text field did not receive actual focus", m.snapshot().FocusedID)
	}
	if keymap.Of(key.NewChord('a', key.CodeA, 1<<key.Meta)) != keymap.SelectAll || keymap.Of(key.NewChord('z', key.CodeZ, 1<<key.Meta)) != keymap.Undo {
		t.Fatal("fixture is not using Command editing mappings")
	}
	return s, m, part.Position
}

func TestMountedValueDraftApplicationKeyboardTextFieldsOwnEditingKeys(t *testing.T) {
	for _, fieldID := range []string{"part-search", "value-entry"} {
		for _, code := range []key.Codes{key.CodeBackspace, key.CodeDelete, key.CodeLeftArrow, key.CodeRightArrow, key.CodeEscape} {
			t.Run(fieldID+"/"+code.String(), func(t *testing.T) {
				s, m, _ := mountedApplicationKeyboard(t, fieldID)
				typeMountedCursor(t, m, fieldID, "123")
				before := s.service.Snapshot()
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: code})
				if !reflect.DeepEqual(before, s.service.Snapshot()) || m.snapshot().FocusedID != fieldID {
					t.Fatal("text editing key changed the circuit or leaked focus from the editor", code)
				}
			})
		}
	}
}

func TestMountedValueDraftApplicationKeyboardUndoRedoStayWithFocusedEditor(t *testing.T) {
	for _, fieldID := range []string{"part-search", "value-entry"} {
		t.Run(fieldID, func(t *testing.T) {
			s, m, original := mountedApplicationKeyboard(t, fieldID)
			before := s.service.Snapshot()
			// Even an empty native text undo stack owns Undo/Redo. It must not
			// borrow the application's already-available circuit history.
			initialText := mountedControlElement(t, m, fieldID).Value
			for _, modifiers := range []key.Modifiers{key.Modifiers(1 << key.Meta), key.Modifiers(1<<key.Meta | 1<<key.Shift)} {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: 'z', Code: key.CodeZ, Modifiers: modifiers})
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("empty text history fell through to circuit Undo/Redo")
				}
				if text := mountedControlElement(t, m, fieldID).Value; text != initialText {
					t.Fatalf("empty text history changed the current field: %q → %q", initialText, text)
				}
			}
			typeMountedCursor(t, m, fieldID, "123")
			m.input(cogent.ManualInput{Type: events.KeyChord, Rune: 'z', Code: key.CodeZ, Modifiers: 1 << key.Meta})
			if !reflect.DeepEqual(before, s.service.Snapshot()) || mountedControlElement(t, m, fieldID).Value == "123" {
				t.Fatal("native text Undo did not consume the edit locally")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Rune: 'z', Code: key.CodeZ, Modifiers: 1<<key.Meta | 1<<key.Shift})
			if !reflect.DeepEqual(before, s.service.Snapshot()) || mountedControlElement(t, m, fieldID).Value != "123" {
				t.Fatal("native text Redo did not restore the private edit locally")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeEscape})
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeTab})
			if m.snapshot().FocusedID != "document-focus" || s.service.Snapshot().Selected != "R1" {
				t.Fatal("Tab did not focus the supported document control with the same selection", m.snapshot().FocusedID)
			}
			before = s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.KeyChord, Rune: 'z', Code: key.CodeZ, Modifiers: 1 << key.Meta})
			after := s.service.Snapshot()
			part, _ := (workbench.Graph{Components: after.Components}).Component("R1")
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || part.Position != original {
				t.Fatal("positive control: non-editor focus did not reach application Undo", s.errorMessage, part.Position)
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Rune: 'z', Code: key.CodeZ, Modifiers: 1<<key.Meta | 1<<key.Shift})
			replayed := s.service.Snapshot()
			part, _ = (workbench.Graph{Components: replayed.Components}).Component("R1")
			if s.errorMessage != "" || replayed.CommandRevision != after.CommandRevision+1 || part.Position.X != original.X+10 || part.Position.Y != original.Y {
				t.Fatal("positive control: non-editor focus did not reach application Redo", s.errorMessage, part.Position)
			}
		})
	}
}
