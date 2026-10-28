package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type meshBuilder struct {
	vertices      []canvas.Vertex
	indices       []uint32
	err           error
	minimumStroke float64
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
	b.add(canvas.StrokePolyline(p, max(width, b.minimumStroke), c))
}
func (b *meshBuilder) quad(x, y, width, height float64, c color.NRGBA) {
	offset := uint32(len(b.vertices))
	b.vertices = append(b.vertices, canvas.Vertex{Position: canvas.Point{X: x, Y: y}, Color: c}, canvas.Vertex{Position: canvas.Point{X: x + width, Y: y}, Color: c}, canvas.Vertex{Position: canvas.Point{X: x, Y: y + height}, Color: c}, canvas.Vertex{Position: canvas.Point{X: x + width, Y: y + height}, Color: c})
	b.indices = append(b.indices, offset, offset+1, offset+2, offset+1, offset+3, offset+2)
}
func (b *meshBuilder) text(x, y float64, text string, c color.NRGBA) {
	b.textAtScale(x, y, text, c, 1.25)
}
func (b *meshBuilder) textAtScale(x, y float64, text string, c color.NRGBA, scale float64) {
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
					b.quad(x+float64(px)*scale, y+float64(py)*scale, scale, scale, c)
				}
			}
		}
		x += float64(advance) / 64 * scale
	}
}
func schematic(state workbench.Snapshot, view canvas.Viewport, phase float64) (*canvas.Mesh, error) {
	return schematicWithPalette(state, view, phase, paletteForTheme(state.Theme))
}

func schematicWithPalette(state workbench.Snapshot, view canvas.Viewport, phase float64, palette appPalette) (*canvas.Mesh, error) {
	b := &meshBuilder{minimumStroke: 1.25 / view.Zoom}
	annotated := map[string]bool{}
	for _, a := range state.SourceAnnotations {
		annotated[a.ID] = true
	}
	ink, muted, accent := palette.Ink, palette.Muted, palette.Accent
	wireColor := traceColor(color.NRGBA{R: 56, G: 125, B: 147, A: 255}, palette)
	grid := palette.Grid
	selectedColor := color.NRGBA{R: 40, G: 158, B: 180, A: 255}
	pendingColor := color.NRGBA{R: 212, G: 138, B: 28, A: 255}
	proposed := map[string]bool{}
	if state.Proposal != nil && !state.Proposal.Stale {
		for _, change := range state.Proposal.Preview.Changes {
			proposed[change.ID] = true
		}
	}
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
	graph := (workbench.Graph{Components: state.Components, Wires: state.Wires}).VisibleGraph()
	state.Components, state.Wires = graph.Components, graph.Wires
	focusObjects, focusTerminals := map[string]bool{}, map[string]bool{}
	if state.TraceFocus != nil && !state.TraceFocus.Unavailable {
		for _, id := range state.TraceFocus.Objects {
			focusObjects[id] = true
		}
		for _, ref := range state.TraceFocus.Terminals {
			focusTerminals[ref] = true
		}
	}
	measurementTerminals, measurementWires := map[string]bool{}, map[string]bool{}
	for _, ref := range state.MeasurementInspection.Terminals {
		measurementTerminals[ref] = true
	}
	for _, id := range state.MeasurementInspection.Wires {
		measurementWires[id] = true
	}
	diagnosticObjects, diagnosticTerminals, diagnosticWires := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, help := range state.DiagnosticPresentation.Items {
		for _, id := range help.Objects {
			diagnosticObjects[id] = true
		}
		for _, ref := range help.Terminals {
			diagnosticTerminals[ref] = true
		}
		for _, id := range help.Wires {
			diagnosticWires[id] = true
		}
	}
	diagnosticColor := color.NRGBA{R: 192, G: 73, B: 50, A: 255}
	measurementColor := color.NRGBA{R: 80, G: 124, B: 64, A: 255}
	focusColor := color.NRGBA{R: 153, G: 78, B: 191, A: 255}
	wireDisplays := map[string]workbench.WireDisplay{}
	for _, wire := range state.Display.Wires {
		wireDisplays[wire.ID] = wire
	}
	connectionCount := map[string]int{}
	for _, w := range state.Wires {
		c := wireColor
		if reading, ok := wireDisplays[w.ID]; ok {
			c = displayColor(reading.Color)
		}
		width := 2.5
		if proposed[w.ID] {
			c, width = palette.Accent, 4
		}
		if measurementWires[w.ID] {
			c, width = measurementColor, 3.5
		}
		if w.ID == state.Selected {
			c = selectedColor
			width = 4
		}
		if diagnosticWires[w.ID] {
			c, width = diagnosticColor, 5
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
		drawComponentSymbol(b, state, part, ink, muted, accent, wireColor, connectionCount[part.ID+".node"] > 0)
		outlineW, outlineH := 64., 46.
		if part.Block != nil {
			outlineW, outlineH = workbench.BlockBodySize(part)
			outlineW += 30
			outlineH += 8
		}
		if workbench.IsLogicKind(part.Kind) {
			outlineW, outlineH = 72, 72
		}
		if part.Kind == "timer555" || workbench.IsSequentialKind(part.Kind) {
			outlineW, outlineH = 108, 118
		}
		if workbench.IsDisplayKind(part.Kind) {
			outlineW, outlineH = 128, 238
		}
		if workbench.IsRegulatorKind(part.Kind) {
			outlineW, outlineH = 108, 98
		}
		if part.Kind == "ldr" {
			outlineW, outlineH = 76, 82
		}
		if part.Kind == "relay" {
			outlineW, outlineH = 128, 138
		}
		if diagnosticObjects[part.ID] {
			stroke(diagnosticColor, 2.5, canvas.Point{X: -outlineW - 12, Y: -outlineH - 12}, canvas.Point{X: outlineW + 12, Y: -outlineH - 12}, canvas.Point{X: outlineW + 12, Y: outlineH + 12}, canvas.Point{X: -outlineW - 12, Y: outlineH + 12}, canvas.Point{X: -outlineW - 12, Y: -outlineH - 12})
			b.text(part.Position.X+82, part.Position.Y-50, "!", diagnosticColor)
		}
		if focusObjects[part.ID] {
			stroke(focusColor, 3, canvas.Point{X: -outlineW - 6, Y: -outlineH - 6}, canvas.Point{X: outlineW + 6, Y: -outlineH - 6}, canvas.Point{X: outlineW + 6, Y: outlineH + 6}, canvas.Point{X: -outlineW - 6, Y: outlineH + 6}, canvas.Point{X: -outlineW - 6, Y: -outlineH - 6})
		}
		if part.ID == state.Selected {
			stroke(selectedColor, 1.8, canvas.Point{X: -outlineW, Y: -outlineH}, canvas.Point{X: outlineW, Y: -outlineH}, canvas.Point{X: outlineW, Y: outlineH}, canvas.Point{X: -outlineW, Y: outlineH}, canvas.Point{X: -outlineW, Y: -outlineH})
		}
		if proposed[part.ID] {
			// A second inset outline identifies the currently affected source;
			// the proposal list supplies exact before/after data without editing.
			stroke(palette.Accent, 2, canvas.Point{X: -outlineW - 10, Y: -outlineH - 10}, canvas.Point{X: outlineW + 10, Y: -outlineH - 10}, canvas.Point{X: outlineW + 10, Y: outlineH + 10}, canvas.Point{X: -outlineW - 10, Y: outlineH + 10}, canvas.Point{X: -outlineW - 10, Y: -outlineH - 10})
		}
		for _, terminal := range part.Terminals {
			ref := part.ID + "." + terminal
			position, _ := workbench.TerminalPosition(part, terminal)
			c := wireColor
			radius := 4.
			if measurementTerminals[ref] {
				c, radius = measurementColor, 7
			}
			if focusTerminals[ref] {
				c = focusColor
				radius = 8
			}
			if ref == state.Drawing.WireStart {
				c = pendingColor
				radius = 7
			}
			if diagnosticTerminals[ref] {
				c, radius = diagnosticColor, 10
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
		if annotated[part.ID] {
			continue
		}
		_, title := workbench.CanvasPartLabel(part)
		at := workbench.CanvasPartLabelPositionAtZoom(part, title, view.Zoom)
		b.textAtScale(at.X, at.Y, title, ink, workbench.CanvasFontScale(view.Zoom))
	}
	for _, a := range state.SourceAnnotations {
		part, visible := graph.Component(a.ID)
		if !visible {
			continue
		}
		layout := workbench.LayoutCanvasAnnotation(a, part.Rotation, view.Zoom)
		for i, line := range a.Lines {
			at := layout.Position(line, i)
			b.textAtScale(at.X, at.Y, line, ink, layout.Scale)
		}
	}
	frame := workbench.MakeDisplayFrame(graph, state.Display, phase)
	for _, marker := range frame.Markers {
		at, dx, dy := marker.At, marker.DX, marker.DY
		c := color.NRGBA{R: 118, G: 67, B: 158, A: 255}
		if marker.Kind == "dot" {
			b.quad(at.X-2.5, at.Y-2.5, 5, 5, c)
			continue
		}
		b.stroke(c, 2, canvas.Point{X: at.X - 7*dx - 5*dy, Y: at.Y - 7*dy + 5*dx}, canvas.Point{X: at.X + 4*dx, Y: at.Y + 4*dy}, canvas.Point{X: at.X - 7*dx + 5*dy, Y: at.Y - 7*dy - 5*dx})
	}
	for _, reading := range state.PinnedReadings {
		min, max := workbench.PinnedReadingBounds(reading)
		c := color.NRGBA{R: 39, G: 110, B: 126, A: 255}
		if reading.State != "live" {
			c = color.NRGBA{R: 151, G: 96, B: 43, A: 255}
		}
		if palette.Paper.R < 100 {
			c = palette.Accent
			if reading.State != "live" {
				c = palette.Danger
			}
		}
		b.quad(min.X, min.Y, max.X-min.X, max.Y-min.Y, palette.Surface)
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

func nativePartLabelPosition(part workbench.Component, title string) workbench.Point {
	return workbench.CanvasPartLabelPosition(part, title)
}

// drawComponentSymbol supplies the sheet and browser with identical symbol geometry.
func drawComponentSymbol(b *meshBuilder, state workbench.Snapshot, part workbench.Component, ink, muted, accent, wireColor color.NRGBA, connected bool) {
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
	case "circuit-block":
		for _, path := range workbench.BlockSymbolPaths(part) {
			points := make([]canvas.Point, len(path))
			for k, q := range path {
				points[k] = canvas.Point{X: q.X, Y: q.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.BlockSymbolLabels(part) {
			b.text(label.At.X, label.At.Y, label.Text, ink)
		}
	case "njfet", "pjfet", "current-source", "electronic-load":
		var paths [][]workbench.Point
		if workbench.IsJFETKind(part.Kind) {
			paths = workbench.JFETSymbolPaths(part.Kind)
		} else {
			paths = workbench.CurrentSymbolPaths(part.Kind)
		}
		for _, path := range paths {
			points := make([]canvas.Point, len(path))
			for k, p := range path {
				points[k] = canvas.Point{X: p.X, Y: p.Y}
			}
			stroke(ink, 2.5, points...)
		}
	case "speaker", "pwm-controller", "shift-register", "piezo", "active-buzzer":
		for _, path := range workbench.ExtendedSymbolPaths(part.Kind) {
			points := make([]canvas.Point, len(path))
			for k, p := range path {
				points[k] = canvas.Point{X: p.X, Y: p.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.ExtendedSymbolLabels(part) {
			first := len(b.vertices)
			b.text(label.At.X, label.At.Y, label.Text, muted)
			for i := first; i < len(b.vertices); i++ {
				at := b.vertices[i].Position
				d := workbench.RotateLocal(workbench.Point{X: at.X - label.At.X, Y: at.Y - label.At.Y}, part.Rotation)
				b.vertices[i].Position = canvas.Point{X: label.At.X + d.X, Y: label.At.Y + d.Y}
			}
		}
	case "hbridge":
		for _, path := range workbench.HBridgeSymbolPaths() {
			points := make([]canvas.Point, len(path))
			for k, p := range path {
				points[k] = canvas.Point{X: p.X, Y: p.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.HBridgeSymbolLabels(part) {
			first := len(b.vertices)
			b.text(label.At.X, label.At.Y, label.Text, muted)
			for i := first; i < len(b.vertices); i++ {
				at := b.vertices[i].Position
				d := workbench.RotateLocal(workbench.Point{X: at.X - label.At.X, Y: at.Y - label.At.Y}, part.Rotation)
				b.vertices[i].Position = canvas.Point{X: label.At.X + d.X, Y: label.At.Y + d.Y}
			}
		}
	case "transformer":
		// Browser artwork uses kind-only placeholders; unconfigured glyphs use normal dots.
		reversed := part.Power != nil && part.Power.Transformer != nil && part.Power.Transformer.Reversed
		for _, path := range workbench.TransformerSymbolPaths(reversed) {
			points := make([]canvas.Point, len(path))
			for k, p := range path {
				points[k] = canvas.Point{X: p.X, Y: p.Y}
			}
			stroke(ink, 2.5, points...)
		}
	case "optocoupler":
		for _, path := range workbench.OptocouplerSymbolPaths() {
			points := make([]canvas.Point, len(path))
			for k, p := range path {
				points[k] = canvas.Point{X: p.X, Y: p.Y}
			}
			stroke(ink, 2.5, points...)
		}
	case "dc-motor":
		for _, path := range workbench.MotorSymbolPaths() {
			points := make([]canvas.Point, len(path))
			for k, p := range path {
				points[k] = canvas.Point{X: p.X, Y: p.Y}
			}
			stroke(ink, 2.5, points...)
		}
	case "battery":
		for _, path := range workbench.BatterySymbolPaths() {
			points := make([]canvas.Point, len(path))
			for k, p := range path {
				points[k] = canvas.Point{X: p.X, Y: p.Y}
			}
			stroke(ink, 2.5, points...)
		}
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
	case "inductor":
		points := workbench.InductorPolyline()
		coil := make([]canvas.Point, len(points))
		for k, p := range points {
			coil[k] = canvas.Point{X: p.X, Y: p.Y}
		}
		stroke(ink, 2.5, coil...)
	case "capacitor":
		stroke(ink, 3, canvas.Point{X: -24, Y: -9}, canvas.Point{X: 24, Y: -9})
		stroke(ink, 3, canvas.Point{X: -24, Y: 9}, canvas.Point{X: 24, Y: 9})
		stroke(wireColor, 2.5, canvas.Point{Y: -32}, canvas.Point{Y: -9})
		stroke(wireColor, 2.5, canvas.Point{Y: 9}, canvas.Point{Y: 32})
	case "linear-regulator", "ldo-regulator":
		for _, path := range workbench.RegulatorSymbolPaths(part.Kind) {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.RegulatorSymbolLabels(part) {
			b.text(label.At.X, label.At.Y, label.Text, muted)
		}
	case "relay", "comparator", "pushbutton", "bouncing-pushbutton":
		memory := workbench.ControlMemory{}
		if state.Mixed != nil && !state.Mixed.Stale {
			for _, c := range state.Mixed.Controls {
				if c.ID == part.ID {
					memory = c.Memory
				}
			}
		}
		for _, path := range workbench.ControlSymbolPaths(part.Kind, memory) {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.ControlSymbolLabels(part) {
			b.text(label.At.X, label.At.Y, label.Text, muted)
		}
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
	case "thermistor", "ldr":
		for _, path := range workbench.SensorSymbolPaths(part.Kind) {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
	case "resistor":
		stroke(ink, 2.5, canvas.Point{X: -55}, canvas.Point{X: -43}, canvas.Point{X: -35, Y: -15}, canvas.Point{X: -21, Y: 15}, canvas.Point{X: -7, Y: -15}, canvas.Point{X: 7, Y: 15}, canvas.Point{X: 21, Y: -15}, canvas.Point{X: 35, Y: 15}, canvas.Point{X: 43}, canvas.Point{X: 55})
	case "diode", "zener", "schottky":
		stroke(ink, 3, canvas.Point{X: -20, Y: -24}, canvas.Point{X: 20, Y: -24}, canvas.Point{Y: 17}, canvas.Point{X: -20, Y: -24})
		switch part.Kind {
		case "zener":
			stroke(ink, 3, canvas.Point{X: -22, Y: 11}, canvas.Point{X: -22, Y: 19}, canvas.Point{X: 22, Y: 19}, canvas.Point{X: 22, Y: 27})
		case "schottky":
			stroke(ink, 3, canvas.Point{X: -14, Y: 10}, canvas.Point{X: -22, Y: 10}, canvas.Point{X: -22, Y: 19}, canvas.Point{X: 22, Y: 19}, canvas.Point{X: 22, Y: 28}, canvas.Point{X: 14, Y: 28})
		default:
			stroke(ink, 3, canvas.Point{X: -22, Y: 19}, canvas.Point{X: 22, Y: 19})
		}
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
	case "logic-not", "logic-and", "logic-or", "logic-nand", "logic-nor", "logic-xor", "logic-xnor":
		for _, path := range workbench.LogicSymbolPaths(part.Kind) {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.LogicSymbolLabels(part) {
			first := len(b.vertices)
			b.text(label.At.X, label.At.Y, label.Text, muted)
			// Pin text rotates with this dense IC symbol so adjacent pin
			// captions retain their spacing at every component orientation.
			for i := first; i < len(b.vertices); i++ {
				at := b.vertices[i].Position
				d := workbench.RotateLocal(workbench.Point{X: at.X - label.At.X, Y: at.Y - label.At.Y}, part.Rotation)
				b.vertices[i].Position = canvas.Point{X: label.At.X + d.X, Y: label.At.Y + d.Y}
			}
		}
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		for _, path := range workbench.SequentialSymbolPaths(part.Kind) {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.SequentialSymbolLabels(part) {
			first := len(b.vertices)
			b.text(label.At.X, label.At.Y, label.Text, muted)
			// Pin text rotates with this dense IC symbol so adjacent pin
			// captions retain their spacing at every component orientation.
			for i := first; i < len(b.vertices); i++ {
				at := b.vertices[i].Position
				d := workbench.RotateLocal(workbench.Point{X: at.X - label.At.X, Y: at.Y - label.At.Y}, part.Rotation)
				b.vertices[i].Position = canvas.Point{X: label.At.X + d.X, Y: label.At.Y + d.Y}
			}
		}
	case "bcd-decoder", "seven-segment":
		for _, path := range workbench.DisplaySymbolPaths(part.Kind) {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
		if part.Kind == "seven-segment" {
			var lit [7]bool
			if state.Mixed != nil && !state.Mixed.Stale {
				for _, d := range state.Mixed.Displays {
					if d.ID == part.ID {
						lit = d.Lit
					}
				}
			}
			for k, path := range workbench.DisplaySegmentPaths() {
				c := muted
				if lit[k] {
					c = color.NRGBA{R: 255, G: 65, B: 50, A: 255}
				}
				points := make([]canvas.Point, len(path))
				for j, at := range path {
					points[j] = canvas.Point{X: at.X, Y: at.Y}
				}
				stroke(c, 7, points...)
			}
		}
		for _, label := range workbench.DisplaySymbolLabels(part) {
			first := len(b.vertices)
			b.text(label.At.X, label.At.Y, label.Text, muted)
			for i := first; i < len(b.vertices); i++ {
				at := b.vertices[i].Position
				d := workbench.RotateLocal(workbench.Point{X: at.X - label.At.X, Y: at.Y - label.At.Y}, part.Rotation)
				b.vertices[i].Position = canvas.Point{X: label.At.X + d.X, Y: label.At.Y + d.Y}
			}
		}
	case "timer555":
		for _, path := range workbench.Timer555SymbolPaths() {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.Timer555SymbolLabels(part) {
			first := len(b.vertices)
			b.text(label.At.X, label.At.Y, label.Text, muted)
			// Pin text rotates with this dense IC symbol so adjacent pin
			// captions retain their spacing at every component orientation.
			for i := first; i < len(b.vertices); i++ {
				at := b.vertices[i].Position
				d := workbench.RotateLocal(workbench.Point{X: at.X - label.At.X, Y: at.Y - label.At.Y}, part.Rotation)
				b.vertices[i].Position = canvas.Point{X: label.At.X + d.X, Y: label.At.Y + d.Y}
			}
		}
	case "opamp":
		for _, path := range workbench.OpAmpSymbolPaths() {
			points := make([]canvas.Point, len(path))
			for k, at := range path {
				points[k] = canvas.Point{X: at.X, Y: at.Y}
			}
			stroke(ink, 2.5, points...)
		}
		for _, label := range workbench.OpAmpSupplyLabels(part) {
			b.text(label.At.X, label.At.Y, label.Text, muted)
		}
	case "nmos", "pmos":
		stroke(wireColor, 2.5, canvas.Point{X: -40}, canvas.Point{X: -16})
		stroke(ink, 3, canvas.Point{X: -16, Y: -20}, canvas.Point{X: -16, Y: 20})
		for _, y := range []float64{-16, 0, 16} {
			stroke(ink, 3, canvas.Point{X: -7, Y: y - 4}, canvas.Point{X: -7, Y: y + 4})
		}
		stroke(ink, 2.5, canvas.Point{X: -7, Y: -16}, canvas.Point{X: 25, Y: -16}, canvas.Point{X: 25, Y: -35})
		stroke(ink, 2.5, canvas.Point{X: -7, Y: 16}, canvas.Point{X: 25, Y: 16}, canvas.Point{X: 25, Y: 35})
		stroke(ink, 2.5, canvas.Point{X: -7}, canvas.Point{X: 25}, canvas.Point{X: 25, Y: 16})
		if part.Kind == "pmos" {
			stroke(accent, 2.5, canvas.Point{X: -2, Y: -5}, canvas.Point{X: 7}, canvas.Point{X: -2, Y: 5})
		} else {
			stroke(accent, 2.5, canvas.Point{X: 4, Y: -5}, canvas.Point{X: -5}, canvas.Point{X: 4, Y: 5})
		}
		for _, pin := range []struct {
			x, y float64
			name string
		}{{-46, -20, "G"}, {32, -36, "D"}, {32, 18, "S"}} {
			at := point(pin.x, pin.y)
			b.text(at.X, at.Y, pin.name, muted)
		}
	case "npn", "pnp":
		stroke(ink, 3, canvas.Point{X: -12, Y: -20}, canvas.Point{X: -12, Y: 20})
		stroke(wireColor, 2.5, canvas.Point{X: -40}, canvas.Point{X: -12})
		stroke(ink, 2.5, canvas.Point{X: -12, Y: -10}, canvas.Point{X: 25, Y: -30}, canvas.Point{X: 25, Y: -35})
		stroke(ink, 2.5, canvas.Point{X: -12, Y: 10}, canvas.Point{X: 25, Y: 30}, canvas.Point{X: 25, Y: 35})
		if part.Kind == "pnp" {
			stroke(accent, 3, canvas.Point{X: 12, Y: -18}, canvas.Point{X: 0, Y: -17}, canvas.Point{X: 5, Y: -28})
		} else {
			stroke(accent, 3, canvas.Point{X: 12, Y: 15}, canvas.Point{X: 25, Y: 30}, canvas.Point{X: 5, Y: 25})
		}
		for _, pin := range []struct {
			x, y float64
			name string
		}{{-46, -20, "B"}, {32, -36, "C"}, {32, 18, "E"}} {
			if part.Kind == "pnp" {
				if pin.name == "C" {
					pin.name = "E"
				} else if pin.name == "E" {
					pin.name = "C"
				}
			}
			p := point(pin.x, pin.y)
			b.text(p.X, p.Y, pin.name, muted)
		}
	case "junction":
		if connected {
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
}
