package workbench

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestTimeQuantityUnitsAndNumericRange(t *testing.T) {
	for _, item := range []struct {
		text    string
		seconds float64
	}{
		{"10 ms", .01}, {".01s", .01}, {"  1e1 ms  ", .01}, {"10000us", .01},
		{"250 µs", .00025}, {"250 μs", .00025}, {"250 us", .00025},
		{"5 ns", 5e-9}, {"5 ps", 5e-12}, {"+1", 1}, {"-2 ms", -.002}, {"0e-999 s", 0},
	} {
		got, err := ParseTimeSeconds(item.text)
		if err != nil || got != item.seconds {
			t.Fatalf("%q: %.17g, %v", item.text, got, err)
		}
	}
	for _, text := range []string{"", "ms", "NaN", "Inf", "1e309 s", "1e-400 s", "1e-320 ps", "1 MS", "10 Hz", "1m", "1 ms + 2 ms", "1,5 ms", "0x1p-1 s", strings.Repeat("0", 129)} {
		if _, err := ParseTimeSeconds(text); err == nil {
			t.Fatalf("accepted invalid time %q", text)
		}
	}
}
func TestTimeCommandsShareBoundsRetryAndPhysics(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.transient.set", map[string]any{"step_s": "1 ms", "high_s": "50 ms", "low_s": ".05 s", "initial_voltage_v": 0, "seconds_per_wall_second": .02}, "settings")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	settled(t, s)
	before := s.Snapshot()
	raw := json.RawMessage(`{"duration_s":"10 ms"}`)
	rev := before.CommandRevision
	request := Request{Version: 1, ID: "step", Command: "simulation.step", Params: raw, ExpectedCommandRevision: &rev, Key: "same-duration"}
	response := s.Dispatch(context.Background(), request, "test")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	state := response.Result.(Snapshot)
	if state.Transient.Sample.Time != .01 || math.Abs(state.Transient.Sample.Voltage-3.160602794142788) > 1e-12 {
		t.Fatal("unit text changes accepted physical sample")
	}
	request.Params = json.RawMessage(`{"duration_s":0.01}`)
	retry := s.Dispatch(context.Background(), request, "test")
	if retry.Error != nil || !retry.Duplicate || s.Snapshot().Transient.Sample.Time != .01 {
		t.Fatal("equivalent numeric retry advances twice or loses intent")
	}
	call(t, s, "scope.cursor.set", map[string]any{"cursor": "a", "time_s": "10 ms"}, "cursor")
	if s.Snapshot().Scope.A.Time != .01 {
		t.Fatal("unit cursor fails to select accepted sample")
	}
	for _, p := range []map[string]any{{"cursor": "a", "time_s": "11 ms"}, {"cursor": "a", "time_s": "10 Hz"}, {"cursor": "a", "time_s": "-1 ms"}} {
		refuse(t, s, "scope.cursor.set", p, "invalid-params")
	}

}
func TestNestedPulseTimeAndStrictStoredDocuments(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{ID: "Pulse", Kind: "finite-pulse-source", Position: Point{}}, "place")
	call(t, s, "source.pulse.set", map[string]any{"id": "Pulse", "waveform": map[string]any{"high_v": 5, "low_v": 0, "high_s": "50 ms", "low_s": ".05 s", "rise_s": "1000 us", "fall_s": "1 ms"}}, "waveform")
	expected := DefaultPulseWaveform()
	if !reflect.DeepEqual(s.Snapshot().Components[0].Pulse, &expected) {
		t.Fatal("nested time fields alter waveform values")
	}
	call(t, s, "edit.batch", map[string]any{"name": "Pulse timing", "edits": []any{map[string]any{"command": "source.pulse.set", "params": map[string]any{"id": "Pulse", "waveform": map[string]any{"high_v": 5, "low_v": 0, "high_s": "100 ms", "low_s": "100 ms", "rise_s": "2 ms", "fall_s": "2 ms"}}}}}, "batch")
	if s.Snapshot().Components[0].Pulse.RiseSeconds != .002 {
		t.Fatal("batch loses shared unit conversion")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if !reflect.DeepEqual(s.Snapshot().Components[0].Pulse, &expected) {
		t.Fatal("timing batch does not undo as one edit")
	}
	s.mu.Lock()
	document := s.documentLocked()
	s.mu.Unlock()
	data, _ := json.Marshal(document)
	var object map[string]any
	_ = json.Unmarshal(data, &object)
	object["transient_settings"].(map[string]any)["step_s"] = "1 ms"
	malformed, _ := json.Marshal(object)
	if _, err := decodeDocument(malformed); err == nil {
		t.Fatal("command convenience relaxed stored numeric settings")
	}
	_ = json.Unmarshal(data, &object)
	components := object["graph"].(map[string]any)["components"].([]any)
	components[0].(map[string]any)["pulse"].(map[string]any)["rise_s"] = "1 ms"
	malformed, _ = json.Marshal(object)
	if _, err := decodeDocument(malformed); err == nil {
		t.Fatal("command convenience relaxed stored waveform timing")
	}
}
func TestTimeDiscoveryChangesOnlyCommandInputs(t *testing.T) {
	var registry map[string]any
	if err := json.Unmarshal(SchemaRegistry()[TimeInputSchemaID], &registry); err != nil {
		t.Fatal(err)
	}
	if registry["maxLength"] != float64(128) {
		t.Fatal("time text budget missing")
	}
	for command, paths := range timeCommandFields {
		cap, ok := capabilityIndex[command]
		if !ok {
			t.Fatalf("missing time command %s", command)
		}
		var schema map[string]any
		_ = json.Unmarshal(cap.Input, &schema)
		for _, path := range paths {
			properties := schema["properties"].(map[string]any)
			names := strings.Split(path, ".")
			if len(names) == 2 {
				properties = properties[names[0]].(map[string]any)["properties"].(map[string]any)
			}
			field := properties[names[len(names)-1]].(map[string]any)
			encoded, _ := json.Marshal(field)
			if !strings.Contains(string(encoded), TimeInputSchemaID) {
				t.Fatalf("%s.%s does not discover unit text", command, path)
			}
		}
	}
	encoded, _ := json.Marshal(typeSchema(reflect.TypeOf(TransientSettings{})))
	if strings.Contains(string(encoded), TimeInputSchemaID) {
		t.Fatal("output/file schema now accepts unit text")
	}
}
