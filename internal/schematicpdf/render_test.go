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
