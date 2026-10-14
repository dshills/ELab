package main

import (
	"math"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/drivers/cogent"
)

type commandCanvas struct {
	*cogent.MeshCanvas
	screen                *screen
	activeTool, dragID    string
	dragOrigin, dragDelta workbench.Point
	dragging              bool
	dragGeneration        uint64
	widget                *core.WidgetBase
	pointer               *pointerBridge
}

func (c *commandCanvas) world(e events.Event, w *core.WidgetBase) workbench.Point {
	p := e.Pos().Sub(w.Geom.Pos.Content.ToPoint())
	world := c.View.World(canvas.Point{X: float64(p.X), Y: float64(p.Y)})
	return workbench.Point{X: world.X, Y: world.Y}
}
func (c *commandCanvas) Mount(parent tree.Node) core.Widget {
	widget := c.MeshCanvas.Mount(parent)
	if widget == nil {
		return nil
	}
	w := widget.AsWidget()
	c.widget = w
	w.OnFirst(events.Scroll, func(e events.Event) {
		if scroll, ok := e.(*events.MouseScroll); ok {
			c.Owner.Do(func() {
				anchor := c.world(e, w)
				factor := math.Max(.25, math.Min(4, math.Exp(-float64(scroll.Delta.Y)*.003)))
				c.screen.invoke("view.zoom", map[string]any{"factor": factor, "anchor": anchor})
			})
			e.SetHandled()
		}
	})
	w.OnFirst(events.MouseDown, func(e events.Event) {
		c.Owner.Do(func() { c.routePointer(0, c.world(e, w), workbench.Point{}) })
		e.SetHandled()
	})
	w.OnFirst(events.SlideMove, func(e events.Event) {
		c.Owner.Do(func() {
			delta := e.PrevDelta()
			c.routePointer(1, c.world(e, w), workbench.Point{X: float64(delta.X) / c.View.Zoom, Y: float64(delta.Y) / c.View.Zoom})
		})
		e.SetHandled()
	})
	w.OnFirst(events.SlideStop, func(e events.Event) {
		c.Owner.Do(func() { c.routePointer(2, c.world(e, w), workbench.Point{}) })
		e.SetHandled()
	})
	w.OnFirst(events.KeyChord, func(e events.Event) {
		chord := strings.ToLower(string(e.KeyChord()))
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
		case "escape":
			c.Owner.Do(func() { c.dragID = ""; c.dragging = false; c.screen.invoke("drawing.cancel", struct{}{}) })
			e.SetHandled()
			return
		case "delete", "backspace":
			c.Owner.Do(func() {
				if id := c.screen.service.Snapshot().Selected; id != "" {
					c.screen.invoke("object.delete", workbench.IDParams{ID: id})
				}
			})
			e.SetHandled()
			return
		default:
			return
		}
		c.Owner.Do(func() {
			state := c.screen.service.Snapshot()
			graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
			if part, ok := graph.Component(state.Selected); ok && state.Drawing.Tool == "select" {
				c.screen.invoke("component.move", workbench.MoveParams{ID: part.ID, Position: workbench.Point{X: part.Position.X + dx, Y: part.Position.Y + dy}})
			} else {
				c.screen.invoke("view.pan", map[string]float64{"dx": dx / c.View.Zoom, "dy": dy / c.View.Zoom})
			}
		})
		e.SetHandled()
	})
	return widget
}
func (c *commandCanvas) Sync(w core.Widget) error { return c.MeshCanvas.Sync(w) }
func hitComponent(state workbench.Snapshot, p workbench.Point) string {
	for i := len(state.Components) - 1; i >= 0; i-- {
		part := state.Components[i]
		local := workbench.RotateLocal(workbench.Point{X: p.X - part.Position.X, Y: p.Y - part.Position.Y}, (360-part.Rotation)%360)
		width, height := 42., 42.
		if part.Kind == "potentiometer" {
			width, height = 65, 50
		}
		if part.Kind == "resistor" || part.Kind == "switch" {
			width, height = 58, 24
		}
		if part.Kind == "junction" {
			width, height = 8, 8
		}
		if part.Kind == "net-label" {
			if local.X >= -8 && local.X <= 22+float64(len(part.Label))*8 && math.Abs(local.Y) <= 14 {
				return part.ID
			}
			continue
		}
		if part.Kind == "ground" {
			width, height = 28, 28
		}
		if math.Abs(local.X) <= width && math.Abs(local.Y) <= height {
			return part.ID
		}
	}
	return ""
}
func terminalHit(state workbench.Snapshot, p workbench.Point, tolerance float64) string {
	for _, part := range state.Components {
		for _, terminal := range part.Terminals {
			position, _ := workbench.TerminalPosition(part, terminal)
			if math.Hypot(position.X-p.X, position.Y-p.Y) <= tolerance {
				return part.ID + "." + terminal
			}
		}
	}
	return ""
}
func wirePath(graph workbench.Graph, w workbench.Wire) []canvas.Point {
	return routedWirePath(graph, w)
}

func hitWire(state workbench.Snapshot, p workbench.Point, tolerance float64) string {
	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	for i := len(state.Wires) - 1; i >= 0; i-- {
		points := wirePath(graph, state.Wires[i])
		for j := 1; j < len(points); j++ {
			a, b := points[j-1], points[j]
			dx, dy := b.X-a.X, b.Y-a.Y
			length := dx*dx + dy*dy
			if length == 0 {
				continue
			}
			u := math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/length))
			if math.Hypot(p.X-a.X-u*dx, p.Y-a.Y-u*dy) <= tolerance {
				return state.Wires[i].ID
			}
		}
	}
	return ""
}
func (s *screen) canvasClick(p workbench.Point) {
	state := s.service.Snapshot()
	switch state.Drawing.Tool {
	case "place":
		if state.Drawing.Kind == "junction" {
			point := workbench.Point{X: math.Round(p.X/workbench.GridSize) * workbench.GridSize, Y: math.Round(p.Y/workbench.GridSize) * workbench.GridSize}
			graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
			ids := []string{}
			for _, wire := range state.Wires {
				if wire.Connected && workbench.WireDistance(graph, wire, point) <= workbench.GridSize/2+1e-9 {
					ids = append(ids, wire.ID)
				}
			}
			if len(ids) > 1 {
				s.invoke("drawing.position.set", workbench.PositionParams{Position: point})
				s.errorMessage = "Several wires cross here. Use the X / Y junction controls to review and explicitly join the named routes."
				s.app.Invalidate("main")
				return
			}
			if len(ids) == 1 {
				s.invoke("wire.junction.insert", workbench.JunctionInsertParams{WireIDs: ids, Position: &point})
				return
			}
		}
		value := ""
		if state.Drawing.Kind == "net-label" {
			value = state.Drawing.Value
		}
		s.invoke("component.place", workbench.PlaceParams{Kind: state.Drawing.Kind, Position: p, Value: value})
	case "wire":
		ref := terminalHit(state, p, math.Max(6, 10/state.View.Zoom))
		if ref == "" {
			s.errorMessage = "Click a terminal dot, or use the named terminal controls."
			s.app.Invalidate("main")
			return
		}
		if state.Drawing.WireStart == "" {
			s.invoke("wire.begin", map[string]string{"terminal": ref})
		} else {
			s.invoke("wire.connect", workbench.ConnectParams{From: state.Drawing.WireStart, To: ref})
		}
	case "select":
		if hitPinnedReading(state, p) != "" {
			return
		}
		id := hitComponent(state, p)
		if id == "" {
			id = hitWire(state, p, 6/state.View.Zoom)
		}
		s.invoke("component.select", workbench.IDParams{ID: id})
	}
}
func (s *screen) mesh(v canvas.Viewport) (*canvas.Mesh, error) {
	state := s.service.Snapshot()
	if s.drawing != nil && s.drawing.widget != nil && s.drawing.pointer != nil {
		w := s.drawing.widget
		origin := w.Geom.Pos.Content
		if w.Scene != nil {
			origin.X += float32(w.Scene.SceneGeom.Pos.X)
			origin.Y += float32(w.Scene.SceneGeom.Pos.Y)
		}
		s.drawing.pointer.publish(float64(origin.X), float64(origin.Y), v)
	}
	if s.drawing != nil && s.drawing.dragging && s.drawing.dragGeneration == state.Document.Generation {
		for i, reading := range state.PinnedReadings {
			if "pin|"+reading.Definition.ID == s.drawing.dragID {
				state.PinnedReadings[i].Definition.Position = workbench.Point{X: math.Round((s.drawing.dragOrigin.X+s.drawing.dragDelta.X)/10) * 10, Y: math.Round((s.drawing.dragOrigin.Y+s.drawing.dragDelta.Y)/10) * 10}
			}
		}
		for i, part := range state.Components {
			if part.ID == s.drawing.dragID {
				state.Components[i].Position = workbench.Point{X: math.Round((s.drawing.dragOrigin.X+s.drawing.dragDelta.X)/10) * 10, Y: math.Round((s.drawing.dragOrigin.Y+s.drawing.dragDelta.Y)/10) * 10}
			}
		}
	}
	return schematic(state, v)
}

// pointer routes precise native packets through the same intent functions and
// commands as the portable canvas handlers. It runs only on the Vellum owner.
func (c *commandCanvas) routePointer(phase int, p, delta workbench.Point) {
	if c.screen.fileBusy || c.screen.pendingCommand != "" {
		c.dragID = ""
		c.dragging = false
		return
	}
	if phase == 1 || phase == 2 {
		state := c.screen.service.Snapshot()
		if state.Document.Generation != c.dragGeneration || state.Closed {
			c.dragID = ""
			c.dragging = false
			return
		}
	}
	switch phase {
	case 0:
		if c.widget != nil {
			c.widget.SetFocus()
		}
		state := c.screen.service.Snapshot()
		c.dragGeneration = state.Document.Generation
		c.activeTool = state.Drawing.Tool
		c.dragID = ""
		c.dragging = false
		c.dragDelta = workbench.Point{}
		if c.activeTool == "select" {
			if id := hitPinnedReading(state, p); id != "" {
				for _, reading := range state.PinnedReadings {
					if reading.Definition.ID == id {
						c.dragID = "pin|" + id
						c.dragOrigin = reading.Definition.Position
					}
				}
			} else if id := hitComponent(state, p); id != "" {
				part, _ := (workbench.Graph{Components: state.Components, Wires: state.Wires}).Component(id)
				c.dragID = id
				c.dragOrigin = part.Position
			}
		}
		c.screen.canvasClick(p)
	case 1:
		if c.activeTool == "place" || c.activeTool == "wire" {
			return
		}
		if c.dragID != "" {
			c.dragging = true
			c.dragDelta.X += delta.X
			c.dragDelta.Y += delta.Y
			c.screen.renderRevision++
			c.Revision = c.screen.renderRevision
			c.Owner.Invalidate(c.Window)
		} else {
			c.screen.invoke("view.pan", map[string]float64{"dx": -delta.X, "dy": -delta.Y})
		}
	case 3:
		factor := math.Max(.25, math.Min(4, math.Exp(delta.Y*.01)))
		c.screen.invoke("view.zoom", map[string]any{"factor": factor, "anchor": p})
	case 2:
		if c.dragging && c.dragID != "" {
			c.dragDelta.X += delta.X
			c.dragDelta.Y += delta.Y
			position := workbench.Point{X: c.dragOrigin.X + c.dragDelta.X, Y: c.dragOrigin.Y + c.dragDelta.Y}
			if strings.HasPrefix(c.dragID, "pin|") {
				c.screen.invoke("measurement.pin.move", workbench.MoveParams{ID: strings.TrimPrefix(c.dragID, "pin|"), Position: position, PreviousPosition: &c.dragOrigin})
			} else {
				c.screen.invoke("component.move", workbench.MoveParams{ID: c.dragID, Position: position, PreviousPosition: &c.dragOrigin})
			}
		}
		c.dragID = ""
		c.dragging = false
		c.screen.renderRevision++
		c.Revision = c.screen.renderRevision
		c.Owner.Invalidate(c.Window)
	}
}
