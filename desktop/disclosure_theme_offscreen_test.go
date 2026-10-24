//go:build offscreen

package main

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorDisclosurePaletteAndSharedExpansion(t *testing.T) {
	for _, mode := range []string{"light", "dark", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.rc", nil)
			uiAction(t, s, "simulation.pause", nil)
			uiAction(t, s, "simulation.restart", nil)
			agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
			uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: mode})
			uiAction(t, s, "inspector-section-simulation", nil)
			s.app.Do(func() {
				field, _ := findNode(s.view(), "cursor-time-a")
				field.Change(ui.ChangeEvent{Text: "oops"})
			})
			before := s.service.Snapshot()
			view := func() ui.Node {
				rail, _ := findNode(s.view(), "inspector-rail")
				return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
			}
			theme := inspectorLayoutTestTheme()
			theme.Default.Background = ui.Some(color.NRGBA{})
			m := newMountedControlWindowSize(t, s, view, 320, 360, theme)
			headingID := ""
			for _, element := range m.snapshot().Elements {
				if element.Kind == ui.Disclosure && element.Name == "Technical details" && strings.HasPrefix(element.ID, "disclosure:") {
					headingID = element.ID
				}
			}
			heading := mountedControlElement(t, m, headingID)
			if !heading.Visible || heading.Bounds.Empty() {
				t.Fatal("disclosure heading is not visible")
			}
			im, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			if artifacts := os.Getenv("ELAB_DISCLOSURE_THEME_ARTIFACTS"); artifacts != "" {
				f, err := os.OpenFile(filepath.Join(artifacts, mode+".png"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				encodeErr, closeErr := png.Encode(f, im), f.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatal(encodeErr, closeErr)
				}
			}
			palette := resolvePalette(mode, false)
			x, y := heading.Bounds.Max.X-12, heading.Bounds.Min.Y+heading.Bounds.Dy()/2
			if got := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA); got != palette.Surface {
				t.Fatal("painted disclosure surface ignores session palette", got, palette.Surface)
			}
			// Inspect actual glyph pixels as well as the descriptor: the old
			// private heading button overrides its parent's controlled foreground.
			inkPixels := 0
			text := heading.Bounds.Inset(8)
			text.Min.X += 24
			for y := text.Min.Y; y < text.Max.Y; y++ {
				for x := text.Min.X; x < text.Max.X; x++ {
					p := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
					sum := int(p.R) + int(p.G) + int(p.B)
					if mode != "light" && sum > 600 || mode == "light" && sum < 240 {
						inkPixels++
					}
				}
			}
			if inkPixels < 10 {
				t.Fatal("painted heading does not use readable session ink", mode, inkPixels)
			}
			clickMountedControl(t, m, headingID)
			after := s.service.Snapshot()
			if !after.DiagnosticPresentation.DetailsVisible || !mountedControlElement(t, m, "diagnostic-0-technical").Visible {
				t.Fatal("themed heading lost its shared expansion action")
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
			agentAction(t, s.service, "diagnostics.details.set", workbench.DiagnosticDetailsParams{Visible: false})
			if s.service.Snapshot().DiagnosticPresentation.DetailsVisible {
				t.Fatal("agent collapse disagrees with the user action")
			}
			assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
		})
	}
}
