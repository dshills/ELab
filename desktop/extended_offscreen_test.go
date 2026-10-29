//go:build offscreen

package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestMountedExtendedInspectorsAcrossThemes(t *testing.T) {
	for _, theme := range []string{"light", "dark", "apollo"} {
		for _, item := range []struct {
			id     string
			fields []string
		}{{"buck", []string{"pwm-title", "pwm-frequency-label", "pwm-frequency", "pwm-duty-label", "pwm-duty"}}, {"opamp-slew", []string{"opamp-dynamics-title", "opamp-dynamics-enabled", "opamp-gbw-label", "opamp-gbw", "opamp-slew-label", "opamp-slew"}}, {"555-tone", []string{"sound-title", "sound-capacitance-label", "sound-capacitance", "audio-render"}}, {"light-alarm", []string{"sound-title", "sound-frequency-label", "sound-frequency", "audio-render"}}} {
			t.Run(theme+"/"+item.id, func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: item.id})
				agentAction(t, s.service, "simulation.pause", struct{}{})
				awaitInductorCircuit(t, s.service)
				agentAction(t, s.service, "workspace.details.set", workbench.WorkspaceDetailsParams{Section: "component", Visible: true})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
				view := func() ui.Node {
					rail, _ := findNode(s.view(), "inspector-rail")
					return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
				}
				m := newMountedControlWindowSize(t, s, view, 340, 1800, inspectorLayoutTestTheme())
				scroll := mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID)
				detailWidth := mountedControlElement(t, m, railLabelBoundsPrefix+"part-terminals").Bounds.Dx()
				if detailWidth < 200 {
					t.Fatal("Details body is narrower than the responsive rail permits", detailWidth)
				}
				previous := 0
				for _, id := range item.fields {
					element := mountedControlElement(t, m, id)
					if !element.Visible || element.Bounds.Empty() || !element.Bounds.In(scroll.Bounds) || element.Bounds.Min.Y < previous {
						t.Fatal("clipped or overlapping", id, element.Bounds)
					}
					if node, ok := findNode(s.view(), id); ok && node.Kind == ui.Label {
						frame := mountedControlElement(t, m, railLabelBoundsPrefix+id)
						if frame.Bounds.Dx() < detailWidth || !element.Bounds.In(frame.Bounds) {
							t.Fatal("label clipped within frame", id, element.Bounds, frame.Bounds)
						}
					}
					previous = element.Bounds.Max.Y
				}
			})
		}
	}
}
