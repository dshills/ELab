package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type meshBuilder struct {
	vertices []canvas.Vertex
	indices  []uint32
	err      error
}

func (b *meshBuilder) add(m *canvas.Mesh, err error) {
	if err != nil {
		b.err = err
		return
	}
	offset := uint32(len(b.vertices))
	b.vertices = append(b.vertices, m.Vertices()...)
	for _, i := range m.Indices() {
		b.indices = append(b.indices, offset+i)
	}
}
func (b *meshBuilder) stroke(c color.NRGBA, width float64, p ...canvas.Point) {
	b.add(canvas.StrokePolyline(p, width, c))
}
func (b *meshBuilder) quad(x, y, width, height float64, c color.NRGBA) {
	offset := uint32(len(b.vertices))
	b.vertices = append(b.vertices, canvas.Vertex{Position: canvas.Point{X: x, Y: y}, Color: c}, canvas.Vertex{Position: canvas.Point{X: x + width, Y: y}, Color: c}, canvas.Vertex{Position: canvas.Point{X: x, Y: y + height}, Color: c}, canvas.Vertex{Position: canvas.Point{X: x + width, Y: y + height}, Color: c})
	b.indices = append(b.indices, offset, offset+1, offset+2, offset+1, offset+3, offset+2)
}
func (b *meshBuilder) text(x, y float64, text string, c color.NRGBA) {
	// Tessellate the bundled bitmap font into retained geometry. Labels follow the
	// same world transform as symbols and need no per-frame CPU image readback.
	text = strings.ReplaceAll(text, "Ω", "ohm")
	text = strings.ReplaceAll(text, "·", "|")
	for _, r := range text {
		rect, mask, origin, advance, ok := basicfont.Face7x13.Glyph(fixed.P(0, 0), r)
		if !ok {
			r = '?'
			rect, mask, origin, advance, _ = basicfont.Face7x13.Glyph(fixed.P(0, 0), r)
		}
		for py := rect.Min.Y; py < rect.Max.Y; py++ {
			for px := rect.Min.X; px < rect.Max.X; px++ {
				_, _, _, alpha := mask.At(origin.X+px-rect.Min.X, origin.Y+py-rect.Min.Y).RGBA()
				if alpha > 0 {
					b.quad(x+float64(px)*1.25, y+float64(py)*1.25, 1.25, 1.25, c)
				}
			}
		}
		x += float64(advance) / 64 * 1.25
	}
}
func schematic(state workbench.Snapshot, view canvas.Viewport) (*canvas.Mesh, error) {
	b := &meshBuilder{}
	ink := color.NRGBA{R: 47, G: 65, B: 78, A: 255}
	wireColor := color.NRGBA{R: 56, G: 125, B: 147, A: 255}
	grid := color.NRGBA{R: 211, G: 222, B: 230, A: 255}
	selectedColor := color.NRGBA{R: 40, G: 158, B: 180, A: 255}
	pendingColor := color.NRGBA{R: 212, G: 138, B: 28, A: 255}
	minPoint, maxPoint := view.World(canvas.Point{}), view.World(canvas.Point{X: view.Width, Y: view.Height})
	minX, maxX := math.Max(-10000, minPoint.X), math.Min(10000, maxPoint.X)
	minY, maxY := math.Max(-10000, minPoint.Y), math.Min(10000, maxPoint.Y)
	// Keep dot spacing legible and geometry bounded when zoomed out.
	step := 20 * math.Max(1, math.Pow(2, math.Ceil(math.Log2(1/view.Zoom))))
	dot := 1.2 / view.Zoom
	for x := math.Ceil(minX/step) * step; x <= maxX; x += step {
		for y := math.Ceil(minY/step) * step; y <= maxY; y += step {
			b.quad(x-dot/2, y-dot/2, dot, dot, grid)
		}
	}
	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	focusObjects, focusTerminals := map[string]bool{}, map[string]bool{}
	if state.TraceFocus != nil && !state.TraceFocus.Unavailable {
		for _, id := range state.TraceFocus.Objects {
			focusObjects[id] = true
		}
		for _, ref := range state.TraceFocus.Terminals {
			focusTerminals[ref] = true
		}
	}
	focusColor := color.NRGBA{R: 153, G: 78, B: 191, A: 255}
	connectionCount := map[string]int{}
	for _, w := range state.Wires {
		c := wireColor
		width := 2.5
		if w.ID == state.Selected {
			c = selectedColor
			width = 4
		}
		b.stroke(c, width, wirePath(graph, w)...)
		if w.Connected {
			connectionCount[w.From]++
			connectionCount[w.To]++
		}
	}
	for _, part := range state.Components {
		point := func(x, y float64) canvas.Point {
			p := workbench.RotateLocal(workbench.Point{X: x, Y: y}, part.Rotation)
			return canvas.Point{X: p.X + part.Position.X, Y: p.Y + part.Position.Y}
		}
		stroke := func(c color.NRGBA, width float64, p ...canvas.Point) {
			transformed := make([]canvas.Point, len(p))
			for i, v := range p {
				transformed[i] = point(v.X, v.Y)
			}
			b.stroke(c, width, transformed...)
		}
		switch part.Kind {
		case "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
			p := make([]canvas.Point, 49)
			for i := range p {
				angle := float64(i) * 2 * math.Pi / 48
				p[i] = canvas.Point{X: 34 * math.Cos(angle), Y: 34 * math.Sin(angle)}
			}
			stroke(ink, 2.5, p...)
			if part.Kind == "sine-source" {
				wave := make([]canvas.Point, 33)
				for i := range wave {
					x := -22 + 44*float64(i)/32
					wave[i] = canvas.Point{X: x, Y: -10 * math.Sin(2*math.Pi*float64(i)/32)}
				}
				stroke(ink, 2, wave...)
			} else if part.Kind == "finite-pulse-source" {
				stroke(ink, 2, canvas.Point{X: -20, Y: 10}, canvas.Point{X: -14, Y: -10}, canvas.Point{X: -4, Y: -10}, canvas.Point{X: 4, Y: 10}, canvas.Point{X: 14, Y: 10}, canvas.Point{X: 20, Y: -10})
			} else if part.Kind == "pulse-source" {
				stroke(ink, 2, canvas.Point{X: -20, Y: 10}, canvas.Point{X: -12, Y: 10}, canvas.Point{X: -12, Y: -10}, canvas.Point{X: 0, Y: -10}, canvas.Point{X: 0, Y: 10}, canvas.Point{X: 12, Y: 10}, canvas.Point{X: 12, Y: -10}, canvas.Point{X: 20, Y: -10})
			} else {
				stroke(ink, 2, canvas.Point{X: -9, Y: -13}, canvas.Point{X: 9, Y: -13})
				stroke(ink, 2, canvas.Point{X: 0, Y: -22}, canvas.Point{X: 0, Y: -4})
				stroke(ink, 2, canvas.Point{X: -9, Y: 15}, canvas.Point{X: 9, Y: 15})
			}
		case "capacitor":
			stroke(ink, 3, canvas.Point{X: -24, Y: -9}, canvas.Point{X: 24, Y: -9})
			stroke(ink, 3, canvas.Point{X: -24, Y: 9}, canvas.Point{X: 24, Y: 9})
			stroke(wireColor, 2.5, canvas.Point{Y: -32}, canvas.Point{Y: -9})
			stroke(wireColor, 2.5, canvas.Point{Y: 9}, canvas.Point{Y: 32})
		case "switch":
			stroke(wireColor, 2.5, canvas.Point{X: -55}, canvas.Point{X: -25})
			stroke(wireColor, 2.5, canvas.Point{X: 25}, canvas.Point{X: 55})
			end := canvas.Point{X: 25, Y: -22}
			closed := part.SwitchClosed != nil && *part.SwitchClosed
			for _, input := range state.SwitchStates {
				if input.ID == part.ID && input.AppliedClosed != nil {
					closed = *input.AppliedClosed
				}
			}
			if closed {
				end.Y = 0
			}
			stroke(ink, 3, canvas.Point{X: -25}, end)
		case "potentiometer":
			stroke(ink, 2.5, canvas.Point{X: -55}, canvas.Point{X: -43}, canvas.Point{X: -35, Y: -15}, canvas.Point{X: -21, Y: 15}, canvas.Point{X: -7, Y: -15}, canvas.Point{X: 7, Y: 15}, canvas.Point{X: 21, Y: -15}, canvas.Point{X: 35, Y: 15}, canvas.Point{X: 43}, canvas.Point{X: 55})
			contact := 35 * (1 - 2*(*part.WiperPosition))
			stroke(ink, 2.5, canvas.Point{Y: -45}, canvas.Point{X: contact, Y: -17}, canvas.Point{X: contact - 5, Y: -25})
			stroke(ink, 2.5, canvas.Point{X: contact, Y: -17}, canvas.Point{X: contact + 5, Y: -25})
		case "resistor":
			stroke(ink, 2.5, canvas.Point{X: -55}, canvas.Point{X: -43}, canvas.Point{X: -35, Y: -15}, canvas.Point{X: -21, Y: 15}, canvas.Point{X: -7, Y: -15}, canvas.Point{X: 7, Y: 15}, canvas.Point{X: 21, Y: -15}, canvas.Point{X: 35, Y: 15}, canvas.Point{X: 43}, canvas.Point{X: 55})
		case "diode":
			stroke(ink, 3, canvas.Point{X: -20, Y: -24}, canvas.Point{X: 20, Y: -24}, canvas.Point{Y: 17}, canvas.Point{X: -20, Y: -24})
			stroke(ink, 3, canvas.Point{X: -22, Y: 19}, canvas.Point{X: 22, Y: 19})
			stroke(wireColor, 2.5, canvas.Point{Y: -32}, canvas.Point{Y: -24})
			stroke(wireColor, 2.5, canvas.Point{Y: 19}, canvas.Point{Y: 32})
		case "led":
			led := color.NRGBA{R: 157, G: 42, B: 50, A: 255}
			current := 0.
			if state.Result != nil && !state.Result.Stale && state.Result.LEDID == part.ID {
				current = state.Result.Current
			}
			if state.NPN != nil && !state.NPN.Stale && state.NPN.LEDID == part.ID {
				current = state.NPN.CollectorCurrent
			}
			if state.Mixed != nil && !state.Mixed.Stale {
				for _, l := range state.Mixed.LEDs {
					if l.ID == part.ID {
						current = l.Current
					}
				}
			}
			if current > 1e-6 {
				strength := math.Min(1, math.Sqrt(current/.003))
				led = color.NRGBA{R: uint8(157 + 81*strength), G: uint8(42 + 45*strength), B: uint8(50 + 18*strength), A: 255}
			}
			stroke(led, 3, canvas.Point{X: -20, Y: -24}, canvas.Point{X: 20, Y: -24}, canvas.Point{Y: 17}, canvas.Point{X: -20, Y: -24})
			stroke(ink, 3, canvas.Point{X: -22, Y: 19}, canvas.Point{X: 22, Y: 19})
			stroke(wireColor, 2.5, canvas.Point{Y: -32}, canvas.Point{Y: -24})
			stroke(wireColor, 2.5, canvas.Point{Y: 19}, canvas.Point{Y: 32})
			for _, y := range []float64{-15, 5} {
				stroke(led, 2, canvas.Point{X: 30, Y: y}, canvas.Point{X: 50, Y: y - 20}, canvas.Point{X: 44, Y: y - 20})
				stroke(led, 2, canvas.Point{X: 50, Y: y - 20}, canvas.Point{X: 50, Y: y - 14})
			}
		case "npn":
			stroke(ink, 3, canvas.Point{X: -12, Y: -20}, canvas.Point{X: -12, Y: 20})
			stroke(wireColor, 2.5, canvas.Point{X: -40}, canvas.Point{X: -12})
			stroke(ink, 2.5, canvas.Point{X: -12, Y: -10}, canvas.Point{X: 25, Y: -30}, canvas.Point{X: 25, Y: -35})
			stroke(ink, 2.5, canvas.Point{X: -12, Y: 10}, canvas.Point{X: 25, Y: 30}, canvas.Point{X: 25, Y: 35})
			stroke(accent, 3, canvas.Point{X: 12, Y: 15}, canvas.Point{X: 25, Y: 30}, canvas.Point{X: 5, Y: 25})
			for _, pin := range []struct {
				x, y float64
				name string
			}{{-46, -20, "B"}, {32, -36, "C"}, {32, 18, "E"}} {
				p := point(pin.x, pin.y)
				b.text(p.X, p.Y, pin.name, muted)
			}
		case "junction":
			if connectionCount[part.ID+".node"] > 0 {
				center := point(0, 0)
				b.quad(center.X-3, center.Y-3, 6, 6, wireColor)
			}
		case "net-label":
			stroke(ink, 2, canvas.Point{}, canvas.Point{X: 10}, canvas.Point{X: 10, Y: -7}, canvas.Point{X: 18}, canvas.Point{X: 10, Y: 7}, canvas.Point{X: 10})
			at := point(22, 4)
			b.text(at.X, at.Y, part.Label, ink)
		case "ground":
			stroke(wireColor, 2.5, canvas.Point{Y: -20}, canvas.Point{})
			for i := 0; i < 3; i++ {
				width := 22. - float64(i)*7
				y := float64(i) * 7
				stroke(ink, 2.5, canvas.Point{X: -width, Y: y}, canvas.Point{X: width, Y: y})
			}
		}
		if focusObjects[part.ID] {
			stroke(focusColor, 3, canvas.Point{X: -70, Y: -52}, canvas.Point{X: 70, Y: -52}, canvas.Point{X: 70, Y: 52}, canvas.Point{X: -70, Y: 52}, canvas.Point{X: -70, Y: -52})
		}
		if part.ID == state.Selected {
			stroke(selectedColor, 1.8, canvas.Point{X: -64, Y: -46}, canvas.Point{X: 64, Y: -46}, canvas.Point{X: 64, Y: 46}, canvas.Point{X: -64, Y: 46}, canvas.Point{X: -64, Y: -46})
		}
		for _, terminal := range part.Terminals {
			ref := part.ID + "." + terminal
			position, _ := workbench.TerminalPosition(part, terminal)
			c := wireColor
			radius := 4.
			if focusTerminals[ref] {
				c = focusColor
				radius = 8
			}
			if ref == state.Drawing.WireStart {
				c = pendingColor
				radius = 7
			}
			if len(state.Diagnostics) > 0 && state.Diagnostics[0].ObjectID == ref {
				c = pendingColor
			}
			p := make([]canvas.Point, 17)
			for i := range p {
				angle := float64(i) * 2 * math.Pi / 16
				p[i] = canvas.Point{X: position.X + radius*math.Cos(angle), Y: position.Y + radius*math.Sin(angle)}
			}
			b.stroke(c, 1.7, p...)
			if connectionCount[ref] > 1 {
				b.quad(position.X-2, position.Y-2, 4, 4, c)
			}
		}
		title := part.ID
		if part.Unit != "" {
			if part.Unit == "F" {
				scale, suffix := 1., "F"
				if part.Value < 1e-9 {
					scale, suffix = 1e12, "pF"
				} else if part.Value < 1e-6 {
					scale, suffix = 1e9, "nF"
				} else if part.Value < 1e-3 {
					scale, suffix = 1e6, "uF"
				} else if part.Value < 1 {
					scale, suffix = 1e3, "mF"
				}
				title += fmt.Sprintf(" %.4g %s", part.Value*scale, suffix)
			} else {
				if part.Unit == "Ω" && part.Value >= 1000 {
					title += fmt.Sprintf(" %.4g kΩ", part.Value/1000)
				} else {
					title += fmt.Sprintf(" %.4g %s", part.Value, part.Unit)
				}
			}
		}
		y := part.Position.Y - 55
		if workbench.IsConnectivityPrimitive(part.Kind) {
			y = part.Position.Y - 18
		}
		if part.Kind == "ground" {
			y = part.Position.Y + 35
		}
		if part.Rotation == 90 || part.Rotation == 270 {
			y = part.Position.Y - 75
		}
		b.text(part.Position.X-float64(len(title))*4.4, y, title, ink)
	}
	for _, reading := range state.PinnedReadings {
		min, max := workbench.PinnedReadingBounds(reading)
		c := color.NRGBA{R: 39, G: 110, B: 126, A: 255}
		if reading.State != "live" {
			c = color.NRGBA{R: 151, G: 96, B: 43, A: 255}
		}
		b.quad(min.X, min.Y, max.X-min.X, max.Y-min.Y, color.NRGBA{R: 243, G: 250, B: 251, A: 255})
		b.stroke(c, 1.5, canvas.Point{X: min.X, Y: min.Y}, canvas.Point{X: max.X, Y: min.Y}, canvas.Point{X: max.X, Y: max.Y}, canvas.Point{X: min.X, Y: max.Y}, canvas.Point{X: min.X, Y: min.Y})
		b.text(min.X+12, min.Y+23, reading.Display, c)
		b.text(min.X+12, min.Y+43, reading.ReferenceDisplay, c)
	}
	if len(state.Components) == 0 {
		b.text(view.Center.X-150, view.Center.Y, "Choose a part and click to place it", ink)
	}
	if b.err != nil {
		return nil, b.err
	}
	return canvas.NewMesh(b.vertices, b.indices)
}
