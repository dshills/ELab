//go:build offscreen

package main

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorFormSpacingAndPaintAcrossThemes(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		for _, page := range []struct{ section, field, label, next string }{
			{"component", "value-entry", "value-title", "apply-value"},
			{"simulation", "setting-step", "setting-label-step", "inspector-field-setting-speed"},
			{"document", "pdf-margin", "pdf-margin-label", "pdf-scale-mode"},
		} {
			t.Run(mode+"/"+page.section, func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				uiAction(t, s, "example.rc", nil)
				uiAction(t, s, "simulation.pause", nil)
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
				agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: page.section})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
				s.app.Do(func() {
					if _, ok := s.reachableAction(page.field); !ok {
						t.Fatal("missing editor", page.field)
					}
				})
				before := s.service.Snapshot()
				view := func() ui.Node {
					rail, _ := findNode(s.view(), "inspector-rail")
					return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
				}
				theme := inspectorLayoutTestTheme()
				theme.Default.Background = ui.Some(color.NRGBA{})
				m := newMountedControlWindowSize(t, s, view, 320, 1100, theme, func(c *cogent.ManualConfig) { c.GPUSceneRendering = true })
				field := mountedControlElement(t, m, page.field)
				caption := mountedControlElement(t, m, page.label)
				following := mountedControlElement(t, m, page.next)
				scroll := mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID)
				t.Logf("label=%v field=%v following=%v scroll=%v", caption.Bounds, field.Bounds, following.Bounds, scroll.Bounds)
				if !field.Visible || field.Bounds.Dx() < 130 || field.Bounds.Dy() < 30 || field.Bounds.Dy() > 34 || !field.Bounds.In(scroll.Bounds) {
					t.Error("inspector editor is clipped or has inconsistent height", field.Bounds)
				}
				if gap := field.Bounds.Min.X - caption.Bounds.Max.X; gap < 6 || gap > 10 || caption.Bounds.Min.Y > field.Bounds.Max.Y || caption.Bounds.Max.Y < field.Bounds.Min.Y {
					t.Error("field label is detached or misaligned", caption.Bounds, field.Bounds)
				}
				if following.Bounds.Min.Y-field.Bounds.Max.Y < 4 {
					t.Error("following content overlaps or crowds the editor", field.Bounds, following.Bounds)
				}
				im, err := m.manual.Capture(m.ctx, "main")
				if err != nil {
					t.Fatal(err)
				}
				palette := paletteForTheme(mode)
				at := field.Bounds.Min.Add(field.Bounds.Size().Div(2))
				painted := false
				for y := field.Bounds.Max.Y - 3; y < field.Bounds.Max.Y; y++ {
					pixel := color.NRGBAModel.Convert(im.At(at.X, y)).(color.NRGBA)
					painted = painted || pixel == nativeHairlineColor(palette)
				}
				if !painted {
					t.Error("editor bottom outline is obscured", nativeHairlineColor(palette))
				}
				assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
				if directory := os.Getenv("ELAB_INSPECTOR_ARTIFACTS"); directory != "" {
					f, err := os.OpenFile(filepath.Join(directory, mode+"-"+page.section+".png"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
					if err != nil {
						t.Fatal(err)
					}
					encodeErr, closeErr := png.Encode(f, im), f.Close()
					if encodeErr != nil || closeErr != nil {
						t.Fatal(encodeErr, closeErr)
					}
				}
			})
		}
	}
}
