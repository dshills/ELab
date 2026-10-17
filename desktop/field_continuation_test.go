package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRepeatedCoordinateEventsContinueWithoutRendering(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "resistor"})
	var x, y ui.Node
	s.app.Do(func() {
		root := s.view()
		x, _ = findNode(root, "position-x")
		y, _ = findNode(root, "position-y")
		// Native key repeats can deliver several events before the next frame.
		for value := float32(-190); value <= 0; value += 10 {
			x.Change(ui.ChangeEvent{Number: value})
			if s.errorMessage != "" {
				t.Fatalf("repeat at %g: %s", value, s.errorMessage)
			}
		}
	})
	state := s.service.Snapshot()
	if state.Drawing.Position != (workbench.Point{X: 0, Y: 0}) || len(state.Components) != 0 || s.fieldContext != nil || s.fieldAccepted != nil {
		t.Fatal("key repeats lost coordinates, placed a part or leaked field context")
	}
	// A different old field still contains X from the old frame. It must not
	// overwrite the accepted X; only the originating field may continue.
	s.app.Do(func() { y.Change(ui.ChangeEvent{Number: 100}) })
	if s.errorMessage == "" || !reflect.DeepEqual(state, s.service.Snapshot()) {
		t.Fatal("old opposite-axis callback overwrote the updated coordinate")
	}
	uiAction(t, s, "position-y", &ui.ChangeEvent{Number: 100})
	if s.service.Snapshot().Drawing.Position != (workbench.Point{X: 0, Y: 100}) {
		t.Fatal("fresh opposite-axis callback lost accepted X")
	}
}

func TestContinuedCoordinateFieldStillRefusesInterveningActions(t *testing.T) {
	for _, tc := range []struct {
		name        string
		replacement bool
	}{{"agent edit", false}, {"document replacement", true}} {
		t.Run(tc.name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "part-list", &ui.ChangeEvent{SelectedID: "resistor"})
			var x ui.Node
			s.app.Do(func() {
				x, _ = findNode(s.view(), "position-x")
				x.Change(ui.ChangeEvent{Number: -190})
			})
			if tc.replacement {
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			} else {
				agentAction(t, s.service, "drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: 100, Y: 200}})
			}
			before := s.service.Snapshot()
			s.app.Do(func() { x.Change(ui.ChangeEvent{Number: -180}) })
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.fieldContext != nil || s.fieldAccepted != nil {
				t.Fatal("continuing field overwrote intervening state or leaked context")
			}
		})
	}
}

func TestFieldContinuationUsesAcceptedResponseBeforeLaterAgentEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	var field ui.Node
	s.app.Do(func() {
		state := s.service.Snapshot()
		first := true
		field = ui.Node{ID: "delayed-return", Kind: ui.TextField, Change: func(e ui.ChangeEvent) {
			s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text})
			if first {
				first = false
				// The accepted command response precedes an unrelated agent edit,
				// even though the callback hasn't returned to its wrapper yet.
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "agent draft"})
			}
		}}
		s.bindFieldContext(&field, actionContext{state.CommandRevision, state.Document.Generation})
		field.Change(ui.ChangeEvent{Text: "first"})
	})
	before := s.service.Snapshot()
	s.app.Do(func() { field.Change(ui.ChangeEvent{Text: "second"}) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || before.Drawing.Value != "agent draft" || s.fieldContext != nil || s.fieldAccepted != nil {
		t.Fatal("continuation borrowed a later agent revision or leaked field state")
	}
}
