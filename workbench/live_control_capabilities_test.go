package workbench

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

var liveContractCommands = []string{"simulation.transient.set", "simulation.input.set", "simulation.pause", "simulation.resume", "simulation.restart", "simulation.step", "simulation.step.start", "simulation.work.cancel", "simulation.wait"}

func TestLiveControlContractsArePureAndTyped(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	for _, id := range liveContractCommands {
		raw, _ := json.Marshal(CapabilityQuery{Command: id})
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: "live-discovery", Command: "capabilities.list", Params: raw}, "agent")
		if response.Error != nil {
			t.Fatal(response.Error)
		}
		var schema map[string]any
		json.Unmarshal(response.Result.(map[string]any)["capabilities"].([]Capability)[0].Input, &schema)
		contract := schema["x-command-contract"].(map[string]any)
		if contract["scope"] != "live-simulation-controls" || contract["completion"] == nil {
			t.Fatal("missing live control contract", id)
		}
		var decoded map[string]any
		if err := json.Unmarshal(catalogInputForLiveTest(t, id), &decoded); err != nil {
			t.Fatal(err)
		}
		// Detailed semantics stay out of ordinary discovery and numeric output types.
		if decoded["x-command-contract"] != nil {
			t.Fatal("ordinary catalog inflated", id)
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("discovery changed live state")
	}
	schema := sweepDetailSchema(t, "simulation.input.set")
	if schema["x-input-mode-inventory"] == nil || len(inputProperty(schema, "mode")["enum"].([]any)) != 5 {
		t.Fatal("mode inventory omitted")
	}
	schema = sweepDetailSchema(t, "simulation.step.start")
	duration := inputProperty(schema, "duration_s")
	if duration["$ref"] != TimeInputSchemaID || schema["x-max-output-steps"] != float64(MaxManualStepSamples) {
		t.Fatal("step units/limit missing")
	}
	var output map[string]any
	json.Unmarshal(SchemaRegistry()[SnapshotSchemaID], &output)
	if inputProperty(output, "transient_settings.step_s")["type"] != "number" || inputProperty(output, "live_input_modes")["items"].(map[string]any)["type"] != "string" {
		t.Fatal("output units/modes widened")
	}
}

func catalogInputForLiveTest(t *testing.T, id string) json.RawMessage {
	t.Helper()
	for _, cap := range Catalog() {
		if cap.ID == id {
			return cap.Input
		}
	}
	t.Fatal("missing command", id)
	return nil
}

func TestLiveModeInventoryMatchesAdmissionAcrossCircuits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		graph Graph
		want  []string
	}{
		{"rc", RCExampleGraph(), []string{"auto", "charge", "discharge"}},
		{"npn", NPNPulseExampleGraph(), []string{"auto", "high", "low"}},
		{"flasher", FlasherExampleGraph(), []string{"auto", "high", "low"}},
		{"network-rc", switchRCGraph(t, true), []string{"auto", "high", "low"}},
		{"blank", EmptyGraph(), []string{}},
		{"led-dc", ExampleGraph(), []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newGraphService(SolveLED, tc.graph)
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "simulation.restart", struct{}{}, "restart")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, err := s.Wait(ctx)
			cancel()
			if err != nil {
				t.Fatal(err)
			}
			before := s.Snapshot()
			if !reflect.DeepEqual(before.LiveInputModes, tc.want) {
				t.Fatal("inventory", before.Analysis, before.LiveInputModes, tc.want)
			}
			for _, mode := range before.LiveInputModes {
				after := call(t, s, "simulation.input.set", InputParams{Mode: mode}, "mode-"+mode).Result.(Snapshot)
				if after.Transient.PendingMode != mode || !reflect.DeepEqual(before.Transient.Sample, after.Transient.Sample) || !reflect.DeepEqual(before.Components, after.Components) || before.Document != after.Document || before.UndoName != after.UndoName {
					t.Fatal("paused input did not queue without physical/source change", mode)
				}
			}
			prior := s.Snapshot()
			code := "invalid-params"
			if len(tc.want) == 0 {
				code = "unavailable"
			}
			refuse(t, s, "simulation.input.set", InputParams{Mode: "invalid"}, code)
			if !reflect.DeepEqual(prior, s.Snapshot()) {
				t.Fatal("rejection changed state")
			}
		})
	}
}

func TestLiveInputInventoryOwnsSnapshotAndRetry(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	response := call(t, s, "simulation.pause", struct{}{}, "inventory-retry")
	response.Result.(Snapshot).LiveInputModes[0] = "corrupt"
	response = call(t, s, "simulation.pause", struct{}{}, "inventory-retry")
	if !response.Duplicate || response.Result.(Snapshot).LiveInputModes[0] != "auto" || s.Snapshot().LiveInputModes[0] != "auto" {
		t.Fatal("inventory aliases retry state")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if len(s.Snapshot().LiveInputModes) != 0 {
		t.Fatal("incomplete graph kept live modes")
	}
}

func TestPublishedStepAndSettingsBoundsMatchRuntime(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	cfg := DefaultTransientSettings()
	cfg.StepSeconds = 1e-6
	cfg.PlaybackSpeed = .001
	call(t, s, "simulation.transient.set", cfg, "micro-step")
	if !s.Snapshot().Transient.Stale || len(s.Snapshot().LiveInputModes) != 3 {
		t.Fatal("settings should retain analysis inventory but stale the paused run")
	}
	refuse(t, s, "simulation.input.set", InputParams{Mode: "auto"}, "unavailable")
	call(t, s, "simulation.restart", struct{}{}, "fresh-micro")
	// All ranges pass, but the independent output-step relation refuses 1001.
	refuse(t, s, "simulation.step.start", StepParams{Duration: .001001}, "invalid-params")
	refuse(t, s, "simulation.step.start", StepParams{Duration: math.Nextafter(.001, math.Inf(1))}, "invalid-params")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "max-steps")
	if math.Abs(s.Snapshot().Transient.Sample.Time-.001) > 1e-15 || s.Snapshot().StepWork.CompletedSamples != MaxManualStepSamples {
		t.Fatal("published 1000-step boundary disagrees with runtime")
	}
	cfg.PlaybackSpeed = math.Nextafter(.001, math.Inf(1))
	refuse(t, s, "simulation.transient.set", cfg, "invalid-params")
	cfg = DefaultTransientSettings()
	cfg.StepSeconds = .1
	cfg.HighSeconds = .05
	refuse(t, s, "simulation.transient.set", cfg, "invalid-params")
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "blank-settings")
	call(t, s, "simulation.transient.set", cfg, "no-ideal-pulse")
	if s.Snapshot().TransientSettings != cfg {
		t.Fatal("legacy phase relation incorrectly applied without ideal pulse")
	}
}

func TestLiveModeInventoryClearsWhenRecognizedTopologyIsIncomplete(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	call(t, s, "object.delete", IDParams{ID: before.Wires[0].ID}, "break-rc")
	state := s.Snapshot()
	if len(state.LiveInputModes) != 0 || state.Analysis != "RC transient" {
		t.Fatal("incomplete but recognized RC should expose no available live input choices", state.Analysis, state.LiveInputModes)
	}
	refuse(t, s, "simulation.input.set", InputParams{Mode: "charge"}, "unavailable")
	call(t, s, "edit.undo", HistoryParams{}, "restore-wire")
	if len(s.Snapshot().LiveInputModes) != 3 {
		t.Fatal("restored complete graph lost its mode inventory")
	}
}

func TestManualStepAtLaterTimeKeepsOutputSampleCap(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.input.set", InputParams{Mode: "charge"}, "charge")
	// Controlled clock origin isolates timestamp arithmetic, not a thousand-second
	// physical-history claim. The paused RC state and requested duration are valid.
	s.mu.Lock()
	s.transient.Sample.Time = 1000
	s.transient.Samples = []TraceSample{s.transient.Sample}
	s.mu.Unlock()
	after := call(t, s, "simulation.step", StepParams{Duration: 1}, "later-time-step").Result.(Snapshot)
	if after.Transient.Sample.Time != 1001 || after.StepWork.CompletedSamples != 1000 || after.StepWork.TotalSamples != 1000 {
		t.Fatal("timestamp accumulation exceeded the 1000-output-step contract", after.StepWork, after.Transient.Sample.Time)
	}
}
