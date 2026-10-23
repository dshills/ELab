package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestDiagnosticDisclosureUsesTheSessionPalette(t *testing.T) {
	for _, mode := range []string{"light", "dark"} {
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
	for _, mode := range []string{"light", "dark", "system"} {
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
