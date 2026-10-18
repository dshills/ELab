//go:build offscreen

package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedInspectorControl(t *testing.T, s *screen, id string) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		var control ui.Node
		for _, p := range state.Components {
			if p.ID == state.Selected {
				for _, node := range s.componentInspector(state, workbench.Graph{Components: state.Components, Wires: state.Wires}, p) {
					if node.ID == id {
						control = node
					}
				}
			}
		}
		control.MinWidth, control.MinHeight = 360, 40
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: id, Sequence: 1}, Children: []ui.Node{control}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func mountedControlElement(t *testing.T, m *mountedControlWindow, id string) cogent.ManualElement {
	t.Helper()
	for _, element := range m.snapshot().Elements {
		if element.ID == id {
			return element
		}
	}
	t.Fatalf("missing mounted control %s", id)
	return cogent.ManualElement{}
}

func TestMountedNumericOwnerDoesNotInheritPrivateWiperDraft(t *testing.T) {
	for _, scenario := range []string{"unchanged-repaint", "document-replacement", "selected-object"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			if err := smokeConstructPotentiometer(s); err != nil {
				t.Fatal(err)
			}
			id := s.service.Snapshot().Selected
			if scenario == "selected-object" {
				agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "P2", Kind: "potentiometer", Value: "10k"})
				for _, pair := range [][2]string{{"V1.positive", "P2.1"}, {"GND.reference", "P2.2"}} {
					agentAction(t, s.service, "wire.connect", workbench.ConnectParams{From: pair[0], To: pair[1]})
				}
			}
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: id})
			path := filepath.Join(t.TempDir(), "numeric-owner.elab")
			agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
			m := mountedInspectorControl(t, s, "pot-wiper-exact")
			original := mountedControlElement(t, m, "pot-wiper-exact")
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
			for _, ch := range []struct {
				rune rune
				code key.Codes
			}{{'7', key.Code7}, {'5', key.Code5}} {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: ch.rune, Code: ch.code})
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("typing prematurely committed numeric edit")
			}
			if scenario == "document-replacement" {
				agentAction(t, s.service, "project.load", workbench.LoadParams{Path: path, DiscardChanges: true})
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: id})
			} else if scenario == "selected-object" {
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "P2"})
			}
			current := s.service.Snapshot()
			m.frame()
			field := mountedControlElement(t, m, "pot-wiper-exact")
			t.Logf("incarnation %d→%d; owner %s→%s", original.Incarnation, field.Incarnation, before.Selected, current.Selected)
			if scenario != "unchanged-repaint" {
				if field.Incarnation == original.Incarnation {
					t.Fatal("numeric owner change retained native editing state")
				}
				for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
					m.input(cogent.ManualInput{Type: kind, Button: events.Left, Target: field.ID, TargetIncarnation: field.Incarnation})
				}
				if m.snapshot().FocusedID != field.ID {
					t.Fatal("replacement numeric field was not actually focused")
				}
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			after, err := s.service.Wait(m.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "unchanged-repaint" {
				if !reflect.DeepEqual(current, after) {
					t.Fatalf("old private numeric edit changed new owner: before %+v after %+v", current.Mixed.Potentiometers, after.Mixed.Potentiometers)
				}
				// Enter may move focus. Explicitly start a fresh edit on the new
				// owner, proving refusal was not just missing keyboard delivery.
				for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
					m.input(cogent.ManualInput{Type: kind, Button: events.Left, Target: field.ID, TargetIncarnation: field.Incarnation})
				}
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: '6', Code: key.Code6})
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: '0', Code: key.Code0})
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
				accepted, err := s.service.Wait(m.ctx)
				if err != nil || accepted.CommandRevision != current.CommandRevision+1 {
					t.Fatal("fresh numeric edit did not commit once", err, s.errorMessage)
				}
				for _, part := range accepted.Components {
					if part.Kind == "potentiometer" {
						want := .5
						if part.ID == current.Selected {
							want = .6
						}
						if *part.WiperPosition != want {
							t.Fatal("fresh numeric edit targeted wrong part", part.ID, *part.WiperPosition)
						}
					}
				}
				return
			}
			if after.CommandRevision != before.CommandRevision+1 {
				t.Fatalf("normal numeric draft did not commit once: %d→%d; %s", before.CommandRevision, after.CommandRevision, s.errorMessage)
			}
			for _, part := range after.Components {
				if part.ID == id && (part.WiperPosition == nil || *part.WiperPosition != .75) {
					t.Fatal("normal numeric draft did not produce 75%", part.WiperPosition)
				}
			}
		})
	}
}

func TestMountedButtonOwnerCannotRetargetHeldRotate(t *testing.T) {
	for _, changeOwner := range []bool{false, true} {
		name := "unchanged-repaint"
		if changeOwner {
			name = "selected-object"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			for i, id := range []string{"R1", "R2"} {
				agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: id, Kind: "resistor", Value: "1k", Position: workbench.Point{X: float64(i * 100)}})
			}
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
			m := mountedInspectorControl(t, s, "rotate-part")
			button := mountedControlElement(t, m, "rotate-part")
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("button activated on press rather than release")
			}
			if changeOwner {
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R2"})
			}
			current := s.service.Snapshot()
			m.frame()
			fresh := mountedControlElement(t, m, "rotate-part")
			t.Logf("incarnation %d→%d; owner %s→%s", button.Incarnation, fresh.Incarnation, before.Selected, current.Selected)
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			after := s.service.Snapshot()
			if changeOwner {
				if !reflect.DeepEqual(current, after) {
					t.Fatal("held rotate changed a different selected object", after.Components)
				}
				if fresh.Incarnation == button.Incarnation {
					t.Fatal("new button owner retained old pressed state")
				}
				for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
					m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
				}
				accepted := s.service.Snapshot()
				if accepted.CommandRevision != current.CommandRevision+1 || accepted.Components[0].Rotation != 0 || accepted.Components[1].Rotation != 90 {
					t.Fatal("fresh button failed to rotate the current object once")
				}
			} else if after.CommandRevision != before.CommandRevision+1 || after.Components[0].Rotation != 90 || after.Components[1].Rotation != 0 {
				t.Fatal("normal press/release did not rotate original object once", after.Components)
			}
		})
	}
}

func TestMountedButtonOwnerDoesNotRestartReplacementDocument(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		root := ui.Node{ID: "root", Kind: ui.Column, Children: []ui.Node{s.button("simulation.restart", "Restart")}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	button := mountedControlElement(t, m, "simulation.restart")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "simulation.restart", struct{}{})
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
	before := s.service.Snapshot()
	m.frame()
	fresh := mountedControlElement(t, m, "simulation.restart")
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || fresh.Incarnation == button.Incarnation {
		t.Fatal("old Restart press changed the replacement experiment")
	}
	for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
		m.input(cogent.ManualInput{Type: kind, Button: events.Left, Position: at})
	}
	after, err := s.service.Wait(m.ctx)
	if err != nil || after.CommandRevision != before.CommandRevision+1 || after.RunID != before.RunID+1 || after.Transient.Sample.Time != 0 || !after.Paused {
		t.Fatal("fresh Restart did not work on current document", err)
	}
}

func TestMountedButtonOwnerPreservesSingleClickValueApply(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	m := newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		var controls []ui.Node
		for _, part := range state.Components {
			if part.ID == state.Selected {
				for _, node := range s.componentInspector(state, workbench.Graph{Components: state.Components, Wires: state.Wires}, part) {
					if node.ID == "value-entry" || node.ID == "apply-value" {
						controls = append(controls, node)
					}
				}
			}
		}
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "value-entry", Sequence: 1}, Children: controls}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
	s.settleField = func(id string, done func(error)) error {
		return cogent.SettleTextField(m.ctx, s.app, "main", id, done)
	}
	before := s.service.Snapshot()
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: key.Meta})
	for _, ch := range []struct {
		rune rune
		code key.Codes
	}{{'2', key.Code2}, {'.', key.CodeFullStop}, {'2', key.Code2}, {'k', key.CodeK}} {
		m.input(cogent.ManualInput{Type: events.KeyChord, Rune: ch.rune, Code: ch.code})
	}
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("unfinished text changed the source before Apply")
	}
	button := mountedControlElement(t, m, "apply-value")
	for _, kind := range []events.Types{events.MouseDown, events.MouseUp} {
		m.input(cogent.ManualInput{Type: kind, Button: events.Left, Target: button.ID, TargetIncarnation: button.Incarnation})
	}
	after, err := s.service.Wait(m.ctx)
	if err != nil || after.CommandRevision != before.CommandRevision+2 || s.pendingValueApply != nil || s.errorMessage != "" {
		t.Fatal("one Apply click failed to settle draft and apply source once", err, s.errorMessage, before.CommandRevision, after.CommandRevision)
	}
	for _, part := range after.Components {
		if part.ID == "R1" && part.Value != 2200 {
			t.Fatal("Apply used old rather than finished text", part.Value)
		}
	}
}
