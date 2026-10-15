package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func statisticsQuery(t *testing.T, s *Service, p any) *RecordingStatistics {
	t.Helper()
	raw, _ := json.Marshal(p)
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "statistics-query", Command: "recording.statistics", Params: raw}, "agent")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(*RecordingStatistics)
}

func nearStatistic(t *testing.T, got, want, tolerance float64) {
	t.Helper()
	if !finite(got) || math.Abs(got-want) > tolerance {
		t.Fatalf("got %.17g, want %.17g ± %.3g", got, want, tolerance)
	}
}

func TestRecordingStatisticsClippedNonuniformLinearSegmentsAndEdges(t *testing.T) {
	s, timing := timingFixture(t)
	defer s.Close()
	p := RecordingStatisticsParams{ID: timing.ID, ProbeID: timing.ProbeID, Start: .002, End: .017}
	// Controlled numerical fixture, not an imported/solver waveform claim.
	record := s.operations[p.ID].record
	record.rows = []VerificationRow{{Time: 0, Values: []float64{1}}, {Time: .003, Values: []float64{1.6}}, {Time: .02, Values: []float64{5}}}
	r := statisticsQuery(t, s, p)
	a, b := 1.4, 4.4
	nearStatistic(t, r.Mean, (a+b)/2, 1e-14)
	nearStatistic(t, r.RMS, math.Sqrt((a*a+a*b+b*b)/3), 1e-14)
	nearStatistic(t, r.ACRMS, (b-a)/math.Sqrt(12), 1e-14)
	nearStatistic(t, r.Minimum, a, 1e-14)
	nearStatistic(t, r.Maximum, b, 1e-14)
	if r.Segments != 2 || r.SampleCount != 3 || r.Method != "piecewise-linear.v1" || r.Duration != p.End-p.Start {
		t.Fatal("interval provenance", r)
	}
	record.rows = []VerificationRow{{Time: 0, Values: []float64{0}}, {Time: .005, Values: []float64{0}}, {Time: .005, Values: []float64{10}}, {Time: .02, Values: []float64{10}}}
	p.Start, p.End = 0, .02
	r = statisticsQuery(t, s, p)
	nearStatistic(t, r.Mean, 7.5, 1e-14)
	nearStatistic(t, r.RMS, math.Sqrt(75), 1e-14)
	nearStatistic(t, r.ACRMS, math.Sqrt(18.75), 1e-14)
	if r.Segments != 2 || r.PeakToPeak != 10 {
		t.Fatal("duplicate edge was weighted as a ramp or sample", r)
	}
	p.Start, p.End = .005, .01
	r = statisticsQuery(t, s, p)
	if r.Mean != 10 || r.RMS != 10 || r.ACRMS != 0 || r.PeakToPeak != 0 {
		t.Fatal("boundary did not use right-hand positive-time segment", r)
	}
	p.Start, p.End = 0, .005
	r = statisticsQuery(t, s, p)
	if r.Mean != 0 || r.RMS != 0 || r.ACRMS != 0 || r.PeakToPeak != 0 {
		t.Fatal("isolated end edge changed interval statistics", r)
	}
}

func TestRecordingStatisticsDynamicRangeAndSmallRipple(t *testing.T) {
	s, timing := timingFixture(t)
	defer s.Close()
	p := RecordingStatisticsParams{ID: timing.ID, ProbeID: timing.ProbeID, End: .02}
	record := s.operations[p.ID].record
	for _, test := range []struct{ a, b, mean, rms, ac float64 }{
		{1e100, 1e100, 1e100, 1e100, 0},
		{-1e300, 1e300, 0, 1e300 / math.Sqrt(3), 1e300 / math.Sqrt(3)},
		{1e8, 1e8 + 1, 1e8 + .5, math.Sqrt(1e16 + 1e8 + 1./3), 1 / math.Sqrt(12)},
	} {
		record.rows = []VerificationRow{{Time: 0, Values: []float64{test.a}}, {Time: .02, Values: []float64{test.b}}}
		r := statisticsQuery(t, s, p)
		nearStatistic(t, r.Mean, test.mean, 2e-15*math.Max(1, math.Abs(test.mean)))
		nearStatistic(t, r.RMS, test.rms, 2e-15*test.rms)
		nearStatistic(t, r.ACRMS, test.ac, 2e-15*math.Max(1, test.ac))
	}
	record.rows = []VerificationRow{{Time: 0, Values: []float64{-1e308}}, {Time: .02, Values: []float64{1e308}}}
	if _, err := s.measureRecordingStatisticsLocked(p); err == nil {
		t.Fatal("nonfinite peak-to-peak was published")
	}
}

func TestRecordingStatisticsAnalyticRCAndIdealPulse(t *testing.T) {
	s, timing := timingFixture(t)
	defer s.Close()
	p := RecordingStatisticsParams{ID: timing.ID, ProbeID: timing.ProbeID, Start: .0015, End: .0195}
	r := statisticsQuery(t, s, p)
	tau, span := .01, p.End-p.Start
	e1 := math.Exp(-p.Start/tau) - math.Exp(-p.End/tau)
	e2 := math.Exp(-2*p.Start/tau) - math.Exp(-2*p.End/tau)
	mean := 5 * (1 - tau/span*e1)
	rms := 5 * math.Sqrt(1-2*tau/span*e1+tau/(2*span)*e2)
	// The analytic exponential is distinct from the saved linear segments;
	// the interpolation error is bounded here at the existing 1 ms sample step.
	nearStatistic(t, r.Mean, mean, .003)
	nearStatistic(t, r.RMS, rms, .003)
	nearStatistic(t, r.ACRMS, math.Sqrt(rms*rms-mean*mean), .003)
	other := pausedNPNPulse(t)
	defer other.Close()
	duration := .1
	_, view := verify(t, other, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}}, "statistics-pulse")
	p = RecordingStatisticsParams{ID: view.Operation.ID, ProbeID: "v:VIN.positive", End: duration}
	r = statisticsQuery(t, other, p)
	nearStatistic(t, r.Mean, 2.5, 1e-12)
	nearStatistic(t, r.RMS, 5/math.Sqrt(2), 1e-12)
	nearStatistic(t, r.ACRMS, 2.5, 1e-12)
	if r.PeakToPeak != 5 || r.Segments >= r.SampleCount-1 {
		t.Fatal("pulse edge rows not distinguished from positive-duration segments", r)
	}
}

func TestRecordingStatisticsPurityOwnershipGuardsAndReopen(t *testing.T) {
	s, timing := timingFixture(t)
	defer s.Close()
	p := RecordingStatisticsParams{ID: timing.ID, ProbeID: timing.ProbeID, End: .02}
	before := s.Snapshot()
	r := statisticsQuery(t, s, map[string]any{"id": p.ID, "probe_id": p.ProbeID, "start_s": "0 ms", "end_s": "20 ms"})
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("query mutated public state")
	}
	call(t, s, "recording.statistics.set", p, "interval-set")
	after := s.Snapshot()
	if !reflect.DeepEqual(after.Recording.Statistics, r) || before.Document != after.Document || before.RunID != after.RunID || before.Transient.Sample != after.Transient.Sample || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("configured result differs or changed live/document/history")
	}
	r.Mean = 999
	after.Recording.Statistics.Minimum = 999
	if s.Snapshot().Recording.Statistics.Minimum == 999 || statisticsQuery(t, s, p).Mean == 999 {
		t.Fatal("result aliases cached state")
	}
	for _, bad := range []any{
		RecordingStatisticsParams{ID: p.ID, ProbeID: p.ProbeID, End: .021},
		RecordingStatisticsParams{ID: p.ID, ProbeID: p.ProbeID, Start: .01, End: .01},
		RecordingStatisticsParams{ID: p.ID, ProbeID: p.ProbeID, Start: -.01, End: .01},
		map[string]any{"id": p.ID, "probe_id": p.ProbeID, "start_s": nil, "end_s": .02},
		map[string]any{"id": p.ID, "probe_id": p.ProbeID, "start_s": 0, "end_s": "20 Hz"},
	} {
		refuse(t, s, "recording.statistics", bad, "invalid-params")
		refuse(t, s, "recording.statistics.set", bad, "invalid-params")
	}
	refuse(t, s, "recording.statistics", RecordingStatisticsParams{ID: "missing", ProbeID: p.ProbeID, End: .02}, "not-ready")
	refuse(t, s, "recording.statistics", RecordingStatisticsParams{ID: p.ID, ProbeID: "v:missing", End: .02}, "not-found")
	refuse(t, s, "recording.statistics.set", RecordingStatisticsParams{ID: p.ID, End: .02}, "invalid-params")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "later-edit")
	retained := s.Snapshot().Recording.Statistics
	if !reflect.DeepEqual(retained, statisticsQuery(t, s, p)) {
		t.Fatal("measurements substituted later circuit")
	}
	beforeRetry := s.Snapshot()
	if !call(t, s, "recording.statistics.set", p, "interval-set").Duplicate || !reflect.DeepEqual(beforeRetry, s.Snapshot()) {
		t.Fatal("retry did not preserve original admission")
	}
	call(t, s, "recording.window.set", RecordingWindowParams{Start: .01, End: .02}, "separate-plot")
	if !reflect.DeepEqual(retained, s.Snapshot().Recording.Statistics) {
		t.Fatal("plot interval changed measurement interval")
	}
	path := filepath.Join(t.TempDir(), "statistics.elab")
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: p.ID, Keep: true}, "keep-statistics-source")
	call(t, s, "project.save", SaveParams{Path: path}, "save-statistics")
	loaded := New()
	defer loaded.Close()
	call(t, loaded, "project.load", LoadParams{Path: path}, "load-statistics")
	if loaded.Snapshot().Recording.Statistics != nil {
		t.Fatal("session measurement persisted")
	}
	loadedID := loaded.Snapshot().Recording.ID
	p.ID = loadedID
	got := statisticsQuery(t, loaded, p)
	nearStatistic(t, got.Mean, retained.Mean, 0)
	call(t, s, "operation.select", IDParams{ID: ""}, "clear-selection")
	if s.Snapshot().Recording != nil {
		t.Fatal("clearing selected capture retained viewer")
	}
	refuse(t, s, "recording.statistics.set", RecordingStatisticsParams{ID: timing.ID, ProbeID: timing.ProbeID, End: .02}, "conflict")
}

func TestRecordingStatisticsDCAndDiscovery(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.reset", ReplaceParams{}, "dc-example")
	call(t, s, "simulation.pause", struct{}{}, "pause-dc")
	_, captured := verify(t, s, VerifyParams{}, "dc-statistics")
	p := RecordingStatisticsParams{ID: captured.Operation.ID, ProbeID: "i:R1"}
	refuse(t, s, "recording.statistics", p, "invalid-params")
	call(t, s, "recording.statistics.set", RecordingStatisticsParams{ID: p.ID}, "clear-dc-statistics")
	for _, command := range []string{"recording.statistics", "recording.statistics.set"} {
		caps, err := detailedCapabilities(command)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(caps[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["x-command-contract"] == nil || schema["x-max-recorded-frames"] != float64(MaxVerificationFrames) || inputProperty(schema, "start_s")["$ref"] != TimeInputSchemaID {
			t.Fatal("measurement discovery does not describe units, bounds and semantics", schema)
		}
	}
	var output map[string]any
	if err := json.Unmarshal(SchemaRegistry()["urn:elab:recording-statistics-settings:1"], &output); err != nil {
		t.Fatal(err)
	}
	if inputProperty(output, "start_s")["type"] != "number" {
		t.Fatal("input-only unit grammar widened numeric output")
	}
}
