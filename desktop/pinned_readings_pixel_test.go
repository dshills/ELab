package main

import (
	"image/color"
	"math"
	"testing"
)

func TestReadingTextCoverageUsesActualResolvedGlyphPixels(t *testing.T) {
	background := color.NRGBA{R: 243, G: 246, B: 249, A: 255}
	ink := color.NRGBA{R: 39, G: 110, B: 126, A: 255}
	// Actual owned native snapshot pixels: quarter/half/three-quarter/full
	// glyph coverage after linear-light multisample resolution and sRGB decode.
	for _, test := range []struct {
		pixel color.NRGBA
		want  float64
	}{{color.NRGBA{R: 215, G: 222, B: 226, A: 255}, .25}, {color.NRGBA{R: 180, G: 194, B: 200, A: 255}, .5}, {color.NRGBA{R: 134, G: 159, B: 168, A: 255}, .75}, {ink, 1}} {
		if got := readingTextCoverage(test.pixel, background, ink); math.Abs(got-test.want) > .01 {
			t.Fatal("resolved glyph coverage incorrect", test.pixel, got, test.want)
		}
	}
	for _, pixel := range []color.NRGBA{background, {R: 255, A: 255}, {R: 39, G: 110, B: 126, A: 0}, {R: 39, G: 200, B: 30, A: 255}} {
		if got := readingTextCoverage(pixel, background, ink); got != 0 {
			t.Fatal("background/unrelated color counted as text", pixel, got)
		}
	}
}
