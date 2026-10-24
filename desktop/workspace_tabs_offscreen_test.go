//go:build offscreen

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorAndDrawingTabsFitAndKeepSharedActions(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.rc", nil)
			uiAction(t, s, "simulation.pause", nil)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			view := func() ui.Node {
				root := s.view()
				root.Focus = ui.FocusRequest{} // The reduced probe excludes the Resume button.
				tools, _ := findNode(root, "tools")
				rail, _ := findNode(root, "inspector-rail")
				// Keep the production inset and complete inspector rail: its scrolling
				// form must not paint over the navigation indicators. Custom drawing
				// extensions are not part of this paint/layout probe.
				root.Children = []ui.Node{{ID: "strip-row", Kind: ui.Row, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(1)), Align: ui.Some(ui.AlignStart)}, Children: []ui.Node{
					{ID: "drawing-header", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(8))}, Children: []ui.Node{tools}},
					rail,
				}}}
				return root
			}
			theme := desktopTheme()
			theme.Default.Background = ui.Some(color.NRGBA{})
			m := newMountedControlWindowSize(t, s, view, 1040, 760, theme, func(config *cogent.ManualConfig) {
				config.GPUSceneRendering = true
			})
			for _, group := range []struct {
				id       string
				children []string
			}{
				{"tools", []string{"tool-select", "tool-wire", "tool-pan", "drawing.cancel", "sweep-open", "view.fit"}},
				{"inspector-navigation", []string{"inspector-section-component", "inspector-section-simulation", "inspector-section-document"}},
			} {
				parent := mountedControlElement(t, m, group.id).Bounds
				for _, id := range group.children {
					child := mountedControlElement(t, m, id)
					if !child.Visible || child.Bounds.Dy() < 28 || !child.Bounds.In(parent) {
						t.Errorf("control is clipped: %s=%v parent %s=%v", id, child.Bounds, group.id, parent)
					}
				}
			}
			modeGroup := mountedControlElement(t, m, "drawing-mode-tabs").Bounds
			cancel := mountedControlElement(t, m, "drawing.cancel").Bounds
			modeButton := mountedControlElement(t, m, "tool-select").Bounds
			t.Logf("mode rail=%v Select=%v Cancel=%v", modeGroup, modeButton, cancel)
			if cancel.Min.X-modeGroup.Max.X < 4 {
				t.Error("Cancel touches the drawing-mode rail", modeGroup, cancel)
			}
			for _, id := range []string{"drawing.cancel", "sweep-open", "view.fit"} {
				bounds := mountedControlElement(t, m, id).Bounds
				if bounds.Min.Y != modeButton.Min.Y || bounds.Max.Y != modeButton.Max.Y {
					t.Error("drawing action is not aligned with the mode buttons", id, bounds, modeButton)
				}
			}
			cancelNode, _ := findNode(view(), "drawing.cancel")
			if cancelNode.Text != "Cancel" || cancelNode.Accessibility.Name != "Cancel" || !strings.Contains(cancelNode.Accessibility.Help, "Escape") || !strings.Contains(cancelNode.Accessibility.Help, "unfinished wire or component placement") {
				t.Fatal("Cancel label or explanatory shortcut tooltip is missing", cancelNode)
			}
			im, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			for _, section := range []string{"component", "simulation", "document"} {
				indicator := mountedControlElement(t, m, "inspector-tab-indicator-"+section)
				t.Logf("%s cell=%v action=%v indicator=%v visible=%v", section, mountedControlElement(t, m, "inspector-tab-cell-"+section).Bounds, mountedControlElement(t, m, "inspector-section-"+section).Bounds, indicator.Bounds, indicator.Visible)
				if !indicator.Visible || indicator.Bounds.Dy() < 2 {
					t.Error("tab underline is clipped", section)
				} else {
					palette := paletteForTheme(mode)
					want := palette.Grid
					if section == "component" {
						want = palette.Accent
					}
					at := indicator.Bounds.Min.Add(indicator.Bounds.Size().Div(2))
					if got := color.NRGBAModel.Convert(im.At(at.X, at.Y)).(color.NRGBA); got != want {
						t.Error("tab underline is not painted", section, got, want)
					}
				}
			}
			if mode == "apollo" {
				// AX rectangles can already be clipped to the toolbar. Inspect the
				// actual bottom border too: the old Cancel/Wire/Pan boxes lose it.
				bounds := mountedControlElement(t, m, "drawing.cancel").Bounds
				palette := resolvePalette(mode, false)
				columns := 0
				for x := bounds.Min.X + 4; x < bounds.Max.X-4; x++ {
					for y := bounds.Max.Y - 2; y < bounds.Max.Y; y++ {
						p := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
						if readingTextCoverage(p, palette.Paper, palette.Outline) > .6 {
							columns++
							break
						}
					}
				}
				if columns < (bounds.Dx()-8)/2 {
					t.Errorf("toolbar clips the painted lower border: %d columns in %v", columns, bounds)
				}
			}
			if directory := os.Getenv("ELAB_TAB_ARTIFACTS"); directory != "" {
				f, err := os.OpenFile(filepath.Join(directory, mode+".png"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				encodeErr, closeErr := png.Encode(f, im), f.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatal(encodeErr, closeErr)
				}
			}
			if t.Failed() {
				return
			}
			before := s.service.Snapshot()
			bottomClick := func(id string) {
				bounds := mountedControlElement(t, m, id).Bounds
				at := image.Pt(bounds.Min.X+bounds.Dx()/2, bounds.Max.Y-2)
				m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
				m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			}
			bottomClick("tool-wire")
			if s.service.Snapshot().Drawing.Tool != "wire" {
				t.Fatal("drawing tab's bottom edge is not clickable")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeRightArrow})
			if m.snapshot().FocusedID != "tool-pan" {
				t.Fatal("drawing group lost toolbar arrow navigation")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			if s.service.Snapshot().Drawing.Tool != "pan" {
				t.Fatal("keyboard activation did not select the drawing mode")
			}
			bottomClick("drawing.cancel")
			if s.service.Snapshot().Drawing.Tool != "select" {
				t.Fatal("Cancel did not return to selection through the shared command")
			}
			bottomClick("inspector-section-document")
			if s.service.Snapshot().InspectorSection != "document" {
				t.Fatal("inspector tab's bottom edge is not clickable")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeLeftArrow})
			if m.snapshot().FocusedID != "inspector-section-simulation" {
				t.Fatal("inspector cells lost toolbar arrow navigation")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			if s.service.Snapshot().InspectorSection != "simulation" {
				t.Fatal("keyboard activation did not select the inspector section")
			}
			agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "select"})
			agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "component"})
			m.frame()
			assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
		})
	}
}
