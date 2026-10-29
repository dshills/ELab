package main

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
)

// Vellum's public theme does not yet expose native scrollbar/private disclosure
// styling. Keep this adapter local to this app's mounted scene, and retain the
// backend's layout, input, and accessibility implementations.
func (s *screen) styleNativeChrome(scene *core.Scene) bool {
	if scene == nil {
		return false
	}
	applyNativeWindowMinimum(scene)
	changed := false
	install := func(w *core.WidgetBase, style func(*styles.Style)) {
		if marker, _ := w.Property("elab.chrome.styled").(bool); marker {
			return
		}
		w.SetProperty("elab.chrome.styled", true)
		w.FinalStyler(style)
		changed = true
	}
	scene.WidgetWalkDown(func(widget core.Widget, w *core.WidgetBase) bool {
		if styleComponentRow(widget, w) {
			changed = true
		}
		if row, ok := widget.(*core.Switch); ok && w.Parent != nil && w.Parent.AsTree().Name == "part-list" {
			install(w, func(style *styles.Style) {
				p := s.nativeColors()
				if p.FontFamily == "" {
					return
				}
				style.Font.Family = rich.Custom
				style.Font.CustomFont = rich.FontName(p.FontFamily)
				style.Border.Radius.Set(units.Dp(p.ControlRadius))
				style.Color = image.NewUniform(p.Ink)
				style.Background = image.NewUniform(p.Paper)
				if row.IsChecked() {
					style.Color = image.NewUniform(p.Paper)
					style.Background = image.NewUniform(p.Accent)
				}
			})
		}
		if frame := core.AsFrame(widget); frame != nil {
			sideHost := isSidePanelHost(w) || w.Name == "inspector-rail" || w.Name == "inspector-navigation" || w.Name == "simulation-tool-navigation"
			install(w, func(style *styles.Style) {
				style.ScrollbarWidth = units.Dp(8)
				// Inspector chrome and each side host own a Scroll viewport. A second
				// auto-scroller on Splits' private host reserves another bar
				// and makes an exactly fitted panel overflow horizontally.
				if sideHost {
					style.Overflow.Set(styles.OverflowHidden)
				}
			})
			for _, scrollbar := range frame.Scrolls {
				if scrollbar == nil {
					continue
				}
				install(scrollbar.AsWidget(), func(style *styles.Style) {
					p := s.nativeColors()
					style.Background = image.NewUniform(p.Paper)
					style.Color = image.NewUniform(p.Muted)
					style.StateLayer = 0
					// Core uses ValueColor for the scrollbar handle.
					scrollbar.ValueColor = image.NewUniform(p.Muted)
					scrollbar.ThumbColor = image.NewUniform(p.Muted)
				})
			}
		}
		if w.Name == "vellum-disclosure-heading" {
			install(w, func(style *styles.Style) {
				style.Color = image.NewUniform(s.nativeColors().Accent)
				style.Background = image.NewUniform(s.nativeColors().Paper)
			})
		}
		return true
	})
	return changed
}

func (s *screen) nativeColors() appPalette {
	if palette := s.nativePalette.Load(); palette != nil {
		return *palette
	}
	return resolvePalette("light", false)
}

// Only the owned left Parts host and right Inspector host are viewports here.
// Preserve the center workspace and every managed Scroll's normal overflow.
func isSidePanelHost(w *core.WidgetBase) bool {
	if w.Parent == nil || w.Parent.AsTree().NumChildren() != 2 {
		return false
	}
	parent := w.Parent.AsTree()
	return parent.Name == "parts-split" && parent.Child(0).AsTree() == w.AsTree() || parent.Name == "inspector-split" && parent.Child(1).AsTree() == w.AsTree()
}
