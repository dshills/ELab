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
	inputs                []float64
}
type sweepCanvas struct {
	*cogent.MeshCanvas
	screen     *screen
	mountEpoch uint64
	widget     *core.WidgetBase
	frame      atomic.Pointer[sweepPlotFrame]
}

func (c *sweepCanvas) Mount(parent tree.Node) core.Widget {
	widget := c.MeshCanvas.Mount(parent)
	if widget == nil {
		return nil
	}
	c.mountEpoch++
	epoch := c.mountEpoch
	w := widget.AsWidget()
	c.widget = w
	w.OnFirst(events.Scroll, func(e events.Event) { e.SetHandled() })
	for _, kind := range []events.Types{events.MouseDown, events.SlideMove} {
		w.OnFirst(kind, func(e events.Event) {
			pos := e.Pos().Sub(w.Geom.Pos.Content.ToPoint())
			c.Owner.Do(func() {
				if c.mountEpoch == epoch {
					c.placeCursor(float64(pos.X))
				}
			})
			e.SetHandled()
		})
	}
	w.OnFirst(events.SlideStop, func(e events.Event) { e.SetHandled() })
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
	if frame == nil || len(frame.inputs) == 0 || x < frame.left || x > frame.right {
		return
	}
	value := frame.min + (x-frame.left)/(frame.right-frame.left)*(frame.max-frame.min)
	best, distance := 0, math.Inf(1)
	for i, input := range frame.inputs {
		d := math.Abs(input - value)
		if d <= distance {
			best, distance = i, d
		}
	}
	c.screen.invokeAtCommandRevision("sweep.cursor.set", workbench.SweepCursorParams{OperationID: frame.id, Index: &best}, frame.revision, frame.generation)
}
