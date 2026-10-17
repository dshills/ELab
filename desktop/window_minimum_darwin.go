//go:build darwin && cgo

package main

import (
	"image"
	"math"
	"sync/atomic"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/units"
	native "cogentcore.org/core/system/driver/desktop"
	"github.com/go-gl/glfw/v3.4/glfw"
)

type windowMinimumStamp struct {
	window *native.Window
	dpi    float32
}

// The public framework currently exposes an initial size, not a resize floor.
// Keep this adapter on the mounted main scene. Queue native work without holding
// the render owner, and recheck lifetime on the same main queue as native close.
func applyNativeWindowMinimum(scene *core.Scene) {
	render := scene.RenderWindow()
	if render == nil {
		return
	}
	window, ok := render.SystemWindow.(*native.Window)
	if !ok || window.IsClosed() {
		return
	}
	stamp := windowMinimumStamp{window, scene.Styles.UnitContext.DPI}
	if stamp.dpi <= 0 || math.IsNaN(float64(stamp.dpi)) || math.IsInf(float64(stamp.dpi), 0) {
		return
	}
	latest, _ := scene.Property("elab.window.minimum").(*atomic.Pointer[windowMinimumStamp])
	if latest == nil {
		latest = &atomic.Pointer[windowMinimumStamp]{}
		scene.SetProperty("elab.window.minimum", latest)
	}
	if previous := latest.Load(); previous != nil && *previous == stamp {
		return
	}
	latest.Store(&stamp)
	width, height := units.Dp(minimumWindowWidthDP), units.Dp(minimumWindowHeightDP)
	pixels := image.Pt(int(math.Ceil(float64(width.ToDots(&scene.Styles.UnitContext)))), int(math.Ceil(float64(height.ToDots(&scene.Styles.UnitContext)))))
	// RunOnMainQueued waits for completion; keep that wait off the render owner.
	go window.App.RunOnMainQueued(func() {
		if latest.Load() != &stamp || window.IsClosed() || window.Glw == nil {
			return
		}
		screen := window.Screen()
		if screen == nil {
			return
		}
		// Use the mounted UI's unit context, which can differ from the screen's
		// initial-size DPI. The native conversion still respects Retina scale.
		size := screen.WindowSizeFromPixels(pixels)
		available := screen.Geometry.Size()
		size.X, size.Y = min(size.X, available.X), min(size.Y, available.Y)
		if size.X <= 0 || size.Y <= 0 {
			return
		}
		window.Glw.SetSizeLimits(size.X, size.Y, glfw.DontCare, glfw.DontCare)
	})
}
