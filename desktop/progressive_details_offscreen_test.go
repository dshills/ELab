//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
)

func TestMountedCompactInspectorKeepsPrimaryActionsAboveDetails(t *testing.T) {
	for _, theme := range []string{"light", "dark", "apollo", "system"} {
		for _, part := range []string{"R1", "C1"} {
			t.Run(theme+"/"+part, func(t *testing.T) {
				s, cleanup := testScreen(t)
				defer cleanup()
				prepareSpectrumDock(t, s.service)
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: part})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
				m := mountedUsabilityInspector(t, s, 480)
				scroll := mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID).Bounds
				for _, id := range []string{"value-entry", "apply-value", "rotate-part", "delete-object", "plot-v:" + part, "plot-i:" + part} {
					n := mountedControlElement(t, m, id)
					if !n.Visible || !n.Bounds.In(scroll) {
						t.Error("basic action needs scrolling", id, n.Bounds, scroll)
					}
				}
				for _, n := range m.snapshot().Elements {
					if (n.ID == "cap-initial" || n.ID == "duplicate-part") && n.Visible {
						t.Fatal("closed details retained keyboard element", n.ID)
					}
				}
				agentAction(t, s.service, "workspace.details.set", workbench.WorkspaceDetailsParams{Section: "component", Visible: true})
				m.frame()
				if !s.service.Snapshot().WorkspaceDetails.Component {
					t.Fatal("agent could not open details")
				}
			})
		}
	}
}

func TestMountedOpeningComponentDetailsPreservesOwnPrivateValue(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	m := mountedValuePreset(t, s, "workspace-details-component")
	before := s.service.Snapshot()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
	for _, r := range "2.2k" {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
	}
	if s.service.Snapshot().Drawing.Value != before.Drawing.Value {
		t.Fatal("fixture committed too early")
	}
	clickMountedControl(t, m, "workspace-details-component")
	after := s.service.Snapshot()
	if !after.WorkspaceDetails.Component || after.Drawing.Value != "2.2k" || !reflect.DeepEqual(after.Components, before.Components) {
		t.Fatal("first Details click lost/refused the private draft or applied it", after.WorkspaceDetails, after.Drawing.Value, s.errorMessage)
	}
}
