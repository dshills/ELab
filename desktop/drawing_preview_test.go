package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestDrawingHoverIsPureAndRetiresAcrossAgentChanges(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
	prepareSchematicPointer(t, s)
	before := s.service.Snapshot()
	s.app.Do(func() { s.routeNativePointer("schematic", 4, workbench.Point{X: 103, Y: 107}, workbench.Point{}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.drawing.hover == nil {
		t.Fatal("hover mutated accepted state or failed to show")
	}
	preview := s.drawing.drawingPreview(before)
	if preview == nil || !preview.Valid || preview.Position != (workbench.Point{X: 100, Y: 110}) {
		t.Fatal("canvas disagrees with shared snapped preview", preview)
	}
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		state := before
		state.Theme = mode
		mesh, err := schematic(state, *s.drawing.View, 0)
		if err != nil {
			t.Fatal(err)
		}
		withPreview, err := s.drawing.withDrawingPreview(mesh, state, *s.drawing.View)
		if err != nil || len(withPreview.Vertices()) <= len(mesh.Vertices()) {
			t.Fatal("ghost not drawn in theme", mode, err)
		}
	}
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "led"})
	if p := s.drawing.drawingPreview(s.service.Snapshot()); p != nil {
		t.Fatal("retired resistor ghost adopted agent's LED intent")
	}
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.routeNativePointer("schematic", 4, workbench.Point{X: 100, Y: 100}, workbench.Point{}) })
	if p := s.drawing.drawingPreview(s.service.Snapshot()); p == nil || p.Component.Kind != "led" {
		t.Fatal("fresh hover did not preview LED")
	}
	s.app.Do(func() { s.routeNativePointer("schematic", 5, workbench.Point{}, workbench.Point{}) })
	if s.drawing.hover != nil {
		t.Fatal("leaving canvas retained ghost")
	}
}

func TestDrawingPreviewClickWiresNearestPinAndUndo(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	for i, x := range []float64{0, 10, 100} {
		agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: []string{"A", "B", "C"}[i], Kind: "junction", Position: workbench.Point{X: x}})
	}
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "wire"})
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.routeNativePointer("schematic", 0, workbench.Point{X: 9}, workbench.Point{}) })
	if s.service.Snapshot().Drawing.WireStart != "B.node" {
		t.Fatal("click did not use nearest preview pin", s.errorMessage)
	}
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.routeNativePointer("schematic", 4, workbench.Point{X: 100}, workbench.Point{}) })
	p := s.drawing.drawingPreview(s.service.Snapshot())
	if p == nil || !p.Valid || len(p.Path) < 2 {
		t.Fatal("pending wire has no valid preview route", p)
	}
	s.app.Do(func() { s.routeNativePointer("schematic", 0, workbench.Point{X: 100}, workbench.Point{}) })
	after := s.service.Snapshot()
	if len(after.Wires) != 1 || after.Wires[0].From != "B.node" || after.Wires[0].To != "C.node" || s.errorMessage != "" {
		t.Fatal("click and query disagree on connection", s.errorMessage)
	}
	s.app.Do(func() { s.invoke("edit.undo", struct{}{}) })
	if len(s.service.Snapshot().Wires) != 0 || s.errorMessage != "" {
		t.Fatal("wire did not undo as one operation", s.errorMessage)
	}
}

func TestDrawingKeyboardToolsAndRotationPreserveConnectedCircuit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	before := s.service.Snapshot()
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.drawing.routeSchematicKey("r", s.drawing.pointerContext.Load()) })
	after := s.service.Snapshot()
	if s.errorMessage != "" || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Nets, after.Nets) || !reflect.DeepEqual(before.Result, after.Result) || before.ElectricalRevision != after.ElectricalRevision || after.UndoName != "Rotate component" {
		t.Fatal("rotation disturbed connected circuit", s.errorMessage)
	}
	for _, chord := range []string{"w", "h", "v"} {
		prepareSchematicPointer(t, s)
		s.app.Do(func() { s.drawing.routeSchematicKey(chord, s.drawing.pointerContext.Load()) })
		want := map[string]string{"w": "wire", "h": "pan", "v": "select"}[chord]
		if s.errorMessage != "" || s.service.Snapshot().Drawing.Tool != want {
			t.Fatal("shortcut did not reach shared tool command", chord, s.errorMessage)
		}
	}
}

func TestDrawingGhostDoesNotBorrowAcceptedSwitchStateByPlaceholderID(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	fixture := workbench.SwitchedNPNExampleGraph()
	for _, part := range fixture.Components {
		id := part.ID
		if id == "S1" {
			id = "Preview"
		}
		value := ""
		if part.Unit != "" {
			value = workbench.FormatValue(part.Value, part.Unit)
		}
		agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: id, Kind: part.Kind, Value: value, Position: part.Position})
	}
	for _, wire := range fixture.Wires {
		agentAction(t, s.service, "wire.connect", workbench.ConnectParams{From: strings.ReplaceAll(wire.From, "S1.", "Preview."), To: strings.ReplaceAll(wire.To, "S1.", "Preview.")})
	}
	agentAction(t, s.service, "switch.set", workbench.SwitchParams{ID: "Preview", Closed: true})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "switch"})
	prepareSchematicPointer(t, s)
	s.app.Do(func() { s.routeNativePointer("schematic", 4, workbench.Point{X: 100, Y: 100}, workbench.Point{}) })
	state := s.service.Snapshot()
	if len(state.SwitchStates) != 1 || state.SwitchStates[0].AppliedClosed == nil || !*state.SwitchStates[0].AppliedClosed {
		t.Fatal("fixture did not establish a real accepted closed switch")
	}
	base, err := schematic(state, *s.drawing.View, 0)
	if err != nil {
		t.Fatal(err)
	}
	withLive, err := s.drawing.withDrawingPreview(base, state, *s.drawing.View)
	if err != nil {
		t.Fatal(err)
	}
	// Visual control only: remove accepted actuation metadata from the ghost
	// renderer's input while retaining the exact same base mesh and graph.
	state.SwitchStates = nil
	withoutLive, err := s.drawing.withDrawingPreview(base, state, *s.drawing.View)
	if err != nil || !reflect.DeepEqual(withLive.Vertices(), withoutLive.Vertices()) {
		t.Fatal("new default switch ghost borrowed the existing Preview component's live state", err)
	}
}
