package workbench

import (
	"context"
	"encoding/json"
	"math"
	"testing"
)

func drawLED(t *testing.T, s *Service) map[string]string {
	t.Helper()
	ids := map[string]string{}
	for i, part := range []struct {
		kind     string
		position Point
	}{{"dc-source", Point{-200, 0}}, {"resistor", Point{0, -110}}, {"led", Point{200, 0}}, {"ground", Point{0, 110}}} {
		response := call(t, s, "component.place", PlaceParams{Kind: part.kind, Position: part.position}, "place-"+part.kind)
		ids[part.kind] = response.Result.(Snapshot).Selected
		if len(s.Snapshot().Components) != i+1 || s.Snapshot().Status != "incomplete" {
			t.Fatal("unfinished drawing was presented as simulated")
		}
	}
	for i, pair := range [][2]string{{ids["dc-source"] + ".positive", ids["resistor"] + ".1"}, {ids["resistor"] + ".2", ids["led"] + ".anode"}, {ids["led"] + ".cathode", ids["ground"] + ".reference"}, {ids["ground"] + ".reference", ids["dc-source"] + ".negative"}} {
		call(t, s, "wire.connect", ConnectParams{From: pair[0], To: pair[1]}, "wire-"+string(rune('a'+i)))
		if i < 3 && s.Snapshot().Result != nil {
			t.Fatal("partial wire graph produced a result")
		}
	}
	return ids
}
func TestBuildLEDFromBlankAndEditArbitraryIDs(t *testing.T) {
	s := New()
	defer s.Close()
	if len(s.Snapshot().Components) != 0 || s.Snapshot().Status != "incomplete" {
		t.Fatal("new session is not blank")
	}
	ids := drawLED(t, s)
	initial := settled(t, s)
	if initial.Result == nil || math.Abs(initial.Result.Current-.00291679) > 2e-8 || len(initial.Nets) != 3 || len(initial.Wires) != 4 {
		t.Fatalf("wrong result from drawn graph: %+v", initial)
	}
	call(t, s, "component.set", ValueParams{ID: ids["resistor"], Value: "2.2k ohm"}, "value")
	edited := settled(t, s)
	if edited.Circuit.Resistance != 2200 || edited.Result.Current >= initial.Result.Current || edited.Result.LEDID != ids["led"] {
		t.Fatal("drawn-part value edit failed")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "blank-again")
	empty := s.Snapshot()
	if len(empty.Components) != 0 || len(empty.Wires) != 0 || !empty.Result.Stale {
		t.Fatal("cleared graph retained current readings")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-clear")
	restored := settled(t, s)
	if len(restored.Components) != 4 || restored.Result.Stale || restored.Circuit.Resistance != 2200 {
		t.Fatal("undo did not restore complete graph")
	}
}
func TestGeometryDoesNotCreateElectricalConnections(t *testing.T) {
	s := NewExample()
	defer s.Close()
	initial := settled(t, s)
	call(t, s, "component.move", MoveParams{ID: "R1", Position: Point{81, 73}}, "move")
	call(t, s, "component.rotate", IDParams{ID: "R1"}, "rotate")
	moved := s.Snapshot()
	if moved.ElectricalRevision != initial.ElectricalRevision || moved.RunID != initial.RunID || moved.Result.Stale {
		t.Fatal("presentation edit restarted the solve")
	}
	if moved.Components[1].Position != (Point{80, 70}) || moved.Components[1].Rotation != 90 {
		t.Fatal("grid/rotation not applied")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-rotate")
	call(t, s, "edit.undo", struct{}{}, "undo-move")
	if s.Snapshot().RunID != initial.RunID {
		t.Fatal("presentation undo restarted solver")
	}
	call(t, s, "object.delete", IDParams{ID: "W2"}, "delete-wire")
	// Place the LED's anode exactly at the resistor's right terminal. Proximity
	// and overlapping paths must still leave the absent electrical wire absent.
	call(t, s, "component.move", MoveParams{ID: "D1", Position: Point{60, -80}}, "near-miss")
	broken := s.Snapshot()
	if broken.Status != "incomplete" || !broken.Result.Stale {
		t.Fatal("geometry silently reconnected a missing wire")
	}
}
func TestDeletePartAndIncidentWiresUndoAtomically(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "object.delete", IDParams{ID: "R1"}, "remove-part")
	deleted := s.Snapshot()
	if len(deleted.Components) != 3 || len(deleted.Wires) != 2 || deleted.Status != "incomplete" {
		t.Fatal("component delete left dangling wires")
	}
	call(t, s, "edit.undo", struct{}{}, "restore-part")
	restored := settled(t, s)
	if len(restored.Wires) != 4 || len(restored.Components) != 4 || restored.Result.Stale {
		t.Fatal("delete undo did not restore atomically")
	}
}
func TestInvalidDrawCommandsPreserveGraphAndHistory(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	for _, test := range []struct{ command, params string }{
		{"component.place", `{"kind":"unknown","position":{"x":10,"y":20}}`},
		{"component.place", `{"kind":"led","position":{"x":10}}`},
		{"component.place", `{"id":"V1","kind":"led","position":{"x":10,"y":20}}`},
		{"component.move", `{"id":"V1","position":{"x":10001,"y":0}}`},
		{"component.move", `{"id":"V1","position":{"x":10,"y":20},"previous_position":{"x":0,"y":0}}`},
		{"wire.connect", `{"from":"V1.positive","to":"missing.1"}`},
		{"wire.connect", `{"from":"V1.positive","to":"V1.positive"}`},
		{"wire.connect", `{"from":"R1.1","to":"V1.positive"}`},
	} {
		before := s.Snapshot()
		revision := before.Revision
		r := s.Dispatch(context.Background(), Request{1, "invalid", test.command, json.RawMessage(test.params), &revision, "invalid"}, "test")
		if r.Error == nil || s.Snapshot().Revision != before.Revision || len(s.Snapshot().Components) != 4 || len(s.Snapshot().Wires) != 4 || s.Snapshot().CanUndo {
			t.Fatalf("invalid draw command changed state: %s %+v", test.params, r)
		}
	}
}
func TestWiringOrderAndResistorDirectionAreIndependentOfLayout(t *testing.T) {
	s := New()
	defer s.Close()
	for _, p := range []PlaceParams{{ID: "battery", Kind: "dc-source", Position: Point{500, 600}, Value: "5 V"}, {ID: "limit", Kind: "resistor", Position: Point{300, 100}, Value: "1k"}, {ID: "light", Kind: "led", Position: Point{-300, -500}}, {ID: "reference", Kind: "ground", Position: Point{700, -100}}} {
		call(t, s, "component.place", p, p.ID)
	}
	// LED first, with resistor reversed, remains the same series equation.
	for i, p := range []ConnectParams{{From: "battery.positive", To: "light.anode"}, {From: "light.cathode", To: "limit.2"}, {From: "limit.1", To: "reference.reference"}, {From: "battery.negative", To: "reference.reference"}} {
		call(t, s, "wire.connect", p, string(rune('a'+i)))
	}
	result := settled(t, s).Result
	if result == nil || math.Abs(result.Current-.00291679) > 2e-8 || result.Nodes["led-anode"] != 5 || result.SourceID != "battery" {
		t.Fatal("compiler depended on IDs/layout/part order")
	}
}
func TestShortAndReversedLEDDoNotPublishValidZero(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "wire.connect", ConnectParams{From: "V1.positive", To: "V1.negative"}, "short")
	state := s.Snapshot()
	if state.Status != "error" || !state.Result.Stale {
		t.Fatal("shorted graph published valid result")
	}
	call(t, s, "edit.undo", struct{}{}, "recover")
	settled(t, s)
	call(t, s, "object.delete", IDParams{ID: "W2"}, "remove-a")
	call(t, s, "object.delete", IDParams{ID: "W3"}, "remove-c")
	call(t, s, "wire.connect", ConnectParams{From: "R1.2", To: "D1.cathode"}, "reverse-a")
	call(t, s, "wire.connect", ConnectParams{From: "D1.anode", To: "GND.reference"}, "reverse-c")
	if state := s.Snapshot(); state.Status != "unsupported" || !state.Result.Stale {
		t.Fatal("reversed LED incorrectly simulated")
	}
}
