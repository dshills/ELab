//go:build offscreen

package main

import (
	"image"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Mount the actual assembled node, including its production context, ownership
// key and platform adaptation. The surrounding layout is a minimal test host.
func mountedAssembledDiscreteControl(t *testing.T, s *screen, id string) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindowSize(t, s, func() ui.Node {
		n, ok := findNode(s.view(), id)
		if !ok {
			t.Fatalf("missing assembled control %s", id)
		}
		return ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: id, Sequence: 1}, Children: []ui.Node{n}}
	}, 700, 500, inspectorLayoutTestTheme())
}

// Spend one originating key event while withholding its posted owner callback,
// then repaint after a real public service agent edit. No Change callback or
// driver private queue is replaced by this fixture. This is routed evidence,
// not a claim about hardware input or the native macOS event queue.
func TestMountedQueuedDiscreteInputCannotBorrowAgentContext(t *testing.T) {
	for _, variant := range []string{"checkbox-output", "checkbox-motor", "keyboard-button", "disclosure", "component-list"} {
		t.Run(variant, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "simulation.pause", struct{}{})
			id, code, r := "pdf-color", key.CodeSpacebar, ' '
			switch variant {
			case "checkbox-output":
				uiAction(t, s, "inspector-section-document", nil)
			case "checkbox-motor":
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "motor-dc"})
				agentAction(t, s.service, "simulation.pause", struct{}{})
				uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "MTR1"})
				id = "motor-locked"
			case "keyboard-button":
				uiAction(t, s, "inspector-section-document", nil)
				id, code, r = "display-voltage", key.CodeReturnEnter, 0
			case "disclosure":
				id, code, r = "diagnostic-details", key.CodeRightArrow, 0
			case "component-list":
				id, code, r = "part-list", key.CodeDownArrow, 0
			}
			m := mountedAssembledDiscreteControl(t, s, id)
			before := s.service.Snapshot()
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.KeyChord, Code: code, Rune: r}); err != nil {
				t.Fatal(err)
			}
			step, err := m.manual.Step(m.ctx, 1)
			if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || s.app.Pending().Ordinary == 0 {
				t.Fatal("originating event was not isolated before owner delivery", m.snapshot().FocusedID, step, err)
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("input changed the model before owner delivery")
			}
			switch variant {
			case "checkbox-output":
				page := before.SchematicPage
				margin := 17.
				page.MarginMM = &margin
				agentAction(t, s.service, "export.page.set", page)
			case "checkbox-motor":
				for _, p := range before.Components {
					if p.ID == "MTR1" {
						config := *p.Power.Motor
						config.Resistance++
						agentAction(t, s.service, "motor.configure", workbench.MotorParams{ID: p.ID, Parameters: config})
					}
				}
			case "component-list":
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "2k"})
			default:
				prefs := before.Display.Preferences
				prefs.VoltageRange = 17
				agentAction(t, s.service, "display.preferences.set", prefs)
			}
			agent := s.service.Snapshot()
			if agent.CommandRevision == before.CommandRevision {
				t.Fatal("fixture did not perform an intervening agent edit")
			}
			if err := s.app.Invalidate("main"); err != nil {
				t.Fatal(err)
			}
			// Refresh retained descriptors without owner drain before the benign
			// move is admitted. The old action remains queued throughout rebind.
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(680, 480)}); err != nil {
				t.Fatal(err)
			}
			m.settle()
			after := s.service.Snapshot()
			if !reflect.DeepEqual(agent, after) {
				t.Fatalf("queued %s borrowed newer agent context: revision %d -> %d, actor %q, drawing %+v -> %+v", id, agent.CommandRevision, after.CommandRevision, after.LastActor, agent.Drawing, after.Drawing)
			}
			// Independent new input must remain usable after the refusal.
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: code, Rune: r})
			fresh := s.service.Snapshot()
			if fresh.CommandRevision != agent.CommandRevision+1 || fresh.LastActor != "user" {
				t.Fatal("fresh input did not resume", s.errorMessage, fresh.CommandRevision, agent.CommandRevision)
			}
		})
	}
}
