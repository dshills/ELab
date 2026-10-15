package workbench

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestMeasurementContractSelectorsAndPinBounds(t *testing.T) {
	for _, id := range []string{"measurement.inspect", "measurement.select", "measurement.pin.add", "measurement.pin.move", "measurement.pin.remove"} {
		schema := sweepDetailSchema(t, id)
		if schema["x-command-contract"].(map[string]any)["scope"] != "live-measurement-inspection-and-pins" || schema["x-reading-provenance"] == "" {
			t.Fatal("missing live measurement contract", id)
		}
		if id == "measurement.pin.add" || id == "measurement.pin.move" {
			if schema["x-max-pinned-readings"] != float64(MaxPinnedReadings) || schema["x-position-snap-grid"] != GridSize {
				t.Fatal("missing pin bounds", id)
			}
			for _, axis := range []string{"x", "y"} {
				p := inputProperty(schema, "position."+axis)
				if p["minimum"] != float64(-10000) || p["maximum"] != float64(10000) || p["multipleOf"] != nil {
					t.Fatal("input bounds must allow unsnapped coordinates", id, axis)
				}
			}
		}
	}
	selectSchema := sweepDetailSchema(t, "measurement.select")
	querySchema := sweepDetailSchema(t, "measurement.inspect")
	queryTarget := inputProperty(querySchema, "target")["anyOf"].([]any)[0].(map[string]any)
	if !reflect.DeepEqual(inputProperty(queryTarget, "kind")["enum"], inputProperty(selectSchema, "kind")["enum"]) {
		t.Fatal("query and select publish different selectors")
	}
}

func TestMeasurementExplicitEmptyQueryDiffersFromClearSelection(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "component.select", IDParams{ID: "R1"}, "edit-selection")
	call(t, s, "measurement.select", MeasurementSelectParams{Kind: "terminal", ID: "C1.1"}, "focus")
	before := s.Snapshot()
	for _, raw := range []string{`{}`, `{"target":null}`} {
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: "query", Command: "measurement.inspect", Params: json.RawMessage(raw)}, "reader")
		if response.Error != nil || response.Result.(MeasurementInspection).Target.ID != "C1.1" {
			t.Fatal("omitted/null target did not use current focus", response.Error)
		}
	}
	empty := inspectMeasurement(t, s, MeasurementQueryParams{Target: &MeasurementSelectParams{}})
	if empty.Target != nil || empty.Available || len(empty.Readings) != 0 || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("explicit empty query changed focus or fabricated data")
	}
	call(t, s, "measurement.select", MeasurementSelectParams{}, "clear")
	if got := s.Snapshot().MeasurementInspection.Target; got == nil || got.ID != "R1" || got.Kind != "object" {
		t.Fatal("clear selection did not fall back to editing selection", got)
	}
}

func TestMeasurementPinPublishedBoundsNoOpAndPreviousPosition(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{Duration: .01}, "charge")
	before := s.Snapshot()
	call(t, s, "measurement.pin.add", PinSpec{ID: "v:C1", Position: Point{X: -10000, Y: 10000}}, "boundary")
	refuse(t, s, "measurement.pin.add", PinSpec{ID: "v:C1", Position: Point{}}, "invalid-params")
	for _, point := range []Point{{X: math.Nextafter(10000, math.Inf(1))}, {Y: math.Nextafter(-10000, math.Inf(-1))}} {
		refuse(t, s, "measurement.pin.move", MoveParams{ID: "v:C1", Position: point}, "invalid-params")
	}
	previous := Point{X: -9999, Y: 10000}
	refuse(t, s, "measurement.pin.move", MoveParams{ID: "v:C1", Position: Point{}, PreviousPosition: &previous}, "conflict")
	previous = Point{X: -10000, Y: 10000}
	call(t, s, "measurement.pin.move", MoveParams{ID: "v:C1", Position: Point{X: -5, Y: 5}, PreviousPosition: &previous}, "half-grid")
	if s.Snapshot().PinnedReadings[0].Definition.Position != (Point{X: -10, Y: 10}) {
		t.Fatal("half-grid did not round away from zero")
	}
	previous = Point{X: -10, Y: 10}
	stable := s.Snapshot()
	call(t, s, "measurement.pin.move", MoveParams{ID: "v:C1", Position: Point{X: -9, Y: 9}, PreviousPosition: &previous}, "same-grid")
	after := s.Snapshot()
	if stable.Document != after.Document || !reflect.DeepEqual(stable.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(stable.PinnedReadings, after.PinnedReadings) || !reflect.DeepEqual(before.Transient, after.Transient) || before.RunID != after.RunID {
		t.Fatal("pin move/no-op changed physics or manufactured history")
	}
}
