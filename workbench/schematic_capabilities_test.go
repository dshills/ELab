package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestComponentMoveRequiresCompletePreviousPosition(t *testing.T) {
	for i, previous := range []any{map[string]any{}, map[string]any{"x": 0}, map[string]any{"y": 0}, map[string]any{"x": nil, "y": 0}} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "component.place", PlaceParams{Kind: "resistor", ID: "R1", Position: Point{}}, "place-origin")
			refuse(t, s, "component.move", map[string]any{"id": "R1", "position": Point{X: 10, Y: 20}, "previous_position": previous}, "invalid-params")
		})
	}
}

func TestSchematicDiscoveryContractsAreOwnedPureAndBounded(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	commands := []string{"parts.category.set", "parts.search.set", "component.duplicate", "component.move", "component.rotate", "component.select", "object.delete", "drawing.tool.set", "drawing.position.set", "drawing.cancel", "wire.connect", "wire.begin", "wire.terminals.set", "wire.junction.insert", "editor.value.set"}
	for _, command := range commands {
		raw, _ := json.Marshal(CapabilityQuery{Command: command})
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: command, Command: "capabilities.list", Params: raw}, "agent")
		if response.Error != nil {
			t.Fatal(command, response.Error)
		}
		caps := response.Result.(map[string]any)["capabilities"].([]Capability)
		var schema map[string]any
		if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		contract, ok := schema["x-command-contract"].(map[string]any)
		if len(caps) != 1 || !ok || contract["scope"] != "schematic-editing" || contract["completion"] == nil || schema["x-inventory"] == nil {
			t.Fatal(command, "missing state/effect/observation contract")
		}
		switch command {
		case "parts.category.set":
			choices, ok := inputProperty(schema, "category")["enum"].([]any)
			if !ok || len(choices) != 8 || choices[0] != "common" || choices[7] != "connections" {
				t.Fatal("category discovery missing", choices)
			}
		case "parts.search.set":
			if inputProperty(schema, "query")["x-max-utf8-bytes"] != float64(128) {
				t.Fatal("search byte bound missing")
			}
		case "component.move", "drawing.position.set":
			if inputProperty(schema, "position.x")["maximum"] != float64(10000) || inputProperty(schema, "position.y")["minimum"] != float64(-10000) || schema["x-position-snap-grid"] != GridSize {
				t.Fatal("position bounds or snap missing")
			}
		case "wire.connect":
			if schema["x-max-wires"] != float64(MaxWires) || inputProperty(schema, "from")["minLength"] != float64(3) || inputProperty(schema, "id")["minLength"] != float64(0) {
				t.Fatal("wire capacity/end/ID bounds missing")
			}
		case "wire.junction.insert":
			if inputProperty(schema, "wire_ids")["maxItems"] != float64(8) || inputProperty(schema, "wire_ids")["uniqueItems"] != true || schema["x-endpoint-clearance"] != GridSize {
				t.Fatal("junction route/capacity bounds missing")
			}
		}
		caps[0].Input[0] = '!'
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("schematic discovery changed public state")
	}
}

func TestSchematicMoveGuardAndGeometryPreservePhysics(t *testing.T) {
	s := NewExample()
	defer s.Close()
	before := settled(t, s)
	call(t, s, "component.move", map[string]any{"id": "R1", "position": Point{15, -15}, "previous_position": nil}, "move-null-preimage")
	moved := s.Snapshot()
	if moved.Components[1].Position != (Point{20, -20}) || moved.RunID != before.RunID || moved.ElectricalRevision != before.ElectricalRevision || moved.Result.Current != before.Result.Current {
		t.Fatal("move snapping or physical preservation")
	}
	refuse(t, s, "component.move", MoveParams{ID: "R1", Position: Point{40, 40}, PreviousPosition: &Point{20, -19}}, "conflict")
	call(t, s, "component.move", MoveParams{ID: "R1", Position: Point{40, 40}, PreviousPosition: &Point{20, -20}}, "move-exact-preimage")
	call(t, s, "component.rotate", IDParams{ID: "R1"}, "rotate-schematic")
	if s.Snapshot().RunID != before.RunID || s.Snapshot().Components[1].Rotation != 90 {
		t.Fatal("rotation changed physics or pin orientation incorrectly")
	}
}

func TestSchematicPlanningDraftsAndCancellationAreNotConnections(t *testing.T) {
	s := NewExample()
	defer s.Close()
	before := settled(t, s)
	call(t, s, "drawing.tool.set", map[string]any{"tool": "wire", "kind": nil}, "wire-retain-kind")
	call(t, s, "wire.terminals.set", TerminalsParams{From: "V1.positive", To: "R1.1"}, "plan-duplicate-pair")
	call(t, s, "drawing.position.set", PositionParams{Position: Point{200, 0}}, "plan-occupied-center")
	call(t, s, "wire.begin", map[string]any{"terminal": "V1.positive"}, "begin-visible-wire")
	call(t, s, "editor.value.set", DraftParams{Value: "not a parsed numeric value"}, "raw-draft")
	refuse(t, s, "drawing.cancel", map[string]any{"unexpected": true}, "invalid-params")
	call(t, s, "drawing.cancel", struct{}{}, "cancel-visible-wire")
	after := s.Snapshot()
	if after.Drawing.Tool != "select" || after.Drawing.WireStart != "" || after.Drawing.From != "V1.positive" || after.Drawing.To != "R1.1" || after.Drawing.Position != (Point{200, 0}) || after.Drawing.Value != "not a parsed numeric value" {
		t.Fatal("planning/cancel did not preserve the documented choices")
	}
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("drawing draft changed graph/physics/history")
	}
	refuse(t, s, "wire.connect", ConnectParams{From: "R1.1", To: "V1.positive"}, "invalid-params")
	refuse(t, s, "wire.terminals.set", map[string]any{"from": nil, "to": ""}, "invalid-params")
	call(t, s, "component.select", IDParams{ID: "R1"}, "select-committed-resistor")
	if s.Snapshot().Drawing.Value != FormatValue(1000, "Ω") {
		t.Fatal("selection did not restore committed value draft")
	}
}

func TestSchematicMovePreimageValidationRollsBackWholeBatch(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{ID: "R1", Kind: "resistor", Position: Point{}}, "batch-origin")
	first, _ := json.Marshal(PlaceParams{ID: "C1", Kind: "capacitor", Position: Point{100, 100}})
	second := json.RawMessage(`{"id":"R1","position":{"x":10,"y":20},"previous_position":{"x":0}}`)
	refuse(t, s, "edit.batch", BatchParams{Name: "Rejected partial preimage", Edits: []BatchEdit{{Command: "component.place", Params: first}, {Command: "component.move", Params: second}}}, "invalid-params")
	// The failed staged placement must not reserve its ID in the real session.
	call(t, s, "component.place", PlaceParams{ID: "C1", Kind: "capacitor", Position: Point{100, 100}}, "reuse-uncommitted-id")
}

func TestSchematicWiresAllowNetCyclesAndLabelCopiesKeepNames(t *testing.T) {
	s := New()
	defer s.Close()
	for i := 1; i <= 3; i++ {
		call(t, s, "component.place", PlaceParams{ID: fmt.Sprintf("R%d", i), Kind: "resistor", Position: Point{X: float64(i) * 100}}, fmt.Sprintf("place-%d", i))
	}
	call(t, s, "wire.connect", map[string]any{"id": nil, "from": "R1.1", "to": "R2.1"}, "allocate-wire-null")
	call(t, s, "wire.connect", ConnectParams{From: "R2.1", To: "R3.1"}, "connect-next")
	call(t, s, "wire.connect", ConnectParams{From: "R1.1", To: "R3.1"}, "same-net-cycle")
	if len(s.Snapshot().Wires) != 3 || s.Snapshot().Wires[0].ID != "W1" {
		t.Fatal("cycle or null auto-ID refused")
	}
	refuse(t, s, "wire.connect", ConnectParams{From: "R2.1", To: "R1.1"}, "invalid-params")
	call(t, s, "component.place", PlaceParams{ID: "LABEL", Kind: "net-label", Position: Point{Y: 200}, Value: "SUPPLY"}, "place-label")
	call(t, s, "component.duplicate", map[string]any{"id": "LABEL", "position": nil}, "duplicate-label")
	copy := s.Snapshot().Selected
	graph := s.graph.Clone()
	part, _ := graph.Component(copy)
	compiled, _, _ := graph.Compile()
	if part.Label != "SUPPLY" || len(graph.Wires) != 3 || compiled.TerminalNet["LABEL.node"] != compiled.TerminalNet[copy+".node"] {
		t.Fatal("label clone did not retain named-net identity without copying wires")
	}
}
