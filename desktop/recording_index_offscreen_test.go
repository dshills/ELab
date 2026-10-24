//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedNumericOwnerRecordedIndexRefusesFractionAndCorrects(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	defer cleanup()
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		field, ok := findNode(s.recordingPanel(state), "recorded-index")
		if !ok {
			t.Fatal("recorded index control missing")
		}
		field.MinWidth, field.MinHeight = 360, 40
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: field.ID, Sequence: 1}, Children: []ui.Node{field}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	before := s.service.Snapshot()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, edit := range []struct {
		r rune
		c key.Codes
	}{{'1', key.Code1}, {'.', key.CodeFullStop}, {'5', key.Code5}} {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: edit.r, Code: edit.c})
	}
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("private numeric entry changed the public viewer")
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" || s.requestHelp == nil {
		t.Fatal("fractional index silently sought a different sample or lost refusal feedback")
	}
	// Explicitly refocus: correction must work, not merely appear protected
	// because Enter moved focus away from the rejected field.
	clickMountedControl(t, m, "recorded-index")
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	m.input(cogent.ManualInput{Type: events.KeyChord, Rune: '3', Code: key.Code3})
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	after := s.service.Snapshot()
	if after.Recording.Index != 3 || after.Recording.Time != .003 || after.CommandRevision != before.CommandRevision+1 || s.errorMessage != "" {
		t.Fatal("fresh integer correction did not seek exactly once", after.Recording.Index, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}
