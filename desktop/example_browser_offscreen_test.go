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

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Mount the actual sheet's content, without claiming native modal positioning.
// Physical native checks separately cover scrim, centering and focus return.
func exampleBrowserBody(s *screen) ui.Node {
	full := s.view()
	if !s.service.Snapshot().ExampleBrowser.Visible {
		// Successful Open returns to the real component search field. Keep
		// that destination in this reduced fixture's focus scope too.
		search, _ := findNode(full, "part-search")
		return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Focus: s.keyboardFocus, Theme: full.Theme, Children: []ui.Node{search}}
	}
	body, _ := findNode(full, exampleBrowserID)
	body.Kind, body.Sheet, body.Open = ui.Column, false, false
	body.DialogDefaultID, body.DialogReturnFocusID, body.Dismiss = "", "", nil
	return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Focus: s.keyboardFocus, Theme: full.Theme, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0))}, Children: []ui.Node{body}}
}

func seedExampleBrowserImage(t *testing.T, s *screen) {
	t.Helper()
	p, err := workbench.PreviewExample("led")
	if err != nil {
		t.Fatal(err)
	}
	pixels, err := renderExampleThumbnail(p)
	if err != nil {
		t.Fatal(err)
	}
	s.exampleThumbnails = exampleThumbnails{requested: true, images: map[string]*canvas.TiledImage{"led": pixels}}
}

func TestMountedExampleBrowserFieldsPreviewAndActionsFitAllThemes(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			uiAction(t, s, "simulation.pause", nil)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			uiAction(t, s, "examples-menu", nil)
			seedExampleBrowserImage(t, s)
			theme := desktopTheme()
			theme.Default.Background = ui.Some(color.NRGBA{})
			m := newMountedControlWindowSize(t, s, func() ui.Node { return exampleBrowserBody(s) }, 840, 760, theme, func(c *cogent.ManualConfig) { c.GPUSceneRendering = true })
			body := mountedControlElement(t, m, exampleBrowserID).Bounds
			if body.Empty() || !body.In(image.Rect(0, 0, 840, 760)) {
				t.Fatal("browser body escaped the compact viewport", body)
			}
			for _, id := range []string{"example-search", "example-category", "example-grid", "example-preview-image", "example-description-scroll", ui.SheetConfirmID(exampleBrowserID), ui.SheetCancelID(exampleBrowserID)} {
				e := mountedControlElement(t, m, id)
				t.Log(id, e.Bounds, "in", body)
				if !e.Visible || e.Bounds.Empty() || !e.Bounds.In(body) {
					t.Error("clipped browser control", id, e.Bounds)
				}
			}
			if dir := os.Getenv("ELAB_EXAMPLE_BROWSER_ARTIFACT_DIR"); dir != "" {
				im, err := m.manual.Capture(m.ctx, "main")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, mode+".png"))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, im)
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			}
		})
	}
}

func TestMountedExampleBrowserKeyboardSearchOpenAndScroll(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "examples-menu", nil)
	seedExampleBrowserImage(t, s)
	theme := desktopTheme()
	theme.Default.Background = ui.Some(color.NRGBA{})
	m := newMountedControlWindowSize(t, s, func() ui.Node { return exampleBrowserBody(s) }, 840, 760, theme)
	for _, r := range "555 flasher" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	if s.service.Snapshot().ExampleBrowser.Query != "" {
		t.Fatal("private typing prematurely changed search")
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	if b := s.service.Snapshot().ExampleBrowser; b.Query != "555 flasher" || !reflect.DeepEqual(b.MatchingIDs, []string{"555-flasher"}) {
		t.Fatal("keyboard search did not reach the matching collection", b, s.errorMessage)
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	if state := s.service.Snapshot(); state.ExampleBrowser.Visible || state.Example == nil || state.Example.Info.ID != "555-flasher" {
		t.Fatal("Return did not open the selected example", state.Example, s.errorMessage)
	}
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "examples-menu", nil)
	agentAction(t, s.service, "examples.category.set", workbench.PartCategoryParams{Category: "all"})
	s.app.Do(func() { s.requestKeyboardFocus("example-grid") })
	m.frame()
	before := s.service.Snapshot()
	grid := mountedControlElement(t, m, "example-grid").Bounds
	m.input(cogent.ManualInput{Type: events.Scroll, Position: grid.Min.Add(grid.Size().Div(2)), Delta: math32.Vec2(0, 240)})
	if after := s.service.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("wheel changed selection, document or physics")
	}
	for steps := 0; steps < len(before.ExampleBrowser.MatchingIDs); steps++ {
		b := s.service.Snapshot().ExampleBrowser
		if b.SelectedID == b.MatchingIDs[len(b.MatchingIDs)-1] {
			return
		}
		m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeRightArrow})
		if s.service.Snapshot().ExampleBrowser.SelectedID == b.SelectedID {
			t.Fatal("arrow navigation stopped before the last example", b.SelectedID, s.errorMessage)
		}
	}
	t.Fatal("arrow navigation exceeded the finite catalog")
}
