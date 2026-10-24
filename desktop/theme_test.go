package main

import (
	"context"
	"image/color"
	"math"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestDiagnosticDisclosureUsesTheSessionPalette(t *testing.T) {
	for _, mode := range []string{"light", "dark", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: mode})
			s.app.Do(func() {
				s.invoke("component.set", workbench.ValueParams{ID: "R1", Value: "oops"})
				node, ok := findNode(s.view(), "diagnostic-details")
				if !ok {
					t.Fatal("missing diagnostic disclosure")
				}
				style, err := ui.ResolveStyle(node, desktopTheme())
				palette := resolvePalette(mode, false)
				if err != nil || style.Foreground.Value != palette.Ink || style.Background.Value != palette.Surface {
					t.Fatal("disclosure falls back to a different session palette", mode, style, err)
				}
			})
		})
	}
}

func TestNativeThemeMenuMatchesSharedStateAndColors(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	before := s.service.Snapshot()
	uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: "dark"})
	after := s.service.Snapshot()
	if after.Theme != "dark" || after.ElectricalRevision != before.ElectricalRevision || after.Document.Dirty != before.Document.Dirty {
		t.Fatal("theme menu bypassed shared presentation command")
	}
	root := s.view()
	dark := resolvePalette("dark", false)
	for _, id := range []string{"part-search", "selection", "schematic", "rc-scope", "inspector-navigation"} {
		node, ok := findNode(root, id)
		if !ok && id == "rc-scope" {
			continue
		}
		if !ok {
			t.Fatal("missing themed control", id)
		}
		if node.Style.Foreground.Value != dark.Ink || !node.Style.Background.Set || node.Style.Background.Value.R > 100 {
			t.Fatal("dark theme failed", id)
		}
	}
	uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: "light"})
	if resolvePalette("system", true) != dark || resolvePalette("system", false) != resolvePalette("light", false) {
		t.Fatal("system appearance resolution")
	}
	// Ensure the driver receives an independently valid local theme.
	if err := desktopTheme().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestThemeMenuUsesValidCheckedActionContract(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		node := s.themeMenu(mode)
		if err := ui.Validate(node); err != nil {
			t.Fatal("portable theme menu invalid", err)
		}
		checked := 0
		for _, option := range node.Options {
			if !option.Checkable {
				t.Fatal("theme is not a controlled check action")
			}
			if option.Checked {
				checked++
				if option.ID != mode {
					t.Fatal("wrong theme checked")
				}
			}
		}
		if checked != 1 || node.SelectedID != "" {
			t.Fatal("theme used selector semantics")
		}
	}
}

func TestApolloThemeMatchesUserAndAgentAndRestoresOtherThemes(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: "apollo"})
	after := s.service.Snapshot()
	assertCursorMeasurementPreservesPhysics(t, before, after)
	if after.Theme != "apollo" || after.Document != before.Document || after.UndoName != before.UndoName {
		t.Fatal("Apollo did not remain a presentation-only shared preference")
	}
	p := resolvePalette("apollo", false)
	if resolvePalette("apollo", true) != p {
		t.Fatal("Apollo follows the system appearance instead of its selected palette")
	}
	for _, mode := range []string{"apollo", "light", "apollo", "dark"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
		root := s.view()
		palette := resolvePalette(mode, false)
		for _, id := range []string{"part-search", "selection", "schematic", "inspector-navigation"} {
			node, ok := findNode(root, id)
			if !ok || node.Style.Foreground.Value != palette.Ink || !node.Style.Background.Set {
				t.Fatal("theme did not reach the application control", mode, id)
			}
			if id == "part-search" && node.Style.CornerRadius.Value != palette.ControlRadius {
				t.Fatal("control shape did not follow the selected theme", mode)
			}
		}
	}
	s.app.Do(func() {
		if err := s.commands.Invoke(context.Background(), "appearance.apollo"); err != nil {
			t.Fatal("native application menu could not select Apollo", err)
		}
	})
	if s.service.Snapshot().Theme != "apollo" {
		t.Fatal("native menu command disagrees with the user and agent preference")
	}
}

func TestApolloPaletteTextContrast(t *testing.T) {
	// Check actual sRGB contrast of text, secondary labels, warnings and selected
	// controls rather than assuming that a darker panel makes them readable.
	luminance := func(c color.NRGBA) float64 {
		linear := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= .04045 {
				return s / 12.92
			}
			return math.Pow((s+.055)/1.055, 2.4)
		}
		return .2126*linear(c.R) + .7152*linear(c.G) + .0722*linear(c.B)
	}
	p := resolvePalette("apollo", false)
	for _, foreground := range []color.NRGBA{p.Ink, p.Muted, p.Accent, p.Danger} {
		for _, background := range []color.NRGBA{p.Paper, p.Surface} {
			a, b := luminance(foreground), luminance(background)
			contrast := (max(a, b) + .05) / (min(a, b) + .05)
			if contrast < 4.5 {
				t.Fatalf("Apollo text contrast %.2f is below 4.5:1: %v on %v", contrast, foreground, background)
			}
		}
	}
}
