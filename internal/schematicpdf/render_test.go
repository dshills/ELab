package schematicpdf

import (
	"bytes"
	"context"
	"image/png"
	"math"
	"testing"
)

func TestRasterPageBounds(t *testing.T) {
	for _, p := range []Page{
		{Width: math.NaN(), Height: 100}, {Width: 100, Height: math.Inf(1)},
		{Width: 100, Height: 100, Margin: 50}, {Width: 100, Height: 100, Scale: -1},
		{Width: 792, Height: 612, RasterDPI: 301}, {Width: 792, Height: 612, RasterDPI: 71},
		{Width: 10000, Height: 10000, RasterDPI: 300},
	} {
		if _, err := Render(context.Background(), nil, p); err == nil {
			t.Fatal("invalid page accepted", p)
		}
	}
}

func TestRasterOrientationColorAndOpaquePaper(t *testing.T) {
	if !Supported() {
		t.Skip("macOS rendering backend")
	}
	// Asymmetric shapes independently check page orientation and the Y flip.
	svg := []byte(`<svg viewBox="0 0 100 200"><rect x="10" y="10" width="20" height="20" fill="#ff0000"/><circle cx="80" cy="180" r="10" fill="#0000ff"/></svg>`)
	for _, gray := range []bool{false, true} {
		data, err := Render(context.Background(), svg, Page{Width: 100, Height: 200, RasterDPI: 72, Grayscale: gray})
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(bytes.NewReader(data))
		if err != nil || im.Bounds().Dx() != 100 || im.Bounds().Dy() != 200 {
			t.Fatal("PNG dimensions", err)
		}
		r, g, b, a := im.At(20, 20).RGBA()
		if a != 65535 || (gray && (r != g || g != b || r == 65535)) || (!gray && (r != 65535 || g != 0 || b != 0)) {
			t.Fatal("top red rect is flipped or incorrectly colored", r, g, b, a)
		}
		r, g, b, a = im.At(80, 180).RGBA()
		if a != 65535 || (gray && (r != g || g != b || r == 65535)) || (!gray && (r != 0 || g != 0 || b != 65535)) {
			t.Fatal("bottom blue circle is flipped or incorrectly colored", r, g, b, a)
		}
		for _, xy := range [][2]int{{0, 0}, {99, 0}, {0, 199}, {99, 199}, {50, 100}} {
			r, g, b, a = im.At(xy[0], xy[1]).RGBA()
			if r != 65535 || g != 65535 || b != 65535 || a != 65535 {
				t.Fatal("paper is not opaque white", xy)
			}
		}
	}
}

func TestRasterTextEndAnchorUsesGlyphAdvance(t *testing.T) {
	if !Supported() {
		t.Skip("macOS raster backend")
	}
	svg := []byte(`<svg viewBox="0 0 200 100"><text x="50" y="40" font-size="20">WAVE</text><text x="150" y="80" font-size="20" text-anchor="end">WAVE</text></svg>`)
	data, err := Render(context.Background(), svg, Page{Width: 200, Height: 100, RasterDPI: 72})
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for index, band := range [][2]int{{15, 45}, {55, 85}} {
		left, right := 200, -1
		for y := band[0]; y < band[1]; y++ {
			for x := 0; x < 200; x++ {
				r, _, _, _ := im.At(x, y).RGBA()
				if r < 50000 {
					left, right = min(left, x), max(right, x)
				}
			}
		}
		if right-left < 40 || index == 0 && (left < 49 || left > 51) || index == 1 && (right > 150 || right < 147 || left < 70) {
			t.Fatal("text start/end anchor ignored or clipped", index, left, right)
		}
	}
}
