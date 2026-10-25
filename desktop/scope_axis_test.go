package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

func TestHighCurrentScopeLabelsFitCanvasAcrossThemes(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "motor-dc"})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "simulation.restart", struct{}{})
	if _, err := s.service.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .004})
	if _, err := s.service.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	peak := 0.
	for _, trace := range s.service.Snapshot().Scope.Traces {
		if trace.Probe.Unit == "A" {
			for _, p := range trace.Points {
				peak = max(peak, p.Value)
			}
		}
	}
	if peak < 1.2 {
		t.Fatal("fixture did not exercise amp-level scientific labels", peak)
	}
	for _, theme := range []string{"light", "dark", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
		for _, width := range []float64{240, 320, 800, 1700} {
			s.app.Do(func() {
				s.view()
				v := canvas.Viewport{Width: width, Height: 160, Zoom: 1}
				mesh, err := s.scopeMesh(v)
				if err != nil {
					t.Fatal(err)
				}
				for _, vertex := range mesh.Vertices() {
					p := v.Screen(vertex.Position)
					if p.X < 0 || p.X > width || p.Y < 0 || p.Y > v.Height {
						t.Fatalf("%s high-current label clipped at width %.0f: %+v", theme, width, p)
					}
				}
			})
		}
	}
}
