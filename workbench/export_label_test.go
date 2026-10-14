package workbench

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"math"
	"testing"

	"github.com/dshills/elab/internal/schematicpdf"
)

func TestExportLabelBackgroundDoesNotEraseNearbyResistor(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS raster backend")
	}
	var bounds [4]float64
	render := func(capacitance float64) image.Image {
		t.Helper()
		g := RectifierExampleGraph()
		g.Components[3].Value = capacitance
		svg := schematicSVG(g, 1, 1)
		decoder := xml.NewDecoder(bytes.NewReader(svg))
		token, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		root := token.(xml.StartElement)
		var next [4]float64
		for _, a := range root.Attr {
			if a.Name.Local == "viewBox" {
				if _, err := fmt.Sscanf(a.Value, "%g %g %g %g", &next[0], &next[1], &next[2], &next[3]); err != nil {
					t.Fatal(err)
				}
			}
		}
		if bounds != [4]float64{} && bounds != next {
			t.Fatal("fixture page bounds changed")
		}
		bounds = next
		data, err := schematicpdf.Render(context.Background(), svg, schematicpdf.Page{Width: 792, Height: 612, Margin: 30, RasterDPI: 144})
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		return im
	}
	short, long := render(1e-6), render(47e-6)
	// Both actual labels end before RL. Only the conservative white background
	// of the longer label reaches its left-hand zigzag. Compare the entire body.
	scale := math.Min(732/bounds[2], 552/bounds[3]) * 2
	x := (1584-bounds[2]*scale)/2 + (340-bounds[0])*scale
	y := (1224-bounds[3]*scale)/2 + (20-bounds[1])*scale
	ink := 0
	for py := int(y - 25*scale); py <= int(y+25*scale); py++ {
		for px := int(x - 12*scale); px <= int(x+12*scale); px++ {
			a, b := short.At(px, py), long.At(px, py)
			ar, ag, ab, aa := a.RGBA()
			br, bg, bb, ba := b.RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				t.Fatalf("longer capacitor label erases adjacent resistor at %d,%d", px, py)
			}
			if ar < 50000 {
				ink++
			}
		}
	}
	if ink < 100 {
		t.Fatal("test crop missed the resistor symbol")
	}
}
