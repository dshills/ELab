package main

import (
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type nativeGlyph struct {
	bounds  image.Rectangle
	alpha   []uint8
	advance float64
}

// Cache immutable glyph coverage behind a synchronized rasterizer. The bundled
// outline font is used by every sheet/scope mesh, independent of OS fonts. The
// mutex protects the face's rasterizer as snapshots may render off-thread.
var nativeCanvasGlyph = sync.OnceValue(func() func(rune) nativeGlyph {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	} // Compile-time bundled Go font, never external input.
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 13, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		panic(err)
	}
	var mu sync.Mutex
	cache := map[rune]nativeGlyph{}
	return func(r rune) nativeGlyph {
		mu.Lock()
		defer mu.Unlock()
		if glyph, ok := cache[r]; ok {
			return glyph
		}
		rect, mask, origin, advance, ok := face.Glyph(fixed.P(0, 0), r)
		if !ok {
			rect, mask, origin, advance, _ = face.Glyph(fixed.P(0, 0), '?')
		}
		glyph := nativeGlyph{bounds: rect, advance: float64(advance) / 64, alpha: make([]uint8, rect.Dx()*rect.Dy())}
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				_, _, _, a := mask.At(origin.X+x-rect.Min.X, origin.Y+y-rect.Min.Y).RGBA()
				glyph.alpha[(y-rect.Min.Y)*rect.Dx()+x-rect.Min.X] = uint8(a >> 8)
			}
		}
		cache[r] = glyph
		return glyph
	}
})
