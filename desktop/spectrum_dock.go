package main

import (
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func (s *screen) ensureSpectrumDrawing() {
	if s.spectrumDrawing != nil {
		return
	}
	v := canvas.Viewport{Width: 800, Height: 160, Zoom: 1}
	// Like the scope, FFT has command-owned axes; native wheel only scrolls its dock.
	s.spectrumDrawing = &spectrumCanvas{screen: s, MeshCanvas: &cogent.MeshCanvas{Owner: s.app, Window: "main", View: &v, Prepare: s.spectrumMesh}}
}
func (s *screen) instrumentPanel(state workbench.Snapshot, width float32) ui.Node {
	tabs := []ui.Node{}
	for _, view := range []string{"scope", "spectrum"} {
		label := "Scope"
		if view == "spectrum" {
			label = "Spectrum"
		}
		action := buttonStyle(s.action("scope-view-"+view, label, "scope.view.set", workbench.ScopeViewParams{View: view}), state.ScopeView == view)
		tabs = append(tabs, inspectorTabCell(action, "plot-"+view, state.ScopeView == view))
	}
	body := s.collapsedScope(state)
	if state.ScopeVisible {
		body = s.scopePanel(state)
		if state.ScopeView == "spectrum" {
			body = s.spectrumPanel(state)
		}
	}
	// The Spectrum settings can be wider than a compact viewport. Bound the
	// explanation to the actual dock width before shaping text; text-only Scope
	// must also not shrink to the longest word's intrinsic width.
	for i := range body.Children {
		if body.Children[i].ID == "instrument-empty" {
			constrainRail(body.Children[i].Children, max(200, width-20))
		}
	}
	return ui.Node{ID: "instrument-panel", Kind: ui.Column, Grow: state.ScopeVisible, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0)), Background: ui.Some(paper)}, Children: []ui.Node{
		{ID: "instrument-tabs", Kind: ui.Row, Style: ui.Style{MinSize: ui.Some(ui.Size{Height: 36}), MaxSize: ui.Some(ui.Size{Height: 36}), Padding: ui.Some(ui.Insets{Left: 10}), Gap: ui.Some(float32(4))}, Accessibility: ui.Accessibility{Name: "Signal display"}, Children: tabs}, body,
	}}
}
func spectrumFixed(n ui.Node, width, height float32) ui.Node {
	n.Style.MinSize = ui.Some(ui.Size{Width: width, Height: height})
	n.Style.MaxSize = ui.Some(ui.Size{Width: width, Height: height})
	if n.Kind == ui.Selector || n.Kind == ui.TextField {
		// The native control adds toolbar padding/border outside its content
		// width. Reserve an explicit outer slot so neighbors cannot overlap.
		return ui.Node{ID: n.ID + "-slot", Kind: ui.Column, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: width + 24, Height: 40}), MaxSize: ui.Some(ui.Size{Width: width + 24, Height: 40}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0))}, Children: []ui.Node{n}}
	}
	return n
}
func (s *screen) spectrumPanel(state workbench.Snapshot) ui.Node {
	cfg := workbench.DefaultLiveSpectrumParams()
	live := state.Spectrum.Live
	if live != nil {
		cfg = live.Settings
	}
	isLive := live != nil && cfg.Enabled
	recorded := state.Spectrum.Settings
	applyLive := func(p workbench.LiveSpectrumParams) {
		s.invokeAtCommandRevision("spectrum.live.set", p, state.CommandRevision, state.Document.Generation)
	}
	applyRecorded := func(p workbench.SpectrumParams) {
		s.invokeAtCommandRevision("spectrum.settings.set", p, state.CommandRevision, state.Document.Generation)
	}
	start := cfg
	start.Enabled = true
	stop := cfg
	stop.Enabled = false
	controls := []ui.Node{
		buttonStyle(s.action("spectrum-source-live", "Live", "spectrum.live.set", start), isLive),
		buttonStyle(s.action("spectrum-source-recorded", "Recorded", "spectrum.live.set", stop), !isLive),
	}
	options := []ui.Option{}
	selected := cfg.ProbeID
	if isLive {
		options = append(options, ui.Option{ID: "", Label: "First scope probe"})
		for _, p := range state.Scope.Traces {
			options = append(options, ui.Option{ID: p.Probe.ID, Label: p.Probe.Label})
		}
		found := selected == ""
		for _, option := range options {
			found = found || option.ID == selected
		}
		if !found {
			options = append(options, ui.Option{ID: selected, Label: selected + " · unavailable"})
		}
		hold := cfg
		hold.Hold = !hold.Hold
		controls = append(controls, buttonStyle(s.action("spectrum-live-hold", "Hold", "spectrum.live.set", hold), cfg.Hold))
	} else {
		ops := []ui.Option{}
		for _, op := range state.Operations {
			if op.Status == "completed" {
				ops = append(ops, ui.Option{ID: op.ID, Label: op.ID})
			}
		}
		controls = append(controls, spectrumFixed(ui.Node{ID: "spectrum-dock-recording", Kind: ui.Selector, Options: ops, SelectedID: recorded.ID, Accessibility: ui.Accessibility{Name: "Spectrum recording"}, Change: func(e ui.ChangeEvent) {
			p := recorded
			p.ID = e.SelectedID
			p.ProbeID = ""
			p.Start, p.End = 0, 0
			applyRecorded(p)
		}}, 170, 24))
		selected = recorded.ProbeID
		for _, p := range state.Spectrum.Available {
			options = append(options, ui.Option{ID: p.ID, Label: p.Label})
		}
	}
	controls = append(controls, spectrumFixed(ui.Node{ID: "spectrum-dock-probe", Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Spectrum probe", Help: "Live choices use the same selected voltage/current scope probes"}, Change: func(e ui.ChangeEvent) {
		if isLive {
			p := cfg
			p.ProbeID = e.SelectedID
			applyLive(p)
		} else {
			p := recorded
			p.ProbeID = e.SelectedID
			applyRecorded(p)
		}
	}}, 240, 24))
	if !isLive {
		n := s.action("spectrum-dock-analyze", "Analyze", "spectrum.analyze", recorded)
		n.Disabled = recorded.ID == "" || recorded.ProbeID == ""
		controls = append(controls, n)
	}
	controls = append(controls, s.scopeVisibilityButton(state))
	window, samples, fundamental, harmonics := cfg.Window, cfg.Samples, cfg.Fundamental, cfg.Harmonics
	if !isLive {
		window, samples, fundamental, harmonics = recorded.Window, recorded.Samples, recorded.Fundamental, recorded.Harmonics
	}
	choices := []ui.Option{{ID: "0", Label: "Automatic samples"}}
	maximum := workbench.MaxLiveSpectrumSamples
	if !isLive {
		maximum = workbench.MaxSpectrumSamples
	}
	for n := 32; n <= maximum; n *= 2 {
		choices = append(choices, ui.Option{ID: strconv.Itoa(n), Label: fmt.Sprintf("%d samples", n)})
	}
	harmonicsOptions := []ui.Option{}
	for n := 2; n <= 10; n++ {
		harmonicsOptions = append(harmonicsOptions, ui.Option{ID: strconv.Itoa(n), Label: fmt.Sprintf("%d harmonics", n)})
	}
	settings := []ui.Node{
		spectrumFixed(ui.Node{ID: "spectrum-dock-window", Kind: ui.Selector, Options: []ui.Option{{ID: "hann", Label: "Hann window"}, {ID: "rectangular", Label: "Rectangular window"}}, SelectedID: window, Accessibility: ui.Accessibility{Name: "Spectrum window"}, Change: func(e ui.ChangeEvent) {
			if isLive {
				p := cfg
				p.Window = e.SelectedID
				applyLive(p)
			} else {
				p := recorded
				p.Window = e.SelectedID
				applyRecorded(p)
			}
		}}, 170, 24),
		spectrumFixed(ui.Node{ID: "spectrum-dock-samples", Kind: ui.Selector, Options: choices, SelectedID: strconv.Itoa(samples), Accessibility: ui.Accessibility{Name: "Spectrum sample count"}, Change: func(e ui.ChangeEvent) {
			n, err := strconv.Atoi(e.SelectedID)
			if err != nil {
				return
			}
			if isLive {
				p := cfg
				p.Samples = n
				applyLive(p)
			} else {
				p := recorded
				p.Samples = n
				applyRecorded(p)
			}
		}}, 150, 24),
		quiet("spectrum-dock-fundamental-label", "Fundamental · Hz"),
		spectrumFixed(ui.Node{ID: "spectrum-dock-fundamental", Kind: ui.TextField, Text: strconv.FormatFloat(fundamental, 'g', -1, 64), Accessibility: ui.Accessibility{Name: "Fundamental frequency in Hz; zero leaves THD unset"}, Change: func(e ui.ChangeEvent) {
			if isLive {
				s.invokeAtCommandRevision("spectrum.live.set", textFieldParams(cfg, "fundamental_hz", e.Text), state.CommandRevision, state.Document.Generation)
			} else {
				s.invokeAtCommandRevision("spectrum.settings.set", textFieldParams(recorded, "fundamental_hz", e.Text), state.CommandRevision, state.Document.Generation)
			}
		}}, 100, 24),
		spectrumFixed(ui.Node{ID: "spectrum-dock-harmonics", Kind: ui.Selector, Options: harmonicsOptions, SelectedID: strconv.Itoa(harmonics), Accessibility: ui.Accessibility{Name: "THD harmonics"}, Change: func(e ui.ChangeEvent) {
			n, err := strconv.Atoi(e.SelectedID)
			if err != nil {
				return
			}
			if isLive {
				p := cfg
				p.Harmonics = n
				applyLive(p)
			} else {
				p := recorded
				p.Harmonics = n
				applyRecorded(p)
			}
		}}, 140, 24),
	}
	result := state.Spectrum.Result
	summary := "Recorded spectrum · choose a recording and Analyze"
	detail := "Advanced recorded interval and export controls are in Simulate → Spectrum & distortion"
	if isLive {
		result = live.Result
		summary = "Live spectrum · " + live.Status
		detail = live.Reason
		if result == nil && state.Transient == nil {
			summary = "Live spectrum · no time-based data"
			detail = "Use a transient circuit or select a recording."
		}
		if result != nil {
			summary = fmt.Sprintf("Live spectrum · %s · %.5g–%.5g s", live.Status, live.Start, live.End)
		}
	}
	if result != nil {
		detail = fmt.Sprintf("%s · Δf %.5g Hz · Nyquist %.5g Hz", result.Probe.Label, result.Resolution, result.Nyquist)
		if result.THD != nil {
			detail += fmt.Sprintf(" · THD %.5g %%", *result.THD)
		} else {
			detail += " · THD unavailable"
		}
	}
	detailNode := spectrumFixed(quiet("spectrum-dock-detail", detail), 760, 20)
	detailNode.Accessibility.Help = detail
	if result != nil && result.THD == nil {
		detailNode.Accessibility.Help += ". " + result.THDReason
	}
	s.ensureSpectrumDrawing()
	s.spectrumDrawing.Revision = s.spectrumAppearance.next(state.Revision, s.colorsForTheme(state.Theme))
	children := []ui.Node{
		{ID: "spectrum-dock-controls", Kind: ui.Toolbar, Style: chromeStyle(), Children: controls},
	}
	if result == nil && state.Transient == nil {
		children = append(children, instrumentEmptyNotice(state, !isLive))
	}
	children = append(children, []ui.Node{
		{ID: "spectrum-dock-settings", Kind: ui.Toolbar, Style: chromeStyle(), Children: settings},
		s.spectrumMeasurementControls(state),
		{ID: "spectrum-plot", Kind: ui.Extension, Grow: true, Custom: s.spectrumDrawing, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 160}), MaxSize: ui.Some(ui.Size{Height: 160})}, Accessibility: ui.Accessibility{Name: "FFT magnitude in decibels versus frequency in hertz"}},
		spectrumFixed(quiet("spectrum-dock-summary", summary), 800, 20), detailNode,
		s.spectrumCursorReadout(state),
	}...)
	return ui.Node{ID: "spectrum-panel", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(10)), Gap: ui.Some(float32(4)), Background: ui.Some(paper)}, Children: children}
}
