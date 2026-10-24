//go:build offscreen

package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMountedFooterFitsOneLineWithLongDocumentName(t *testing.T) {
	for _, mode := range []string{"light", "dark", "apollo"} {
		for _, width := range []int{1040, 1440} {
			t.Run(fmt.Sprintf("%s/%d", mode, width), func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
				agentAction(t, s.service, "simulation.pause", struct{}{})
				path := filepath.Join(t.TempDir(), strings.Repeat("Long RC experiment ", 8)+".elab")
				agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
				theme := inspectorLayoutTestTheme()
				theme.Default.Background = ui.Some(color.NRGBA{})
				// Mount the production footer with its real root style. The manual
				// driver deliberately excludes ELab's custom schematic extension.
				view := func() ui.Node {
					root := s.view()
					footer, ok := findNode(root, "footer-row")
					if !ok {
						t.Fatal("missing production footer")
					}
					root.Children = []ui.Node{{ID: "workspace-space", Kind: ui.Column, Grow: true}, footer}
					return root
				}
				m := newMountedControlWindowSize(t, s, view, float32(width), 120, theme)
				bounds := mountedControlElement(t, m, "footer").Bounds
				row := mountedControlElement(t, m, "footer-row").Bounds
				t.Logf("footer label=%v row=%v", bounds, row)
				if bounds.Empty() || bounds.Dy() > 20 || !bounds.In(row) {
					t.Error("footer text wraps or paints outside its allocated row")
				}
				if artifacts := os.Getenv("ELAB_FOOTER_LAYOUT_ARTIFACTS"); artifacts != "" {
					im, err := m.manual.Capture(m.ctx, "main")
					if err != nil {
						t.Fatal(err)
					}
					file, err := os.OpenFile(filepath.Join(artifacts, fmt.Sprintf("%s-%d.png", mode, width)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
					if err != nil {
						t.Fatal(err)
					}
					encodeErr := png.Encode(file, im)
					closeErr := file.Close()
					if encodeErr != nil || closeErr != nil {
						t.Fatal(encodeErr, closeErr)
					}
				}
			})
		}
	}
}
