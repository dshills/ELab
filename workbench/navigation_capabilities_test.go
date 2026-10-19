package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func navigationSchema(t *testing.T, command string) map[string]any {
	t.Helper()
	caps, err := detailedCapabilities(command)
	if err != nil || len(caps) != 1 {
		t.Fatal(command, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func TestNavigationDiscoveryOwnedPureComplete(t *testing.T) {
	s := New()
	defer s.Close()
	awaitTestInitialization(t, s)
	before := s.Snapshot()
	for _, command := range []string{"selection.navigate", "view.selection.center", "view.pan", "view.zoom"} {
		t.Run(command, func(t *testing.T) {
			raw, _ := json.Marshal(CapabilityQuery{Command: command})
			response := s.Dispatch(t.Context(), Request{Version: 1, ID: command, Command: "capabilities.list", Params: raw}, "agent")
			if response.Error != nil {
				t.Fatal(response.Error)
			}
			caps := response.Result.(map[string]any)["capabilities"].([]Capability)
			var schema map[string]any
			if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
				t.Fatal(err)
			}
			contract, ok := schema["x-command-contract"].(map[string]any)
			if !ok || contract["scope"] != "schematic-navigation" || contract["completion"] == nil || len(contract["effects"].([]any)) == 0 || len(contract["errors"].([]any)) == 0 {
				t.Fatal("missing navigation contract", command)
			}
			caps[0].Input[0] = '!'
			if fresh := navigationSchema(t, command); fresh["x-command-contract"] == nil {
				t.Fatal("borrowed discovery")
			}
		})
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("discovery changed public state")
	}
}

func TestNavigationDiscoveredSelectionAndDefaultSemantics(t *testing.T) {
	schema := navigationSchema(t, "selection.navigate")
	directions, ok := inputProperty(schema, "direction")["enum"].([]any)
	if !ok {
		t.Fatal("direction choices absent")
	}
	kinds, ok := inputProperty(schema, "kind")["enum"].([]any)
	if !ok {
		t.Fatal("kind choices absent")
	}
	if inputProperty(schema, "center")["default"] != false {
		t.Fatal("centering default absent")
	}
	for _, direction := range directions {
		for _, kind := range kinds {
			t.Run(fmt.Sprint(direction, "/", kind), func(t *testing.T) {
				s := NewExample()
				defer s.Close()
				call(t, s, "simulation.pause", struct{}{}, "pause")
				call(t, s, "component.select", IDParams{}, "clear")
				before := s.Snapshot()
				ids := []string{}
				if kind != "wires" {
					for _, p := range before.Components {
						ids = append(ids, p.ID)
					}
				}
				if kind != "components" {
					for _, w := range before.Wires {
						ids = append(ids, w.ID)
					}
				}
				call(t, s, "selection.navigate", map[string]any{"direction": direction, "kind": kind, "center": nil}, "navigate")
				after := s.Snapshot()
				want := ids[0]
				if direction == "previous" || direction == "last" {
					want = ids[len(ids)-1]
				}
				if after.Selected != want || after.View != before.View || after.Viewport != before.Viewport || after.Drawing.Tool != "select" || after.Drawing.WireStart != "" {
					t.Fatal("published selection/default behavior differs", after.Selected, want)
				}
				call(t, s, "selection.navigate", map[string]any{"direction": direction, "kind": kind, "center": true}, "center")
				if s.Snapshot().Viewport.Following {
					t.Fatal("centered selection kept following")
				}
			})
		}
	}
}

func TestNavigationDiscoveredPanAndZoomBounds(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	pan := navigationSchema(t, "view.pan")
	bound := inputProperty(pan, "dx")["maximum"].(float64)
	if bound != 2e6 || inputProperty(pan, "dy")["minimum"] != -bound || pan["x-resulting-center-domain"] == nil {
		t.Fatal("pan needs relative and resulting bounds")
	}
	call(t, s, "view.pan", map[string]float64{"dx": -1e6 - s.Snapshot().View.Center.X, "dy": -s.Snapshot().View.Center.Y}, "left")
	call(t, s, "view.pan", map[string]float64{"dx": bound, "dy": 0}, "right")
	if s.Snapshot().View.Center.X != 1e6 {
		t.Fatal("published delta boundary refused")
	}
	refuse(t, s, "view.pan", map[string]float64{"dx": math.Nextafter(1e6, math.Inf(1)) - 1e6, "dy": 0}, "invalid-params")
	zoom := navigationSchema(t, "view.zoom")
	factor := inputProperty(zoom, "factor")
	for _, value := range []float64{factor["minimum"].(float64), factor["maximum"].(float64)} {
		call(t, s, "view.zoom", map[string]any{"factor": value, "anchor": nil}, fmt.Sprint("zoom", value))
	}
	for _, value := range []float64{math.Nextafter(factor["minimum"].(float64), 0), math.Nextafter(factor["maximum"].(float64), math.Inf(1))} {
		refuse(t, s, "view.zoom", map[string]any{"factor": value}, "invalid-params")
	}
	for _, axis := range []string{"x", "y"} {
		if p := inputProperty(zoom, "anchor."+axis); p["minimum"] != float64(-1e6) || p["maximum"] != float64(1e6) {
			t.Fatal("anchor domain absent")
		}
	}
	call(t, s, "view.zoom", map[string]any{"factor": 1., "anchor": Point{X: 1e6, Y: -1e6}}, "anchor-boundary")
	refuse(t, s, "view.zoom", map[string]any{"factor": 1., "anchor": Point{X: math.Nextafter(1e6, math.Inf(1))}}, "invalid-params")
	call(t, s, "view.pan", map[string]float64{"dx": -2e6, "dy": 0}, "left-again")
	refuse(t, s, "view.zoom", map[string]any{"factor": .25, "anchor": Point{X: 1e6}}, "invalid-params")
}

func TestNavigationContractsPreserveExperimentAndDescribeRefusals(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.rc", ReplaceParams{}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "step")
	call(t, s, "component.select", IDParams{ID: "R1"}, "select")
	before := s.Snapshot()
	for _, request := range []struct {
		command string
		params  any
	}{
		{"view.pan", map[string]float64{"dx": 40, "dy": -20}}, {"view.zoom", map[string]any{"factor": 2., "anchor": Point{}}},
		{"view.selection.center", CenterSelectionParams{ID: "R1"}}, {"selection.navigate", NavigateParams{Direction: "last", Center: true}},
	} {
		call(t, s, request.command, request.params, request.command)
	}
	after := s.Snapshot()
	for _, pair := range [][2]any{{before.Components, after.Components}, {before.Wires, after.Wires}, {before.Transient, after.Transient}, {before.Mixed, after.Mixed}, {before.Document, after.Document}, {before.UndoEntry, after.UndoEntry}, {before.RedoEntry, after.RedoEntry}, {before.RunID, after.RunID}, {before.ElectricalRevision, after.ElectricalRevision}} {
		if !reflect.DeepEqual(pair[0], pair[1]) {
			t.Fatal("navigation changed protected experiment")
		}
	}
	for _, tc := range []struct {
		command, code string
		params        any
	}{
		{"selection.navigate", "invalid-params", NavigateParams{Direction: "sideways"}},
		{"selection.navigate", "invalid-params", NavigateParams{Direction: "next", Kind: "resistors"}},
		{"view.selection.center", "conflict", CenterSelectionParams{ID: "absent"}},
		{"view.pan", "invalid-params", map[string]any{"dx": 0}},
		{"view.zoom", "invalid-params", map[string]any{"factor": 1., "anchor": map[string]any{"x": 0}}},
	} {
		contract := navigationSchema(t, tc.command)["x-command-contract"].(map[string]any)
		known := false
		for _, e := range contract["errors"].([]any) {
			if e.(map[string]any)["code"] == tc.code {
				known = true
			}
		}
		if !known {
			t.Fatal("runtime refusal missing from discovery", tc)
		}
		refuse(t, s, tc.command, tc.params, tc.code)
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	for _, command := range []string{"selection.navigate", "view.selection.center"} {
		params := map[string]any{}
		if command == "selection.navigate" {
			params["direction"] = "next"
		}
		refuse(t, s, command, params, "unavailable")
	}
}
