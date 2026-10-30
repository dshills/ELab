//go:build offscreen

package main

import (
	"image"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedSpectrumMeasurements(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	s.app.Do(func() {
		if _, ok := s.reachableAction("spectrum-cursor-a"); !ok {
			t.Fatal("missing spectrum measurement controls")
		}
	})
	view := func() ui.Node {
		root := s.view()
		dock, _ := findNode(root, "instrument-panel")
		root.Kind = ui.Column // Reduced fixtures stack controls independently of the full-height native shell.
		root.Children = []ui.Node{dock}
		var omit func(*ui.Node)
		omit = func(n *ui.Node) {
			if n.Kind == ui.Extension {
				n.Kind = ui.Row
				n.Custom = nil
			}
			for i := range n.Children {
				omit(&n.Children[i])
			}
		}
		omit(&root)
		return root
	}
	m := newMountedControlWindowSize(t, s, view, 900, 1100, inspectorLayoutTestTheme())
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	return m
}
func TestMountedSpectrumMeasurementLayoutAndEdits(t *testing.T) {
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		t.Run(theme, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			prepareSpectrumMeasurements(t, s)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
			m := mountedSpectrumMeasurements(t, s)
			ids := []string{"spectrum-peaks", "spectrum-cursor-select-a", "spectrum-cursor-select-b", "spectrum-cursor-label-a", "spectrum-cursor-a", "spectrum-cursor-label-b", "spectrum-cursor-b", "spectrum-cursor-peak", "spectrum-cursor-clear-a", "spectrum-cursor-clear-b", "spectrum-cursor-values", "spectrum-cursor-delta", "spectrum-plot"}
			bounds := map[string]image.Rectangle{}
			for _, id := range ids {
				e := mountedControlElement(t, m, id)
				bounds[id] = e.Bounds
				if !e.Visible || e.Bounds.Empty() || !e.Bounds.In(image.Rect(0, 0, 900, 1100)) {
					t.Errorf("not fully visible %s %v", id, e.Bounds)
				}
			}
			for i, a := range ids {
				for _, b := range ids[i+1:] {
					if bounds[a].Overlaps(bounds[b]) {
						t.Errorf("overlap %s %v / %s %v", a, bounds[a], b, bounds[b])
					}
				}
			}
			before := s.service.Snapshot()
			typeMountedCursor(t, m, "spectrum-cursor-a", "100 Hz")
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("typing changed public state")
			}
			m.frame()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			m.frame()
			if c := s.service.Snapshot().Spectrum.Live.Measurements.A; c == nil || !c.Valid || c.Requested != 100 {
				t.Fatal("typed frequency not admitted", s.errorMessage, c)
			}
			clickMountedControl(t, m, "spectrum-cursor-select-b")
			m.frame()
			clickMountedControl(t, m, "spectrum-cursor-peak")
			m.frame()
			if c := s.service.Snapshot().Spectrum.Live.Measurements.B; c == nil || !c.Valid {
				t.Fatal("mounted peak action", s.errorMessage)
			}
			clickMountedControl(t, m, "spectrum-peaks")
			m.frame()
			if s.service.Snapshot().Spectrum.Live.Measurements.Visible {
				t.Fatal("mounted peak visibility")
			}
			p := s.service.Snapshot().Spectrum.Live.Settings
			p.Hold = false
			agentAction(t, s.service, "spectrum.live.set", p)
			for _, duration := range []float64{.01, .1} {
				old := s.service.Snapshot().Spectrum.Live.End
				agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: duration})
				waitSpectrumEnd(t, s.service, old)
				m.frame()
				for _, id := range ids {
					if got := mountedControlElement(t, m, id).Bounds; got != bounds[id] {
						t.Errorf("measurement update moved %s %v -> %v", id, bounds[id], got)
					}
				}
			}
		})
	}
}
func TestMountedSpectrumFrequencyFocusLossThenClear(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSpectrumMeasurements(t, s)
	m := mountedSpectrumMeasurements(t, s)
	typeMountedCursor(t, m, "spectrum-cursor-a", "100 Hz")
	clickMountedControl(t, m, "spectrum-cursor-clear-a")
	m.frame()
	if s.service.Snapshot().Spectrum.Live.Measurements.A != nil || s.errorMessage != "" {
		t.Fatal("focus-loss clear needs second click", s.errorMessage)
	}
}
func TestMountedSpectrumPrivateFrequencyDoesNotCrossSources(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSpectrumMeasurements(t, s)
	m := mountedSpectrumMeasurements(t, s)
	typeMountedCursor(t, m, "spectrum-cursor-a", "100 Hz")
	p := s.service.Snapshot().Spectrum.Live.Settings
	p.ProbeID = "i:R1"
	p.Hold = false
	agentAction(t, s.service, "spectrum.live.set", p)
	awaitSpectrumDock(t, s.service)
	m.frame()
	if got := mountedControlElement(t, m, "spectrum-cursor-a").Value; got != "" {
		t.Fatal("voltage draft crossed to current source", got)
	}
}
