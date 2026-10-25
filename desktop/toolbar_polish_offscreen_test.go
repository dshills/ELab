//go:build offscreen

package main

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func toolbarPaintView(s *screen) ui.Node {
	root := s.view()
	header, _ := findNode(root, "header")
	tools, _ := findNode(root, "tools")
	scope, _ := findNode(root, "scope-panel")
	footer, _ := findNode(root, "footer-row")
	scope.Style.MaxSize = ui.Some(ui.Size{})
	root.Children = []ui.Node{header, tools, scope, footer}
	var omitCanvas func(*ui.Node)
	omitCanvas = func(n *ui.Node) {
		if n.Kind == ui.Extension {
			n.Kind, n.Custom = ui.Row, nil
		}
		for i := range n.Children {
			omitCanvas(&n.Children[i])
		}
	}
	omitCanvas(&root)
	return root
}

func TestMountedToolbarPolishFitsAndPreservesSharedSelection(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.rc", nil)
			uiAction(t, s, "simulation.pause", nil)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			theme := desktopTheme()
			theme.Default.Background = ui.Some(color.NRGBA{})
			view := func() ui.Node { return toolbarPaintView(s) }
			m := newMountedControlWindowSize(t, s, view, 1040, 760, theme, func(config *cogent.ManualConfig) { config.GPUSceneRendering = true })
			im, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range []struct {
				id       string
				controls []string
			}{
				{"toolbar", []string{"file-menu", "file.save", "examples-menu", "edit.undo", "edit.redo", "theme-menu"}},
				{"simulation-toolbar", []string{"simulation.resume"}},
				{"tools", []string{"drawing.cancel", "sweep-open", "view.fit"}},
				{"scope-toolbar", []string{"scope-visibility", "input-auto", "input-charge", "input-discharge", "transient-step", "simulation.restart"}},
				{"scope-trace-row", []string{"export-csv", "scope-trace-focus"}},
				{"scope-cursors", []string{"cursor-select-a", "cursor-time-a", "cursor-clear-a", "cursor-select-b", "cursor-time-b", "cursor-clear-b"}},
			} {
				parent := mountedControlElement(t, m, row.id).Bounds
				for _, id := range row.controls {
					control := mountedControlElement(t, m, id)
					t.Logf("%s %v in %s %v", id, control.Bounds, row.id, parent)
					if !control.Visible || control.Bounds.Dy() < 30 || control.Bounds.Dy() > 32 || !control.Bounds.In(parent) {
						t.Error("toolbar control is clipped or has inconsistent height", id, control.Bounds, parent)
					}
					if !control.Enabled || control.Bounds.Empty() {
						continue
					}
					node, _ := findNode(view(), id)
					outline := node.Style.Border.Value.Color
					if control.Focused {
						// Focus uses the same scoped semantic accent as the app.
						root := view()
						if root.Theme == nil || !root.Theme.Palette.Set {
							t.Fatal("missing scoped focus palette")
						}
						outline = root.Theme.Palette.Value.Accent
					}
					columns := 0
					for x := control.Bounds.Min.X + 8; x < control.Bounds.Max.X-8; x++ {
						for y := control.Bounds.Max.Y - 3; y < control.Bounds.Max.Y; y++ {
							pixel := color.NRGBAModel.Convert(im.At(x, y)).(color.NRGBA)
							if readingTextCoverage(pixel, node.Style.Background.Value, outline) > .6 {
								columns++
								break
							}
						}
					}
					if columns < (control.Bounds.Dx()-16)/2 {
						t.Error("toolbar lower outline is not painted", id, columns, control.Bounds)
					}
				}
			}
			if directory := os.Getenv("ELAB_TOOLBAR_ARTIFACTS"); directory != "" {
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
			clickMountedControl(t, m, "cursor-select-b")
			if s.service.Snapshot().Scope.ActiveCursor != "b" {
				t.Fatal("toolbar selection bypassed the shared cursor command")
			}
			agentAction(t, s.service, "scope.cursor.select", workbench.CursorSelectParams{Cursor: "a"})
			m.frame()
			assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
			selected := mountedControlElement(t, m, "cursor-select-a").Bounds
			paint, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			// A one-point resting stroke is inset within Vellum's two-point
			// focus allowance. Inspect the whole lower stroke area, as above.
			selectedOutline := false
			for y := selected.Max.Y - 3; y < selected.Max.Y; y++ {
				pixel := color.NRGBAModel.Convert(paint.At(selected.Min.X+selected.Dx()/2, y)).(color.NRGBA)
				selectedOutline = selectedOutline || pixel == paletteForTheme(mode).Accent
			}
			if !selectedOutline {
				t.Fatal("agent-selected cursor did not acquire its accent outline", selected)
			}
			resume := mountedControlElement(t, m, "simulation.resume").Bounds
			clickMountedControl(t, m, "simulation.resume")
			pause := mountedControlElement(t, m, "simulation.pause").Bounds
			if s.service.Snapshot().Paused || resume.Size() != pause.Size() {
				t.Fatal("Pause/Resume transition changed its target size", resume, pause)
			}
			agentAction(t, s.service, "simulation.pause", struct{}{})
		})
	}
}

func TestMountedToolbarOverflowKeepsLastActionReachable(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	view := func() ui.Node {
		toolbar, _ := findNode(s.view(), "scope-toolbar")
		return ui.Node{ID: "overflow-probe", Kind: ui.Column, Focus: ui.FocusRequest{ID: "scope-visibility", Sequence: 1}, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0))}, Children: []ui.Node{toolbar, quiet("following-content", "Following content must not cover toolbar outlines")}}
	}
	theme := desktopTheme()
	theme.Default.Typography.Value.Family = "Noto Sans"
	theme.Default.Background = ui.Some(paper)
	m := newMountedControlWindowSize(t, s, view, 380, 120, theme)
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeEnd})
	last := mountedControlElement(t, m, "simulation.restart")
	if m.snapshot().FocusedID != last.ID || !last.Visible || !last.Bounds.In(mountedControlElement(t, m, "scope-toolbar").Bounds) || last.Bounds.Dy() < 30 {
		t.Fatal("overflow hides or clips the final action", last)
	}
	before := s.service.Snapshot()
	clickMountedControl(t, m, "simulation.restart")
	if s.service.Snapshot().RunID == before.RunID {
		t.Fatal("overflowed restart action is not clickable")
	}
}
