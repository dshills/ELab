package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func spectrumDisplay(state workbench.Snapshot) (*workbench.SpectrumResult, *workbench.SpectrumMeasurements, string) {
	if state.Spectrum == nil {
		return nil, nil, "recorded"
	}
	if state.ScopeView == "spectrum" && state.Spectrum.Live != nil && state.Spectrum.Live.Settings.Enabled {
		return state.Spectrum.Live.Result, state.Spectrum.Live.Measurements, "live"
	}
	return state.Spectrum.Result, state.Spectrum.Measurements, "recorded"
}
func activeSpectrumCursor(m *workbench.SpectrumMeasurements) string {
	if m != nil && m.Active == "b" {
		return "b"
	}
	return "a"
}
func spectrumAmplitudeRange(r *workbench.SpectrumResult) (float64, float64) {
	peak := -300.
	for _, bin := range r.Bins {
		peak = math.Max(peak, bin.DB)
	}
	high := 0.
	if peak > -300 {
		high = math.Ceil(peak/20) * 20
	}
	return high - 100, high
}
func (s *screen) spectrumMeasurementControls(state workbench.Snapshot) ui.Node {
	result, m, source := spectrumDisplay(state)
	if m == nil {
		m = &workbench.SpectrumMeasurements{Visible: true, Active: "a"}
	}
	owner := spectrumMeasurementOwner(state, source)
	action := func(id, text, command string, p any) ui.Node {
		return s.spectrumMeasurementAction(state, owner, id, text, command, p)
	}
	children := []ui.Node{buttonStyle(action("spectrum-peaks", "Peaks", "spectrum.markers.set", workbench.ScopeVisibilityParams{Visible: !m.Visible}), m.Visible)}
	for _, name := range []string{"a", "b"} {
		action := action("spectrum-cursor-select-"+name, strings.ToUpper(name), "spectrum.cursor.set", workbench.SpectrumCursorParams{Source: source, Cursor: name, Select: true})
		children = append(children, buttonStyle(action, m.Active == name))
	}
	for _, name := range []string{"a", "b"} {
		name := name
		value := ""
		c := m.A
		if name == "b" {
			c = m.B
		}
		if c != nil {
			value = strconv.FormatFloat(c.Requested, 'g', -1, 64)
		}
		children = append(children, quiet("spectrum-cursor-label-"+name, strings.ToUpper(name)+" · Hz"), spectrumFixed(ui.Node{ID: "spectrum-cursor-" + name, Kind: ui.TextField, Text: value, Disabled: result == nil, Accessibility: ui.Accessibility{Name: "Spectrum cursor " + strings.ToUpper(name) + " frequency", Help: "Nearest FFT bin; accepts Hz/kHz/mHz; empty clears"}, Change: func(e ui.ChangeEvent) {
			p := workbench.SpectrumCursorParams{Source: source, Cursor: name}
			if strings.TrimSpace(e.Text) == "" {
				p.Clear = true
				s.changeSpectrumFrequency(state, owner, p)
			} else {
				s.changeSpectrumFrequency(state, owner, textFieldParams(p, "frequency_hz", e.Text))
			}
		}}, 100, 24))
	}
	jump := action("spectrum-cursor-peak", "Peak → cursor", "spectrum.cursor.set", workbench.SpectrumCursorParams{Source: source, Cursor: m.Active, Peak: 1})
	jump.Disabled = len(m.Peaks) == 0
	children = append(children, jump)
	for _, name := range []string{"a", "b"} {
		children = append(children, action("spectrum-cursor-clear-"+name, "Clear "+strings.ToUpper(name), "spectrum.cursor.set", workbench.SpectrumCursorParams{Source: source, Cursor: name, Clear: true}))
	}
	controls := ui.Node{ID: "spectrum-measurements", Kind: ui.Toolbar, Style: chromeStyle(), Children: children}
	scopeOwnedControlState(&controls, owner)
	return controls
}
func (s *screen) spectrumCursorReadout(state workbench.Snapshot) ui.Node {
	_, m, _ := spectrumDisplay(state)
	a, b, delta := "A —", "B —", "Δ —"
	if m != nil {
		label := func(name string, c *workbench.SpectrumCursor) string {
			if c == nil {
				return name + " —"
			}
			if !c.Valid || c.Reading == nil {
				return name + " · " + c.Reason
			}
			r := c.Reading
			db := fmt.Sprintf("%.4g dB", r.DB)
			if r.DBFloored {
				db = "≤−300 dB floor"
			}
			return fmt.Sprintf("%s %.5g Hz · %.5g %s peak · %s", name, r.Frequency, r.Peak, m.Unit, db)
		}
		a, b = label("A", m.A), label("B", m.B)
		if m.DeltaHz != nil {
			delta = fmt.Sprintf("Δf %.5g Hz · Δpeak %.5g %s", *m.DeltaHz, *m.DeltaPeak, m.Unit)
			if m.DeltaDB != nil {
				delta += fmt.Sprintf(" · Δ %.5g dB", *m.DeltaDB)
			} else {
				delta += " · dB difference unavailable at floor"
			}
		}
		if len(m.Peaks) > 0 {
			p := m.Peaks[0].Reading
			delta += fmt.Sprintf(" · P1 %.5g Hz / %.5g %s peak", p.Frequency, p.Peak, m.Unit)
		}
	}
	return ui.Node{ID: "spectrum-cursor-readouts", Kind: ui.Column, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(2)), MinSize: ui.Some(ui.Size{Height: 42}), MaxSize: ui.Some(ui.Size{Height: 42})}, Children: []ui.Node{spectrumFixed(quiet("spectrum-cursor-values", a+"    "+b), 800, 20), spectrumFixed(quiet("spectrum-cursor-delta", delta), 800, 20)}}
}

type spectrumFieldContinuation struct {
	from, to actionContext
	owner    string
}

type spectrumPlotFrame struct {
	revision, generation     uint64
	source, resultID, cursor string
	left, right, nyquist     float64
	dock                     bool
}
type spectrumGesture struct {
	origin   *spectrumPlotFrame
	revision uint64
	position float64
	valid    bool
}
type spectrumCanvas struct {
	*cogent.MeshCanvas
	screen  *screen
	frame   atomic.Pointer[spectrumPlotFrame]
	epoch   uint64
	gesture *spectrumGesture
}

func (c *spectrumCanvas) Mount(parent tree.Node) core.Widget {
	widget := c.MeshCanvas.Mount(parent)
	if widget == nil {
		return nil
	}
	c.epoch++
	c.frame.Store(nil)
	c.gesture = nil
	epoch := c.epoch
	w := widget.AsWidget()
	owner, window, ok := cogent.WindowScope(w)
	if !ok || owner != c.Owner {
		return nil
	}
	w.OnFirst(events.Scroll, func(e events.Event) { forwardScopeScroll(w, e) })
	for _, kind := range []events.Types{events.MouseDown, events.MouseUp, events.SlideStart, events.SlideMove, events.SlideStop} {
		w.OnFirst(kind, func(e events.Event) {
			frame := c.frame.Load()
			pos := e.Pos().Sub(w.Geom.Pos.Content.ToPoint())
			withWindowOwner(c.Owner, window, func() {
				if c.epoch == epoch {
					c.pointer(frame, kind, e.MouseButton(), float64(pos.X))
				}
			})
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
func (c *spectrumCanvas) apply(frame *spectrumPlotFrame, px float64, revision uint64) (uint64, bool) {
	if c.screen == nil || frame == nil || c.frame.Load() == nil || frame.right <= frame.left || !finitePixel(px) || c.screen.fileBusy || c.screen.pendingCommand != "" {
		return revision, false
	}
	state := c.screen.service.Snapshot()
	_, _, source := spectrumDisplay(state)
	if source != frame.source || sourcePaneHidden(state) || state.ExampleBrowser.Visible || frame.dock && (state.ScopeView != "spectrum" || !state.ScopeVisible) {
		return revision, false
	}
	f := math.Max(0, math.Min(1, (px-frame.left)/(frame.right-frame.left))) * frame.nyquist
	p := workbench.SpectrumCursorParams{Source: frame.source, Cursor: frame.cursor, Frequency: &f}
	if frame.source == "recorded" {
		p.ResultID = frame.resultID
	}
	response := c.screen.invokeAtCommandRevision("spectrum.cursor.set", p, revision, frame.generation)
	if response.Error != nil {
		return revision, false
	}
	return response.Result.(workbench.Snapshot).CommandRevision, true
}
func (c *spectrumCanvas) pointer(frame *spectrumPlotFrame, kind events.Types, button events.Buttons, px float64) {
	if button != events.Left {
		c.gesture = nil
		return
	}
	finish := kind == events.MouseUp || kind == events.SlideStop
	if kind == events.MouseDown {
		c.gesture = nil
		if frame != nil {
			c.gesture = &spectrumGesture{origin: frame, revision: frame.revision}
		}
	}
	g := c.gesture
	if g == nil || frame == nil {
		c.gesture = nil
		return
	}
	if finish && g.valid && g.position == px {
		c.gesture = nil
		return
	}
	origin := g.origin
	if frame.revision < origin.revision || frame.revision > g.revision || frame.generation != origin.generation || frame.source != origin.source || frame.cursor != origin.cursor || frame.dock != origin.dock || frame.source == "recorded" && frame.resultID != origin.resultID {
		c.gesture = nil
		return
	}
	next, accepted := c.apply(frame, px, g.revision)
	if !accepted {
		c.gesture = nil
		return
	}
	g.revision = next
	g.position = px
	g.valid = true
	if finish {
		c.gesture = nil
	}
}
func finitePixel(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Private frequency drafts belong to one source, not to a rolling frame. Hold
// and ordinary acquisition preserve it; probe/run/settings changes retire it.
func spectrumMeasurementOwner(state workbench.Snapshot, source string) string {
	if source == "recorded" {
		if state.Spectrum != nil && state.Spectrum.Result != nil {
			return "recorded:" + state.Spectrum.Result.ID
		}
		return "recorded:empty"
	}
	cfg := state.Spectrum.Live.Settings
	cfg.Enabled, cfg.Hold = false, false
	window := state.Scope.Window
	if window.Mode != "fixed" {
		window.Start, window.End = 0, 0
	}
	return fmt.Sprintf("live:%d:%d:%d:%v:%v:%v", state.ElectricalRevision, state.RunID, state.ActuationRevision, cfg, state.Scope.Selected, window)
}
func (s *screen) changeSpectrumFrequency(state workbench.Snapshot, owner string, p any) {
	var origin *actionContext
	if s.fieldContext != nil {
		copy := *s.fieldContext
		origin = &copy
	}
	response := s.invokeAtCommandRevision("spectrum.cursor.set", p, state.CommandRevision, state.Document.Generation)
	if response.Error != nil || origin == nil {
		s.spectrumFieldAccepted = nil
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	from := *origin
	if previous := s.spectrumFieldAccepted; previous != nil && previous.to == from && previous.owner == owner {
		from = previous.from
	}
	s.spectrumFieldAccepted = &spectrumFieldContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, owner: owner}
}

// Focus loss may commit a frequency before a Select/Clear/Peak click arrives.
// Bridge only that field's consecutive accepted writes in this exact source.
func (s *screen) spectrumMeasurementAction(state workbench.Snapshot, owner, id, text, command string, p any) ui.Node {
	context := actionContext{state.CommandRevision, state.Document.Generation}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Activate: func() {
		current := s.service.Snapshot()
		if accepted := s.spectrumFieldAccepted; accepted != nil && accepted.owner == owner && accepted.from.generation == context.generation && accepted.from.revision <= context.revision && context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
			context = accepted.to
		}
		s.invokeAtCommandRevision(command, p, context.revision, context.generation)
	}}, false)
}
