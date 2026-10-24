package main

import (
	"image/color"

	"cogentcore.org/core/colors/matcolor"
	"github.com/dshills/vellum/ui"
)

type appPalette struct {
	Ink, Muted, Accent, Paper, Surface, Grid, Danger color.NRGBA
	ControlRadius                                    float32
}

func paletteForTheme(mode string) appPalette { return resolvePalette(mode, matcolor.SchemeIsDark) }
func resolvePalette(mode string, systemDark bool) appPalette {
	if mode == "apollo" {
		// Gray-green instrument panels, engraved cream legends and amber
		// annunciators. Keep signal/warning colors distinct from panel ink.
		return appPalette{
			Ink: color.NRGBA{R: 239, G: 235, B: 218, A: 255}, Muted: color.NRGBA{R: 182, G: 188, B: 170, A: 255},
			Accent: color.NRGBA{R: 236, G: 195, B: 106, A: 255}, Paper: color.NRGBA{R: 29, G: 35, B: 33, A: 255},
			Surface: color.NRGBA{R: 49, G: 58, B: 54, A: 255}, Grid: color.NRGBA{R: 67, G: 77, B: 70, A: 255},
			Danger: color.NRGBA{R: 255, G: 151, B: 129, A: 255}, ControlRadius: 2,
		}
	}
	if mode == "dark" || mode == "system" && systemDark {
		return appPalette{
			Ink: color.NRGBA{R: 240, G: 240, B: 242, A: 255}, Muted: color.NRGBA{R: 164, G: 164, B: 170, A: 255},
			Accent: color.NRGBA{R: 100, G: 173, B: 255, A: 255}, Paper: color.NRGBA{R: 30, G: 30, B: 32, A: 255},
			Surface: color.NRGBA{R: 42, G: 42, B: 45, A: 255}, Grid: color.NRGBA{R: 57, G: 57, B: 61, A: 255},
			Danger: color.NRGBA{R: 255, G: 147, B: 153, A: 255}, ControlRadius: 5,
		}
	}
	return appPalette{Ink: ink, Muted: muted, Accent: accent, Paper: paper, Surface: surface, Grid: color.NRGBA{R: 224, G: 224, B: 228, A: 255}, Danger: color.NRGBA{R: 168, G: 65, B: 68, A: 255}, ControlRadius: 5}
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
	case ui.TextField, ui.NumberField, ui.MultilineText, ui.Selector, ui.MenuButton, ui.Button, ui.Disclosure:
		node.Style.CornerRadius = ui.Some(palette.ControlRadius)
		if node.Kind == ui.Button && node.Appearance == "" {
			node.Appearance = ui.ButtonSecondary
		}
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
	case "apollo":
		text = "Apollo"
	}
	node := buttonStyle(ui.Node{ID: "theme-menu", Kind: ui.MenuButton, Text: text, Accessibility: ui.Accessibility{Name: "Appearance theme"}, Options: []ui.Option{{ID: "light", Label: "Light", Checkable: true, Checked: mode == "light"}, {ID: "dark", Label: "Dark", Checkable: true, Checked: mode == "dark"}, {ID: "system", Label: "System", Checkable: true, Checked: mode == "system"}, {ID: "apollo", Label: "Apollo", Checkable: true, Checked: mode == "apollo"}}, Change: func(e ui.ChangeEvent) { s.invoke("appearance.theme.set", map[string]string{"theme": e.SelectedID}) }}, false)
	return node
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
