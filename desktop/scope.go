package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

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
	screen *screen
}

func (c *scopeCanvas) Mount(parent tree.Node) core.Widget {
	widget := c.MeshCanvas.Mount(parent)
	if widget == nil {
		return nil
	}
	w := widget.AsWidget()
	w.OnFirst(events.Scroll, func(e events.Event) { e.SetHandled() })
	for _, kind := range []events.Types{events.MouseDown, events.SlideMove} {
		w.OnFirst(kind, func(e events.Event) {
			if c.screen != nil {
				pos := e.Pos().Sub(w.Geom.Pos.Content.ToPoint())
				c.Owner.Do(func() { c.placeCursor(float64(pos.X)) })
			}
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
func (s *screen) scopeMesh(v canvas.Viewport) (*canvas.Mesh, error) {
	b := &meshBuilder{}
	state := s.service.Snapshot()
	t := state.Transient
	plot := state.Scope
	step := state.TransientSettings.StepSeconds
	stale := false
	events := []workbench.InputEvent{}
	recorded := state.Recording != nil && state.Recording.Viewing
	if recorded {
		plot = state.Recording.Scope
		step = state.Recording.Step
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
	left, right, top, bottom := 58., v.Width-78, 12., v.Height-28
	if right <= left || bottom <= top {
		return canvas.NewMesh(nil, nil)
	}
	first, last := plot.Window.Start, plot.Window.End
	span := math.Max(step, last-first)
	vmin, vmax, imin, imax := 0., 1., 0., 1.
	for _, trace := range plot.Traces {
		for _, p := range trace.Points {
			if trace.Probe.Unit == "A" {
				imin = math.Min(imin, p.Value*1000)
				imax = math.Max(imax, p.Value*1000)
			} else {
				vmin = math.Min(vmin, p.Value)
				vmax = math.Max(vmax, p.Value)
			}
		}
	}
	vmin = math.Floor(vmin)
	vmax = math.Ceil(vmax)
	imin = math.Floor(imin)
	imax = math.Ceil(imax)
	grid := color.NRGBA{R: 228, G: 235, B: 240, A: 255}
	for i := 0; i <= 4; i++ {
		fraction := float64(i) / 4
		py := bottom - (bottom-top)*fraction
		line(grid, 1, canvas.Point{X: left, Y: py}, canvas.Point{X: right, Y: py})
		text(4, py+4, fmt.Sprintf("%.3g V", vmin+(vmax-vmin)*fraction), muted)
		text(right+6, py+4, fmt.Sprintf("%.2g mA", imin+(imax-imin)*fraction), muted)
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
		c := probeColor(scopeProbeColors(plot.Traces)[i])
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
			line(color.NRGBA{R: 192, G: 208, B: 216, A: 255}, 1, canvas.Point{X: px, Y: top}, canvas.Point{X: px, Y: bottom})
		}
	}
	text(left, bottom+22, fmt.Sprintf("%.4g s", first), muted)
	text(math.Max(left, right-80), bottom+22, fmt.Sprintf("%.4g s", last), muted)
	if b.err != nil {
		return nil, b.err
	}
	return canvas.NewMesh(b.vertices, b.indices)
}
func (s *screen) scopePanel(state workbench.Snapshot) ui.Node {
	if state.Recording != nil && state.Recording.Viewing {
		return s.recordingPanel(state)
	}
	t := state.Transient
	npn := workbench.IsNPNTransientAnalysis(state.Analysis) || t != nil && workbench.IsNPNTransientEngine(t.Engine)
	caption := "Transient · waiting for a valid run"
	if t != nil {
		caption = fmt.Sprintf("t = %.4g s · τ = %.3g ms · run %d", t.Sample.Time, t.TimeConstant*1000, t.RunID)
		if npn {
			caption = fmt.Sprintf("t = %.4g s · %s · run %d", t.Sample.Time, t.Sample.State, t.RunID)
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
	if t != nil && (t.Engine == workbench.MixedEngineID || t.Engine == workbench.NetworkEngineID || t.Engine == workbench.WaveformEngineID) {
		titleText = "Live circuit network"
	}
	title := section("scope-title", titleText)
	title.MinWidth = 110
	captionNode := quiet("scope-caption", caption)
	captionNode.MinWidth = 300
	controls := []ui.Node{title, captionNode}
	modes := []struct{ id, text string }{{"auto", "Auto"}, {"charge", "Charge"}, {"discharge", "Discharge"}}
	if npn {
		modes = []struct{ id, text string }{{"auto", "Auto"}, {"high", "High"}, {"low", "Low"}}
	}
	if t != nil && (t.Engine == workbench.MixedEngineID || t.Engine == workbench.NetworkEngineID || t.Engine == workbench.WaveformEngineID) {
		for _, c := range state.Components {
			if c.ID == t.SourceID && c.Kind == "dc-source" {
				modes = []struct{ id, text string }{{"auto", "Auto"}, {"high", "Power on"}, {"low", "Power off"}}
			}
		}
	}
	for _, mode := range modes {
		n := s.action("input-"+mode.id, mode.text, "simulation.input.set", workbench.InputParams{Mode: mode.id})
		n.Disabled = t == nil || t.Stale
		n = buttonStyle(n, t != nil && t.Mode == mode.id && t.PendingMode == "")
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
	s.scope.Revision = state.Revision
	return ui.Node{ID: "scope-panel", Kind: ui.Column, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 240}), Padding: ui.Some(ui.UniformInsets(10)), Gap: ui.Some(float32(4)), Background: ui.Some(paper)}, Children: []ui.Node{
		{ID: "scope-toolbar", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 36}), Gap: ui.Some(float32(6)), Align: ui.Some(ui.AlignCenter)}, Children: controls},
		{ID: "rc-scope", Kind: ui.Extension, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 120}), MaxSize: ui.Some(ui.Size{Height: 120})}, Custom: s.scope}, {ID: "scope-trace-row", Kind: ui.Row, Style: ui.Style{Gap: ui.Some(float32(8))}, Children: []ui.Node{traceSelector, quiet("scope-legend", legend)}}, quiet("step-work-status", stepStatus), s.cursorControls(state),
	}}
}
func (s *screen) transientInspector(state workbench.Snapshot) []ui.Node {
	npn := workbench.IsNPNTransientAnalysis(state.Analysis) || state.Transient != nil && workbench.IsNPNTransientEngine(state.Transient.Engine)
	title, help := "RC experiment", "Ideal RC response. Changing settings restarts from the initial voltage."
	if npn {
		title = "Pulse experiment"
		help = "Instantaneous NPN response. Settings restart time; no device delay or storage."
	}
	mixed := workbench.IsNetworkAnalysis(state.Analysis) || state.Transient != nil && (state.Transient.Engine == workbench.NetworkEngineID || state.Transient.Engine == workbench.WaveformEngineID)
	if mixed {
		title = "Network experiment"
		help = "Capacitor state evolves with the coupled circuit. Select each C to set signed startup voltage."
	}
	nodes := s.probeInspector(state)
	nodes = append(nodes, section("transient-title", title), quiet("transient-help", help))
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
				s.invokeAtCommandRevision("simulation.transient.set", timeParams(cfg, field, text), state.CommandRevision)
			}))
			continue
		}
		nodes = append(nodes, quiet("setting-label-"+setting.id, setting.text), ui.Node{ID: "setting-" + setting.id, Kind: ui.NumberField, Number: float32(setting.value * setting.scale), Numeric: ui.Numeric{Min: float32(setting.min), Max: float32(setting.max), Step: float32(.001)}, Accessibility: ui.Accessibility{Name: setting.text}, Change: func(e ui.ChangeEvent) {
			next := cfg
			setting.set(&next, float64(e.Number)/setting.scale)
			s.invoke("simulation.transient.set", next)
		}})
	}
	return nodes
}
