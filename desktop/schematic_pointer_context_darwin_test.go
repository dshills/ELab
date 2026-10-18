//go:build darwin

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

// Exercise the actual owner-delivery method without installing an OS monitor.
func TestSchematicNativeQueuedFrameCannotAdoptRepaint(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	prepareSchematicPointer(t, s)
	b := &pointerBridge{route: s.routeNativePointer}
	b.enabled.Store(true)
	b.publish(0, 0, *s.drawing.View, "schematic", s.drawing.pointerToken())
	old := b.frame
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "led"})
	prepareSchematicPointer(t, s)
	b.publish(0, 0, *s.drawing.View, "schematic", s.drawing.pointerToken())
	before := s.service.Snapshot()
	s.app.Do(func() { b.deliver(old, 0, workbench.Point{}, workbench.Point{}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("queued native click adopted the new frame's document/tool")
	}
	s.app.Do(func() { b.deliver(b.frame, 0, workbench.Point{}, workbench.Point{}) })
	if s.errorMessage != "" || len(s.service.Snapshot().Components) != 1 || s.service.Snapshot().Components[0].Kind != "led" {
		t.Fatal("fresh native packet did not place current LED", s.errorMessage)
	}
}

func TestSchematicNativeQueuedReleaseSurvivesOwnSelectionRepaint(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "R1", Kind: "resistor", Value: "1k", Position: workbench.Point{X: 100, Y: 100}})
	agentAction(t, s.service, "component.select", workbench.IDParams{})
	prepareSchematicPointer(t, s)
	b := &pointerBridge{route: s.routeNativePointer}
	b.enabled.Store(true)
	b.publish(0, 0, *s.drawing.View, "schematic", s.drawing.pointerToken())
	pressed := b.frame
	s.app.Do(func() { b.deliver(pressed, 0, workbench.Point{X: 100, Y: 100}, workbench.Point{}) })
	prepareSchematicPointer(t, s)
	b.publish(0, 0, *s.drawing.View, "schematic", s.drawing.pointerToken())
	before := s.service.Snapshot()
	s.app.Do(func() {
		b.deliver(pressed, 1, workbench.Point{}, workbench.Point{X: 30, Y: -20})
		b.deliver(pressed, 2, workbench.Point{}, workbench.Point{X: 10})
	})
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || after.Components[0].Position != (workbench.Point{X: 140, Y: 80}) {
		t.Fatal("own selection repaint swallowed the queued drag/release", s.errorMessage)
	}
}

func TestSchematicNativeQueuedPanContinuesOnlyOwnWrites(t *testing.T) {
	for _, intervene := range []bool{false, true} {
		t.Run(map[bool]string{false: "own-pan", true: "agent-pan"}[intervene], func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			prepareSchematicPointer(t, s)
			b := &pointerBridge{route: s.routeNativePointer}
			b.enabled.Store(true)
			b.publish(0, 0, *s.drawing.View, "schematic", s.drawing.pointerToken())
			pressed := b.frame
			s.app.Do(func() {
				b.deliver(pressed, 0, workbench.Point{}, workbench.Point{})
				b.deliver(pressed, 1, workbench.Point{}, workbench.Point{X: 20})
			})
			first := s.service.Snapshot()
			if intervene {
				agentAction(t, s.service, "view.pan", map[string]float64{"dx": 50, "dy": 0})
			}
			before := s.service.Snapshot()
			prepareSchematicPointer(t, s)
			view := *s.drawing.View
			view.Center = canvas.Point{X: before.View.Center.X, Y: before.View.Center.Y}
			b.publish(0, 0, view, "schematic", s.drawing.pointerToken())
			s.app.Do(func() {
				b.deliver(pressed, 1, workbench.Point{}, workbench.Point{X: 30})
				b.deliver(pressed, 2, workbench.Point{}, workbench.Point{})
			})
			after := s.service.Snapshot()
			if intervene {
				if !reflect.DeepEqual(before, after) {
					t.Fatal("queued pan overwrote another actor's viewport")
				}
				return
			}
			if s.errorMessage != "" || first.View.Center.X != -20 || after.View.Center.X != -50 || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("queued pan failed to continue its own accepted write", s.errorMessage, after.View)
			}
		})
	}
}
