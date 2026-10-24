//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedNumericOwnerDecimalStepsReachSharedState(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name, id := "live", "setting-speed"
		if recorded {
			name, id = "recorded", "recorded-speed"
		}
		t.Run(name, func(t *testing.T) {
			s, cleanup := recordedCursorRC(t)
			t.Cleanup(cleanup)
			if recorded {
				agentAction(t, s.service, "recording.speed.set", workbench.RecordingPlayParams{Speed: .03})
			} else {
				uiAction(t, s, id, &ui.ChangeEvent{Number: .03})
			}
			m := newMountedControlWindow(t, s, func() ui.Node {
				state := s.service.Snapshot()
				field, ok := findNode(s.view(), id)
				if !ok || field.Kind != ui.NumberField || field.Disabled {
					t.Fatal("missing enabled production playback field", id)
				}
				field.MinWidth, field.MinHeight = 200, 40
				root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: id, Sequence: 1}, Children: []ui.Node{field}}
				s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
				return root
			})
			before := s.service.Snapshot()
			for i, expected := range []float32{.031, .032} {
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeUpArrow})
				m.frame()
				after := s.service.Snapshot()
				value := after.TransientSettings.PlaybackSpeed
				if recorded {
					value = after.Recording.Speed
				}
				if value != float64(expected) || mountedControlElement(t, m, id).Value != expected || after.CommandRevision != before.CommandRevision+uint64(i+1) || s.errorMessage != "" {
					t.Fatalf("step %d did not reach shared playback state exactly: value=%g want=%g revision=%d error=%s", i, value, expected, after.CommandRevision, s.errorMessage)
				}
				if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !after.Paused {
					t.Fatal("playback step changed the circuit or resumed simulation")
				}
			}
		})
	}
}
