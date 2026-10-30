//go:build offscreen

package main

import (
	"fmt"
	"image"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorTextGroupsUseAvailableRailWidth(t *testing.T) {
	for _, example := range []string{"rc", "nmos", "pmos", "cmos"} {
		t.Run(example, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: example})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "simulation.restart", struct{}{})
			if _, err := s.service.Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
			uiAction(t, s, "inspector-section-simulation", nil)
			uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: "circuit"})
			before := s.service.Snapshot()
			view := func() ui.Node {
				group, ok := findNode(s.view(), "inspector-group-circuit")
				if !ok {
					t.Fatal("missing production circuit-information group")
				}
				return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{panel("test-rail", 240, group)}}
			}
			m := newMountedControlWindowSize(t, s, view, 320, 1100, inspectorLayoutTestTheme())
			for pass := range 2 {
				if pass > 0 {
					m.frame()
				}
				window := m.snapshot()
				bounds := map[string]image.Rectangle{}
				for _, element := range window.Elements {
					bounds[element.ID] = element.Bounds
				}
				group := bounds["inspector-group-circuit"]
				if group.Dx() < 239 || group.Dx() > 241 {
					t.Errorf("circuit text group shrank inside the production 240-unit content rail: %v", group)
				}
				ids := []string{"circuit-capacity", "selection-navigation-help"}
				if example != "rc" {
					ids = append(ids, "mixed-help")
				}
				for _, id := range ids {
					label := bounds[id]
					frame := bounds[railLabelBoundsPrefix+id]
					if frame.Dx() < 227 || frame.Dx() > 229 {
						t.Errorf("%s has a %d-pixel wrapping frame instead of the available 228 pixels: %v", id, frame.Dx(), frame)
					}
					if label.Empty() || !label.In(frame) || !frame.In(group) {
						t.Errorf("reading %s escaped its frame or accessibility group: label=%v frame=%v group=%v", id, label, frame, group)
					}
					if label.Dy() > 74 {
						t.Errorf("%s still wraps into a tall narrow column: %v", id, label)
					}
					node, ok := findNode(view(), id)
					if !ok || node.Kind != ui.Label || node.Text == "" {
						t.Fatal("reading identity or source text disappeared", id)
					}
					t.Logf("%s pass=%d label=%v frame=%v group=%v", id, pass, label, frame, group)
				}
				if after := s.service.Snapshot(); !reflect.DeepEqual(before, after) {
					t.Fatal("mount or repaint changed accepted circuit state")
				}
			}
		})
	}
}

func TestMountedInspectorWidthRespectsConstrainedRails(t *testing.T) {
	for _, width := range []float32{152, 248} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			view := func() ui.Node {
				nodes := inspectorGroup("text", "Electrical readings", []ui.Node{quiet("reading", "Source voltage and current remain tied to their named references.")})
				constrainRail(nodes, width)
				return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{panel("test-rail", width, nodes...)}}
			}
			m := newMountedControlWindowSize(t, s, view, width+64, 500, inspectorLayoutTestTheme())
			found := false
			for _, element := range m.snapshot().Elements {
				if element.ID == "inspector-group-text" {
					found = true
					if element.Bounds.Dx() < int(width)-1 || element.Bounds.Dx() > int(width)+1 {
						t.Fatal("text group does not fill its bounded rail", width, element.Bounds)
					}
				}
			}
			if !found {
				t.Fatal("mounted text group disappeared")
			}
		})
	}
}

// Manual rendering requires an explicit bundled font; production keeps the
// native family. Width policy is tested independently of OS font discovery.
func inspectorLayoutTestTheme() ui.Theme {
	theme := desktopTheme()
	theme.Default.Background = ui.Some(paper)
	typography := theme.Default.Typography.Value
	typography.Family = "Noto Sans"
	theme.Default.Typography = ui.Some(typography)
	return theme
}
