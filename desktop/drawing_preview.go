package main

import (
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

type drawingHover struct {
	position workbench.Point
	context  actionContext
}

func (c *commandCanvas) clearHover() {
	if c.hover == nil {
		return
	}
	c.hover = nil
	c.invalidatePreview()
}

func (c *commandCanvas) invalidatePreview() {
	c.screen.renderRevision++
	c.Revision = c.screen.renderRevision
	c.Owner.Invalidate(c.Window)
}

func (c *commandCanvas) previewPointer(context *actionContext, p workbench.Point) {
	state := c.screen.service.Snapshot()
	if context == nil || state.Closed || c.screen.fileBusy || c.screen.pendingCommand != "" || state.ExampleBrowser.Visible || sourcePaneHidden(state) || c.dragging || context.revision != state.CommandRevision || context.generation != state.Document.Generation || state.Drawing.Tool != "place" && state.Drawing.Tool != "wire" {
		c.clearHover()
		return
	}
	next := drawingHover{position: p, context: *context}
	if c.hover != nil && *c.hover == next {
		return
	}
	c.hover = &next
	c.invalidatePreview()
}

func (c *commandCanvas) drawingPreview(state workbench.Snapshot) *workbench.DrawingPreview {
	if c.hover == nil || state.Closed || c.screen.fileBusy || c.screen.pendingCommand != "" || state.ExampleBrowser.Visible || sourcePaneHidden(state) || c.hover.context.revision != state.CommandRevision || c.hover.context.generation != state.Document.Generation || state.Drawing.Tool != "place" && state.Drawing.Tool != "wire" {
		return nil
	}
	preview := workbench.PreviewDrawing(state, c.hover.position)
	return &preview
}

func (c *commandCanvas) withDrawingPreview(mesh *canvas.Mesh, state workbench.Snapshot, view canvas.Viewport) (*canvas.Mesh, error) {
	preview := c.drawingPreview(state)
	if preview == nil {
		return mesh, nil
	}
	palette := c.screen.colorsForTheme(state.Theme)
	ink := palette.Accent
	if !preview.Valid {
		ink = palette.Danger
	}
	b := &meshBuilder{minimumStroke: 1.25 / view.Zoom}
	b.add(mesh, nil)
	if preview.Component != nil {
		// A ghost shows its own configured defaults, never an existing device's
		// accepted actuation/frame that happens to share the placeholder ID.
		drawComponentSymbol(b, workbench.Snapshot{}, *preview.Component, ink, ink, ink, ink, false)
		for _, pin := range preview.Component.Terminals {
			at, _ := workbench.TerminalPosition(*preview.Component, pin)
			b.quad(at.X-3/view.Zoom, at.Y-3/view.Zoom, 6/view.Zoom, 6/view.Zoom, ink)
		}
	}
	// Dashed pending routes cannot be confused with committed electrical wires.
	for i := 1; i < len(preview.Path); i++ {
		a, z := preview.Path[i-1], preview.Path[i]
		dx, dy := z.X-a.X, z.Y-a.Y
		length := math.Hypot(dx, dy)
		for d := 0.; d < length; d += 12 / view.Zoom {
			end := math.Min(length, d+7/view.Zoom)
			b.stroke(ink, 2/view.Zoom, canvas.Point{X: a.X + dx*d/length, Y: a.Y + dy*d/length}, canvas.Point{X: a.X + dx*end/length, Y: a.Y + dy*end/length})
		}
	}
	if preview.Terminal != "" {
		ring := make([]canvas.Point, 33)
		for i := range ring {
			a := float64(i) * 2 * math.Pi / 32
			ring[i] = canvas.Point{X: preview.Position.X + 9/view.Zoom*math.Cos(a), Y: preview.Position.Y + 9/view.Zoom*math.Sin(a)}
		}
		b.stroke(ink, 2/view.Zoom, ring...)
	}
	// A fixed top-left hint stays readable without chasing the pointer or
	// reflowing any controls. Wrap to the viewport, including compact windows.
	message := []rune(preview.Message + " | Escape cancels")
	maxChars := max(12, int((view.Width-32)/7))
	lines := []string{}
	for len(message) > maxChars {
		lines = append(lines, string(message[:maxChars]))
		message = message[maxChars:]
	}
	lines = append(lines, string(message))
	at := view.World(canvas.Point{X: 12, Y: 12})
	b.quad(at.X, at.Y, (view.Width-24)/view.Zoom, float64(12+len(lines)*16)/view.Zoom, palette.Surface)
	for i, line := range lines {
		b.textAtScale(at.X+8/view.Zoom, at.Y+float64(17+i*16)/view.Zoom, line, ink, 1/view.Zoom)
	}
	if b.err != nil {
		return nil, b.err
	}
	return canvas.NewMesh(b.vertices, b.indices)
}
