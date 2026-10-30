//go:build offscreen

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedDCInstrumentNavigationAndExplanation(t *testing.T) {
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		for _, width := range []int{640, 1040} {
			t.Run(theme+"/"+strconv.Itoa(width), func(t *testing.T) {
				s, cleanup := testScreen(t)
				defer cleanup()
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "npn"})
				agentAction(t, s.service, "simulation.pause", struct{}{})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
				view := func() ui.Node {
					root := s.workspaceView(float32(width) - 16)
					dock, ok := findNode(root, "instrument-panel")
					if !ok {
						t.Fatal("DC circuit hid the instrument dock")
					}
					// This fixture exercises retained controls and text layout, not
					// GPU plots. The ordinary spectrum tests cover plot meshes.
					var omitPlots func(*ui.Node)
					omitPlots = func(n *ui.Node) {
						if n.Kind == ui.Extension {
							n.Kind, n.Custom = ui.Row, nil
						}
						for i := range n.Children {
							omitPlots(&n.Children[i])
						}
					}
					omitPlots(&dock)
					root.Children = []ui.Node{{ID: "test-scroll", Kind: ui.Scroll, Grow: true, Children: []ui.Node{dock}}}
					return root
				}
				m := newMountedControlWindowSize(t, s, view, float32(width), 600, inspectorLayoutTestTheme())
				capture := func(name string) {
					t.Helper()
					if destination := os.Getenv("ELAB_INSTRUMENT_CAPTURE"); destination != "" {
						if err := os.MkdirAll(destination, 0700); err != nil {
							t.Fatal(err)
						}
						pixels, err := m.manual.Capture(m.ctx, "main")
						if err != nil {
							t.Fatal(err)
						}
						file, err := os.Create(filepath.Join(destination, theme+"-"+strconv.Itoa(width)+"-"+name+".png"))
						if err != nil {
							t.Fatal(err)
						}
						encodeErr := png.Encode(file, pixels)
						closeErr := file.Close()
						if encodeErr != nil || closeErr != nil {
							t.Fatal(encodeErr, closeErr)
						}
					}
				}
				assertVisible := func(ids ...string) {
					t.Helper()
					for _, id := range ids {
						e := mountedControlElement(t, m, id)
						if !e.Visible || e.Bounds.Empty() || !e.Bounds.In(image.Rect(0, 0, width, 600)) {
							t.Fatalf("%s not fully visible: %+v", id, e)
						}
						if id == "instrument-empty-help" {
							// Text bounds follow shaped lines (and vary with font).
							// Check the allocation that actually controls wrapping.
							frame := mountedControlElement(t, m, railLabelBoundsPrefix+id).Bounds
							if frame.Dx() < width-60 || !frame.In(image.Rect(0, 0, width, 600)) {
								t.Fatalf("explanation lacks its viewport wrapping allocation: %v", frame)
							}
						}
					}
				}
				click := func(id string) {
					t.Helper()
					e := mountedControlElement(t, m, id)
					at := e.Bounds.Min.Add(e.Bounds.Size().Div(2))
					for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
						m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
					}
					m.frame()
				}
				assertVisible("scope-view-scope", "scope-view-spectrum", "instrument-empty-title", "instrument-empty-help")
				capture("scope")
				click("scope-view-spectrum")
				if s.service.Snapshot().ScopeView != "spectrum" {
					t.Fatal("DC Spectrum tab did not route the shared command")
				}
				assertVisible("scope-view-scope", "scope-view-spectrum", "instrument-empty-title", "instrument-empty-help")
				capture("spectrum")
				click("scope-visibility")
				assertVisible("scope-view-scope", "scope-view-spectrum", "scope-expand")
				click("scope-view-scope")
				if state := s.service.Snapshot(); state.ScopeView != "scope" || state.ScopeVisible {
					t.Fatal("collapsed tabs lost shared state")
				}
				click("scope-expand")
				assertVisible("scope-view-scope", "scope-view-spectrum", "instrument-empty-title", "instrument-empty-help")
				if s.errorMessage != "" {
					t.Fatal(s.errorMessage)
				}
			})
		}
	}
}
