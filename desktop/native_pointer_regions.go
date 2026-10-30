package main

import (
	"image"

	"cogentcore.org/core/core"
)

// The AppKit event bridge owns only exposed sheet pixels. Floating managed
// controls are siblings above the mesh, so their complete painted rectangles
// must pass through to the normal UI event path, including disabled controls.
func nativeSheetExclusions(canvas *core.WidgetBase) [4]image.Rectangle {
	var out [4]image.Rectangle
	if canvas == nil || canvas.Parent == nil {
		return out
	}
	count := 0
	for _, child := range canvas.Parent.AsTree().Children {
		w, ok := child.(core.Widget)
		if !ok {
			continue
		}
		b := w.AsWidget()
		switch b.Name {
		case "tools", "native-zoom", "native-sampling-notice", "operation-notice", "operation-notice-area":
		default:
			continue
		}
		if !b.IsVisible() || b.Geom.TotalBBox.Empty() {
			continue
		}
		rect := b.Geom.TotalBBox
		if b.Scene != nil {
			rect = rect.Add(b.Scene.SceneGeom.Pos)
		}
		out[count] = rect
		count++
		if count == len(out) {
			break
		}
	}
	return out
}

// Core sends primary positional events to every widget underneath the point,
// even when an overlapping sibling owns the click. Apply the same exclusion to
// this fallback path as to the AppKit bridge. Core event positions are relative
// to this scene; bridge positions include its offset in the content view.
func nativeSheetOverlayHit(canvas *core.WidgetBase, position image.Point) bool {
	if canvas != nil && canvas.Scene != nil {
		position = position.Add(canvas.Scene.SceneGeom.Pos)
	}
	for _, rect := range nativeSheetExclusions(canvas) {
		if position.In(rect) {
			return true
		}
	}
	return false
}
