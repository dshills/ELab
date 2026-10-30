package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

type nativeProbePill struct {
	ID, Text                 string
	Left, Top, Width, Height float64
	Anchor                   canvas.Point
	Color                    color.NRGBA
}

func nativeQuantity(value float64, unit string) string {
	if math.Abs(value) < 1e-12 {
		return "0 " + unit
	}
	magnitude := math.Abs(value)
	for _, entry := range []struct {
		scale  float64
		prefix string
	}{{1e9, "G"}, {1e6, "M"}, {1e3, "k"}, {1, ""}, {1e-3, "m"}, {1e-6, "µ"}, {1e-9, "n"}, {1e-12, "p"}} {
		if magnitude >= entry.scale {
			scaled := value / entry.scale
			decimals := max(0, 2-int(math.Floor(math.Log10(math.Abs(scaled)))))
			return fmt.Sprintf("%.*f %s%s", decimals, scaled, entry.prefix, unit)
		}
	}
	return "0 " + unit
}

// Pills mirror accepted selected trace samples. They never synthesize a value
// for a stale/absent result, a fixed historical window, or an unrelated recording.
func nativeProbePills(state workbench.Snapshot, view canvas.Viewport, palette appPalette) []nativeProbePill {
	if state.Transient == nil || state.Transient.Stale || state.Scope.Stale || state.Scope.Window.Mode == "fixed" || state.Recording != nil && state.Recording.Viewing {
		return nil
	}
	graph := (workbench.Graph{Components: state.Components, Wires: state.Wires}).VisibleGraph()
	names := scopeProbeColors(state.Scope.Traces)
	var out []nativeProbePill
	for i, trace := range state.Scope.Traces {
		if len(trace.Points) == 0 {
			continue
		}
		part, terminal, ok := graph.Terminal(trace.Probe.Positive)
		if !ok {
			continue
		}
		at, ok := workbench.TerminalPosition(part, terminal)
		if !ok {
			continue
		}
		value := trace.Points[len(trace.Points)-1].Value
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		text := nativeQuantity(value, trace.Probe.Unit)
		width := float64(len([]rune(text))*7 + 20)
		pill := nativeProbePill{ID: trace.Probe.ID, Text: text, Width: width / view.Zoom, Height: 22 / view.Zoom, Anchor: canvas.Point{X: at.X, Y: at.Y}, Color: traceColor(probeColor(names[i]), palette)}
		pill.Left = at.X - pill.Width/2
		pill.Top = at.Y - 32/view.Zoom
		collides := func() bool {
			for _, earlier := range out {
				gap := 4 / view.Zoom
				if pill.Left-gap < earlier.Left+earlier.Width && pill.Left+pill.Width+gap > earlier.Left && pill.Top-gap < earlier.Top+earlier.Height && pill.Top+pill.Height+gap > earlier.Top {
					return true
				}
			}
			return false
		}
		limit := view.World(canvas.Point{}).Y
		for attempt := 0; attempt < len(out)+1 && collides(); attempt++ {
			pill.Top -= 26 / view.Zoom
		}
		if pill.Top < limit {
			pill.Top = at.Y + 10/view.Zoom
			for attempt := 0; attempt < len(out)+1 && collides(); attempt++ {
				pill.Top += 26 / view.Zoom
			}
		}
		out = append(out, pill)
	}
	return out
}
func (b *meshBuilder) nativePill(p nativeProbePill, view canvas.Viewport) {
	b.stroke(p.Color, 1.6/view.Zoom, p.Anchor, canvas.Point{X: p.Left + p.Width/2, Y: p.Top + p.Height})
	radius := p.Height / 2
	b.quad(p.Left+radius, p.Top, p.Width-2*radius, p.Height, p.Color)
	for _, cx := range []float64{p.Left + radius, p.Left + p.Width - radius} {
		center := uint32(len(b.vertices))
		b.vertices = append(b.vertices, canvas.Vertex{Position: canvas.Point{X: cx, Y: p.Top + radius}, Color: p.Color})
		for i := 0; i <= 20; i++ {
			angle := float64(i) * math.Pi / 10
			b.vertices = append(b.vertices, canvas.Vertex{Position: canvas.Point{X: cx + radius*math.Cos(angle), Y: p.Top + radius + radius*math.Sin(angle)}, Color: p.Color})
			if i > 0 {
				b.indices = append(b.indices, center, center+uint32(i), center+uint32(i)+1)
			}
		}
	}
	b.textAtScale(p.Left+10/view.Zoom, p.Top+15/view.Zoom, p.Text, nativePillTextColor(p.Color), 1/view.Zoom)
}

// Choose ink by actual sRGB luminance, rather than assuming all trace fills
// support white text. Amber and bright dark-theme traces need dark lettering.
func nativePillTextColor(fill color.NRGBA) color.NRGBA {
	linear := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= .04045 {
			return x / 12.92
		}
		return math.Pow((x+.055)/1.055, 2.4)
	}
	luminance := .2126*linear(fill.R) + .7152*linear(fill.G) + .0722*linear(fill.B)
	if luminance > .179 {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}
