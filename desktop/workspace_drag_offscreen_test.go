//go:build offscreen

package main

import (
	"image"
	"math"
	"reflect"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// This uses the retained driver's originating event and owner queues, not a
// captured Change callback. It intentionally withholds owner delivery so an
// intervening public agent command can repaint the same retained divider.
func TestMountedWorkspaceQueuedDividerCannotBorrowAgentContext(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	m := newMountedWorkspaceDivider(t, s)
	var handle cogent.ManualElement
	for _, e := range m.snapshot().Elements {
		if e.ID == "split:d29ya3NwYWNlLXNwbGl0" {
			handle = e
		}
	}
	if handle.Bounds.Empty() {
		t.Fatal("missing retained divider")
	}
	at := handle.Bounds.Min.Add(handle.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if m.snapshot().FocusedID != "workspace-split" {
		t.Fatalf("divider did not receive keyboard focus: %q, handle %q %v", m.snapshot().FocusedID, handle.ID, handle.Bounds)
	}
	before := s.service.Snapshot()
	if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.KeyChord, Code: key.CodeDownArrow}); err != nil {
		t.Fatal(err)
	}
	step, err := m.manual.Step(m.ctx, 1)
	if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || s.app.Pending().Ordinary == 0 {
		t.Fatal("fixture did not isolate the originating event before owner delivery", step, err)
	}
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("divider wrote before owner delivery")
	}
	agentAction(t, s.service, "workspace.layout.set", workbench.WorkspaceLayout{SchematicFraction: .75})
	agent := s.service.Snapshot()
	if err := s.app.Invalidate("main"); err != nil {
		t.Fatal(err)
	}
	// Input refreshes dirty descriptors without draining owner callbacks. The
	// benign move is separately queued and does not select or resize anything.
	if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(10, 10)}); err != nil {
		t.Fatal(err)
	}
	m.settle()
	after := s.service.Snapshot()
	if !reflect.DeepEqual(agent, after) {
		t.Fatalf("queued divider borrowed refreshed agent context: fraction %g -> %g, command revision %d -> %d, actor %q", agent.WorkspaceLayout.SchematicFraction, after.WorkspaceLayout.SchematicFraction, agent.CommandRevision, after.CommandRevision, after.LastActor)
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeDownArrow})
	fresh := s.service.Snapshot()
	if fresh.CommandRevision != agent.CommandRevision+1 || fresh.WorkspaceLayout.SchematicFraction <= agent.WorkspaceLayout.SchematicFraction || fresh.LastActor != "user" {
		t.Fatal("fresh divider key did not use current context", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, agent, fresh)
}

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

func newMountedWorkspaceDivider(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	view := func() ui.Node {
		state := s.service.Snapshot()
		dock := s.workspaceDock(ui.Node{ID: "schematic-fixture", Kind: ui.Column, Grow: true}, ui.Node{ID: "scope-fixture", Kind: ui.Column, Grow: true}, state)
		root := ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{dock}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	}
	return newMountedControlWindowSize(t, s, view, 700, 600, inspectorLayoutTestTheme())
}

// A held pointer gesture is one originating action even when its own accepted
// moves repaint. A new agent write must invalidate the rest of that gesture.
func TestMountedWorkspaceHeldDividerCannotBorrowAgentContext(t *testing.T) {
	for _, variant := range []string{"unchanged-repaint", "agent-layout", "agent-input", "document-replacement"} {
		t.Run(variant, func(t *testing.T) {
			s, cleanup := scopeContextRC(t)
			defer cleanup()
			m := newMountedWorkspaceDivider(t, s)
			grip := func() image.Point {
				t.Helper()
				for _, e := range m.snapshot().Elements {
					if e.ID == "split:d29ya3NwYWNlLXNwbGl0" {
						return e.Bounds.Min.Add(e.Bounds.Size().Div(2)).Add(image.Pt(2, 1))
					}
				}
				t.Fatal("missing retained divider")
				return image.Point{}
			}
			start := grip()
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
			if _, err := m.clock.Advance(core.TimingSettings.SlideStartTime + time.Millisecond); err != nil {
				t.Fatal(err)
			}
			m.settle()
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(0, -20))})
			if s.service.Snapshot().CommandRevision != before.CommandRevision+1 {
				t.Fatal("first held move was not accepted")
			}
			switch variant {
			case "agent-layout":
				agentAction(t, s.service, "workspace.layout.set", workbench.WorkspaceLayout{SchematicFraction: .75})
			case "agent-input":
				agentAction(t, s.service, "simulation.input.set", workbench.InputParams{Mode: "charge"})
			case "document-replacement":
				agentAction(t, s.service, "example.rc", workbench.ReplaceParams{DiscardChanges: true})
				agentAction(t, s.service, "simulation.pause", struct{}{})
			}
			accepted := s.service.Snapshot()
			m.frame()
			m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(0, -40))})
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(0, -40))})
			after := s.service.Snapshot()
			if variant == "unchanged-repaint" {
				if after.WorkspaceLayout.SchematicFraction >= accepted.WorkspaceLayout.SchematicFraction || after.LastActor != "user" {
					t.Fatal("own accepted repaint interrupted the held gesture", s.errorMessage)
				}
				assertCursorMeasurementPreservesPhysics(t, before, after)
			} else if !reflect.DeepEqual(accepted, after) {
				t.Fatalf("held divider borrowed %s context: fraction %g -> %g, revision %d -> %d", variant, accepted.WorkspaceLayout.SchematicFraction, after.WorkspaceLayout.SchematicFraction, accepted.CommandRevision, after.CommandRevision)
			}
			// Independent input after release must use the current context.
			start = grip()
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(0, 10))})
			if s.service.Snapshot().WorkspaceLayout.SchematicFraction <= after.WorkspaceLayout.SchematicFraction {
				t.Fatal("fresh divider gesture did not resume", s.errorMessage)
			}
		})
	}
}
