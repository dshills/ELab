//go:build offscreen

package main

import (
	"image"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMountedLiveScopeControlsKeepGeometryAcrossSamples(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		for _, example := range []string{"rc", "blink", "flasher", "display-counter"} {
			t.Run(mode+"/"+example, func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: example})
				agentAction(t, s.service, "simulation.pause", struct{}{})
				settings := s.service.Snapshot().TransientSettings
				settings.PlaybackSpeed = min(.02, settings.PlaybackSpeed)
				agentAction(t, s.service, "simulation.transient.set", settings)
				agentAction(t, s.service, "simulation.restart", struct{}{})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
				// Paused stepping gives deterministic real accepted samples across
				// digit boundaries and transistor regions. No snapshot/model data
				// is fabricated, and all updates use the public service commands.
				agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .001})
				s.app.Do(func() {
					if _, ok := s.reachableAction("cursor-select-a"); !ok {
						t.Fatal("missing live controls")
					}
				})
				view := func() ui.Node { return toolbarPaintView(s) }
				m := newMountedControlWindowSize(t, s, view, 1040, 1100, inspectorLayoutTestTheme())
				ids := []string{"scope-title", "scope-caption", "scope-caption-time", "scope-caption-detail", "scope-visibility", "transient-step", "simulation.restart", "scope-trace-row", "export-csv", "scope-trace-focus", "scope-cursors", "cursor-select-a", "cursor-select-b"}
				// Contextual readings now live in Inspect and have their own
				// stationary-reading fixture; the removed strip is not mounted.

				for _, input := range s.service.Snapshot().LiveInputModes {
					ids = append(ids, "input-"+input)
				}
				bounds := make(map[string]image.Rectangle)
				for _, id := range ids {
					bounds[id] = mountedControlElement(t, m, id).Bounds
				}
				captions := map[string]bool{}
				for _, duration := range []float64{.001, .002, .003, .004, .005, .04} {
					agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: duration})
					m.frame()
					caption, _ := findNode(s.view(), "scope-caption-time")
					captions[caption.Text] = true
					for _, id := range ids {
						if got := mountedControlElement(t, m, id).Bounds; got != bounds[id] {
							t.Errorf("sample t=%g caption=%q moved %s from %v to %v", s.service.Snapshot().Transient.Sample.Time, caption.Text, id, bounds[id], got)
						}
					}
				}
				if len(captions) < 4 {
					t.Fatal("fixture did not vary the accepted time readout")
				}
				// Also observe genuine background integration after Resume. Frame
				// geometry must remain fixed without a sequence of Step commands.
				agentAction(t, s.service, "simulation.resume", struct{}{})
				previous := s.service.Snapshot().Transient.Sample.Time
				for range 3 {
					deadline := time.Now().Add(2 * time.Second)
					for s.service.Snapshot().Transient.Sample.Time <= previous && time.Now().Before(deadline) {
						time.Sleep(10 * time.Millisecond)
					}
					current := s.service.Snapshot().Transient.Sample.Time
					if current <= previous {
						t.Fatal("real background integration did not advance")
					}
					m.frame()
					for _, id := range ids {
						if got := mountedControlElement(t, m, id).Bounds; got != bounds[id] {
							t.Errorf("live t=%g moved %s from %v to %v", current, id, bounds[id], got)
						}
					}
					previous = current
				}
			})
		}
	}
}
