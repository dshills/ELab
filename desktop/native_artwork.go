package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"sync"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/svg"
	"github.com/dshills/vellum/canvas"
)

var nativeIconCache sync.Map

// Render the runtime's bundled Material Symbols artwork, not a new icon system.
// Fixed immutable artwork avoids depending on mutable private Button parts.
func nativeArtwork(icon icons.Icon, ink color.NRGBA) *canvas.TiledImage {
	key := fmt.Sprintf("%s/%x", icon, ink)
	if cached, ok := nativeIconCache.Load(key); ok {
		return cached.(*canvas.TiledImage)
	}
	drawing := svg.NewSVG(math32.Vec2(24, 24))
	drawing.DefaultFill = image.NewUniform(ink)
	if err := drawing.ReadXML(strings.NewReader(string(icon))); err != nil {
		return nil
	}
	rendered := drawing.RenderImage()
	bitmap := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	draw.Draw(bitmap, bitmap.Bounds(), rendered, rendered.Bounds().Min, draw.Src)
	pixels, err := canvas.NewTiledImage(bitmap)
	if err != nil {
		return nil
	}
	actual, _ := nativeIconCache.LoadOrStore(key, pixels)
	return actual.(*canvas.TiledImage)
}
