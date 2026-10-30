package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"

	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/dshills/elab/workbench"
)

var browserSymbols = sync.OnceValue(func() map[string]icons.Icon {
	out := make(map[string]icons.Icon)
	for _, part := range workbench.Library() {
		b := &meshBuilder{minimumStroke: 5}
		wiper := .5
		component := workbench.Component{Kind: part.Kind, WiperPosition: &wiper}
		ink := color.NRGBA{A: 255}
		drawComponentSymbol(b, workbench.Snapshot{}, component, ink, ink, ink, ink, true)
		// Tessellate through the very same renderer as the sheet. Core's Icon
		// renderer applies the current theme/focus foreground to this vector mask.
		var svg strings.Builder
		minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, v := range b.vertices {
			minX, minY = min(minX, v.Position.X), min(minY, v.Position.Y)
			maxX, maxY = max(maxX, v.Position.X), max(maxY, v.Position.Y)
		}
		if part.Kind == "junction" {
			minX, minY, maxX, maxY = -14, -14, 14, 14
		}
		fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%g %g %g %g" width="32" height="32"><path d="`, minX-6, minY-6, maxX-minX+12, maxY-minY+12)
		for i := 0; i+2 < len(b.indices); i += 3 {
			a, c, d := b.vertices[b.indices[i]].Position, b.vertices[b.indices[i+1]].Position, b.vertices[b.indices[i+2]].Position
			fmt.Fprintf(&svg, "M%g %gL%g %gL%g %gZ", a.X, a.Y, c.X, c.Y, d.X, d.Y)
		}
		svg.WriteString(`"/></svg>`)
		out[part.Kind] = icons.Icon(svg.String())
	}
	return out
})

// The portable ListBox retains all keyboard, stale-action and accessibility
// behavior. This app-local presentation adapter supplies its leading symbols
// until Vellum exposes artwork on Option; no backend files are modified.
func styleComponentRow(widget core.Widget, w *core.WidgetBase) bool {
	row, ok := widget.(*core.Switch)
	if !ok || w.Parent == nil || w.Parent.AsTree().Name != "part-list" {
		return false
	}
	symbol, ok := browserSymbols()[strings.TrimPrefix(w.Name, "choice:")]
	if !ok {
		return false
	}
	changed := false
	if row.IconOn != symbol || row.IconOff != symbol {
		row.IconOn, row.IconOff = symbol, symbol
		changed = true
	}
	if styled, _ := w.Property("elab.component.styled").(bool); !styled {
		w.SetProperty("elab.component.styled", true)
		w.FinalStyler(func(s *styles.Style) {
			s.Min.Y = units.Dp(28)
			s.Max.Y = units.Dp(28)
			s.IconSize.Set(units.Dp(20))
			s.Font.Size = units.Dp(13)
			s.Padding.Set(units.Dp(4))
			s.Gap.X = units.Dp(6)
			s.Border.Width.Zero()
			s.Border.Radius.Set(units.Dp(5))
		})
		changed = true
	}
	if changed {
		row.Update()
	}
	return changed
}
