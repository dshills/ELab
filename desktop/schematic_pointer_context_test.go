package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
)

func prepareSchematicPointer(t *testing.T, s *screen) {
	t.Helper()
	if err := s.app.Do(func() {
		if _, err := s.mesh(*s.drawing.View); err != nil {
			t.Fatal(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSchematicClickReleaseAfterViewportUpdateIsAnUnchangedNoop(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.routeNativePointer("schematic", 0, workbench.Point{X: 120, Y: 80}, workbench.Point{}) })
	agentAction(t, s.service, "view.pan", map[string]float64{"dx": 20, "dy": 10})
	before := s.service.Snapshot()
	s.app.Do(func() { s.routeNativePointer("schematic", 2, workbench.Point{X: 120, Y: 80}, workbench.Point{}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage != "" || s.drawing.pointerGesture != nil {
		t.Fatal("non-editing release reported a stale action or changed circuit", s.errorMessage)
	}
}

func TestSchematicPreparedPointerCannotPlaceIntoReplacement(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	prepareSchematicPointer(t, s)
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "led"})
	before := s.service.Snapshot()
	s.app.Do(func() { s.routeNativePointer("schematic", 0, workbench.Point{X: 120, Y: 80}, workbench.Point{}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old schematic click placed a component into replacement document")
	}
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.routeNativePointer("schematic", 0, workbench.Point{X: 120, Y: 80}, workbench.Point{}) })
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Components) != 1 || after.Components[0].Kind != "led" || after.Components[0].Position != (workbench.Point{X: 120, Y: 80}) || after.CommandRevision != before.CommandRevision+1 {
		t.Fatal("fresh placement did not use replacement's LED tool", s.errorMessage)
	}
}

func TestSchematicPreparedPointerCannotAdoptAgentWireStart(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareNamedWiringLED(t, s.service)
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "wire"})
	agentAction(t, s.service, "wire.begin", map[string]string{"terminal": "V1.positive"})
	prepareSchematicPointer(t, s)
	agentAction(t, s.service, "wire.begin", map[string]string{"terminal": "D1.cathode"})
	before := s.service.Snapshot()
	part, _ := (workbench.Graph{Components: before.Components, Wires: before.Wires}).Component("R1")
	target, _ := workbench.TerminalPosition(part, "1")
	s.app.Do(func() { s.routeNativePointer("schematic", 0, target, workbench.Point{}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old schematic click connected the agent's newly selected wire start")
	}
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.routeNativePointer("schematic", 0, target, workbench.Point{}) })
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Wires) != 1 || after.Wires[0].From != "D1.cathode" || after.Wires[0].To != "R1.1" {
		t.Fatal("fresh wiring did not use the current start", s.errorMessage)
	}
}

func TestSchematicPreparedPointerZoomAndCapturedContextRefuseChanges(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSchematicPointer(t, s)
	captured := s.drawing.pointerContext.Load()
	agentAction(t, s.service, "view.pan", map[string]float64{"dx": 30, "dy": -20})
	before := s.service.Snapshot()
	s.app.Do(func() { s.routeNativePointer("schematic", 3, workbench.Point{}, workbench.Point{Y: 3}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("stale wheel used a changed viewport")
	}
	prepareSchematicPointer(t, s)
	s.app.Do(func() {
		s.drawing.withPreparedPointer(captured, func() { t.Fatal("captured portable event adopted a newly prepared frame") })
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("retired portable callback changed state")
	}
	s.app.Do(func() { s.routeNativePointer("schematic", 3, workbench.Point{}, workbench.Point{Y: 3}) })
	if s.errorMessage != "" || s.service.Snapshot().View.Zoom <= before.View.Zoom {
		t.Fatal("fresh wheel did not zoom", s.errorMessage)
	}
}

func TestSchematicPreparedPointerDragMatchesPublicMoveAndUndo(t *testing.T) {
	for _, intervene := range []bool{false, true} {
		t.Run(map[bool]string{false: "own-repaint", true: "agent-edit"}[intervene], func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agent := workbench.New()
			defer agent.Close()
			for _, service := range []*workbench.Service{s.service, agent} {
				agentAction(t, service, "component.place", workbench.PlaceParams{ID: "R1", Kind: "resistor", Value: "1k", Position: workbench.Point{X: 100, Y: 100}})
				agentAction(t, service, "component.select", workbench.IDParams{})
			}
			prepareSchematicPointer(t, s)
			s.app.Do(func() { s.routeNativePointer("schematic", 0, workbench.Point{X: 100, Y: 100}, workbench.Point{}) })
			if s.errorMessage != "" || s.drawing.pointerGesture == nil {
				t.Fatal("fresh press did not retain its accepted selection", s.errorMessage)
			}
			agentAction(t, agent, "component.select", workbench.IDParams{ID: "R1"})
			if intervene {
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			}
			prepareSchematicPointer(t, s)
			before := s.service.Snapshot()
			s.app.Do(func() {
				s.routeNativePointer("schematic", 1, workbench.Point{}, workbench.Point{X: 30, Y: -20})
				s.routeNativePointer("schematic", 2, workbench.Point{}, workbench.Point{X: 10})
			})
			after := s.service.Snapshot()
			if intervene {
				if !reflect.DeepEqual(before, after) || s.drawing.dragging || s.drawing.dragID != "" {
					t.Fatal("held drag overwrote intervening value edit or retained preview")
				}
				return
			}
			agentAction(t, agent, "component.move", workbench.MoveParams{ID: "R1", Position: workbench.Point{X: 140, Y: 80}, PreviousPosition: &workbench.Point{X: 100, Y: 100}})
			expected := agent.Snapshot()
			if s.errorMessage != "" || !reflect.DeepEqual(after.Components, expected.Components) || after.UndoName != expected.UndoName || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("gesture did not match one shared move", s.errorMessage)
			}
			s.app.Do(func() { s.invoke("edit.undo", struct{}{}) })
			if s.errorMessage != "" || s.service.Snapshot().Components[0].Position != (workbench.Point{X: 100, Y: 100}) {
				t.Fatal("Undo did not restore complete drag", s.errorMessage)
			}
		})
	}
}

func TestSchematicPreparedPointerDispatchKeepsOriginalPreconditions(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	prepareSchematicPointer(t, s)
	s.app.Do(func() {
		s.drawing.withPreparedPointer(s.drawing.pointerContext.Load(), func() {
			// Reproduce the narrow interval after the UI's predicate check.
			agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "led"})
			before := s.service.Snapshot()
			s.canvasClick(workbench.Point{})
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" {
				t.Fatal("public dispatcher did not refuse stale pointer context")
			}
		})
	})
}
