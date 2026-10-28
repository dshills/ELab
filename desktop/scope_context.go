package main

import (
	"math"

	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
)

// Immutable context for the last successful preparation, before GPU submission.
// The backend has no present callback; this does not claim OS display completion.
type scopePlotFrame struct {
	revision, generation    uint64
	runID                   uint64
	recordingID, cursor     string
	left, right, start, end float64
}
type scopeGesture struct {
	origin        *scopePlotFrame
	revision      uint64
	position      float64
	positionValid bool
}

// A quick drag may release before native drag classification, or contain
// SlideStart and SlideStop without any SlideMove. Admit both positions through the original gesture's revision guard.
func (c *scopeCanvas) cursorPointer(frame *scopePlotFrame, kind events.Types, button events.Buttons, px float64) {
	if button != events.Left {
		c.gesture = nil
		return
	}
	finish := kind == events.MouseUp || kind == events.SlideStop
	if g := c.gesture; finish && g != nil && g.positionValid && g.position == px {
		c.gesture = nil
		return
	}
	c.cursorGesture(frame, px, kind == events.MouseDown)
	if finish {
		c.gesture = nil
	}
}

func (c *scopeCanvas) applyCursorFrame(frame *scopePlotFrame, px float64, revision uint64) (uint64, bool) {
	if c.screen == nil || frame == nil || c.frame.Load() == nil || frame.right <= frame.left || c.screen.fileBusy || c.screen.pendingCommand != "" || math.IsNaN(px) || math.IsInf(px, 0) {
		return revision, false
	}
	state := c.screen.service.Snapshot()
	if !state.ScopeVisible || state.ScopeView == "spectrum" || sourcePaneHidden(state) || state.ExampleBrowser.Visible {
		return revision, false
	}
	if frame.recordingID == "" && (state.Transient == nil || state.Transient.Stale || state.Transient.RunID != frame.runID) {
		return revision, false
	}
	fraction := math.Max(0, math.Min(1, (px-frame.left)/(frame.right-frame.left)))
	at := frame.start + fraction*(frame.end-frame.start)
	command := "scope.cursor.set"
	var params any = workbench.CursorParams{Cursor: frame.cursor, Time: at}
	if frame.recordingID != "" {
		command = "recording.cursor.set"
		params = workbench.RecordingCursorParams{OperationID: frame.recordingID, Cursor: frame.cursor, Time: at}
	}
	response := c.screen.invokeAtCommandRevision(command, params, revision, frame.generation)
	if response.Error != nil {
		return revision, false
	}
	return response.Result.(workbench.Snapshot).CommandRevision, true
}

func (c *scopeCanvas) cursorGesture(frame *scopePlotFrame, px float64, begin bool) {
	if begin {
		c.gesture = nil
		if frame != nil {
			c.gesture = &scopeGesture{origin: frame, revision: frame.revision}
		}
	}
	g := c.gesture
	if g == nil || frame == nil {
		c.gesture = nil
		return
	}
	// Only this gesture's own successful cursor writes may bridge a repaint.
	// Other actor/setting/selection changes still fail the shared revision guard.
	if frame.revision < g.origin.revision || frame.revision > g.revision || frame.generation != g.origin.generation || frame.recordingID != g.origin.recordingID || frame.runID != g.origin.runID || frame.cursor != g.origin.cursor {
		c.gesture = nil
		return
	}
	next, ok := c.applyCursorFrame(frame, px, g.revision)
	if !ok {
		c.gesture = nil
		return
	}
	g.revision = next
	g.position, g.positionValid = px, true
}

func (c *scopeCanvas) placeCursor(px float64) {
	frame := c.frame.Load()
	if frame != nil {
		c.applyCursorFrame(frame, px, frame.revision)
	}
}
