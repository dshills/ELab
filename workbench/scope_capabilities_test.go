package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

var scopeContractCommands = []string{"scope.visibility.set", "scope.probes.set", "scope.probe.remove", "scope.trace.select", "scope.window.set", "scope.cursor.set", "scope.cursor.select", "scope.cursor.clear", "export.scope.csv"}

func TestScopeContractsAreOwnedPureAndKeepNumericOutput(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	for _, id := range scopeContractCommands {
		raw, _ := json.Marshal(CapabilityQuery{Command: id})
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: id, Command: "capabilities.list", Params: raw}, "agent")
		if response.Error != nil {
			t.Fatal(response.Error)
		}
		cap := response.Result.(map[string]any)["capabilities"].([]Capability)[0]
		var schema map[string]any
		json.Unmarshal(cap.Input, &schema)
		if schema["x-command-contract"].(map[string]any)["scope"] != "live-scope-controls-and-csv" || schema["x-max-selected-probes"] != float64(MaxScopeProbes) || schema["x-max-live-history-samples"] != float64(MaxTraceSamples) {
			t.Fatal("missing scope contract", id)
		}
		cap.Input[0] = '!'
		if next, err := detailedCapabilities(id); err != nil || !json.Valid(next[0].Input) {
			t.Fatal("caller owns later discovery", id)
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("scope discovery changed state")
	}
	var out map[string]any
	json.Unmarshal(SchemaRegistry()[SnapshotSchemaID], &out)
	if inputProperty(out, "scope.window.start_s")["type"] != "number" {
		t.Fatal("scope output acquired unit-text input schema")
	}
}

func TestPublishedScopeSelectionAndRemovalContracts(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	schema := sweepDetailSchema(t, "scope.probes.set")
	idsProperty := inputProperty(schema, "ids")
	if idsProperty["maxItems"] != float64(6) || idsProperty["uniqueItems"] != true || idsProperty["minItems"] != float64(0) {
		t.Fatal("selection bounds missing")
	}
	available := s.Snapshot().Scope.Available
	ids := []string{}
	for i := 0; i < 6; i++ {
		ids = append(ids, available[i].ID)
	}
	before := s.Snapshot()
	call(t, s, "scope.probes.set", ProbeParams{IDs: ids}, "six")
	if !reflect.DeepEqual(s.Snapshot().Scope.Selected, ids) {
		t.Fatal("selection order changed")
	}
	refuse(t, s, "scope.probes.set", ProbeParams{IDs: append(append([]string{}, ids...), available[6].ID)}, "invalid-params")
	refuse(t, s, "scope.probes.set", ProbeParams{IDs: []string{ids[0], ids[0]}}, "invalid-params")
	refuse(t, s, "scope.probes.set", ProbeParams{IDs: nil}, "invalid-params")
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"i:C1"}}, "capacitor")
	call(t, s, "scope.trace.select", IDParams{ID: "i:C1"}, "focus")
	viewState := s.Snapshot()
	if before.Document != viewState.Document || before.RunID != viewState.RunID || !reflect.DeepEqual(before.Transient, viewState.Transient) || !reflect.DeepEqual(before.UndoEntry, viewState.UndoEntry) {
		t.Fatal("scope controls changed source/physics/history")
	}
	call(t, s, "object.delete", IDParams{ID: "C1"}, "delete-source")
	state := s.Snapshot()
	if len(state.Scope.Unavailable) != 1 || state.Scope.Unavailable[0].ID != "i:C1" {
		t.Fatal("removed requested channel not discoverable")
	}
	call(t, s, "scope.probe.remove", IDParams{ID: "i:C1"}, "remove-unavailable")
	if len(s.Snapshot().Scope.Unavailable) != 0 || s.Snapshot().TraceFocus != nil {
		t.Fatal("remove retained unavailable channel/focus")
	}
	refuse(t, s, "scope.probe.remove", IDParams{ID: "i:C1"}, "not-found")
}

func TestScopeMembershipToleranceCannotDwarfWindow(t *testing.T) {
	for _, span := range []float64{1, .01, 1e-6, 1e-12, 1e-20} {
		start, end := .4*span, .6*span
		if scopeWindowTimeInside(0, start, end) || scopeWindowTimeInside(span, start, end) || !scopeWindowTimeInside(start, start, end) || !scopeWindowTimeInside(end, start, end) {
			t.Fatal("window dilation", span)
		}
	}
	// Conventional decimal boundaries still tolerate the pre-existing tiny
	// roundoff; import grid/range validation keeps its original helper unchanged.
	if !scopeWindowTimeInside(.020000000000000004, .01, .02) || !scopeTimeInside(.02+5e-13, .02, .02) {
		t.Fatal("ordinary boundary or legacy import policy changed")
	}
}
