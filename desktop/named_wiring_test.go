package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
)

func prepareNamedWiringLED(t *testing.T, service *workbench.Service) {
	t.Helper()
	for _, part := range []workbench.PlaceParams{
		{ID: "V1", Kind: "dc-source", Value: "5V", Position: workbench.Point{X: -200}},
		{ID: "R1", Kind: "resistor", Value: "1k", Position: workbench.Point{Y: -120}},
		{ID: "D1", Kind: "led", Position: workbench.Point{X: 200}},
		{ID: "GND", Kind: "ground", Position: workbench.Point{X: -200, Y: 120}},
	} {
		agentAction(t, service, "component.place", part)
	}
	agentAction(t, service, "drawing.tool.set", workbench.ToolParams{Tool: "wire"})
}

func namedLEDConnections() []workbench.ConnectParams {
	return []workbench.ConnectParams{
		{From: "V1.positive", To: "R1.1"},
		{From: "R1.2", To: "D1.anode"},
		{From: "D1.cathode", To: "GND.reference"},
		{From: "GND.reference", To: "V1.negative"},
	}
}

func assertNamedWiringParity(t *testing.T, actual, expected workbench.Snapshot) {
	t.Helper()
	if !reflect.DeepEqual(actual.Components, expected.Components) || !reflect.DeepEqual(actual.Wires, expected.Wires) || !reflect.DeepEqual(actual.Nets, expected.Nets) || actual.Drawing != expected.Drawing || actual.Selected != expected.Selected {
		t.Fatalf("wiring differs from public agent: actual drawing=%+v wires=%+v expected drawing=%+v wires=%+v", actual.Drawing, actual.Wires, expected.Drawing, expected.Wires)
	}
}

func recordNamedWiringObservation(t *testing.T, name string, state workbench.Snapshot) {
	t.Helper()
	var current *float64
	stale := state.Result == nil || state.Result.Stale
	if !stale {
		value := state.Result.Current
		current = &value
	}
	encoded, err := json.Marshal(map[string]any{"case": name, "components": state.Components, "wires": state.Wires, "nets": state.Nets, "drawing": state.Drawing, "selected": state.Selected, "undo_name": state.UndoName, "result_stale": stale, "current_a": current})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("named-wiring-observation %s", encoded)
}
