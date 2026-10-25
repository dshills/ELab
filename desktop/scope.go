package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/dshills/vellum/drivers/cogent"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

// A live scope has a fixed rolling time axis. Native canvas pan/zoom handlers
// are swallowed here so every exposed interaction remains a shared command.
type scopeCanvas struct {
	*cogent.MeshCanvas
	screen     *screen
	mountEpoch uint64
	frame      atomic.Pointer[scopePlotFrame]
	gesture    *scopeGesture // Owner-held; only accepted cursor commands advance it.
	widget     *core.WidgetBase
}

func (c *scopeCanvas) Mount(parent tree.Node) core.Widget {
	widget := c.MeshCanvas.Mount(parent)
	if widget == nil {
		return nil
	}
	c.mountEpoch++
	c.frame.Store(nil)
	c.gesture = nil
	epoch := c.mountEpoch
	w := widget.AsWidget()
	owner, window, scoped := cogent.WindowScope(w)
	if !scoped || owner != c.Owner {
		return nil
	}
	c.widget = w
	w.OnFirst(events.Scroll, func(e events.Event) { forwardScopeScroll(w, e) })
	for _, kind := range []events.Types{events.MouseDown, events.MouseUp, events.SlideStart, events.SlideMove, events.SlideStop} {
		w.OnFirst(kind, func(e events.Event) {
			if c.screen != nil {
				frame := c.frame.Load()
				pos := e.Pos().Sub(w.Geom.Pos.Content.ToPoint())
				withWindowOwner(c.Owner, window, func() {
					if c.mountEpoch == epoch {
						c.cursorPointer(frame, kind, e.MouseButton(), float64(pos.X))
					}
				})
			}
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
func (s *screen) scopeMesh(v canvas.Viewport) (*canvas.Mesh, error) {
	s.scope.frame.Store(nil)
	b := &meshBuilder{}
	state := s.service.Snapshot()
	palette := s.colorsForTheme(state.Theme)
	muted := palette.Muted
	t := state.Transient
	plot := state.Scope
	step := state.TransientSettings.StepSeconds
	stale := false
	events := []workbench.InputEvent{}
	recorded := state.Recording != nil && state.Recording.Viewing
	var reference []workbench.ScopeTrace
	if recorded {
		plot = state.Recording.Scope
		step = state.Recording.Step
		if overlay := state.Recording.ReferenceOverlay; overlay != nil && overlay.Scope.WindowValid {
			reference = overlay.Scope.Traces
		}
	} else if t != nil {
		stale = t.Stale
		events = t.Events
	}
	if !recorded && (t == nil || len(t.Samples) == 0) {
		return canvas.NewMesh(nil, nil)
	}
	line := func(c color.NRGBA, width float64, points ...canvas.Point) {
		world := make([]canvas.Point, len(points))
		for i, p := range points {
			world[i] = v.World(p)
		}
		b.stroke(c, width, world...)
	}
	text := func(x, y float64, content string, c color.NRGBA) {
		p := v.World(canvas.Point{X: x, Y: y})
		b.text(p.X, p.Y, content, c)
	}
	vmin, vmax, imin, imax := scopeRanges(plot.Traces, reference)
	var currentLabels [5]string
	rightPadding := 78.
	for i := range currentLabels {
		currentLabels[i] = fmt.Sprintf("%.2g mA", imin+(imax-imin)*float64(i)/4)
		// Scientific notation for amp-level currents can be wider than the
		// ordinary milliamp labels. Reserve its actual bitmap-font advance.
		rightPadding = max(rightPadding, 6+workbench.CanvasTextWidth(currentLabels[i], 1.25)+4)
	}
	left, right, top, bottom := 58., v.Width-rightPadding, 12., v.Height-28
	if right <= left || bottom <= top {
		return canvas.NewMesh(nil, nil)
	}
	first, last := plot.Window.Start, plot.Window.End
	span := last - first
	if span <= 0 {
		span = step // A DC operating point has no time extent.
	}
	grid := palette.Grid
	for i := 0; i <= 4; i++ {
		fraction := float64(i) / 4
		py := bottom - (bottom-top)*fraction
		line(grid, 1, canvas.Point{X: left, Y: py}, canvas.Point{X: right, Y: py})
		text(4, py+4, fmt.Sprintf("%.3g V", vmin+(vmax-vmin)*fraction), muted)
		text(right+6, py+4, currentLabels[i], muted)
	}
	x := func(at float64) float64 { return left + (right-left)*(at-first)/span }
	for i, trace := range plot.Traces {
		points := []canvas.Point{}
		for _, p := range trace.Points {
			fraction := (p.Value - vmin) / (vmax - vmin)
			if trace.Probe.Unit == "A" {
				fraction = (p.Value*1000 - imin) / (imax - imin)
			}
			points = append(points, canvas.Point{X: x(p.Time), Y: bottom - (bottom-top)*fraction})
		}
		c := traceColor(probeColor(scopeProbeColors(plot.Traces)[i]), palette)
		if stale {
			c = muted
		}
		if len(points) > 1 {
			line(c, 2, points...)
		} else if len(points) == 1 {
			p := points[0]
			line(c, 3, canvas.Point{X: p.X - 2, Y: p.Y}, canvas.Point{X: p.X + 2, Y: p.Y})
		}
	}
	for _, trace := range reference {
		points := []canvas.Point{}
		for _, p := range trace.Points {
			fraction := (p.Value - vmin) / (vmax - vmin)
			if trace.Probe.Unit == "A" {
				fraction = (p.Value*1000 - imin) / (imax - imin)
			}
			points = append(points, canvas.Point{X: x(p.Time), Y: bottom - (bottom-top)*fraction})
		}
		for i, a := range plot.Traces {
			if a.Probe.ID != trace.Probe.ID {
				continue
			}
			c := traceColor(probeColor(scopeProbeColors(plot.Traces)[i]), palette)
			if len(points) == 1 {
				p := points[0]
				line(c, 2, canvas.Point{X: p.X - 3, Y: p.Y - 3}, canvas.Point{X: p.X + 3, Y: p.Y + 3})
				line(c, 2, canvas.Point{X: p.X - 3, Y: p.Y + 3}, canvas.Point{X: p.X + 3, Y: p.Y - 3})
			} else {
				dashedScopeLine(points, func(points ...canvas.Point) { line(c, 3, points...) })
			}
			break
		}
	}
	if recorded && state.Recording.Time >= first && state.Recording.Time <= last {
		px := x(state.Recording.Time)
		line(accent, 1.5, canvas.Point{X: px, Y: top}, canvas.Point{X: px, Y: bottom})
	}
	for _, cursor := range []struct {
		label string
		value *workbench.ScopeCursor
	}{{"A", plot.A}, {"B", plot.B}} {
		if cursor.value != nil && cursor.value.Valid && cursor.value.Time >= first && cursor.value.Time <= last {
			px := x(cursor.value.Time)
			line(ink, 1.5, canvas.Point{X: px, Y: top}, canvas.Point{X: px, Y: bottom})
			text(px+3, top+10, cursor.label, ink)
		}
	}
	for _, e := range events {
		if e.Time >= first && e.Time <= last && e.Mode != "restart" {
			px := x(e.Time)
			line(palette.Muted, 1, canvas.Point{X: px, Y: top}, canvas.Point{X: px, Y: bottom})
		}
	}
	text(left, bottom+22, fmt.Sprintf("%.4g s", first), muted)
	text(math.Max(left, right-80), bottom+22, fmt.Sprintf("%.4g s", last), muted)
	if b.err != nil {
		return nil, b.err
	}
	mesh, err := canvas.NewMesh(b.vertices, b.indices)
	if err == nil && state.ScopeVisible && !sourcePaneHidden(state) && plot.WindowValid && !stale {
		frame := &scopePlotFrame{revision: state.CommandRevision, generation: state.Document.Generation, left: left, right: right, start: first, end: last, cursor: plot.ActiveCursor}
		if recorded {
			frame.recordingID = state.Recording.ID
		} else if t != nil {
			frame.runID = t.RunID
		}
		s.scope.frame.Store(frame)
	}
	return mesh, err
}
func (s *screen) scopePanel(state workbench.Snapshot) ui.Node {
	if !state.ScopeVisible {
		return s.collapsedScope(state)
	}
	if state.Recording != nil && state.Recording.Viewing {
		return s.recordingPanel(state)
	}
	t := state.Transient
	npn := workbench.IsNPNTransientAnalysis(state.Analysis) || t != nil && workbench.IsNPNTransientEngine(t.Engine)
	caption := "Transient · waiting for a valid run"
	if t != nil {
		caption = fmt.Sprintf("t = %.4g s · τ = %.3g ms · run %d", t.Sample.Time, t.TimeConstant*1000, t.RunID)
		if npn {
			region := t.Sample.State
			if deviceID, _ := workbench.MixedRegion(state.Mixed); deviceID != "" {
				region = deviceID + " " + region
			}
			caption = fmt.Sprintf("t = %.4g s · %s · run %d", t.Sample.Time, region, t.RunID)
		}
		if t.Stale {
			caption += " · stale"
		} else if state.Paused {
			caption += " · paused"
		}
	}
	titleText := "Live RC scope"
	if npn {
		titleText = "Live NPN scope"
	}
	if t != nil && (t.Engine == workbench.MixedEngineID || t.Engine == workbench.NetworkEngineID || t.Engine == workbench.WaveformEngineID || t.Engine == workbench.InductorEngineID || t.Engine == workbench.Timer555EngineID || t.Engine == workbench.SequentialEngineID || t.Engine == workbench.ControlEngineID) {
		titleText = "Live circuit network"
	}
	if t != nil && t.Engine == workbench.Timer555EngineID {
		titleText = "Live 555 scope"
	}
	if state.LogicModel != nil {
		titleText = "Live logic scope"
	}
	if state.SequentialModel != nil {
		titleText = "Live sequential scope"
	}
	if state.ControlModel != nil {
		titleText = "Live control scope"
	}
	title := section("scope-title", titleText)
	title.MinWidth = 110
	captionNode := quiet("scope-caption", caption)
	captionNode.Style.MaxSize = ui.Some(ui.Size{Width: 240})
	controls := []ui.Node{s.scopeVisibilityButton(state), title, captionNode}
	modeLabels := map[string]string{"auto": "Auto", "charge": "Charge", "discharge": "Discharge", "high": "High", "low": "Low"}
	if t != nil && workbench.IsNetworkAnalysis(state.Analysis) {
		for _, c := range state.Components {
			if c.ID == t.SourceID && c.Kind == "dc-source" {
				modeLabels["high"], modeLabels["low"] = "Power on", "Power off"
			}
		}
	}
	for _, mode := range state.LiveInputModes {
		n := s.action("input-"+mode, modeLabels[mode], "simulation.input.set", workbench.InputParams{Mode: mode})
		n.Disabled = t == nil || t.Stale
		n = buttonStyle(n, t != nil && t.Mode == mode && t.PendingMode == "")
		controls = append(controls, n)
	}
	traceOptions := []ui.Option{{ID: "__none", Label: "Inspect trace…"}}
	for _, trace := range state.Scope.Traces {
		traceOptions = append(traceOptions, ui.Option{ID: trace.Probe.ID, Label: trace.Probe.Label})
	}
	focused := "__none"
	if state.TraceFocus != nil {
		focused = state.TraceFocus.ProbeID
	}
	traceSelector := ui.Node{ID: "scope-trace-focus", Kind: ui.Selector, Options: traceOptions, SelectedID: focused, Accessibility: ui.Accessibility{Name: "Inspect trace source"}, Change: func(e ui.ChangeEvent) {
		id := e.SelectedID
		if id == "__none" {
			id = ""
		}
		s.invoke("scope.trace.select", workbench.IDParams{ID: id})
	}}
	step := s.action("transient-step", "Step +1 sample", "simulation.step.start", workbench.StepParams{Duration: state.TransientSettings.StepSeconds})
	step.Disabled = !state.Paused || t == nil || t.Stale || state.StepWork != nil && (state.StepWork.Status == "queued" || state.StepWork.Status == "running" || state.StepWork.Status == "canceling")
	controls = append(controls, step, s.button("simulation.restart", "Restart"))
	stepStatus := ""
	if state.StepWork != nil {
		work := state.StepWork
		stepStatus = fmt.Sprintf("Step %s · %d/%d staged samples", work.Status, work.CompletedSamples, work.TotalSamples)
		if work.RunID != state.RunID {
			stepStatus = "Previous " + stepStatus
		}
		if work.Status == "queued" || work.Status == "running" {
			controls = append(controls, s.action("step-work-cancel", "Cancel step", "simulation.work.cancel", workbench.IDParams{ID: work.ID}))
		}
	}
	legendParts := []string{}
	for i, trace := range state.Scope.Traces {
		legendParts = append(legendParts, scopeProbeColors(state.Scope.Traces)[i]+": "+trace.Probe.Label)
	}
	for _, missing := range state.Scope.Unavailable {
		legendParts = append(legendParts, missing.ID+" unavailable")
	}
	legend := strings.Join(legendParts, " · ")
	if t != nil && t.PendingMode != "" {
		legend += " · pending: " + t.PendingMode
	}
	traceStyle := chromeStyle()
	traceStyle.MinSize = ui.Some(ui.Size{Width: 320, Height: 32})
	traceStyle.MaxSize = ui.Some(ui.Size{Height: 32})
	traceSelector.Style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 10, Right: 10})
	export := ui.Node{ID: "export-csv", Kind: ui.Button, Text: "Export CSV", Accessibility: ui.Accessibility{Name: "Export scope CSV"}, Disabled: t == nil || t.Stale || !state.Scope.WindowValid, Activate: func() { s.beginFile("export.scope.csv", true, workbench.SaveParams{}, nil) }}
	export.Style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 10, Right: 10})
	s.scope.Revision = s.scopeAppearance.next(state.Revision, s.colorsForTheme(state.Theme))
	return ui.Node{ID: "scope-panel", Kind: ui.Column, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 288}), Padding: ui.Some(ui.UniformInsets(10)), Gap: ui.Some(float32(4)), Background: ui.Some(paper)}, Children: []ui.Node{
		{ID: "scope-toolbar", Kind: ui.Toolbar, Grow: true, Style: chromeStyle(), Children: controls},
		{ID: "rc-scope", Kind: ui.Extension, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 120}), MaxSize: ui.Some(ui.Size{Height: 120})}, Custom: s.scope}, {ID: "scope-trace-row", Kind: ui.Toolbar, Style: traceStyle, Children: []ui.Node{export, traceSelector, quiet("scope-legend", legend)}}, quiet("step-work-status", stepStatus), s.cursorControls(state),
	}}
}
func (s *screen) transientInspector(state workbench.Snapshot) []ui.Node {
	npn := workbench.IsNPNTransientAnalysis(state.Analysis) || state.Transient != nil && workbench.IsNPNTransientEngine(state.Transient.Engine)
	title, help := "RC experiment", "Ideal RC response. Changing settings restarts from the initial voltage."
	if npn {
		title = "Pulse experiment"
		help = "Instantaneous NPN response. Settings restart time; no device delay or storage."
	}
	mixed := workbench.IsNetworkAnalysis(state.Analysis) || state.Transient != nil && (state.Transient.Engine == workbench.NetworkEngineID || state.Transient.Engine == workbench.WaveformEngineID || state.Transient.Engine == workbench.InductorEngineID || state.Transient.Engine == workbench.Timer555EngineID || state.Transient.Engine == workbench.SequentialEngineID || state.Transient.Engine == workbench.ControlEngineID)
	if mixed {
		title = "Network experiment"
		help = "Capacitor state evolves with the coupled circuit. Select each C to set signed startup voltage."
	}
	if state.Analysis == workbench.InductorAnalysis {
		title = "RLC experiment"
		help = "Coil current and capacitor voltage stay continuous at source/switch events. Select L or C to set signed startup state."
	}
	if state.Analysis == workbench.Timer555Analysis {
		title = "555 experiment"
		help = "External timing storage controls the latch. Restart restores capacitor startup and clears the latch before settling its inputs."
	}
	if state.LogicModel != nil {
		title = "Logic experiment"
		help = "Combinational gates have no internal delay. Intermediate input voltages are analog transitions; external capacitors retain circuit timing."
	}
	if state.SequentialModel != nil {
		title = "Sequential experiment"
		help = "Rising clock edges sample valid inputs. Reset is asynchronous and active high. Restart clears memory; power-up establishes clock without an edge."
	}
	if state.ControlModel != nil {
		title = "Controlled experiment"
		help = "Buttons press/release at the accepted time; simulation time advances bounce and automatic release. Clock/reset states preserve capacitor and coil energy. Restart clears runtime controls."
	}
	nodes := []ui.Node{section("transient-title", title), quiet("transient-help", help)}
	if state.StepWork != nil && state.StepWork.Error != nil {
		nodes = append(nodes, quiet("step-work-error", state.StepWork.Error.Message))
	}
	cfg := state.TransientSettings
	for _, setting := range []struct {
		id, text               string
		value, scale, min, max float64
		set                    func(*workbench.TransientSettings, float64)
	}{
		{"step", "Sample step", cfg.StepSeconds, 1000, .001, 1000, func(c *workbench.TransientSettings, v float64) { c.StepSeconds = v }},
		{"high", "Pulse high", cfg.HighSeconds, 1000, .001, 10000, func(c *workbench.TransientSettings, v float64) { c.HighSeconds = v }},
		{"low", "Pulse low", cfg.LowSeconds, 1000, .001, 10000, func(c *workbench.TransientSettings, v float64) { c.LowSeconds = v }},
		{"initial", "Initial voltage · V", cfg.InitialVoltage, 1, 0, 12, func(c *workbench.TransientSettings, v float64) { c.InitialVoltage = v }},
		{"speed", "Playback · sim s / wall s", cfg.PlaybackSpeed, 1, .001, 1, func(c *workbench.TransientSettings, v float64) { c.PlaybackSpeed = v }},
	} {
		setting := setting
		if mixed && (setting.id == "high" || setting.id == "low") {
			hasPulse := false
			for _, c := range state.Components {
				if c.Kind == "pulse-source" {
					hasPulse = true
				}
			}
			if !hasPulse {
				continue
			}
		}
		if npn && !mixed && setting.id == "initial" {
			continue
		}
		if setting.scale == 1000 {
			field := map[string]string{"step": "step_s", "high": "high_s", "low": "low_s"}[setting.id]
			nodes = append(nodes, quiet("setting-label-"+setting.id, setting.text), timeField("setting-"+setting.id, setting.text, setting.value, false, func(text string) {
				s.invokeAtCommandRevision("simulation.transient.set", textFieldParams(cfg, field, text), state.CommandRevision)
			}))
			continue
		}
		nodes = append(nodes, quiet("setting-label-"+setting.id, setting.text), ui.Node{ID: "setting-" + setting.id, Kind: ui.NumberField, Number: float32(setting.value * setting.scale), Numeric: ui.Numeric{Min: float32(setting.min), Max: float32(setting.max), Step: float32(.001)}, Accessibility: ui.Accessibility{Name: setting.text}, Change: func(e ui.ChangeEvent) {
			next := cfg
			// Vellum number events are float32. Map displayed range endpoints
			// back to their declared float64 values, so 0.001 is not above
			// the exact minimum playback/step boundary after widening.
			value := float64(e.Number)
			if e.Number == float32(setting.min) {
				value = setting.min
			}
			if e.Number == float32(setting.max) {
				value = setting.max
			}
			setting.set(&next, value/setting.scale)
			s.invoke("simulation.transient.set", next)
		}})
	}
	return nodes
}
