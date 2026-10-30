//go:build offscreen

package main

import (
	"image"
	"image/color"
	"testing"

	"github.com/dshills/vellum/ui"
)

func TestMountedNativePrimaryActionPaintsBothRoundedEnds(t *testing.T) {
	for _, theme := range []string{"light", "dark", "apollo"} {
		t.Run(theme, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
			uiAction(t, s, "simulation.pause", nil)
			view := func() ui.Node {
				root := s.workspaceView(1440)
				header, _ := findNode(root, "header")
				root.Kind = ui.Column
				root.Children = []ui.Node{header}
				root.Focus = ui.FocusRequest{}
				return root
			}
			m := newMountedControlWindowSize(t, s, view, 884, 90, inspectorLayoutTestTheme())
			e := mountedControlElement(t, m, "simulation.resume")
			pixels, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			p := paletteForTheme(theme)
			fill := image.Rectangle{}
			for y := e.Bounds.Min.Y; y < e.Bounds.Max.Y; y++ {
				for x := e.Bounds.Min.X; x < e.Bounds.Max.X; x++ {
					c := color.NRGBAModel.Convert(pixels.At(x, y)).(color.NRGBA)
					if readingTextCoverage(c, p.Surface, p.Accent) > .95 {
						box := image.Rect(x, y, x+1, y+1)
						if fill.Empty() {
							fill = box
						} else {
							fill = fill.Union(box)
						}
					}
				}
			}
			if fill.Empty() {
				t.Fatal("primary action has no filled accent background", e.Bounds)
			}
			for _, x := range []int{fill.Min.X, fill.Max.X - 1} {
				c := color.NRGBAModel.Convert(pixels.At(x, fill.Min.Y)).(color.NRGBA)
				if readingTextCoverage(c, p.Surface, p.Accent) > .95 {
					t.Error("primary action is clipped to a square edge", fill, x, e.Bounds)
				}
			}
			before := s.service.Snapshot()
			clickMountedControl(t, m, "simulation.resume")
			if s.service.Snapshot().Paused || s.service.Snapshot().Document != before.Document {
				t.Fatal("filled action changed command behavior")
			}
			if next := mountedControlElement(t, m, "simulation.pause"); next.Bounds != e.Bounds {
				t.Fatal("Pause and Resume targets differ", e.Bounds, next.Bounds)
			}
		})
	}
}

func TestMountedNativeSweepPopoverPaintsItsEditors(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "sweep-open", nil)
	view := func() ui.Node {
		root := s.view()
		var omit func(*ui.Node)
		omit = func(n *ui.Node) {
			if n.Kind == ui.Extension {
				n.Kind = ui.Column
				n.Custom = nil
			}
			for i := range n.Children {
				omit(&n.Children[i])
			}
		}
		omit(&root)
		root.Focus = ui.FocusRequest{}
		return root
	}
	m := newMountedControlWindowSize(t, s, view, 1440, 900, inspectorLayoutTestTheme())
	for _, id := range []string{"sweep-target", "sweep-range-start", "sweep-range-stop"} {
		e := mountedControlElement(t, m, id)
		if !e.Visible || e.Bounds.Empty() {
			t.Error("sweep popup editor has no painted allocation", id, e.Bounds)
		}
	}
}

func TestMountedNativeCompactHeaderKeepsEveryActionVisible(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	view := func() ui.Node {
		state := s.service.Snapshot()
		n := s.nativeToolbar(state, 484)
		s.bindFieldContext(&n, actionContext{state.CommandRevision, state.Document.Generation})
		s.platformSelectors(&n, state.CommandRevision)
		p := s.colorsForTheme(state.Theme)
		applyPalette(&n, p)
		polishNativeDesign(&n, p)
		reserveManagedFocusOutline(&n)
		return ui.Node{ID: "root", Kind: ui.Column, Children: []ui.Node{n}}
	}
	m := newMountedControlWindowSize(t, s, view, 500, 100, inspectorLayoutTestTheme())
	header := mountedControlElement(t, m, "header").Bounds
	for _, id := range []string{"edit.undo", "edit.redo", "examples-menu", "sweep-open", "view.fit", "simulation.resume", "native-share", "inspector-toggle"} {
		e := mountedControlElement(t, m, id)
		if e.Bounds.Empty() || !e.Bounds.In(header) {
			t.Error("compact toolbar action clipped", id, e.Bounds, header)
		}
	}
}
