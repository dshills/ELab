//go:build offscreen

package main

import (
	"image"
	"math"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Complement native endpoint observations with controlled event timing and
// actual retained hit geometry. Manual clock timing is not hardware-rate proof.
func TestMountedWorkspaceRepeatedOffCenterDragsUseUpdatedHandle(t *testing.T) {
	for _, slow := range []bool{false, true} {
		t.Run(map[bool]string{false: "quick", true: "classified-slide"}[slow], func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			defer cleanup()
			view := func() ui.Node {
				state := s.service.Snapshot()
				dock := s.workspaceDock(ui.Node{ID: "schematic-fixture", Kind: ui.Column, Grow: true}, ui.Node{ID: "scope-fixture", Kind: ui.Column, Grow: true}, state)
				root := ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{dock}, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0))}}
				s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
				return root
			}
			m := newMountedControlWindowSize(t, s, view, 700, 600, inspectorLayoutTestTheme())
			grip := func() image.Rectangle {
				t.Helper()
				for _, e := range m.snapshot().Elements {
					if e.ID == "split:d29ya3NwYWNlLXNwbGl0" {
						return e.Bounds
					}
				}
				t.Fatal("missing retained handle")
				return image.Rectangle{}
			}
			original := s.service.Snapshot()
			b := grip()
			start := b.Min.Add(b.Size().Div(2)).Add(image.Pt(2, 1))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start})
			if s.service.Snapshot().CommandRevision != original.CommandRevision {
				t.Fatal("focus click changed layout")
			}
			for _, delta := range []int{-60, 60, -40, 40} {
				b = grip()
				start = b.Min.Add(b.Size().Div(2)).Add(image.Pt(2, 1))
				before := s.service.Snapshot()
				m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
				if slow {
					if _, err := m.clock.Advance(core.TimingSettings.SlideStartTime + time.Millisecond); err != nil {
						t.Fatal(err)
					}
					m.settle()
				}
				m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(0, delta/2))})
				m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(0, delta))})
				m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(0, delta))})
				after := s.service.Snapshot()
				moved := grip().Min.Y - b.Min.Y
				if math.Abs(float64(moved-delta)) > 1 {
					t.Fatalf("new retained grip moved %d, requested %d (layout %g -> %g): %s", moved, delta, before.WorkspaceLayout.SchematicFraction, after.WorkspaceLayout.SchematicFraction, s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			}
			if math.Abs(s.service.Snapshot().WorkspaceLayout.SchematicFraction-original.WorkspaceLayout.SchematicFraction) > .003 {
				t.Fatal("inverse gestures did not restore the accepted share")
			}
		})
	}
}
