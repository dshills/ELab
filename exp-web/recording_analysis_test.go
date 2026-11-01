package main

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestBrowserRecordingAnalysesUseCapturedDataAndRefuseStaleSources(t *testing.T) {
	f := fixture(t)
	f.command("simulation.pause", struct{}{})
	f.command("simulation.restart", struct{}{})
	f.wait()
	a := captureForControls(t, f, .2, []string{"v:C1", "i:R1"})
	f.command("component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
	f.wait()
	b := captureForControls(t, f, .2, []string{"i:R1", "v:C1"})
	f.command("operation.select", workbench.IDParams{ID: a})
	f.command("recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	if saved := f.project("save", "", "Recording analysis fixture", false); saved.code != 200 {
		t.Fatal("could not establish clean document", string(saved.body))
	}
	before := f.state()
	if before.Document.Dirty {
		t.Fatal("analysis fixture must start clean")
	}
	rows := f.command("operation.samples", workbench.VerificationPageParams{ID: a, Limit: 200}).Result
	f.command("recording.compare.set", map[string]any{"id": a, "reference_id": b, "time_s": "11.4 ms", "show_overlay": true, "show_statistics": true})
	r := f.state().Recording
	if r.Comparison == nil || !r.Comparison.Available || len(r.Comparison.Readings) != 2 || r.Comparison.A.ID != a || r.Comparison.B.ID != b || r.ReferenceOverlay == nil || len(r.ReferenceOverlay.Scope.Traces) != 2 {
		t.Fatal("missing captured comparison/overlay", r)
	}
	for _, reading := range r.Comparison.Readings {
		if reading.Delta != reading.B-reading.A {
			t.Fatal("wrong native comparison delta", reading)
		}
	}
	f.command("recording.timing.set", map[string]any{"id": a, "probe_id": "v:C1", "level": "2.5 V", "second_level": "4 V", "direction": "both", "start_s": "0 ms", "end_s": "200 ms", "offset": 0, "limit": 1})
	timing := f.state().Recording.Timing
	if timing == nil || timing.Total < 4 || len(timing.Crossings) != 1 || timing.NextOffset != 1 || timing.Transitions == nil {
		t.Fatal("missing timing page/transitions", timing)
	}
	crossing := timing.Crossings[0]
	if crossing.Before.Value >= 2.5 || crossing.After.Value < 2.5 || crossing.Before.Index >= crossing.After.Index || !(crossing.Before.Time <= .01 && crossing.After.Time >= .005) {
		t.Fatal("threshold bracket differs from native captured rows", crossing)
	}
	next := timing.Settings
	next.Offset = timing.NextOffset
	f.command("recording.timing.set", next)
	if got := f.state().Recording.Timing; got.Settings.Offset != 1 || got.NextOffset != 2 || len(got.Crossings) != 1 || got.Crossings[0].Before.Index <= crossing.Before.Index {
		t.Fatal("crossing pagination did not advance", got)
	}
	f.command("recording.step-response.set", map[string]any{"id": a, "probe_id": "v:C1", "start_s": "0 ms", "end_s": "49 ms", "initial_value": 0, "final_value": 5, "tolerance_fraction": .02, "minimum_hold_s": "5 ms"})
	step := f.state().Recording.StepResponse
	if step == nil || step.Settling == nil || math.Abs(*step.Settling-(-.01*math.Log(.02))) > .001 || step.Overshoot != 0 || step.Held < .005 {
		t.Fatal("step response differs from native first-order capture", step)
	}
	after := f.state()
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || before.Document.Dirty != after.Document.Dirty || !reflect.DeepEqual(rows, f.command("operation.samples", workbench.VerificationPageParams{ID: a, Limit: 200}).Result) {
		t.Fatal("analysis changed the live circuit, dirty state or captured samples")
	}
	f.command("recording.compare.set", workbench.RecordingCompareParams{ID: a})
	f.command("recording.timing.set", workbench.RecordingTimingParams{ID: a, Level: "0", Direction: "both"})
	f.command("recording.step-response.set", workbench.StepResponseParams{ID: a})
	r = f.state().Recording
	if r.Comparison != nil || r.ReferenceOverlay != nil || r.Timing != nil || r.StepResponse != nil {
		t.Fatal("clear commands retained analysis state", r)
	}
	f.command("operation.select", workbench.IDParams{ID: b})
	fresh := f.state()
	params, err := json.Marshal(workbench.RecordingCompareParams{ID: a, ReferenceID: b})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(workbench.Request{Version: 1, ID: token(), Key: token(), Command: "recording.compare.set", Params: params, ExpectedCommandRevision: &fresh.CommandRevision, ExpectedDocumentGeneration: &fresh.Document.Generation})
	if err != nil {
		t.Fatal(err)
	}
	response := f.request("POST", "/api/command", raw, "")
	var reply workbench.Response
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil || reply.Error == nil || reply.Error.Code != "conflict" {
		t.Fatal("stale analysis capture accepted", response.Body.String())
	}
}

func TestBrowserInstrumentModulesAreEmbeddedWithoutStaleCaching(t *testing.T) {
	f := fixture(t)
	for _, path := range []string{"/recording-analysis.mjs", "/spectrum.mjs", "/sweep.mjs", "/instrument-plot.mjs"} {
		response := f.request("GET", path, nil, "")
		if response.Code != 200 || !strings.Contains(response.Header().Get("Content-Type"), "javascript") || response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Body.String(), "export function") {
			t.Errorf("embedded module %s unavailable or cacheable: %d %v", path, response.Code, response.Header())
		}
	}
}
