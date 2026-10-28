package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func sweepDetailSchema(t *testing.T, id string) map[string]any {
	t.Helper()
	caps, err := detailedCapabilities(id)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}
func TestSweepDetailedDiscoveryCoversFamilyAndKeepsSchemasLocal(t *testing.T) {
	for _, cap := range Catalog() {
		if !strings.HasPrefix(cap.ID, "sweep.") {
			continue
		}
		schema := sweepDetailSchema(t, cap.ID)
		if schema["x-command-contract"] == nil {
			t.Fatal("missing sweep contract", cap.ID)
		}
		if cap.ID == "sweep.inspect" || cap.ID == "sweep.cancel" || cap.ID == "sweep.select" || cap.ID == "sweep.trace.select" {
			required, _ := json.Marshal(schema["required"])
			if !strings.Contains(string(required), `"id"`) {
				t.Fatal("required ID not discoverable", cap.ID)
			}
		}
	}
	start := sweepDetailSchema(t, "sweep.start")
	nested := inputProperty(start, "settings")["anyOf"].([]any)[0].(map[string]any)
	if nested["x-parameter-domains"] == nil || inputProperty(nested, "points")["maximum"] != float64(MaxDCSweepPoints) {
		t.Fatal("nested start settings not detailed")
	}
	// The normal registry remains compact and independent of detailed command annotations.
	var ordinary map[string]any
	_ = json.Unmarshal(SchemaRegistry()["urn:elab:sweep-settings:1"], &ordinary)
	if ordinary["x-parameter-domains"] != nil {
		t.Fatal("detailed input leaked into shared/output schemas")
	}
}
func TestPublishedSweepDomainsAndLimitsMatchAdmission(t *testing.T) {
	schema := sweepDetailSchema(t, "sweep.settings.set")
	domains := schema["x-parameter-domains"].([]any)
	for _, raw := range domains {
		d := raw.(map[string]any)
		parameter := d["parameter"].(string)
		min, max := d["minimum"].(float64), d["maximum"].(float64)
		graph := ExampleGraph()
		target := "V1"
		if parameter == "frequency" {
			graph = ACRCExampleGraph()
		}
		if parameter == "loop-frequency" {
			graph = FeedbackLoopExampleGraph()
			target = "U1"
		}
		if parameter == "temperature" {
			graph = thermalResistorGraph(0, false)
			target = "R1"
		}
		if parameter == "resistance" {
			target = "R1"
		}
		if parameter == "wiper" {
			graph = potGraph(t, "resistor")
			target = "P1"
		}
		cfg := SweepSettings{ComponentID: target, Parameter: parameter, Start: fmt.Sprint(min), Stop: fmt.Sprint(max), Points: 2}
		if _, err := newSweepPlan(graph, cfg); err != nil {
			t.Fatal("published endpoints rejected", parameter, err)
		}
		cfg.Start, cfg.Stop = cfg.Stop, cfg.Start
		if _, err := newSweepPlan(graph, cfg); parameter == "loop-frequency" {
			if err == nil {
				t.Fatal("descending loop band accepted")
			}
		} else if err != nil {
			t.Fatal("published descending range rejected", parameter, err)
		}
		for _, outside := range []float64{math.Nextafter(min, math.Inf(-1)), math.Nextafter(max, math.Inf(1))} {
			bad := cfg
			bad.Start = fmt.Sprintf("%.17g", outside)
			if _, err := newSweepPlan(graph, bad); err == nil {
				t.Fatal("outside advertised domain accepted", parameter, outside)
			}
		}
	}
	graph := ExampleGraph()
	cfg := sweepSettingsForLED()
	points := inputProperty(schema, "points")
	for _, n := range []int{int(points["minimum"].(float64)), int(points["maximum"].(float64))} {
		cfg.Points = n
		if _, err := newSweepPlan(graph, cfg); err != nil {
			t.Fatal("published count rejected", n, err)
		}
	}
	for _, n := range []int{1, MaxDCSweepPoints + 1} {
		cfg.Points = n
		if _, err := newSweepPlan(graph, cfg); err == nil {
			t.Fatal("outside count accepted", n)
		}
	}
	cfg = sweepSettingsForLED()
	cfg.ProbeIDs = []string{}
	if _, err := newSweepPlan(graph, cfg); err == nil {
		t.Fatal("empty probes accepted")
	}
	cfg.ProbeIDs = nil
	if _, err := newSweepPlan(graph, cfg); err != nil {
		t.Fatal("automatic probes rejected", err)
	}
	cfg.ProbeIDs = []string{"i:D1", "i:D1"}
	if _, err := newSweepPlan(graph, cfg); err == nil {
		t.Fatal("duplicate probes accepted")
	}
}
func TestSweepCursorExplicitNullClearsWhileOmissionAndStaleIDRefuse(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	cfg := sweepSettingsForLED()
	call(t, s, "sweep.start", SweepStartParams{Settings: &cfg}, "start")
	awaitSweep(t, s, "completed")
	call(t, s, "sweep.cursor.set", SweepCursorParams{Index: ptrSweepIndex(5)}, "point")
	before := s.Snapshot()
	refuse(t, s, "sweep.cursor.set", map[string]any{}, "invalid-params")
	refuse(t, s, "sweep.cursor.set", map[string]any{"index": 5.5}, "invalid-params")
	refuse(t, s, "sweep.cursor.set", SweepCursorParams{OperationID: "retired"}, "conflict")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("refused cursor changes view or live state")
	}
	response := call(t, s, "sweep.cursor.set", SweepCursorParams{OperationID: before.DCSweep.Selected}, "clear").Result.(Snapshot)
	if response.DCSweep.Cursor != nil || !reflect.DeepEqual(response.DCSweep.Result, before.DCSweep.Result) || !reflect.DeepEqual(response.Components, before.Components) || response.RunID != before.RunID || !reflect.DeepEqual(response.Result, before.Result) || !reflect.DeepEqual(response.UndoEntry, before.UndoEntry) {
		t.Fatal("clear changes captured or live state")
	}
	// Discovery's explicit-null example must execute against a complete selected curve.
	schema := sweepDetailSchema(t, "sweep.cursor.set")
	for i, example := range schema["examples"].([]any) {
		call(t, s, "sweep.cursor.set", example, fmt.Sprint("example", i))
	}
	if s.Snapshot().DCSweep.Cursor != nil {
		t.Fatal("discovered clear example does not clear")
	}
}
func TestSweepDiscoveredErrorCodesMatchRefusals(t *testing.T) {
	for _, tc := range []struct {
		id   string
		p    any
		code string
	}{
		{"sweep.start", SweepStartParams{}, "invalid-params"},
		{"sweep.cancel", IDParams{ID: "missing"}, "not-found"},
		{"sweep.inspect", IDParams{ID: "missing"}, "not-found"},
		{"sweep.select", IDParams{ID: "missing"}, "not-found"},
		{"sweep.cursor.set", SweepCursorParams{}, "not-ready"},
		{"sweep.trace.select", IDParams{ID: "i:D1"}, "unavailable"},
		{"sweep.keep.set", RecordingKeepParams{ID: "missing", Keep: true}, "not-ready"},
	} {
		s := New()
		schema := sweepDetailSchema(t, tc.id)
		contract := schema["x-command-contract"].(map[string]any)
		found := false
		for _, entry := range contract["errors"].([]any) {
			if entry.(map[string]any)["code"] == tc.code {
				found = true
			}
		}
		if !found {
			t.Fatal("expected refusal not advertised", tc.id, tc.code)
		}
		refuse(t, s, tc.id, tc.p, tc.code)
		s.Close()
	}
}
