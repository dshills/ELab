package main

import (
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

type schematicKeySequence struct {
	origin, accepted actionContext
}

func schematicKeySupported(chord string) bool {
	if navigationDirection(chord) != "" {
		return true
	}
	switch chord {
	case "leftarrow", "rightarrow", "uparrow", "downarrow", "escape", "delete", "backspace":
		return true
	}
	return false
}

func (c *commandCanvas) bindSchematicKeys(w *core.WidgetBase, window app.WindowRef) {
	w.OnFirst(events.KeyChord, func(e events.Event) {
		chord := strings.ToLower(string(e.KeyChord()))
		if !schematicKeySupported(chord) {
			return
		}
		// Immutable mesh context, captured before acquiring the application owner.
		captured := c.pointerContext.Load()
		withWindowOwner(c.Owner, window, func() { c.routeSchematicKey(chord, captured) })
		e.SetHandled()
	})
}

func (c *commandCanvas) routeSchematicKey(chord string, captured *actionContext) {
	if !schematicKeySupported(chord) {
		return
	}
	state := c.screen.service.Snapshot()
	if captured == nil || state.Closed || c.screen.fileBusy || c.screen.pendingCommand != "" || sourcePaneHidden(state) || c.dragging && chord != "escape" {
		c.keySequence = nil
		return
	}
	expected := *captured
	sequence := c.keySequence
	if sequence != nil && captured.generation == sequence.origin.generation && captured.revision >= sequence.origin.revision && captured.revision <= sequence.accepted.revision {
		expected = sequence.accepted
	} else {
		sequence = nil
	}
	if expected.generation != state.Document.Generation || expected.revision != state.CommandRevision {
		c.keySequence = nil
		c.screen.errorMessage = "The schematic changed before this key could be applied. Try again."
		c.screen.app.Invalidate("main")
		return
	}
	command, params := schematicKeyCommand(state, chord)
	if command == "" {
		return
	}
	response := c.screen.invokeAtCommandRevision(command, params, expected.revision, expected.generation)
	accepted, ok := response.Result.(workbench.Snapshot)
	if response.Error != nil || !ok {
		c.keySequence = nil
		return
	}
	if sequence == nil {
		sequence = &schematicKeySequence{origin: *captured}
	}
	sequence.accepted = actionContext{accepted.CommandRevision, accepted.Document.Generation}
	c.keySequence = sequence
	if chord == "escape" {
		c.dragID, c.dragging, c.dragGeneration, c.pointerGesture = "", false, 0, nil
	}
}

// Plan from the exact locally admitted snapshot. Dispatch retains its original
// preconditions, so an agent changing selection/position after planning refuses.
func schematicKeyCommand(state workbench.Snapshot, chord string) (string, any) {
	if direction := navigationDirection(chord); direction != "" {
		return "selection.navigate", workbench.NavigateParams{Direction: direction, Center: true}
	}
	switch chord {
	case "escape":
		return "drawing.cancel", struct{}{}
	case "delete", "backspace":
		if state.Selected != "" {
			return "object.delete", workbench.IDParams{ID: state.Selected}
		}
		return "", nil
	}
	dx, dy := 0., 0.
	switch chord {
	case "leftarrow":
		dx = -10
	case "rightarrow":
		dx = 10
	case "uparrow":
		dy = -10
	case "downarrow":
		dy = 10
	default:
		return "", nil
	}
	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	if part, ok := graph.Component(state.Selected); ok && state.Drawing.Tool == "select" {
		return "component.move", workbench.MoveParams{ID: part.ID, Position: workbench.Point{X: part.Position.X + dx, Y: part.Position.Y + dy}, PreviousPosition: &part.Position}
	}
	return "view.pan", map[string]float64{"dx": dx / state.View.Zoom, "dy": dy / state.View.Zoom}
}
