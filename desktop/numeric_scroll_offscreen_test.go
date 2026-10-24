//go:build offscreen

package main

import (
	"fmt"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedNumericOwnerWheelScrollPreservesValues(t *testing.T) {
	for _, fieldID := range []string{"setting-initial", "recorded-speed"} {
		for _, focused := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/focused=%t", fieldID, focused), func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
				agentAction(t, s.service, "simulation.pause", struct{}{})
				agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "simulation"})
				duration := .02
				agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}})
				completedVerification(t, s.service, s.service.Snapshot().LastOperation)
				agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
				focus := "wheel-neutral"
				if focused {
					focus = fieldID
				}
				m := newMountedControlWindowSize(t, s, func() ui.Node {
					field, ok := findNode(s.view(), fieldID)
					if !ok || field.Kind != ui.NumberField || field.Disabled {
						t.Fatal("missing enabled production number field", fieldID)
					}
					field.MinWidth, field.MinHeight = 120, 40
					children := []ui.Node{field, {ID: "wheel-witness", Kind: ui.Label, Text: "Following controls", MinHeight: 32}}
					for i := range 16 {
						children = append(children, ui.Node{ID: fmt.Sprintf("wheel-padding-%d", i), Kind: ui.Label, Text: "Further controls", MinHeight: 32})
					}
					return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Focus: ui.FocusRequest{ID: focus, Sequence: 1}, Children: []ui.Node{
						{ID: "wheel-neutral", Kind: ui.Button, Text: "Outside the editor", Activate: func() {}},
						{ID: "wheel-scroll", Kind: ui.Scroll, Grow: true, Children: []ui.Node{{ID: "wheel-content", Kind: ui.Column, Children: children}}},
					}}
				}, 400, 300, inspectorLayoutTestTheme())
				if m.snapshot().FocusedID != focus {
					t.Fatal("fixture did not establish actual editor focus", m.snapshot().FocusedID)
				}
				before := s.service.Snapshot()
				field := mountedControlElement(t, m, fieldID)
				witness := mountedControlElement(t, m, "wheel-witness").Bounds
				m.input(cogent.ManualInput{Type: events.Scroll, Target: field.ID, TargetIncarnation: field.Incarnation, Delta: math32.Vec2(0, 12)})
				m.frame()
				if !reflect.DeepEqual(before, s.service.Snapshot()) {
					t.Fatal("wheel changed accepted numeric state instead of scrolling")
				}
				if next := mountedControlElement(t, m, "wheel-witness").Bounds; next.Min.Y >= witness.Min.Y {
					t.Fatal("wheel did not move the enclosing scroll viewport", witness, next)
				}
				if focused {
					// Numeric editing remains available through deliberate keys.
					node, _ := findNode(s.view(), fieldID)
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeUpArrow})
					after, _ := findNode(s.view(), fieldID)
					if after.Number != node.Number+node.Numeric.Step || reflect.DeepEqual(before, s.service.Snapshot()) {
						t.Fatal("scroll repair disabled deliberate numeric editing", node.Number, after.Number)
					}
				}
			})
		}
	}
}
