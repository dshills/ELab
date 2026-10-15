package main

import (
	"math"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

func TestOverlaySharesAxesAndBoundsDashedGeometry(t *testing.T) {
	a := []workbench.ScopeTrace{{Probe: workbench.Probe{Unit: "V"}, Points: []workbench.ScopePoint{{Value: 2}}}, {Probe: workbench.Probe{Unit: "A"}, Points: []workbench.ScopePoint{{Value: .002}}}}
	b := []workbench.ScopeTrace{{Probe: workbench.Probe{Unit: "V"}, Points: []workbench.ScopePoint{{Value: 11}, {Value: -3}}}, {Probe: workbench.Probe{Unit: "A"}, Points: []workbench.ScopePoint{{Value: -.01}, {Value: .02}}}}
	vmin, vmax, imin, imax := scopeRanges(a, b)
	if vmin != -3 || vmax != 11 || imin != -10 || imax != 20 {
		t.Fatal("reference amplitudes clipped or independently scaled", vmin, vmax, imin, imax)
	}
	points := []canvas.Point{}
	for i := 0; i < 258; i++ {
		points = append(points, canvas.Point{X: float64(i), Y: float64(i%2) * 100})
	}
	points = append(points, points[len(points)-1]) // Repeated/zero-length vertices are valid.
	count := 0
	dashedScopeLine(points, func(segment ...canvas.Point) {
		count++
		if len(segment) != 2 {
			t.Fatal("invalid dash segment")
		}
		for _, p := range segment {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) || p.X < 0 || p.X > 257 || p.Y < 0 || p.Y > 100 {
				t.Fatal("dash escaped original screen path", p)
			}
		}
	})
	if count == 0 || count > 1600 {
		t.Fatal("unbounded or absent dashed reference geometry", count)
	}
	dashedScopeLine([]canvas.Point{{}, {}, {}}, func(...canvas.Point) { t.Fatal("zero-length path generated a dash") })
}

func TestRecordedNarrowWindowDrawsCursorAtItsLabeledEndpoint(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	duration, step := .004, .002
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, Step: &step, InputMode: "charge", ProbeIDs: []string{"v:C1"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	agentAction(t, s.service, "recording.window.set", workbench.RecordingWindowParams{Start: .001, End: .002})
	agentAction(t, s.service, "recording.cursor.set", workbench.CursorParams{Cursor: "a", Time: .002})
	s.app.Do(func() {
		view := canvas.Viewport{Width: 800, Height: 160, Zoom: 1}
		mesh, err := s.scopeMesh(view)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, vertex := range mesh.Vertices() {
			p := view.Screen(vertex.Position)
			if vertex.Color == ink && p.Y > 129 && math.Abs(p.X-722) < 2 {
				found = true
			}
		}
		if !found {
			t.Fatal("cursor at the labeled end of a sub-step interval is drawn inside the plot instead of at its right edge")
		}
	})
}
