package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func stepQuery(t *testing.T, s *Service, p any) *StepResponse {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "step-query", Command: "recording.step-response", Params: raw}, "agent")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	out, ok := r.Result.(*StepResponse)
	if !ok || out == nil {
		t.Fatalf("unexpected step response type %T", r.Result)
	}
	return out
}
func TestRecordedStepRCIndependentSettlingPurityAndReopen(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "step-response"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration, step := .02, .00002
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}}, "capture")
	p := StepResponseParams{ID: op.Operation.ID, ProbeID: "v:C1", End: .02, Initial: 0, Final: 5, Tolerance: .02, Hold: .002}
	before := s.Snapshot()
	result := stepQuery(t, s, p)
	want := -.001 * math.Log(.02)
	if result.Settling == nil || math.Abs(*result.Settling-want) > 6e-8 || result.Overshoot != 0 || result.Held < .015 || result.OriginID == "" {
		t.Fatal("independent RC settling", result, want)
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("step query mutated state")
	}
	call(t, s, "recording.step-response.set", p, "measure")
	after := s.Snapshot()
	if !reflect.DeepEqual(after.Recording.StepResponse, result) || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Mixed, after.Mixed) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("step set changed live/document/history")
	}
	*after.Recording.StepResponse.Settling = 999
	if *s.Snapshot().Recording.StepResponse.Settling == 999 {
		t.Fatal("step measurement aliases")
	}
	path := filepath.Join(t.TempDir(), "step.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	settled(t, fresh)
	p.ID = fresh.Snapshot().Recording.ID
	if fresh.Snapshot().Recording.StepResponse != nil || !reflect.DeepEqual(stepQuery(t, fresh, p).Settling, result.Settling) {
		t.Fatal("recomputed captured step differs")
	}
}
func TestRecordedStepOvershootFallingJumpsAndUnsettledReasons(t *testing.T) {
	s, timing := timingFixture(t)
	defer s.Close()
	p := StepResponseParams{ID: timing.ID, ProbeID: timing.ProbeID, End: .02, Initial: 0, Final: 1, Tolerance: .02, Hold: .005}
	set := func(rows []VerificationRow) { s.mu.Lock(); s.operations[p.ID].record.rows = rows; s.mu.Unlock() }
	set([]VerificationRow{{Time: 0, Values: []float64{0}}, {Time: .005, Values: []float64{1.2}}, {Time: .01, Values: []float64{1}}, {Time: .02, Values: []float64{1}}})
	r := stepQuery(t, s, p)
	want := .005 + (.2-.02)/.2*.005
	if math.Abs(r.Overshoot-20) > 1e-12 || r.Settling == nil || math.Abs(*r.Settling-want) > 1e-12 {
		t.Fatal("known triangular overshoot", r, want)
	}
	set([]VerificationRow{{Time: 0, Values: []float64{1}}, {Time: .005, Values: []float64{-.2}}, {Time: .01, Values: []float64{0}}, {Time: .02, Values: []float64{0}}})
	p.Initial, p.Final = 1, 0
	r = stepQuery(t, s, p)
	if math.Abs(r.Overshoot-20) > 1e-12 || r.Settling == nil || math.Abs(*r.Settling-want) > 1e-12 {
		t.Fatal("falling normalization", r)
	}
	p.Initial, p.Final = 0, 1
	set([]VerificationRow{{Time: 0, Values: []float64{0}}, {Time: .005, Values: []float64{0}}, {Time: .005, Values: []float64{1.5}}, {Time: .005, Values: []float64{1}}, {Time: .02, Values: []float64{1}}})
	r = stepQuery(t, s, p)
	if r.Overshoot != 50 || r.Settling == nil || *r.Settling != .005 {
		t.Fatal("instantaneous limits", r)
	}
	p.Hold = .019
	r = stepQuery(t, s, p)
	if r.Settling != nil || r.Reason == "" {
		t.Fatal("short hold fabricated settling")
	}
	set([]VerificationRow{{Time: 0, Values: []float64{0}}, {Time: .02, Values: []float64{.9}}})
	r = stepQuery(t, s, p)
	if r.Settling != nil || r.Reason == "" {
		t.Fatal("unsettled fabricated final")
	}
	bad := p
	bad.Final = bad.Initial
	refuse(t, s, "recording.step-response", bad, "invalid-params")
	bad = p
	bad.End = .021
	refuse(t, s, "recording.step-response", bad, "invalid-params")
	bad = p
	bad.ProbeID = "missing"
	refuse(t, s, "recording.step-response", bad, "not-found")
	set([]VerificationRow{{Time: .001, Values: []float64{0}}, {Time: .02, Values: []float64{1}}})
	refuse(t, s, "recording.step-response", p, "unavailable")
}

func TestRecordedStepSettingsClearSelectionAndUnitSyntax(t *testing.T) {
	s, timing := timingFixture(t)
	defer s.Close()
	p := StepResponseParams{ID: timing.ID, ProbeID: timing.ProbeID, End: .02, Final: 5, Tolerance: .1, Hold: .001}
	call(t, s, "recording.step-response.set", p, "set-step")
	result := stepQuery(t, s, map[string]any{"id": p.ID, "probe_id": p.ProbeID, "start_s": "0 ms", "end_s": "20 ms", "initial_value": 0, "final_value": 5, "tolerance_fraction": .1, "minimum_hold_s": "1 ms"})
	if result.Settings != StepResponseSettings(p) {
		t.Fatal("time syntax changed values", result.Settings)
	}
	call(t, s, "recording.step-response.set", map[string]any{"id": p.ID, "probe_id": p.ProbeID, "start_s": "0 ms", "end_s": "20 ms", "initial_value": 0, "final_value": 5, "tolerance_fraction": .1, "minimum_hold_s": "1 ms"}, "unit-step-set")
	before := s.Snapshot()
	bad := p
	bad.Hold = 0
	refuse(t, s, "recording.step-response.set", bad, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("invalid setting changed measurement")
	}
	call(t, s, "recording.step-response.set", StepResponseParams{ID: p.ID}, "clear-step")
	if s.Snapshot().Recording.StepResponse != nil {
		t.Fatal("measurement not cleared")
	}
	_, op := verify(t, s, VerifyParams{}, "other-record")
	if op.Operation.ID == p.ID {
		t.Fatal("fixture capture identity")
	}
	refuse(t, s, "recording.step-response.set", p, "conflict")
	for _, id := range []string{"recording.step-response", "recording.step-response.set"} {
		caps, err := detailedCapabilities(id)
		if err != nil || len(caps) != 1 {
			t.Fatal("step capability", err, len(caps))
		}
		var schema map[string]any
		if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["x-method-id"] != StepResponseMethod || schema["x-model-id"] != nil || schema["x-command-contract"] == nil {
			t.Fatal("missing step contract", schema)
		}
	}
}
