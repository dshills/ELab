package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeSourceEditsAndFitUseSharedAnnotations(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.rectifier"})
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "simulation.restart", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "V1"})
	uiAction(t, s, "sine-frequency", &ui.ChangeEvent{Text: "25"})
	uiAction(t, s, "sine-offset", &ui.ChangeEvent{Text: "1"})
	uiAction(t, s, "sine-phase", &ui.ChangeEvent{Text: "45"})
	uiAction(t, s, "simulation.restart", nil)
	before := s.service.Snapshot()
	a := before.SourceAnnotations[0]
	want := []string{"V1 | sine", "5 Vpk | 25 Hz", "offset 1 V | phase 45 deg"}
	if !reflect.DeepEqual(a.Lines, want) || a.State != "current" {
		t.Fatal("native parameter fields omit/round different source text", a)
	}
	uiAction(t, s, "view.fit", nil)
	after := s.service.Snapshot()
	if before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) || before.Document != after.Document {
		t.Fatal("native source Fit changed physics/dirty state")
	}
	min, max := workbench.SourceAnnotationBounds(after.SourceAnnotations[0])
	for _, p := range []workbench.Point{min, max} {
		x := (p.X-after.View.Center.X)*after.View.Zoom + s.drawing.View.Width/2
		y := (p.Y-after.View.Center.Y)*after.View.Zoom + s.drawing.View.Height/2
		if x < 0 || x > s.drawing.View.Width || y < 0 || y > s.drawing.View.Height {
			t.Fatal("native Fit clips annotation", p)
		}
	}
	s.app.Do(func() {
		mesh, err := s.mesh(*s.drawing.View)
		if err != nil || mesh == nil {
			t.Fatal("annotated native schematic cannot render", err)
		}
		// New annotation geometry is left of the source and contains actual ink,
		// rather than only an API label that the native canvas ignores.
		ink := 0
		for _, v := range mesh.Vertices() {
			if v.Position.X >= min.X && v.Position.X <= max.X && v.Position.Y >= min.Y && v.Position.Y <= max.Y && math.Abs(v.Position.X+300) > 30 {
				ink++
			}
		}
		if ink < 100 {
			t.Fatal("source annotation text is absent from native mesh")
		}
	})
}
