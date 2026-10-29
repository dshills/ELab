//go:build offscreen

package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
	"testing"
)

func TestMountedContextReadingsStayStationaryAsCircuitAdvances(t *testing.T) {
	for _, theme := range []string{"light", "dark", "apollo", "system"} {
		t.Run(theme, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "measurement.select", workbench.MeasurementSelectParams{Kind: "terminal", ID: "C1.1"})
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
			m := newMountedControlWindowSize(t, s, func() ui.Node {
				root := s.view()
				row, _ := findNode(root, "readings")
				root.Children = []ui.Node{row}
				return root
			}, 900, 64, inspectorLayoutTestTheme())
			id := "context-reading-v:C1.1-value"
			before := mountedControlElement(t, m, id)
			for range 3 {
				agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .001})
				m.frame()
				after := mountedControlElement(t, m, id)
				if !after.Visible || after.Bounds != before.Bounds || after.Bounds.Max.Y > 64 {
					t.Fatal("live digits move their control", before.Bounds, after.Bounds)
				}
			}
		})
	}
}
