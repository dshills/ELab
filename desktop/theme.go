package main

import (
	"image/color"

	"cogentcore.org/core/colors/matcolor"
	"github.com/dshills/vellum/ui"
)

type appPalette struct{ Ink, Muted, Accent, Paper, Surface, Grid, Danger color.NRGBA }

func paletteForTheme(mode string) appPalette { return resolvePalette(mode, matcolor.SchemeIsDark) }
func resolvePalette(mode string, systemDark bool) appPalette {
	if mode == "dark" || mode == "system" && systemDark {
		return appPalette{
			Ink: color.NRGBA{R: 229, G: 237, B: 243, A: 255}, Muted: color.NRGBA{R: 163, G: 181, B: 195, A: 255},
			Accent: color.NRGBA{R: 88, G: 211, B: 197, A: 255}, Paper: color.NRGBA{R: 27, G: 38, B: 49, A: 255},
			Surface: color.NRGBA{R: 39, G: 53, B: 67, A: 255}, Grid: color.NRGBA{R: 52, G: 70, B: 85, A: 255},
			Danger: color.NRGBA{R: 255, G: 147, B: 153, A: 255},
		}
	}
	return appPalette{Ink: ink, Muted: muted, Accent: accent, Paper: paper, Surface: surface, Grid: color.NRGBA{R: 211, G: 222, B: 230, A: 255}, Danger: color.NRGBA{R: 168, G: 65, B: 68, A: 255}}
}

// Resolve colors per descriptor tree, without mutating a live driver theme or
// a process-wide native palette. This also keeps independent sessions isolated.
func applyPalette(node *ui.Node, palette appPalette) {
	selected := node.Style.Background.Set && node.Style.Background.Value == accent
	remap := func(c color.NRGBA) color.NRGBA {
		switch c {
		case ink:
			return palette.Ink
		case muted:
			return palette.Muted
		case accent:
			return palette.Accent
		case paper:
			return palette.Paper
		case surface:
			return palette.Surface
		case color.NRGBA{R: 168, G: 65, B: 68, A: 255}:
			return palette.Danger
		}
		return c
	}
	if node.Style.Foreground.Set {
		node.Style.Foreground.Value = remap(node.Style.Foreground.Value)
	} else {
		node.Style.Foreground = ui.Some(palette.Ink)
	}
	if node.Style.Background.Set {
		node.Style.Background.Value = remap(node.Style.Background.Value)
	}
	if selected {
		node.Style.Foreground = ui.Some(palette.Paper)
	}
	switch node.Kind {
	case ui.TextField, ui.NumberField, ui.MultilineText, ui.Selector, ui.MenuButton, ui.Button:
		if !node.Style.Background.Set {
			node.Style.Background = ui.Some(palette.Surface)
		}
	case ui.Extension:
		node.Style.Background = ui.Some(palette.Paper)
	}
	for i := range node.Children {
		applyPalette(&node.Children[i], palette)
	}
}

func (s *screen) themeMenu(mode string) ui.Node {
	text := "Theme"
	switch mode {
	case "light":
		text = "Light"
	case "dark":
		text = "Dark"
	case "system":
		text = "System"
	}
	node := buttonStyle(ui.Node{ID: "theme-menu", Kind: ui.MenuButton, Text: text, Accessibility: ui.Accessibility{Name: "Appearance theme"}, Options: []ui.Option{{ID: "light", Label: "Light"}, {ID: "dark", Label: "Dark"}, {ID: "system", Label: "System"}}, Change: func(e ui.ChangeEvent) { s.invoke("appearance.theme.set", map[string]string{"theme": e.SelectedID}) }}, false)
	return s.platformMenu(node, "Appearance")
}

func traceColor(c color.NRGBA, palette appPalette) color.NRGBA {
	if c == accent {
		return palette.Accent
	}
	if palette.Paper.R < 100 {
		c.R = uint8((int(c.R)*4 + 255) / 5)
		c.G = uint8((int(c.G)*4 + 255) / 5)
		c.B = uint8((int(c.B)*4 + 255) / 5)
	}
	return c
}
