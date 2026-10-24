//go:build offscreen

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorWindowBackgroundMatchesTheSessionRoot(t *testing.T) {
	for _, mode := range []string{"light", "dark", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "simulation.pause", nil)
			uiAction(t, s, "theme-menu", &ui.ChangeEvent{SelectedID: mode})
			// Isolate the actual application root's surface contract from its
			// schematic renderer. This is a paint probe, not a full app capture.
			view := func() ui.Node {
				root := s.view()
				return ui.Node{ID: root.ID, Kind: root.Kind, Grow: root.Grow, Style: root.Style}
			}
			theme := inspectorLayoutTestTheme()
			theme.Default.Background = ui.Some(color.NRGBA{})
			m := newMountedControlWindowSize(t, s, view, 320, 180, theme)
			before := s.service.Snapshot()
			im, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			if artifacts := os.Getenv("ELAB_WINDOW_BACKGROUND_ARTIFACTS"); artifacts != "" {
				f, err := os.OpenFile(filepath.Join(artifacts, mode+".png"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					t.Fatal(err)
				}
				encodeErr, closeErr := png.Encode(f, im), f.Close()
				if encodeErr != nil || closeErr != nil {
					t.Fatal(encodeErr, closeErr)
				}
			}
			want := resolvePalette(mode, false).Surface
			for _, point := range []image.Point{{0, 0}, {319, 0}, {0, 179}, {319, 179}} {
				if got := color.NRGBAModel.Convert(im.At(point.X, point.Y)).(color.NRGBA); got != want {
					t.Fatal("window border disagrees with application root", mode, point, got, want)
				}
			}
			if bounds := mountedControlElement(t, m, "root").Bounds; bounds != image.Rect(8, 8, 312, 172) {
				t.Fatal("background update moved the existing content", bounds)
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("painting the host changed accepted state")
			}
		})
	}
}
