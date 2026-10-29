package main

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

var sweepBitmapUnits = strings.NewReplacer("µ", "u", "°", "deg ")

func sweepVisible(state workbench.Snapshot) bool {
	return state.DCSweep != nil && state.DCSweep.Visible
}
func sourcePaneHidden(state workbench.Snapshot) bool {
	return previewVisible(state) || sweepVisible(state)
}
func selectedSweep(state *workbench.SweepState) *workbench.SweepSummary {
	if state != nil {
		for _, op := range state.Operations {
			if op.ID == state.Selected {
				v := op
				return &v
			}
		}
	}
	return nil
}
func (s *screen) ensureSweepCanvas() {
	if s.sweepDrawing != nil {
		return
	}
	view := canvas.Viewport{Width: 800, Height: 500, Zoom: 1}
	s.sweepDrawing = &sweepCanvas{MeshCanvas: &cogent.MeshCanvas{Owner: s.app, Window: "main", View: &view, Prepare: s.sweepMesh}, screen: s}
}
func (s *screen) sweepInspector(state workbench.Snapshot) []ui.Node {
	sweep := state.DCSweep
	if sweep == nil {
		return nil
	}
	s.pruneSweepFieldRejections(state)
	cfg := sweep.Settings
	nodes := []ui.Node{section("sweep-settings-title", "Frequency, noise & DC sweeps"), quiet("sweep-settings-help", "AC response, white noise or DC points."), quiet("sweep-live-help", "Live circuit unchanged.")}
	nodes = append(nodes, buttonStyle(ui.Node{ID: "sweep-import", Kind: ui.Button, Text: "Open sweep JSON…", Accessibility: ui.Accessibility{Name: "Open captured frequency or DC sweep JSON"}, Disabled: s.fileBusy || s.pendingCommand != "", Activate: func() { s.beginFile("sweep.import", true, workbench.RecordingImportParams{}, nil) }}, false))
	options := []ui.Option{}
	selected := cfg.ComponentID + ":" + cfg.Parameter
	valid := false
	for _, t := range sweep.Targets {
		id := t.ComponentID + ":" + t.Parameter
		options = append(options, ui.Option{ID: id, Label: t.Label})
		if id == selected {
			valid = true
		}
	}
	if !valid {
		options = append(options, ui.Option{ID: selected, Label: "Choose target", Disabled: true})
	}
	nodes = append(nodes, ui.Node{ID: "sweep-target", Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Sweep parameter"}, Change: func(e ui.ChangeEvent) {
		pieces := strings.SplitN(e.SelectedID, ":", 2)
		if len(pieces) != 2 {
			return
		}
		next := cfg
		next.ComponentID, next.Parameter = pieces[0], pieces[1]
		next.ProbeIDs = nil
		next.Spacing = ""
		switch next.Parameter {
		case "frequency", "loop-frequency", "noise":
			next.Start, next.Stop, next.Spacing = "1 Hz", "100 kHz", "logarithmic"
		case "resistance":
			next.Start, next.Stop = "100 Ω", "10k Ω"
		case "light":
			next.Start, next.Stop = "0", "100"
		case "temperature":
			next.Start, next.Stop = "-20", "80"
		case "wiper":
			next.Start, next.Stop = "0", "1"
		default:
			next.Start, next.Stop = "0 V", "5 V"
		}
		s.invoke("sweep.settings.set", next)
	}})
	spacing := cfg.Spacing
	if spacing == "" {
		spacing = "linear"
	}
	nodes = append(nodes, ui.Node{ID: "sweep-spacing", Kind: ui.Selector, Options: []ui.Option{{ID: "linear", Label: "Linear spacing"}, {ID: "logarithmic", Label: "Logarithmic spacing"}}, SelectedID: spacing, Accessibility: ui.Accessibility{Name: "Sweep point spacing"}, Change: func(e ui.ChangeEvent) {
		next := cfg
		next.Spacing = e.SelectedID
		s.invoke("sweep.settings.set", next)
	}})
	for _, field := range []struct{ id, label, value string }{{"start", "Start", cfg.Start}, {"stop", "Stop", cfg.Stop}} {
		field := field
		value := field.value
		if rejected, ok := s.sweepFieldRejected["sweep-range-"+field.id]; ok {
			value = rejected.text
		}
		nodes = append(nodes, quiet("sweep-"+field.id+"-label", field.label), ui.Node{ID: "sweep-range-" + field.id, Kind: ui.TextField, Text: value, ControlStateKey: sweepSettingsOwner(cfg), Accessibility: ui.Accessibility{Name: "Sweep " + field.label}, Change: func(e ui.ChangeEvent) {
			s.changeSweepField(state, "sweep-range-"+field.id, e.Text)
		}})
	}
	points := strconv.Itoa(cfg.Points)
	if rejected, ok := s.sweepFieldRejected["sweep-points"]; ok {
		points = rejected.text
	}
	nodes = append(nodes, quiet("sweep-points-label", "Points · 2–201 · inclusive endpoints"), ui.Node{ID: "sweep-points", Kind: ui.TextField, Text: points, ControlStateKey: sweepSettingsOwner(cfg), Accessibility: ui.Accessibility{Name: "Sweep point count"}, Change: func(e ui.ChangeEvent) {
		s.changeSweepField(state, "sweep-points", e.Text)
	}})
	probeOptions := []ui.Option{{ID: "__auto", Label: "Auto readings"}}
	for _, p := range sweep.Available {
		if cfg.Parameter == "loop-frequency" && p.ID != "loop:"+cfg.ComponentID {
			continue
		}
		if (cfg.Parameter == "frequency" || cfg.Parameter == "noise") && p.Unit != "V" {
			continue
		}
		probeOptions = append(probeOptions, ui.Option{ID: p.ID, Label: p.Label + " · " + p.Unit})
	}
	choice := "__auto"
	if len(cfg.ProbeIDs) > 0 {
		choice = cfg.ProbeIDs[0]
		found := false
		for _, option := range probeOptions {
			if option.ID == choice {
				found = true
			}
		}
		if !found {
			probeOptions = append(probeOptions, ui.Option{ID: choice, Label: "Missing probe " + choice, Disabled: true})
		}
	}
	nodes = append(nodes, ui.Node{ID: "sweep-probe", Kind: ui.Selector, Options: probeOptions, SelectedID: choice, Accessibility: ui.Accessibility{Name: "Next sweep reading"}, Change: func(e ui.ChangeEvent) {
		next := cfg
		next.ProbeIDs = nil
		if e.SelectedID != "__auto" {
			next.ProbeIDs = append([]string{}, cfg.ProbeIDs...)
			for _, id := range next.ProbeIDs {
				if id == e.SelectedID {
					return
				}
			}
			limit := workbench.MaxScopeProbes
			if workbench.IsSpectralSweep(cfg.Parameter) {
				limit = 3
			}
			if len(next.ProbeIDs) >= limit {
				s.errorMessage = "Remove a sweep reading before adding another"
				s.app.Invalidate("main")
				return
			}
			next.ProbeIDs = append(next.ProbeIDs, e.SelectedID)
		}
		s.invoke("sweep.settings.set", next)
	}})
	if len(cfg.ProbeIDs) > 0 {
		nodes = append(nodes, quiet("sweep-readings-list", "Next readings: "+strings.Join(cfg.ProbeIDs, ", ")))
		for _, id := range cfg.ProbeIDs {
			next := cfg
			next.ProbeIDs = []string{}
			for _, other := range cfg.ProbeIDs {
				if other != id {
					next.ProbeIDs = append(next.ProbeIDs, other)
				}
			}
			if len(next.ProbeIDs) == 0 {
				next.ProbeIDs = nil
			}
			nodes = append(nodes, s.action("sweep-probe-remove-"+id, "Remove "+id, "sweep.settings.set", next))
		}
	}

	active := false
	for _, op := range sweep.Operations {
		if op.Status == "queued" || op.Status == "running" || op.Status == "canceling" {
			active = true
			cancel := s.action("sweep-cancel-"+op.ID, "Cancel sweep", "sweep.cancel", workbench.IDParams{ID: op.ID})
			cancel.Disabled = op.Status == "canceling"
			nodes = append(nodes, quiet("sweep-progress-"+op.ID, fmt.Sprintf("%s · %d/%d points staged", op.Status, op.Completed, op.Total)), cancel)
		}
	}
	start := s.sweepStartAction(state)
	_, _, diagnostic := (workbench.Graph{Components: state.Components, Wires: state.Wires}).Compile()
	start.Disabled = start.Disabled || active || !valid || diagnostic != nil
	if cfg.Parameter == "noise" {
		nodes = append(nodes, start, quiet("sweep-assumptions", "White noise · linearized DC bias"), quiet("sweep-source-help", "Declared independent sources; no flicker or avalanche"), quiet("sweep-time-help", "RMS integrates sampled linear PSD across this band"))
	} else if cfg.Parameter == "loop-frequency" {
		nodes = append(nodes, start, quiet("sweep-assumptions", "Loop return ratio · closed-bias control injection"), quiet("sweep-source-help", "One dynamic op-amp differential control; loading retained"), quiet("sweep-time-help", "Other loops stay closed; no stability certification"))
	} else if cfg.Parameter == "frequency" {
		nodes = append(nodes, start, quiet("sweep-assumptions", "AC · small signal at DC bias"), quiet("sweep-source-help", "Selected source: normalized 1 V"), quiet("sweep-time-help", "Other sources: AC ground"), quiet("sweep-ac-bandwidth", "Bandwidth follows configured op-amp dynamics"))
	} else {
		nodes = append(nodes, start, quiet("sweep-assumptions", "Capacitors: open"), quiet("sweep-source-help", "Sine: offset / pulse: low"), quiet("sweep-time-help", "Startup/time ignored"))
	}
	if len(sweep.Operations) > 0 {
		ops := []ui.Option{}
		for _, op := range sweep.Operations {
			ops = append(ops, ui.Option{ID: op.ID, Label: op.ID + " · " + op.Status})
		}
		nodes = append(nodes, ui.Node{ID: "sweep-record", Kind: ui.Selector, Options: ops, SelectedID: sweep.Selected, Accessibility: ui.Accessibility{Name: "Captured sweep"}, Change: func(e ui.ChangeEvent) { s.invoke("sweep.select", workbench.IDParams{ID: e.SelectedID}) }})
		nodes = append(nodes, s.action("sweep-show", "View sweep", "sweep.visibility.set", workbench.SweepVisibilityParams{Visible: true}))
		if op := selectedSweep(sweep); op != nil {
			if op.Error != nil {
				nodes = append(nodes, quiet("sweep-error", op.Error.Message))
			}
			if op.Status == "completed" {
				nodes = append(nodes, ui.Node{ID: "sweep-keep", Kind: ui.Checkbox, Text: "Keep in project", Checked: op.Kept, Accessibility: ui.Accessibility{Name: "Keep in project", Help: "Save this captured sweep with the project."}, Change: func(e ui.ChangeEvent) {
					s.invoke("sweep.keep.set", workbench.RecordingKeepParams{ID: op.ID, Keep: e.Checked})
				}})
				for _, format := range []string{"json", "csv"} {
					format := format
					nodes = append(nodes, buttonStyle(ui.Node{ID: "sweep-export-" + format, Kind: ui.Button, Text: "Export sweep " + strings.ToUpper(format), Accessibility: ui.Accessibility{Name: "Export sweep " + format}, Activate: func() {
						s.beginFile("sweep.export", true, workbench.VerificationExportParams{ID: op.ID, Format: format}, nil)
					}}, false))
				}
			}
		}
	}
	if sweep.Result != nil && sweep.Result.Noise != nil {
		nodes = append(nodes, noiseReadouts(sweep.Result)...)
	}
	if sweep.Result != nil && sweep.Result.Loop != nil {
		nodes = append(nodes, loopMarginReadouts(sweep.Result.Loop)...)
	}
	return s.focusSimulationForm(state, "sweeps", nodes)
}
func (s *screen) sweepPanel(state workbench.Snapshot) []ui.Node {
	sweep := state.DCSweep
	s.ensureSweepCanvas()
	s.sweepDrawing.Revision = s.sweepAppearance.next(state.Revision, s.colorsForTheme(state.Theme))
	toolbar := ui.Node{ID: "sweep-toolbar", Kind: ui.Toolbar, Style: chromeStyle(), Children: []ui.Node{heading("sweep-title", "Frequency, noise & DC sweeps"), s.action("sweep-back", "Back to circuit", "sweep.visibility.set", workbench.SweepVisibilityParams{Visible: false})}}
	nodes := []ui.Node{toolbar}
	op := selectedSweep(sweep)
	if op == nil {
		return append(nodes, quiet("sweep-no-record", "Build a supported circuit, choose a range, then run a sweep."))
	}
	caption := op.ID + " · " + op.Status + " · independent DC points"
	if sweep.Result != nil && workbench.IsFrequencySweep(sweep.Result.Target.Parameter) {
		caption = op.ID + " · " + op.Status + " · AC small-signal at DC bias"
	}
	if sweep.Result != nil && sweep.Result.Loop != nil {
		caption = op.ID + " · " + op.Status + " · differential loop return ratio"
	}
	if op.Kind == "sweep-import" {
		caption += " · imported recorded data"
	}
	if !op.SourceUnchanged {
		caption += " · captured result; current-source association unavailable"
	}
	nodes = append(nodes, quiet("sweep-caption", caption))
	if sweep.Result == nil {
		text := fmt.Sprintf("%d/%d points staged. Partial results are never published.", op.Completed, op.Total)
		if op.Error != nil {
			text = op.Error.Message
		}
		nodes = append(nodes, quiet("sweep-pending", text))
		return nodes
	}
	result := sweep.Result
	for j, band := range result.AC {
		nodes = append(nodes, quiet(fmt.Sprintf("sweep-ac-peak-%d", j), fmt.Sprintf("%s · sampled peak %.5g dB at %.5g Hz", band.ProbeID, band.PeakDB, band.PeakFrequency)))
		for k, f := range band.HalfPowerHz {
			nodes = append(nodes, quiet(fmt.Sprintf("sweep-ac-cutoff-%d-%d", j, k), fmt.Sprintf("Half-power estimate %.5g Hz", f)))
		}
	}
	yLabel := ""
	for _, p := range result.Probes {
		if p.ID == sweep.TraceID {
			_, unit := sweepDisplayUnit(p.Unit)
			yLabel = p.Label + " · " + unit
		}
	}
	options := []ui.Option{}
	for _, p := range result.Probes {
		options = append(options, ui.Option{ID: p.ID, Label: p.Label + " · " + p.Unit})
	}
	nodes = append(nodes, ui.Node{ID: "sweep-trace", Kind: ui.Selector, Options: options, SelectedID: sweep.TraceID, Accessibility: ui.Accessibility{Name: "Sweep plotted reading"}, Change: func(e ui.ChangeEvent) { s.invoke("sweep.trace.select", workbench.IDParams{ID: e.SelectedID}) }}, quiet("sweep-axis", fmt.Sprintf("%s · %s · %.9g → %.9g %s", result.Target.ComponentID, result.Target.Parameter, result.Start, result.Stop, result.Target.Unit)+" · "+sweepSpacingLabel(result.Spacing)+" · "+yLabel), ui.Node{ID: "sweep-plot", Kind: ui.Extension, Grow: true, Custom: s.sweepDrawing, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 200})}})
	if sweep.Cursor != nil && *sweep.Cursor < len(result.Points) {
		p := result.Points[*sweep.Cursor]
		nodes = append(nodes, quiet("sweep-readout", fmt.Sprintf("Point %d · %.9g %s", *sweep.Cursor+1, p.Input, result.Target.Unit)))
		for j, z := range p.AC {
			if !z.PhaseDefined {
				nodes = append(nodes, quiet(fmt.Sprintf("sweep-ac-zero-%d", j), "Zero transfer · phase undefined"))
			}
		}
		for i, probe := range result.Probes {
			nodes = append(nodes, quiet("sweep-reading-"+probe.ID, fmt.Sprintf("%s · %.9g %s", probe.ID, p.Values[i], probe.Unit)))
		}
	} else {
		nodes = append(nodes, quiet("sweep-readout", "Click a calculated point. Values are never interpolated."))
	}
	nodes = append(nodes, s.action("sweep-cursor-clear", "Clear point", "sweep.cursor.set", workbench.SweepCursorParams{OperationID: sweep.Selected}))

	return nodes
}
func (s *screen) sweepMesh(v canvas.Viewport) (*canvas.Mesh, error) {
	state := s.service.Snapshot()
	b := &meshBuilder{}
	sweep := state.DCSweep
	if sweep == nil || sweep.Result == nil {
		if s.sweepDrawing != nil {
			s.sweepDrawing.frame.Store(nil)
		}
		return canvas.NewMesh(nil, nil)
	}
	result := sweep.Result
	index := -1
	for i, p := range result.Probes {
		if p.ID == sweep.TraceID {
			index = i
			break
		}
	}
	if index < 0 {
		return canvas.NewMesh(nil, nil)
	}
	left, right, top, bottom := 66., v.Width-28, 20., v.Height-40
	if right <= left || bottom <= top {
		return canvas.NewMesh(nil, nil)
	}
	palette := s.colorsForTheme(state.Theme)
	probe := result.Probes[index]
	factor, _ := sweepDisplayUnit(probe.Unit)
	xmin, xmax := math.Min(result.Start, result.Stop), math.Max(result.Start, result.Stop)
	ymin, ymax := math.Inf(1), math.Inf(-1)
	for _, p := range result.Points {
		value := p.Values[index] * factor
		ymin = math.Min(ymin, value)
		ymax = math.Max(ymax, value)
	}
	if ymin == ymax {
		pad := math.Max(1e-3, math.Abs(ymin)*.1)
		ymin -= pad
		ymax += pad
	} else {
		pad := (ymax - ymin) * .05
		ymin -= pad
		ymax += pad
	}
	line := func(c color.NRGBA, width float64, points ...canvas.Point) {
		world := make([]canvas.Point, len(points))
		for i, p := range points {
			world[i] = v.World(p)
		}
		b.stroke(c, width, world...)
	}
	text := func(x, y float64, label string) {
		p := v.World(canvas.Point{X: x, Y: y})
		b.text(p.X, p.Y, label, palette.Muted)
	}
	for i := 0; i <= 4; i++ {
		f := float64(i) / 4
		x := left + (right-left)*f
		y := bottom - (bottom-top)*f
		line(palette.Grid, 1, canvas.Point{X: left, Y: y}, canvas.Point{X: right, Y: y})
		text(2, y-5, fmt.Sprintf("%.4g", ymin+(ymax-ymin)*f))
		label := fmt.Sprintf("%.4g", result.AxisValue(f))
		width := workbench.CanvasTextWidth(label, 1.25)
		text(math.Max(2, math.Min(x-width/2, v.Width-width-6)), bottom+12, label)
	}
	text(left, bottom+27, sweepBitmapUnits.Replace(result.Target.Unit))
	points := make([]canvas.Point, len(result.Points))
	inputs := make([]float64, len(result.Points))
	for i, p := range result.Points {
		inputs[i] = result.AxisFraction(p.Input)
		points[i] = canvas.Point{X: left + inputs[i]*(right-left), Y: bottom - (p.Values[index]*factor-ymin)/(ymax-ymin)*(bottom-top)}
	}
	line(palette.Accent, 2, points...)
	if sweep.Cursor != nil && *sweep.Cursor < len(points) {
		p := points[*sweep.Cursor]
		line(palette.Accent, 1, canvas.Point{X: p.X, Y: top}, canvas.Point{X: p.X, Y: bottom})
		point := v.World(p)
		b.quad(point.X-3, point.Y-3, 6, 6, palette.Accent)
	}
	if w := s.sweepDrawing.widget; w != nil && s.drawing.pointer != nil && sweepVisible(state) && !previewVisible(state) {
		origin := w.Geom.Pos.Content
		if w.Scene != nil {
			origin.X += float32(w.Scene.SceneGeom.Pos.X)
			origin.Y += float32(w.Scene.SceneGeom.Pos.Y)
		}
		s.drawing.pointer.publish(float64(origin.X), float64(origin.Y), v, "sweep", fmt.Sprintf("%s/%d/%d", sweep.Selected, state.CommandRevision, state.Document.Generation))
	}
	s.sweepDrawing.frame.Store(&sweepPlotFrame{id: sweep.Selected, revision: state.CommandRevision, generation: state.Document.Generation, left: left, right: right, min: xmin, max: xmax, inputs: inputs})
	return canvas.NewMesh(b.vertices, b.indices)
}

func sweepSpacingLabel(spacing string) string {
	if spacing == "logarithmic" {
		return "log axis"
	}
	return "linear axis"
}

func sweepDisplayUnit(unit string) (float64, string) {
	switch unit {
	case "A":
		return 1000, "mA (display)"
	case "V/√Hz":
		return 1e9, "nV/√Hz (display)"
	}
	return 1, unit
}
