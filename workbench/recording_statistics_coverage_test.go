package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func importedStatisticsTail(t *testing.T) (*Service, RecordingStatisticsParams) {
	t.Helper()
	s := New()
	t.Cleanup(s.Close)
	path, err := filepath.Abs("testdata/recorded-statistics-short-tail.json")
	if err != nil {
		t.Fatal(err)
	}
	call(t, s, "verification.import", RecordingImportParams{Path: path}, "import-tail")
	r := s.Snapshot().Recording
	if r == nil || r.Duration != .02 || r.LastSampleTime != .02-5e-13 || r.LastSampleTime >= r.Duration {
		t.Fatal("fixture needs an accepted recording whose final saved timestamp is before its nominal duration", r)
	}
	return s, RecordingStatisticsParams{ID: r.ID, ProbeID: "v:V1.positive"}
}

func TestRecordingStatisticsRequireCompleteSavedTimeCoverage(t *testing.T) {
	s, p := importedStatisticsTail(t)
	last := s.Snapshot().Recording.LastSampleTime
	p.End = last
	call(t, s, "recording.statistics.set", p, "previous-valid-measurement")
	for _, interval := range [][2]float64{
		{0, .02},
		{.02 - 1e-12, .02},
		{last, .02},
		{.02 - 1e-12, math.Nextafter(last, math.Inf(1))},
	} {
		p.Start, p.End = interval[0], interval[1]
		for _, command := range []string{"recording.statistics", "recording.statistics.set"} {
			before := s.Snapshot()
			raw, _ := json.Marshal(p)
			revision := before.CommandRevision
			r := s.Dispatch(context.Background(), Request{Version: 1, ID: command, Key: command, Command: command, Params: raw, ExpectedCommandRevision: &revision}, "agent.coverage")
			if r.Error == nil || r.Error.Code != "unavailable" || r.Result != nil {
				t.Fatalf("partly unsupported %g→%g interval published statistics: error=%v result=%+v", p.Start, p.End, r.Error, r.Result)
			}
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("refused coverage changed prior statistics, rows or live/document/history state")
			}
		}
	}
}

func TestRecordingStatisticsKeepFullyCoveredTinyIntervals(t *testing.T) {
	s, p := importedStatisticsTail(t)
	last := s.Snapshot().Recording.LastSampleTime
	for _, interval := range [][2]float64{{0, last}, {.02 - 1e-12, last}, {math.Nextafter(last, 0), last}} {
		p.Start, p.End = interval[0], interval[1]
		before := s.Snapshot()
		result := statisticsQuery(t, s, p)
		nearStatistic(t, result.Mean, 5, 1e-13)
		nearStatistic(t, result.RMS, 5, 1e-13)
		nearStatistic(t, result.ACRMS, 0, 1e-13)
		if result.Minimum != 5 || result.Maximum != 5 || result.Settings.Start != p.Start || result.Settings.End != p.End || result.Duration != p.End-p.Start || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("covered interval was widened, normalized, extrapolated or mutated", result)
		}
	}
}

func TestRecordingStatisticsCanDiscoverCoverageThroughPublicSampleReads(t *testing.T) {
	s, p := importedStatisticsTail(t)
	before := s.Snapshot()
	inspect, _ := json.Marshal(IDParams{ID: p.ID})
	op := s.Dispatch(context.Background(), Request{Version: 1, ID: "source", Command: "operation.inspect", Params: inspect}, "agent.coverage")
	if op.Error != nil {
		t.Fatal(op.Error)
	}
	count := op.Result.(OperationView).Metadata.SampleCount
	raw, _ := json.Marshal(VerificationPageParams{ID: p.ID, Offset: count - 1, Limit: 1})
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "saved-end", Command: "operation.samples", Params: raw}, "agent.coverage")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	page := response.Result.(VerificationPage)
	if len(page.Rows) != 1 || page.Rows[0].Time >= .02 {
		t.Fatal("sample inventory did not expose the actual saved endpoint", page)
	}
	p.End = page.Rows[0].Time
	result := statisticsQuery(t, s, p)
	nearStatistic(t, result.Mean, 5, 1e-13)
	nearStatistic(t, result.RMS, 5, 1e-13)
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("discovering saved bounds or measuring them changed state")
	}
}
