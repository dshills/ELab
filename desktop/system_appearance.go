package main

import (
	"image/color"
	"math"

	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func (s *screen) observeSystemAppearance(value cogent.SystemAppearance) {
	if !value.Known || value.Accent.A != 255 {
		return
	}
	s.systemAppearance.Store(&value)
	s.reduceMotion.Store(value.ReduceMotion)
	// Preference observation is presentation state. It must not create a
	// document edit, change a run, or overwrite the user's explicit theme.
	s.app.Invalidate("main")
}

func (s *screen) colorsForTheme(mode string) appPalette {
	p := paletteForTheme(mode)
	value := s.systemAppearance.Load()
	if value == nil {
		return p
	}
	if mode == "system" {
		p = resolvePalette(mode, value.Dark)
		p.Accent = value.Accent
	}
	if value.HighContrast {
		if p.Paper.R < 100 {
			p.Ink = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		} else {
			p.Ink = color.NRGBA{A: 255}
		}
		p.Muted, p.Outline = p.Ink, p.Ink
	}
	return p
}

func (s *screen) scopedTheme(p appPalette, reducedMotion bool) ui.Theme {
	theme := desktopTheme()
	// Composite native choices read the scoped default for their private rows,
	// rather than inheriting the outer Node's explicitly remapped ink.
	theme.Default.Foreground = ui.Some(p.Ink)
	if p.FontFamily != "" {
		typography := theme.Default.Typography.Value
		typography.Family = p.FontFamily
		theme.Default.Typography = ui.Some(typography)
	}
	theme.Palette = ui.Some(ui.Palette{Window: p.Paper, Panel: p.Paper, Field: p.Surface, Popup: p.Paper,
		Ink: p.Ink, Secondary: p.Muted, Line: toolbarOutline(p), Accent: p.Accent, AccentInk: accentForeground(p.Accent)})
	theme.ReducedMotion = s.reduceMotion.Load() || reducedMotion
	return theme
}

func accentForeground(accent color.NRGBA) color.NRGBA {
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= .04045 {
			return s / 12.92
		}
		return math.Pow((s+.055)/1.055, 2.4)
	}
	luminance := .2126*linear(accent.R) + .7152*linear(accent.G) + .0722*linear(accent.B)
	if (luminance+.05)/.05 >= 1.05/(luminance+.05) {
		return color.NRGBA{A: 255}
	}
	return paper
}

// Custom canvas paints also depend on environmental appearance, which does
// not advance the service's document/simulation revision.
type canvasAppearanceRevision struct {
	model, serial uint64
	palette       appPalette
}

func (r *canvasAppearanceRevision) next(model uint64, palette appPalette) uint64 {
	if model != r.model || palette != r.palette || r.serial == 0 {
		r.model, r.palette = model, palette
		r.serial++
	}
	return r.serial
}
