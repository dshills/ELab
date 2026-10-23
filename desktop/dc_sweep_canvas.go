package main

import (
	"math"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
)

type sweepPlotFrame struct {
	id                    string
	revision, generation  uint64
	left, right, min, max float64
	inputs                []float64 // ascending-axis fractions in stored point order
}
type sweepCanvas struct {
	*cogent.MeshCanvas
	screen     *screen
	mountEpoch uint64
	widget     *core.WidgetBase
	frame      atomic.Pointer[sweepPlotFrame]
	gesture    *sweepGesture
}

type sweepGesture struct {
	origin   *sweepPlotFrame
	revision uint64
}

func (c *sweepCanvas) Mount(parent tree.Node) core.Widget {
	widget := c.MeshCanvas.Mount(parent)
	if widget == nil {
		return nil
	}
	c.mountEpoch++
	c.gesture = nil
	epoch := c.mountEpoch
	w := widget.AsWidget()
	owner, window, scoped := cogent.WindowScope(w)
	if !scoped || owner != c.Owner {
		return nil
	}
	c.widget = w
	w.OnFirst(events.Scroll, func(e events.Event) { e.SetHandled() })
	for _, kind := range []events.Types{events.MouseDown, events.SlideStart, events.SlideMove, events.SlideStop} {
		w.OnFirst(kind, func(e events.Event) {
			frame := c.frame.Load()
			pos := e.Pos().Sub(w.Geom.Pos.Content.ToPoint())
			withWindowOwner(c.Owner, window, func() {
				if c.mountEpoch == epoch {
					c.cursorPointer(frame, kind, e.MouseButton(), float64(pos.X))
				}
			})
			e.SetHandled()
		})
	}
	w.OnFirst(events.KeyChord, func(e events.Event) {
		switch strings.ToLower(string(e.KeyChord())) {
		case "leftarrow", "rightarrow", "uparrow", "downarrow":
			e.SetHandled()
		}
	})
	return widget
}
func (c *sweepCanvas) placeCursor(x float64) {
	frame := c.frame.Load()
	if frame != nil {
		c.applyCursorFrame(frame, x, frame.revision)
	}
}

func (c *sweepCanvas) cursorPointer(frame *sweepPlotFrame, kind events.Types, button events.Buttons, x float64) {
	if button != events.Left {
		c.gesture = nil
		return
	}
	if kind == events.MouseDown {
		c.gesture = nil
		if frame != nil {
			c.gesture = &sweepGesture{origin: frame, revision: frame.revision}
		}
	}
	g := c.gesture
	if g == nil || frame == nil || frame.revision < g.origin.revision || frame.revision > g.revision || frame.generation != g.origin.generation || frame.id != g.origin.id {
		c.gesture = nil
		return
	}
	next, ok := c.applyCursorFrame(frame, x, g.revision)
	if !ok || kind == events.SlideStop {
		c.gesture = nil
		return
	}
	g.revision = next
}

func (c *sweepCanvas) applyCursorFrame(frame *sweepPlotFrame, x float64, revision uint64) (uint64, bool) {
	if frame == nil || c.frame.Load() == nil || c.screen.fileBusy || c.screen.pendingCommand != "" || !sweepVisible(c.screen.service.Snapshot()) || len(frame.inputs) == 0 || frame.right <= frame.left || math.IsNaN(x) || math.IsInf(x, 0) || x < frame.left || x > frame.right {
		return revision, false
	}
	value := (x - frame.left) / (frame.right - frame.left)
	best, distance := 0, math.Inf(1)
	for i, input := range frame.inputs {
		d := math.Abs(input - value)
		if d <= distance {
			best, distance = i, d
		}
	}
	response := c.screen.invokeAtCommandRevision("sweep.cursor.set", workbench.SweepCursorParams{OperationID: frame.id, Index: &best}, revision, frame.generation)
	if response.Error != nil {
		return revision, false
	}
	return response.Result.(workbench.Snapshot).CommandRevision, true
}
