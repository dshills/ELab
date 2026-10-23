package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
)

// A prepared mesh is the input target until the next preparation. Live samples
// do not change its command revision; another actor/tool/document does.
func (c *commandCanvas) pointerToken() string {
	return fmt.Sprintf("%d:%d", c.preparedGeneration, c.preparedCommandRevision)
}

func (c *commandCanvas) routePreparedPointer(phase int, p, delta workbench.Point) {
	// Click-only placement/wiring/inspection has already completed on down.
	// A subsequent auto-fit can advance the revision before release; there is
	// no edit to admit on that release, and it must not report a stale gesture.
	if phase == 2 && !c.dragging {
		c.pointerGesture = nil
		c.dragID = ""
		return
	}
	context := c.pointerContext.Load()
	if phase == 1 || phase == 2 {
		context = c.pointerGesture
	}
	next := c.withPreparedPointer(context, func() { c.routePointer(phase, p, delta) })
	if phase != 3 {
		c.pointerGesture = next
		if phase == 2 {
			c.pointerGesture = nil
		}
	}
}

func (c *commandCanvas) withPreparedPointer(context *actionContext, apply func()) *actionContext {
	state := c.screen.service.Snapshot()
	if context == nil || state.Closed || c.screen.fileBusy || c.screen.pendingCommand != "" || sourcePaneHidden(state) {
		c.dragID, c.dragging, c.dragGeneration = "", false, 0
		return nil
	}
	if context.generation != state.Document.Generation || context.revision != state.CommandRevision {
		c.dragID, c.dragging = "", false
		c.dragGeneration = 0
		c.screen.errorMessage = "The schematic changed before this input could be applied. Try again."
		c.screen.app.Invalidate("main")
		return nil
	}
	previous, previousAccepted := c.screen.pointerContext, c.screen.pointerAccepted
	c.screen.pointerContext = context
	c.screen.pointerAccepted = nil
	defer func() { c.screen.pointerContext, c.screen.pointerAccepted = previous, previousAccepted }()
	// The same public command carries this guard through dispatch, including
	// an agent change after the snapshot check above.
	serial := c.screen.serial
	apply()
	if c.screen.serial != serial && c.screen.pointerAccepted == nil {
		c.dragID, c.dragging = "", false
		c.dragGeneration = 0
		return nil
	}
	if c.screen.pointerAccepted != nil {
		return c.screen.pointerAccepted
	}
	return context
}
