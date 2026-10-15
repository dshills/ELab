package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func comparisonFixture(t *testing.T) (*Service, string, string) {
	t.Helper()
	s := pausedRC(t)
	duration, step := .02, .001
	_, a := verify(t, s, VerifyParams{Duration: &duration, Step: &step, InputMode: "charge", ProbeIDs: []string{"v:C1", "i:R1"}}, "first")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "resistor")
	step = .002
	_, b := verify(t, s, VerifyParams{Duration: &duration, Step: &step, InputMode: "charge", ProbeIDs: []string{"i:R1", "v:C1"}}, "second")
	return s, a.Operation.ID, b.Operation.ID
}
func comparisonQuery(t *testing.T, s *Service, p any) *RecordingComparison {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "compare-query", Command: "recording.compare", Params: raw}, "agent")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(*RecordingComparison)
}
func TestRecordingCompareExactRCDataPurityAndOwnedViews(t *testing.T) {
	s, a, b := comparisonFixture(t)
	defer s.Close()
	before := s.Snapshot()
	p := RecordingCompareParams{ID: b, ReferenceID: a, Time: .0114}
	result := comparisonQuery(t, s, map[string]any{"id": b, "reference_id": a, "time_s": "11.4 ms"})
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("pure comparison changes public state")
	}
	if !result.Available || len(result.Readings) != 2 || result.A.Time != .012 || math.Abs(result.B.Time-.011) > 1e-15 || result.A.Index != 6 || result.B.Index != 11 {
		t.Fatal("nearest stored rows not reported", result)
	}
	for _, r := range result.Readings {
		if r.StatisticsA.Count != 11 || r.StatisticsB.Count != 21 || r.Delta != r.B-r.A {
			t.Fatal("statistics/delta wrong", r)
		}
		if r.Probe.ID == "v:C1" {
			av, bv := 5*(1-math.Exp(-.012/.02)), 5*(1-math.Exp(-.011/.01))
			if math.Abs(r.A-av) > 1e-12 || math.Abs(r.B-bv) > 1e-12 {
				t.Fatal("RC comparison differs from analytic charging", r)
			}
		}
	}
	call(t, s, "recording.compare.set", p, "configure")
	view := s.Snapshot().Recording.Comparison
	if !reflect.DeepEqual(result, view) {
		t.Fatal("query and configured view differ")
	}
	after := s.Snapshot()
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Transient.Sample != after.Transient.Sample || before.Document != after.Document || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("comparison altered experiment/history")
	}
	view.Readings[0].A = 999
	view.UnmatchedA = append(view.UnmatchedA, Probe{ID: "bad"})
	result.Readings[0].Probe.Label = "bad"
	fresh := comparisonQuery(t, s, p)
	if fresh.Readings[0].A == 999 || fresh.Readings[0].Probe.Label == "bad" || len(s.Snapshot().Recording.Comparison.UnmatchedA) != 0 {
		t.Fatal("comparison responses alias retained state")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "30k"}, "later-live-edit")
	if !reflect.DeepEqual(fresh, comparisonQuery(t, s, p)) {
		t.Fatal("comparison substituted later live inputs")
	}
	p.ShowStatistics = true
	call(t, s, "recording.compare.set", p, "statistics")
	if !s.Snapshot().Recording.Comparison.Settings.ShowStatistics {
		t.Fatal("statistics preference not shared")
	}
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: b}, "clear")
	if s.Snapshot().Recording.Comparison != nil {
		t.Fatal("clear failed")
	}
}
func TestRecordingCompareRefusalsAndSelectionGuard(t *testing.T) {
	s, a, b := comparisonFixture(t)
	defer s.Close()
	p := RecordingCompareParams{ID: b, ReferenceID: a, Time: .01}
	call(t, s, "recording.compare.set", p, "initial")
	for _, test := range []struct {
		command string
		p       any
		code    string
	}{
		{"recording.compare", RecordingCompareParams{ID: a, ReferenceID: a}, "invalid-params"},
		{"recording.compare", RecordingCompareParams{ID: a, ReferenceID: b, Time: .021}, "invalid-params"},
		{"recording.compare", RecordingCompareParams{ID: a, ReferenceID: b, Time: -1}, "invalid-params"},
		{"recording.compare", RecordingCompareParams{ID: a, ReferenceID: "missing"}, "not-ready"},
		{"recording.compare", map[string]any{"id": a, "reference_id": b}, "invalid-params"},
		{"recording.compare", map[string]any{"id": a, "reference_id": b, "time_s": "10 Hz"}, "invalid-params"},
		{"recording.compare.set", RecordingCompareParams{ID: a, ReferenceID: b}, "conflict"},
		{"recording.compare.set", RecordingCompareParams{ID: b, ReferenceID: a, Time: .03}, "invalid-params"},
	} {
		before := s.Snapshot()
		refuse(t, s, test.command, test.p, test.code)
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("failed comparison mutates state", test.command, test.p)
		}
	}
	call(t, s, "operation.select", IDParams{ID: a}, "select")
	if s.Snapshot().Recording.Comparison != nil {
		t.Fatal("selection retained comparison of old A")
	}
}
func TestRecordingCompareProbeMatchingEdgeTiesAndEviction(t *testing.T) {
	s, a, b := comparisonFixture(t)
	defer s.Close()
	s.mu.Lock()
	// A controlled record isolates identity/tie behavior independently of solver
	// output-grid conventions. Real imports still undergo strict model validation.
	ar, br := s.operations[a].record, s.operations[b].record
	ar.rows = []VerificationRow{{Time: 0, Values: []float64{0, 0}}, {Time: .01, Values: []float64{1, 2}}, {Time: .01, Values: []float64{3, 4}}, {Time: .02, Values: []float64{5, 6}}}
	br.rows = []VerificationRow{{Time: 0, Values: []float64{10, 20}}, {Time: .02, Values: []float64{30, 40}}}
	br.metadata.Probes[1].Label = "Different display name"
	s.mu.Unlock()
	result := comparisonQuery(t, s, RecordingCompareParams{ID: a, ReferenceID: b, Time: .01})
	if result.A.Index != 2 || result.A.Time != .01 || result.B.Index != 1 || result.B.Time != .02 || len(result.Readings) != 2 {
		t.Fatal("duplicate edge/right-hand nearest tie or label matching changed", result)
	}
	s.mu.Lock()
	br.metadata.Probes[1].Reference = "different.reference"
	s.mu.Unlock()
	result = comparisonQuery(t, s, RecordingCompareParams{ID: a, ReferenceID: b})
	if len(result.Readings) != 1 || len(result.UnmatchedA) != 1 || len(result.UnmatchedB) != 1 {
		t.Fatal("different polarity/reference treated as matching")
	}
	s.mu.Lock()
	br.metadata.Probes[0].Unit = "other"
	s.mu.Unlock()
	result = comparisonQuery(t, s, RecordingCompareParams{ID: a, ReferenceID: b})
	if result.Available || result.Reason == "" {
		t.Fatal("no-match reason missing")
	}
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: b, ReferenceID: a}, "configure")
	s.mu.Lock()
	delete(s.operations, a)
	for i, id := range s.operationOrder {
		if id == a {
			s.operationOrder = append(s.operationOrder[:i], s.operationOrder[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
	view := s.Snapshot().Recording.Comparison
	if view.Available || view.Reason == "" {
		t.Fatal("evicted reference silently reused cached data")
	}
}
func TestRecordingCompareReopenedAndDCRecords(t *testing.T) {
	s, a, b := comparisonFixture(t)
	defer s.Close()
	expected := comparisonQuery(t, s, RecordingCompareParams{ID: b, ReferenceID: a, Time: .01})
	call(t, s, "recording.compare.set", RecordingCompareParams{ID: b, ReferenceID: a, Time: .01}, "view")
	path := filepath.Join(t.TempDir(), "captures.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	loaded := New()
	defer loaded.Close()
	call(t, loaded, "project.load", LoadParams{Path: path}, "open")
	state := loaded.Snapshot()
	if state.Recording.Comparison != nil || len(state.Operations) != 2 {
		t.Fatal("session-only comparison or retained sources lost")
	}
	first, second := state.Operations[1].ID, state.Operations[0].ID
	actual := comparisonQuery(t, loaded, RecordingCompareParams{ID: second, ReferenceID: first, Time: .01})
	if !reflect.DeepEqual(expected.Readings, actual.Readings) {
		t.Fatal("reopened comparison data changed")
	}
	dc := NewExample()
	defer dc.Close()
	settled(t, dc)
	zero := 0.
	_, x := verify(t, dc, VerifyParams{Duration: &zero}, "dc-a")
	call(t, dc, "component.set", ValueParams{ID: "R1", Value: "2k"}, "dc-r")
	_, y := verify(t, dc, VerifyParams{Duration: &zero}, "dc-b")
	result := comparisonQuery(t, dc, RecordingCompareParams{ID: x.Operation.ID, ReferenceID: y.Operation.ID})
	if !result.Available || result.A.Time != 0 || result.B.Time != 0 || result.A.Count != 1 || result.B.Count != 1 {
		t.Fatal("DC comparison unavailable")
	}
}
func TestRecordingCompareOutputTimeSchemaIsNumeric(t *testing.T) {
	var output map[string]any
	if err := json.Unmarshal(SchemaRegistry()["urn:elab:recording-compare-settings:1"], &output); err != nil {
		t.Fatal(err)
	}
	if inputProperty(output, "time_s")["type"] != "number" {
		t.Fatal("command unit grammar leaked into comparison output")
	}
}

func TestRecordingComparePreservesLaterChoiceDuringImport(t *testing.T) {
	path := importFixture(t)
	s, a, b := comparisonFixture(t)
	defer s.Close()
	entered, release := blockRecordingImport(t, s)
	defer release()
	promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	p := RecordingCompareParams{ID: b, ReferenceID: a, Time: .01}
	promptLiveCommand(t, s, "recording.compare.set", p)
	before := s.Snapshot().Recording
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.ViewerSelected == nil || *after.ProjectWork.ViewerSelected || !reflect.DeepEqual(before, after.Recording) {
		t.Fatal("late import replaced newer comparison")
	}
}
func TestRecordingCompareDiscoveryContracts(t *testing.T) {
	for _, command := range []string{"recording.compare", "recording.compare.set"} {
		caps, err := detailedCapabilities(command)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["x-command-contract"] == nil || schema["x-relations"] == nil || inputProperty(schema, "time_s")["x-normalized-si-range"] == nil {
			t.Fatal("comparison contracts missing", command)
		}
	}
}
