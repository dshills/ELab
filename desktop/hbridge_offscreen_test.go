//go:build offscreen

package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMountedHBridgeInspectorReadingsAcrossThemes(t *testing.T) {
	for _, theme := range []string{"light", "dark", "apollo"} {
		t.Run(theme, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "hbridge-motor"})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			awaitInductorCircuit(t, s.service)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
			before := s.service.Snapshot()
			view := func() ui.Node {
				rail, _ := findNode(s.view(), "inspector-rail")
				return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
			}
			m := newMountedControlWindowSize(t, s, view, 320, 1400, inspectorLayoutTestTheme())
			scroll := mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID)
			previous := 0
			for _, id := range []string{"hbridge-title", "hbridge-controls-help", "hbridge-truth-table", "hbridge-live-help", "hbridge-reading-title", "hbridge-state", "hbridge-inputs", "hbridge-voltage", "hbridge-current", "hbridge-loss", "hbridge-clamps"} {
				field := mountedControlElement(t, m, id)
				frame := mountedControlElement(t, m, railLabelBoundsPrefix+id)
				if !field.Visible || frame.Bounds.Dx() < 230 || !field.Bounds.In(frame.Bounds) || !frame.Bounds.In(scroll.Bounds) || field.Bounds.Min.Y < previous {
					t.Fatal("clipped or overlapping driver inspector", theme, id, field.Bounds, scroll.Bounds)
				}
				previous = field.Bounds.Max.Y
			}
			assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
		})
	}
}
