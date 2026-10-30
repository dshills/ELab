//go:build offscreen

package main

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorAndDrawingTabsFitAndKeepSharedActions(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			uiAction(t, s, "example.rc", nil)
			uiAction(t, s, "simulation.pause", nil)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			view := func() ui.Node {
				root := s.view()
				root.Focus = ui.FocusRequest{}
				tools, _ := findNode(root, "tools")
				rail, _ := findNode(root, "inspector-rail")
				root.Kind = ui.Column
				root.Children = []ui.Node{{ID: "strip-row", Kind: ui.Row, Grow: true, Children: []ui.Node{{ID: "drawing-header", Kind: ui.Column, Grow: true, Children: []ui.Node{tools}}, rail}}}
				return root
			}
			m := newMountedControlWindowSize(t, s, view, 1040, 760, inspectorLayoutTestTheme())
			check := func(parent string, ids ...string) {
				t.Helper()
				p := mountedControlElement(t, m, parent).Bounds
				for _, id := range ids {
					e := mountedControlElement(t, m, id)
					if !e.Visible || e.Bounds.Dy() < 24 || !e.Bounds.In(p) {
						t.Fatalf("clipped %s %v in %v", id, e.Bounds, p)
					}
				}
			}
			check("tools", "tool-select", "tool-wire", "tool-pan")
			check("inspector-navigation", "inspector-section-component", "inspector-section-simulation", "inspector-section-document")
			before := s.service.Snapshot()
			bottomClick := func(id string) {
				t.Helper()
				e := mountedControlElement(t, m, id)
				at := image.Pt(e.Bounds.Min.X+e.Bounds.Dx()/2, e.Bounds.Max.Y-3)
				for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
					m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
				}
			}
			bottomClick("tool-wire")
			if s.service.Snapshot().Drawing.Tool != "wire" {
				t.Fatal("bottom edge did not activate Wire")
			}
			check("tools", "tool-select", "tool-wire", "tool-pan", "drawing.cancel")
			cancel, _ := findNode(view(), "drawing.cancel")
			if cancel.Text != "Cancel" || !strings.Contains(cancel.Accessibility.Help, "Escape") || !strings.Contains(cancel.Accessibility.Help, "unfinished wire or component placement") {
				t.Fatal("Cancel lost its explanatory shortcut", cancel)
			}
			bottomClick("drawing.cancel")
			if s.service.Snapshot().Drawing.Tool != "select" {
				t.Fatal("Cancel bypassed shared drawing command")
			}
			bottomClick("tool-wire")
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeRightArrow})
			if m.snapshot().FocusedID != "tool-pan" {
				t.Fatal("tool toolbar lost arrow navigation", m.snapshot().FocusedID)
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			if s.service.Snapshot().Drawing.Tool != "pan" {
				t.Fatal("keyboard did not select Pan")
			}
			bottomClick("inspector-section-document")
			if s.service.Snapshot().InspectorSection != "document" {
				t.Fatal("bottom edge did not activate Document")
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeLeftArrow})
			if m.snapshot().FocusedID != "inspector-section-simulation" {
				t.Fatal("inspector lost arrow navigation", m.snapshot().FocusedID)
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			if s.service.Snapshot().InspectorSection != "simulation" {
				t.Fatal("keyboard did not select Simulate")
			}
			assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
			pixels, err := m.manual.Capture(m.ctx, "main")
			if err != nil {
				t.Fatal(err)
			}
			selected := mountedControlElement(t, m, "inspector-section-simulation").Bounds
			p := paletteForTheme(mode)
			// A selected segment is visibly distinct from the track; inspect its fill,
			// away from text, focus rings and rounded corners.
			at := image.Pt(selected.Min.X+8, selected.Min.Y+5)
			got := color.NRGBAModel.Convert(pixels.At(at.X, at.Y)).(color.NRGBA)
			if got == nativeSegmentColor(p) {
				t.Fatal("selected segment has the inactive track fill", got)
			}
		})
	}
}
