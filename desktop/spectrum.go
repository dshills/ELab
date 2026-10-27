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

func (s *screen) spectrumInspector(state workbench.Snapshot) []ui.Node {
	if state.Spectrum == nil {
		return nil
	}
	cfg := state.Spectrum.Settings
	nodes := []ui.Node{section("spectrum-title", "Spectrum & distortion"), quiet("spectrum-help", "Analyze a completed recorded run. Live simulation stays independent; choose a steady interval containing whole cycles for THD.")}
	apply := func(next workbench.SpectrumParams) {
		s.invokeAtCommandRevision("spectrum.settings.set", next, state.CommandRevision, state.Document.Generation)
	}
	ops := []ui.Option{}
	for _, op := range state.Operations {
		if op.Status == "completed" {
			ops = append(ops, ui.Option{ID: op.ID, Label: op.ID + " · recorded"})
		}
	}
	nodes = append(nodes, ui.Node{ID: "spectrum-operation", Kind: ui.Selector, SelectedID: cfg.ID, Options: ops, Accessibility: ui.Accessibility{Name: "Spectrum recording"}, Change: func(e ui.ChangeEvent) {
		next := cfg
		next.ID = e.SelectedID
		next.ProbeID = ""
		next.Start, next.End = 0, 0
		apply(next)
	}})
	probes := []ui.Option{}
	for _, p := range state.Spectrum.Available {
		probes = append(probes, ui.Option{ID: p.ID, Label: p.Label})
	}
	nodes = append(nodes, ui.Node{ID: "spectrum-probe", Kind: ui.Selector, SelectedID: cfg.ProbeID, Options: probes, Accessibility: ui.Accessibility{Name: "Spectrum recorded probe"}, Change: func(e ui.ChangeEvent) { next := cfg; next.ProbeID = e.SelectedID; apply(next) }})
	for _, field := range []struct {
		id, label string
		value     float64
	}{{"start", "Start · s", cfg.Start}, {"end", "End · s (0 = capture end)", cfg.End}, {"fundamental", "Fundamental · Hz (0 = unset)", cfg.Fundamental}} {
		f := field
		nodes = append(nodes, quiet("spectrum-"+f.id+"-label", f.label), ui.Node{ID: "spectrum-" + f.id, Kind: ui.TextField, Text: strconv.FormatFloat(f.value, 'g', -1, 64), Accessibility: ui.Accessibility{Name: f.label}, Change: func(e ui.ChangeEvent) {
			key := f.id + "_s"
			if f.id == "fundamental" {
				key = "fundamental_hz"
			}
			s.invokeAtCommandRevision("spectrum.settings.set", textFieldParams(cfg, key, e.Text), state.CommandRevision, state.Document.Generation)
		}})
	}
	nodes = append(nodes, ui.Node{ID: "spectrum-window", Kind: ui.Selector, SelectedID: cfg.Window, Options: []ui.Option{{ID: "rectangular", Label: "Rectangular · coherent capture"}, {ID: "hann", Label: "Hann · reduced leakage"}}, Accessibility: ui.Accessibility{Name: "Spectrum window"}, Change: func(e ui.ChangeEvent) { next := cfg; next.Window = e.SelectedID; apply(next) }})
	samples := []ui.Option{{ID: "0", Label: "Automatic sample count"}}
	for _, n := range []int{32, 64, 128, 256, 512, 1024, 2048, 4096} {
		samples = append(samples, ui.Option{ID: strconv.Itoa(n), Label: strconv.Itoa(n) + " samples"})
	}
	nodes = append(nodes, ui.Node{ID: "spectrum-samples", Kind: ui.Selector, SelectedID: strconv.Itoa(cfg.Samples), Options: samples, Accessibility: ui.Accessibility{Name: "Spectrum uniform sample count"}, Change: func(e ui.ChangeEvent) {
		n, err := strconv.Atoi(e.SelectedID)
		if err == nil {
			next := cfg
			next.Samples = n
			apply(next)
		}
	}})
	harmonics := []ui.Option{}
	for n := 2; n <= 10; n++ {
		harmonics = append(harmonics, ui.Option{ID: strconv.Itoa(n), Label: fmt.Sprintf("Through harmonic %d", n)})
	}
	nodes = append(nodes, ui.Node{ID: "spectrum-harmonics", Kind: ui.Selector, SelectedID: strconv.Itoa(cfg.Harmonics), Options: harmonics, Accessibility: ui.Accessibility{Name: "THD harmonic count"}, Change: func(e ui.ChangeEvent) {
		n, err := strconv.Atoi(e.SelectedID)
		if err == nil {
			next := cfg
			next.Harmonics = n
			apply(next)
		}
	}})
	run := s.action("spectrum-analyze", "Analyze recorded spectrum", "spectrum.analyze", cfg)
	run.Disabled = len(ops) == 0 || cfg.ID == "" || cfg.ProbeID == ""
	nodes = append(nodes, run)
	if r := state.Spectrum.Result; r != nil {
		nodes = append(nodes, quiet("spectrum-summary", fmt.Sprintf("Recorded %s · %s\n%d samples · resolution %.5g Hz\nNyquist %.5g Hz · mean %.5g %s", r.Settings.ID, r.Probe.Label, r.Settings.Samples, r.Resolution, r.Nyquist, r.Mean, r.Probe.Unit)))
		if r.THD != nil {
			nodes = append(nodes, quiet("spectrum-thd", fmt.Sprintf("THD %.6g %% · harmonics 2–%d", *r.THD, r.Settings.Harmonics)))
		} else {
			nodes = append(nodes, quiet("spectrum-thd-unavailable", r.THDReason))
		}
		if s.spectrumDrawing == nil {
			v := canvas.Viewport{Width: 300, Height: 190, Zoom: 1}
			s.spectrumDrawing = &cogent.MeshCanvas{Owner: s.app, Window: "main", View: &v, Prepare: s.spectrumMesh}
		}
		s.spectrumDrawing.Revision = s.spectrumAppearance.next(state.CommandRevision, s.colorsForTheme(state.Theme))
		nodes = append(nodes, ui.Node{ID: "spectrum-plot", Kind: ui.Extension, Custom: s.spectrumDrawing, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 190})}, Accessibility: ui.Accessibility{Name: "Recorded FFT magnitude in decibels versus frequency in hertz"}})
		for _, h := range r.Harmonics {
			nodes = append(nodes, quiet(fmt.Sprintf("spectrum-harmonic-%d", h.Order), fmt.Sprintf("H%d · %.5g Hz · %.6g %s peak", h.Order, h.Frequency, h.Peak, r.Probe.Unit)))
		}
		for _, format := range []string{"json", "csv"} {
			f := format
			id := r.ID
			nodes = append(nodes, buttonStyle(ui.Node{ID: "spectrum-export-" + f, Kind: ui.Button, Text: "Export spectrum " + strings.ToUpper(f), Accessibility: ui.Accessibility{Name: "Export spectrum " + f}, Activate: func() {
				s.beginFile("spectrum.export", true, workbench.VerificationExportParams{ID: id, Format: f}, nil)
			}}, false))
		}
	}
	return nodes
}
func (s *screen) spectrumMesh(v canvas.Viewport) (*canvas.Mesh, error) {
	state := s.service.Snapshot()
	b := &meshBuilder{}
	if state.Spectrum == nil || state.Spectrum.Result == nil {
		return canvas.NewMesh(nil, nil)
	}
	r := state.Spectrum.Result
	if r.Nyquist <= 0 || math.IsNaN(r.Nyquist) || math.IsInf(r.Nyquist, 0) {
		return canvas.NewMesh(nil, nil)
	}
	left, right, top, bottom := 48., v.Width-14, 16., v.Height-36
	if right <= left || bottom <= top {
		return canvas.NewMesh(nil, nil)
	}
	palette := s.colorsForTheme(state.Theme)
	line := func(c color.NRGBA, points ...canvas.Point) {
		world := make([]canvas.Point, len(points))
		for i, p := range points {
			world[i] = v.World(p)
		}
		b.stroke(c, 1, world...)
	}
	text := func(x, y float64, label string) {
		p := v.World(canvas.Point{X: x, Y: y})
		b.text(p.X, p.Y, label, palette.Muted)
	}
	peak := -300.
	for _, p := range r.Bins {
		peak = math.Max(peak, p.DB)
	}
	high := math.Max(0, math.Ceil(peak/20)*20)
	low := high - 100
	for i := 0; i <= 4; i++ {
		f := float64(i) / 4
		y := bottom - f*(bottom-top)
		line(palette.Grid, canvas.Point{X: left, Y: y}, canvas.Point{X: right, Y: y})
		text(2, y-4, fmt.Sprintf("%.0f", low+f*(high-low)))
		label := fmt.Sprintf("%.3g", f*r.Nyquist)
		width := workbench.CanvasTextWidth(label, 1.25)
		text(math.Max(2, math.Min(left+f*(right-left)-width/2, v.Width-width-4)), bottom+10, label)
	}
	text(left, bottom+23, "Hz · dB / 1 "+r.Probe.Unit)
	points := make([]canvas.Point, len(r.Bins))
	for i, p := range r.Bins {
		points[i] = canvas.Point{X: left + p.Frequency/r.Nyquist*(right-left), Y: bottom - (math.Max(low, math.Min(high, p.DB))-low)/(high-low)*(bottom-top)}
	}
	line(palette.Accent, points...)
	return canvas.NewMesh(b.vertices, b.indices)
}
