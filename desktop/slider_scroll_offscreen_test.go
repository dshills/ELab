//go:build offscreen

package main

import (
	"fmt"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedSliderWheelScrollPreservesCircuit(t *testing.T) {
	for _, focused := range []bool{false, true} {
		t.Run(fmt.Sprintf("focused=%t", focused), func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			if err := smokeConstructPotentiometer(s); err != nil {
				t.Fatal(err)
			}
			agentAction(t, s.service, "simulation.pause", struct{}{})
			var potID string
			for _, part := range s.service.Snapshot().Components {
				if part.Kind == "potentiometer" {
					potID = part.ID
				}
			}
			if potID == "" {
				t.Fatal("fixture has no potentiometer")
			}
			focus := "wheel-neutral"
			if focused {
				focus = "pot-wiper"
			}
			m := newMountedControlWindowSize(t, s, func() ui.Node {
				state := s.service.Snapshot()
				var slider ui.Node
				for _, part := range state.Components {
					if part.ID != potID {
						continue
					}
					for _, node := range s.potentiometerControls(state, part) {
						if node.ID == "pot-wiper" {
							slider = node
						}
					}
				}
				if slider.Kind != ui.Slider || slider.Disabled {
					t.Fatal("missing enabled production wiper control")
				}
				slider.MinWidth, slider.MinHeight = 240, 40
				children := []ui.Node{slider, {ID: "wheel-witness", Kind: ui.Label, Text: "Following controls", MinHeight: 32}}
				for i := range 16 {
					children = append(children, ui.Node{ID: fmt.Sprintf("wheel-padding-%d", i), Kind: ui.Label, Text: "Further controls", MinHeight: 32})
				}
				root := ui.Node{ID: "root", Kind: ui.Column, Grow: true, Focus: ui.FocusRequest{ID: focus, Sequence: 1}, Children: []ui.Node{
					{ID: "wheel-neutral", Kind: ui.Button, Text: "Outside the slider", Activate: func() {}},
					{ID: "wheel-scroll", Kind: ui.Scroll, Grow: true, Children: []ui.Node{{ID: "wheel-content", Kind: ui.Column, Children: children}}},
				}}
				s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
				return root
			}, 400, 300, inspectorLayoutTestTheme())
			if m.snapshot().FocusedID != focus {
				t.Fatal("fixture did not establish requested focus", m.snapshot().FocusedID)
			}
			before := s.service.Snapshot()
			slider := mountedControlElement(t, m, "pot-wiper")
			witness := mountedControlElement(t, m, "wheel-witness").Bounds
			m.input(cogent.ManualInput{Type: events.Scroll, Target: slider.ID, TargetIncarnation: slider.Incarnation, Delta: math32.Vec2(0, 12)})
			m.frame()
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("wheel changed circuit state instead of scrolling")
			}
			if next := mountedControlElement(t, m, "wheel-witness").Bounds; next.Min.Y >= witness.Min.Y {
				t.Fatal("wheel did not move the enclosing viewport", witness, next)
			}
			if focused {
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeRightArrow})
				if _, err := s.service.Wait(m.ctx); err != nil {
					t.Fatal(err)
				}
				after := s.service.Snapshot()
				for _, part := range after.Components {
					if part.ID == potID && *part.WiperPosition != .51 {
						t.Fatal("explicit Right key did not increment wiper once", *part.WiperPosition)
					}
				}
				if after.CommandRevision != before.CommandRevision+1 || after.UndoEntry == nil || after.UndoEntry.CommandRevision != after.CommandRevision {
					t.Fatal("explicit key lost its single shared history entry")
				}
			}
		})
	}
}
