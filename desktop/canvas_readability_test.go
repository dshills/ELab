package main

import (
	"image/color"
	"math"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

func TestCanvasCaptionsAndWireStrokesRemainVisibleAtCompactFit(t *testing.T) {
	for _, zoom := range []float64{.42, .6, 1} {
		ink := color.NRGBA{A: 255}
		caption := &meshBuilder{}
		caption.textAtScale(0, 0, "R2 10 kohm", ink, workbench.CanvasFontScale(zoom))
		minY, maxY := math.Inf(1), math.Inf(-1)
		for _, v := range caption.vertices {
			minY, maxY = min(minY, v.Position.Y), max(maxY, v.Position.Y)
		}
		if (maxY-minY)*zoom < 8 {
			t.Fatalf("caption shrinks below eight ink pixels at zoom %g: %g", zoom, (maxY-minY)*zoom)
		}
		wire := &meshBuilder{minimumStroke: 1.25 / zoom}
		wire.stroke(ink, 2.5, canvas.Point{}, canvas.Point{X: 100})
		minY, maxY = math.Inf(1), math.Inf(-1)
		for _, v := range wire.vertices {
			minY, maxY = min(minY, v.Position.Y), max(maxY, v.Position.Y)
		}
		if (maxY-minY)*zoom < 1.25-1e-12 {
			t.Fatalf("wire is subpixel at zoom %g", zoom)
		}
	}
}

func TestCanvasAnnotationLinesKeepSpacingAndRightAlignment(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	state := s.service.Snapshot()
	if len(state.SourceAnnotations) != 1 {
		t.Fatal("RC source annotation missing")
	}
	for _, a := range state.SourceAnnotations {
		for _, rotation := range []int{0, 90, 180, 270} {
			layout := workbench.LayoutCanvasAnnotation(a, rotation, .42)
			if layout.LineHeight < 13*layout.Scale {
				t.Fatal("source annotation lines overlap at compact Fit")
			}
			for i, text := range a.Lines {
				at := layout.Position(text, i)
				right := at.X + workbench.CanvasTextWidth(text, layout.Scale)
				if math.Abs(right-layout.Right) > 1e-10 {
					t.Fatal("source annotation lost right alignment")
				}
			}
		}
	}
}
