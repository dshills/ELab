//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
)

func TestMountedSimulationPrimaryActionsStayOutsideScrollingForms(t *testing.T) {
	for _, tool := range []struct{ name, id string }{{"recordings", "verify-start"}, {"sweeps", "sweep-start"}, {"tolerances", "tolerance-run"}} {
		t.Run(tool.name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "simulation"})
			agentAction(t, s.service, "workspace.simulation.set", workbench.SimulationToolParams{Tool: tool.name})
			s.keyboardFocus.ID = ""
			m := mountedUsabilityInspector(t, s, 480)
			before := mountedControlElement(t, m, tool.id)
			chooser := mountedControlElement(t, m, "simulation-tool")
			if !before.Visible || before.Bounds.Max.Y > 480 {
				t.Fatal("primary action needs scrolling", before.Bounds)
			}
			scroll := mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID).Bounds
			m.input(cogent.ManualInput{Type: events.Scroll, Position: scroll.Min.Add(scroll.Size().Div(2)), Delta: math32.Vec2(0, 200)})
			after := mountedControlElement(t, m, tool.id)
			if after.Bounds != before.Bounds || mountedControlElement(t, m, "simulation-tool").Bounds != chooser.Bounds {
				t.Fatal("primary action/chooser moved with form")
			}
		})
	}
}

func TestMountedRecordFromBasicFormUsesAcceptedAdvancedSettings(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "simulation"})
	agentAction(t, s.service, "workspace.simulation.set", workbench.SimulationToolParams{Tool: "recordings"})
	s.keyboardFocus.ID = ""
	m := mountedUsabilityInspector(t, s, 600)
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	s.settleNumber = func(id string, done func(error)) error {
		return cogent.SettleNumberField(m.ctx, s.app, "main", id, done)
	}
	before := s.service.Snapshot()
	clickMountedControl(t, m, "verify-start")
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Operations) != 1 || after.WorkspaceDetails.Recordings || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("basic recording failed or changed live experiment", s.errorMessage)
	}
}

// The actual private TextField commits on focus transfer. Advanced must use
// that exact own-form continuation, never a fresh agent snapshot.
func TestMountedAdvancedFirstClickKeepsPrivateDuration(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "simulation"})
	agentAction(t, s.service, "workspace.simulation.set", workbench.SimulationToolParams{Tool: "recordings"})
	s.keyboardFocus.ID = ""
	m := mountedUsabilityInspector(t, s, 700)
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	s.settleNumber = func(id string, done func(error)) error {
		return cogent.SettleNumberField(m.ctx, s.app, "main", id, done)
	}
	typeMountedCursor(t, m, "verify-duration", "20 ms")
	clickMountedControl(t, m, "workspace-details-recordings")
	after := s.service.Snapshot()
	if s.errorMessage != "" || !after.WorkspaceDetails.Recordings || after.VerificationSettings.Duration != .02 || len(after.Operations) != 0 {
		t.Fatal("Advanced lost own unfinished duration or started a run", s.errorMessage, after.VerificationSettings)
	}
}
