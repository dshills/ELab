package schematicpdf

import (
	"bytes"
	"context"
	"image/png"
	"math"
	"testing"
)

func TestTiledPlanCoverageAndOverlap(t *testing.T) {
	svg := []byte(`<svg viewBox="-20 -30 900 600"/>`)
	page := Page{Width: 500, Height: 400, Margin: 20, Scale: 1, Tile: true, Overlap: 10}
	plan, err := Plan(svg, page)
	if err != nil || plan.Rows != 2 || plan.Columns != 2 || len(plan.Pages) != 4 {
		t.Fatal("unexpected tile grid", plan, err)
	}
	for i, p := range plan.Pages {
		if p.Scale != 1 || p.Index != i || p.Row != i/2 || p.Column != i%2 || p.Footer == "" {
			t.Fatal("tile changed scale/order", p)
		}
		// Independent mapping back from page points must cover each corner and
		// maintain ten points of shared content between neighboring tile views.
		left, top := (20-p.X)/p.Scale, (20-p.Y)/p.Scale
		if left != -20+float64(p.Column)*450 || top != -30+float64(p.Row)*332 {
			t.Fatal("tile origin", left, top)
		}
	}
	for _, point := range [][2]float64{{-20, -30}, {880, -30}, {-20, 570}, {880, 570}} {
		covered := false
		for _, p := range plan.Pages {
			x, y := point[0]*p.Scale+p.X, point[1]*p.Scale+p.Y
			if x >= 20 && x <= 480 && y >= 20 && y <= 362 {
				covered = true
			}
		}
		if !covered {
			t.Fatal("layout dropped schematic corner", point)
		}
	}
	page.Scale = 0
	if _, err := Plan(svg, page); err == nil {
		t.Fatal("tile fit silently rescaled")
	}
	page.Scale = 100
	if _, err := Plan(svg, page); err == nil {
		t.Fatal("unbounded tile count")
	}
	page.Scale = 1
	page.Overlap = math.NaN()
	if _, err := Plan(svg, page); err == nil {
		t.Fatal("invalid overlap")
	}
}

func TestTiledRasterClippingAndPageSelection(t *testing.T) {
	if !Supported() {
		t.Skip("macOS backend")
	}
	svg := []byte(`<svg viewBox="0 0 300 100"><rect x="0" y="0" width="300" height="100" fill="#ff0000"/></svg>`)
	page := Page{Width: 200, Height: 150, Margin: 10, Scale: 1, Tile: true, Overlap: 10, RasterDPI: 72}
	if _, err := Render(context.Background(), svg, page); err == nil {
		t.Fatal("multi-page PNG silently cropped")
	}
	for index := 0; index < 2; index++ {
		page.PageIndex = &index
		data, err := Render(context.Background(), svg, page)
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		for _, point := range [][2]int{{0, 50}, {199, 50}, {50, 0}} {
			r, g, b, _ := im.At(point[0], point[1]).RGBA()
			if r != 65535 || g != 65535 || b != 65535 {
				t.Fatal("content leaked into tile margins", index, point)
			}
		}
		r, g, b, _ := im.At(20, 20).RGBA()
		if r != 65535 || g != 0 || b != 0 {
			t.Fatal("selected tile lost content", index)
		}
	}
	bad := 2
	page.PageIndex = &bad
	if _, err := Render(context.Background(), svg, page); err == nil {
		t.Fatal("out-of-range raster page")
	}
	page.RasterDPI = 0
	page.PageIndex = nil
	data, err := Render(context.Background(), svg, page)
	if err != nil || !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("tiled PDF", err)
	}
}
