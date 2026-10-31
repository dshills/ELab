package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// Vellum has no public stacking layout in the pinned runtime. Keep this small
// app-local adapter on the existing managed nodes: normal button focus, hit
// testing and callback guards stay with the driver. The sheet alone has custom
// positions; its canvas gets the complete allocation, with controls above it.
func (s *screen) styleNativeSheet(widget core.Widget, w *core.WidgetBase) bool {
	if w.Name == "root" {
		changed := false
		for ancestor := w.Parent; ancestor != nil; ancestor = ancestor.AsTree().Parent {
			if parent, ok := ancestor.(core.Widget); ok {
				pw := parent.AsWidget()
				if _, marked := pw.Property("elab.native-root").(bool); !marked {
					pw.SetProperty("elab.native-root", true)
					pw.FinalStyler(func(st *styles.Style) { st.Padding.Zero(); st.Margin.Zero(); st.Border.Width.Zero() })
					changed = true
				}
			}
		}
		if changed {
			return true
		}
	}
	if w.Name == "native-center" {
		if _, ok := w.Property("elab.native-center").(bool); !ok {
			w.SetProperty("elab.native-center", true)
			w.FinalStyler(func(st *styles.Style) { st.Min.Zero(); st.Grow.Set(1, 1); st.Overflow.Set(styles.OverflowHidden) })
			return true
		}
	}
	if w.Name == "schematic-panel" {
		if _, ok := w.Property("elab.native-sheet").(bool); !ok {
			w.SetProperty("elab.native-sheet", true)
			w.FinalStyler(func(st *styles.Style) {
				st.Display = styles.Custom
				st.Overflow.Set(styles.OverflowHidden)
				st.Padding.Zero()
				st.Grow.Set(1, 1)
			})
			return true
		}
	}
	if w.Parent != nil && w.Parent.AsTree().Name == "schematic-panel" {
		if _, ok := w.Property("elab.native-overlay").(bool); !ok {
			w.SetProperty("elab.native-overlay", true)
			w.FinalStyler(func(st *styles.Style) { nativeOverlayStyle(w.Name, w.NumChildren() > 4, st) })
			return true
		}
	}
	return false
}

func nativeToolPillWidth(compact, cancel bool) float32 {
	if compact {
		if cancel {
			return 250
		}
		return 172
	}
	if cancel {
		return 460
	}
	return 340
}

// Custom units are evaluated by Core after the parent receives its current
// allocation. Reading Geom in a Styler instead uses the previous frame and can
// strand a floating control offscreen after a resize or drawer change.
func nativeOverlayStyle(name string, cancel bool, st *styles.Style) {
	st.Overflow.Set(styles.OverflowHidden)
	scale := func(uc *units.Context) float32 {
		if uc.DPI <= 0 {
			return 1
		}
		return uc.DPI / units.DpPerInch
	}
	pill := func(uc *units.Context) float32 {
		width := uc.Paw / scale(uc)
		return nativeToolPillWidth(width < 700, cancel)
	}
	switch name {
	case "schematic":
		st.Grow.Set(1, 1)
		st.Min.Zero()
		st.Max.Zero()
		st.Pos.Set(units.Dp(0))
	case "tools":
		st.Min.Set(units.Dp(300), units.Dp(34))
		st.Min.X = units.Custom(func(uc *units.Context) float32 { return pill(uc) * scale(uc) })
		st.Max = st.Min
		st.Grow.Set(0, 0)
		st.Pos.X = units.Custom(func(uc *units.Context) float32 { return max(0, (uc.Paw-(pill(uc)+10)*scale(uc))/2) })
		st.Pos.Y = units.Custom(func(uc *units.Context) float32 { return max(0, uc.Pah-64*scale(uc)) })
		st.BoxShadow = nil
	case "native-zoom":
		st.Min.Set(units.Dp(100), units.Dp(28))
		st.Max = st.Min
		st.Grow.Set(0, 0)
		st.Pos.X = units.Custom(func(uc *units.Context) float32 { return max(0, uc.Paw-130*scale(uc)) })
		st.Pos.Y = units.Custom(func(uc *units.Context) float32 {
			// At narrow widths the labeled tool bar can reach the zoom control.
			offset := float32(52)
			if width := uc.Paw / scale(uc); width < nativeToolPillWidth(width < 700, true)+260 {
				offset += 44
			}
			return max(0, uc.Pah-offset*scale(uc))
		})
	case "native-sampling-notice", "operation-notice", "operation-notice-area":
		maxWidth, height := float32(650), float32(34)
		if name == "native-sampling-notice" {
			maxWidth, height = 480, 26
		}
		notice := func(uc *units.Context) float32 { return max(0, min(maxWidth, uc.Paw/scale(uc)-40)) * scale(uc) }
		st.Min.Set(units.Dp(maxWidth), units.Dp(height))
		st.Min.X = units.Custom(notice)
		st.Max = st.Min
		if name != "native-sampling-notice" {
			st.Max.Y = units.Dp(60)
		}
		st.Grow.Set(0, 0)
		st.Pos.X = units.Custom(func(uc *units.Context) float32 { return max(0, (uc.Paw-notice(uc))/2) })
		st.Pos.Y = units.Dp(14)
	}
}
