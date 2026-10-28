//go:build offscreen

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/events"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedSpectrumControlsRemainStationaryDuringLiveUpdates(t *testing.T) {
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		t.Run(theme, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareSpectrumDock(t, s.service)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
			agentAction(t, s.service, "simulation.restart", struct{}{})
			agentAction(t, s.service, "scope.view.set", workbench.ScopeViewParams{View: "spectrum"})
			view := func() ui.Node {
				root := s.view()
				dock, _ := findNode(root, "instrument-panel")
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
			m := newMountedControlWindowSize(t, s, view, 1040, 600, inspectorLayoutTestTheme())
			ids := []string{"scope-view-scope", "scope-view-spectrum", "spectrum-source-live", "spectrum-source-recorded", "spectrum-live-hold", "scope-visibility", "spectrum-dock-probe", "spectrum-dock-window", "spectrum-dock-samples", "spectrum-dock-fundamental-label", "spectrum-dock-fundamental", "spectrum-dock-harmonics", "spectrum-dock-summary", "spectrum-dock-detail", "spectrum-plot", "spectrum-peaks", "spectrum-cursor-select-a", "spectrum-cursor-select-b", "spectrum-cursor-label-a", "spectrum-cursor-a", "spectrum-cursor-label-b", "spectrum-cursor-b", "spectrum-cursor-peak", "spectrum-cursor-clear-a", "spectrum-cursor-clear-b", "spectrum-cursor-values", "spectrum-cursor-delta"}
			bounds := map[string]image.Rectangle{}
			for _, id := range ids {
				element := mountedControlElement(t, m, id)
				bounds[id] = element.Bounds
				if !element.Visible || !element.Bounds.In(image.Rect(0, 0, 1040, 600)) {
					t.Errorf("control not fully visible: %s visible=%v bounds=%v", id, element.Visible, element.Bounds)
				}
				if os.Getenv("ELAB_SPECTRUM_CAPTURE") != "" {
					t.Logf("%s %v", id, element.Bounds)
				}
			}
			// Every exposed control must fit its row; fixed readout text cannot push
			// buttons offscreen as result availability, times and FFT values change.
			for _, row := range []string{"spectrum-dock-controls", "spectrum-dock-settings", "spectrum-measurements"} {
				parent := mountedControlElement(t, m, row).Bounds
				for _, id := range ids {
					n, _ := findNode(s.view(), row)
					if _, ok := findNode(n, id); ok && !bounds[id].In(parent) {
						t.Errorf("%s outside %s: %v / %v", id, row, bounds[id], parent)
					}
				}
			}
			if destination := os.Getenv("ELAB_SPECTRUM_CAPTURE"); destination != "" {
				if err := os.MkdirAll(destination, 0700); err != nil {
					t.Fatal(err)
				}
				pixels, err := m.manual.Capture(m.ctx, "main")
				if err != nil {
					t.Fatal(err)
				}
				file, err := os.OpenFile(filepath.Join(destination, theme+"-controls.png"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				encodeErr := png.Encode(file, pixels)
				closeErr := file.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatal(encodeErr, closeErr)
				}
			}
			for _, row := range []string{"spectrum-dock-controls", "spectrum-dock-settings", "spectrum-measurements"} {
				n, _ := findNode(s.view(), row)
				for i, a := range ids {
					if _, ok := findNode(n, a); !ok {
						continue
					}
					for _, b := range ids[i+1:] {
						if _, ok := findNode(n, b); ok && bounds[a].Overlaps(bounds[b]) {
							t.Errorf("controls overlap: %s %v and %s %v", a, bounds[a], b, bounds[b])
						}
					}
				}
			}
			previous := 0.
			for _, duration := range []float64{.1, .001, .01, .1} {
				agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: duration})
				waitSpectrumEnd(t, s.service, previous)
				previous = s.service.Snapshot().Spectrum.Live.End
				m.frame()
				for _, id := range ids {
					if got := mountedControlElement(t, m, id).Bounds; got != bounds[id] {
						t.Errorf("FFT update moved %s: %v -> %v", id, bounds[id], got)
					}
				}
			}
			agentAction(t, s.service, "simulation.resume", struct{}{})
			for range 2 {
				waitSpectrumEnd(t, s.service, previous)
				previous = s.service.Snapshot().Spectrum.Live.End
				m.frame()
				for _, id := range ids {
					if got := mountedControlElement(t, m, id).Bounds; got != bounds[id] {
						t.Errorf("running FFT moved %s: %v -> %v", id, bounds[id], got)
					}
				}
			}
		})
	}
}
func waitSpectrumEnd(t *testing.T, s *workbench.Service, after float64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Snapshot().Spectrum.Live
		if v != nil && v.Result != nil && v.End > after {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("accepted live FFT did not advance", after, s.Snapshot().Spectrum.Live)
}

func TestMountedSpectrumTabsAndHoldRouteSharedCommands(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSpectrumDock(t, s.service)
	view := func() ui.Node {
		root := s.view()
		dock, _ := findNode(root, "instrument-panel")
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
	m := newMountedControlWindowSize(t, s, view, 1040, 600, inspectorLayoutTestTheme())
	click := func(id string) {
		t.Helper()
		at := mountedControlElement(t, m, id).Bounds.Min.Add(mountedControlElement(t, m, id).Bounds.Size().Div(2))
		for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
			m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
		}
		m.frame()
	}
	click("scope-view-spectrum")
	if state := s.service.Snapshot(); state.ScopeView != "spectrum" || state.Spectrum.Live == nil || !state.Spectrum.Live.Settings.Enabled {
		t.Fatal("mounted Spectrum tab did not dispatch")
	}
	click("spectrum-live-hold")
	if !s.service.Snapshot().Spectrum.Live.Settings.Hold {
		t.Fatal("mounted Hold did not dispatch")
	}
	click("spectrum-source-recorded")
	if s.service.Snapshot().Spectrum.Live.Settings.Enabled {
		t.Fatal("mounted Recorded did not dispatch")
	}
	click("scope-view-scope")
	if s.service.Snapshot().ScopeView != "scope" || s.errorMessage != "" {
		t.Fatal("mounted Scope tab did not dispatch", s.errorMessage)
	}
}
