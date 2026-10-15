package workbench

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func timingQuery(t *testing.T, s *Service, p any) *RecordingTiming {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "timing-query", Command: "recording.timing", Params: raw}, "agent")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	return response.Result.(*RecordingTiming)
}
func timingFixture(t *testing.T) (*Service, RecordingTimingParams) {
	t.Helper()
	s := pausedRC(t)
	duration := .02
	_, view := verify(t, s, VerifyParams{Duration: &duration, InputMode: "charge", ProbeIDs: []string{"v:C1"}}, "charge")
	return s, RecordingTimingParams{ID: view.Operation.ID, ProbeID: "v:C1", Level: "2.5 V", Direction: "rising", End: duration}
}
func TestRecordingTimingRCChargeDischargeBoundsAndPurity(t *testing.T) {
	s, p := timingFixture(t)
	defer s.Close()
	before := s.Snapshot()
	result := timingQuery(t, s, p)
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("timing query changed public state")
	}
	expected := .01 * math.Log(2)
	if result.Total != 1 || result.Level != 2.5 || len(result.Crossings) != 1 {
		t.Fatal("charge crossing missing", result)
	}
	crossing := result.Crossings[0]
	if crossing.Before.Index != 6 || crossing.After.Index != 7 || !(crossing.Before.Time < expected && expected < crossing.After.Time) || crossing.Before.Value >= 2.5 || crossing.After.Value < 2.5 {
		t.Fatal("analytic RC crossing outside saved bracket", crossing, expected)
	}
	call(t, s, "recording.timing.set", p, "set")
	cached := s.Snapshot().Recording.Timing
	if !reflect.DeepEqual(result, cached) {
		t.Fatal("query/configured timing differ")
	}
	after := s.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Transient.Sample != after.Transient.Sample || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("timing setting changed experiment/history")
	}
	cached.Crossings[0].Before.Value = 999
	result.Crossings[0].After.Time = 999
	if s.Snapshot().Recording.Timing.Crossings[0].Before.Value == 999 || timingQuery(t, s, p).Crossings[0].After.Time == 999 {
		t.Fatal("timing responses alias retained data")
	}
	call(t, s, "recording.window.set", RecordingWindowParams{Start: .01, End: .02}, "plot-window")
	if s.Snapshot().Recording.Timing.Total != 1 {
		t.Fatal("plot window silently changed timing interval")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "later-edit")
	if !reflect.DeepEqual(s.Snapshot().Recording.Timing, timingQuery(t, s, p)) {
		t.Fatal("timing substituted later live inputs")
	}
	clipped := p
	clipped.Start = .0065
	if timingQuery(t, s, clipped).Total != 0 {
		t.Fatal("fabricated crossing at interval boundary")
	}
	duration, initial := .02, 5.
	_, discharge := verify(t, s, VerifyParams{Duration: &duration, InputMode: "discharge", InitialVoltage: &initial, ProbeIDs: []string{"v:C1"}}, "discharge")
	if s.Snapshot().Recording.Timing != nil {
		t.Fatal("new capture retained old timing")
	}
	dp := RecordingTimingParams{ID: discharge.Operation.ID, ProbeID: "v:C1", Level: "2500 mV", Direction: "falling", End: duration}
	result = timingQuery(t, s, dp)
	expected = .02 * math.Log(2)
	if result.Total != 1 || result.Crossings[0].Before.Time > expected || result.Crossings[0].After.Time < expected {
		t.Fatal("analytic discharge crossing outside bracket")
	}
}
func TestRecordingTimingIdealNPNEdgesAndSpacing(t *testing.T) {
	s := pausedNPNPulse(t)
	defer s.Close()
	duration := .25
	_, view := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive"}}, "pulse")
	p := RecordingTimingParams{ID: view.Operation.ID, ProbeID: "v:VIN.positive", Level: "2.5 V", Direction: "both", End: duration}
	result := timingQuery(t, s, p)
	if result.Total < 4 || result.RisingSpacing == nil || result.FallingSpacing == nil {
		t.Fatal("repeated pulse edges not measured", result)
	}
	for _, c := range result.Crossings {
		if c.Before.Time != c.After.Time {
			t.Fatal("ideal duplicate edge rows not preserved", c)
		}
	}
	for _, spacing := range []*TimingSpacing{result.RisingSpacing, result.FallingSpacing} {
		if math.Abs(spacing.Minimum-.1) > 1e-12 || math.Abs(spacing.Maximum-.1) > 1e-12 {
			t.Fatal("known pulse spacing outside bounds", spacing)
		}
	}
	call(t, s, "recording.timing.set", p, "configure")
	copy := s.Snapshot()
	copy.Recording.Timing.RisingSpacing.First.Before.Value = 999
	if s.Snapshot().Recording.Timing.RisingSpacing.First.Before.Value == 999 {
		t.Fatal("spacing bracket aliases state")
	}
}
func TestRecordingTimingPlateausPagingAndStrictIntervals(t *testing.T) {
	s, p := timingFixture(t)
	defer s.Close()
	// Isolate sample threshold-arrival semantics; these controlled rows are not
	// imported or used as a claim about a particular device/solver waveform.
	values := []float64{-1, 0, 0, 1, 0, 0, -1, 0, -1}
	s.mu.Lock()
	record := s.operations[p.ID].record
	record.rows = nil
	for i, v := range values {
		record.rows = append(record.rows, VerificationRow{Time: float64(i) / 1000, Values: []float64{v}})
	}
	s.mu.Unlock()
	p.Level = "0"
	p.Direction = "both"
	p.Limit = 1
	result := timingQuery(t, s, p)
	if result.Total != 3 || result.NextOffset != 1 || result.Crossings[0].After.Index != 1 || result.RisingSpacing == nil || result.FallingSpacing != nil {
		t.Fatal("plateau arrival semantics", result)
	}
	p.Offset = 1
	second := timingQuery(t, s, p)
	if second.Crossings[0].Direction != "falling" || second.Crossings[0].After.Index != 4 || !reflect.DeepEqual(second.RisingSpacing, result.RisingSpacing) {
		t.Fatal("paging changes whole-window summary")
	}
	p.Offset = 3
	last := timingQuery(t, s, p)
	if len(last.Crossings) != 0 || last.NextOffset != 3 || last.Total != 3 {
		t.Fatal("end page incorrect")
	}
	p.Offset = 0
	p.Start = .001
	p.End = .001
	if timingQuery(t, s, p).Total != 0 {
		t.Fatal("interval boundary fabricated a row")
	}
	p.Start = 0
	p.End = .02
	p.Direction = "falling"
	if timingQuery(t, s, p).Total != 1 {
		t.Fatal("direction filter ignored")
	}
}
func TestRecordingTimingFailuresClearAndReopen(t *testing.T) {
	s, p := timingFixture(t)
	defer s.Close()
	call(t, s, "recording.timing.set", p, "set")
	bads := []RecordingTimingParams{}
	for _, change := range []func(*RecordingTimingParams){func(q *RecordingTimingParams) { q.End = .021 }, func(q *RecordingTimingParams) { q.Start = .02; q.End = .01 }, func(q *RecordingTimingParams) { q.Offset = -1 }, func(q *RecordingTimingParams) { q.Offset = 2 }, func(q *RecordingTimingParams) { q.Limit = 101 }, func(q *RecordingTimingParams) { q.Direction = "up" }, func(q *RecordingTimingParams) { q.Level = "2 A" }, func(q *RecordingTimingParams) { q.Level = "NaN" }} {
		bad := p
		change(&bad)
		bads = append(bads, bad)
	}
	for _, bad := range bads {
		before := s.Snapshot()
		refuse(t, s, "recording.timing", bad, "invalid-params")
		refuse(t, s, "recording.timing.set", bad, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid timing changed state", bad)
		}
	}
	bad := p
	bad.ID = "other"
	refuse(t, s, "recording.timing.set", bad, "conflict")
	refuse(t, s, "recording.timing", bad, "not-ready")
	bad = p
	bad.ProbeID = "i:C1"
	refuse(t, s, "recording.timing", bad, "not-found")
	raw, _ := json.Marshal(p)
	var missing map[string]any
	_ = json.Unmarshal(raw, &missing)
	delete(missing, "level")
	refuse(t, s, "recording.timing", missing, "invalid-params")
	path := filepath.Join(t.TempDir(), "timing.elab")
	expected := timingQuery(t, s, p)
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	loaded := New()
	defer loaded.Close()
	call(t, loaded, "project.load", LoadParams{Path: path}, "open")
	if loaded.Snapshot().Recording.Timing != nil {
		t.Fatal("session timing leaked into project")
	}
	p.ID = loaded.Snapshot().Recording.ID
	if !reflect.DeepEqual(timingQuery(t, loaded, p).Crossings, expected.Crossings) {
		t.Fatal("reopened data changed timing")
	}
	clear := RecordingTimingParams{ID: s.Snapshot().Recording.ID, Level: "0", Direction: "both"}
	call(t, s, "recording.timing.set", clear, "clear")
	if s.Snapshot().Recording.Timing != nil {
		t.Fatal("timing clear failed")
	}
}
func TestThresholdUnitsAndNumericTimingSchemas(t *testing.T) {
	for _, test := range []struct {
		text, unit string
		want       float64
	}{{"2.5 V", "V", 2.5}, {"2500 mV", "V", 2.5}, {"-2.5 mA", "A", -.0025}, {"20 µA", "A", 20e-6}, {"20 μA", "A", 20e-6}, {"20 uA", "A", 20e-6}, {"1e-3", "A", .001}} {
		value, err := parseThresholdLevel(test.text, test.unit)
		if err != nil || math.Abs(value-test.want) > 1e-18 {
			t.Fatal(test, value, err)
		}
	}
	for _, text := range []string{"1 V", "1 m", "NaN", "Inf", "1e13 A", "1e-323 mA", "1e-999 A"} {
		if _, err := parseThresholdLevel(text, "A"); err == nil {
			t.Fatal("invalid threshold accepted", text)
		}
	}
	for _, command := range []string{"recording.timing", "recording.timing.set"} {
		caps, err := detailedCapabilities(command)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		_ = json.Unmarshal(caps[0].Input, &schema)
		if schema["x-command-contract"] == nil || inputProperty(schema, "limit")["maximum"] != float64(MaxTimingPage) || inputProperty(schema, "limit")["minimum"] != float64(0) {
			t.Fatal("timing details missing")
		}
	}
	var output map[string]any
	_ = json.Unmarshal(SchemaRegistry()["urn:elab:recording-timing-settings:1"], &output)
	if inputProperty(output, "start_s")["type"] != "number" || inputProperty(output, "end_s")["type"] != "number" {
		t.Fatal("time grammar leaked into numeric output")
	}
}

// Use a post-startup interval: the first startup edge spacing is not a period.
func TestRecordingTimingBothFlasherSides(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	duration := 1.2
	_, view := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"i:D1", "i:D2"}}, "timed-flasher")
	for _, probe := range []string{"i:D1", "i:D2"} {
		result := timingQuery(t, s, RecordingTimingParams{ID: view.Operation.ID, ProbeID: probe, Level: "1 mA", Direction: "rising", Start: .3, End: duration})
		if result.RisingSpacing == nil || result.RisingSpacing.Minimum < .47 || result.RisingSpacing.Maximum > .51 {
			t.Fatal("flasher repeat spacing outside existing reference envelope", probe, result)
		}
		c := result.Cycle
		if c == nil || c.Duty == nil || c.High.Minimum <= 0 || c.Low.Minimum <= 0 || c.Period.Minimum < .47 || c.Period.Maximum > .51 || c.Duty.Minimum <= 0 || c.Duty.Maximum >= 1 || c.Duty.Minimum > c.Duty.Maximum {
			t.Fatal("flasher pulse/cycle summary missing or invalid", probe)
		}
	}
}
func TestRecordingTimingLaterSelectionSurvivesImport(t *testing.T) {
	path := importFixture(t)
	s, p := timingFixture(t)
	defer s.Close()
	entered, release := blockRecordingImport(t, s)
	defer release()
	promptLiveCommand(t, s, "verification.import.start", RecordingImportParams{Path: path})
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "recording.timing.set", p)
	before := s.Snapshot().Recording
	release()
	after := awaitProjectStatus(t, s, "completed")
	if after.ProjectWork.ViewerSelected == nil || *after.ProjectWork.ViewerSelected || !reflect.DeepEqual(before, after.Recording) {
		t.Fatal("late import discarded newer timing")
	}
}

func TestRecordingTimingDCAndEmptyCapture(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	duration := 0.
	_, view := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:D1"}}, "dc-timing")
	p := RecordingTimingParams{ID: view.Operation.ID, ProbeID: "v:D1", Level: "0", Direction: "both"}
	result := timingQuery(t, s, p)
	if result.Total != 0 || result.WindowSamples != 1 || result.RisingSpacing != nil || result.FallingSpacing != nil {
		t.Fatal("DC fabricated temporal arrivals")
	}
	_, empty := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{}}, "empty-timing")
	call(t, s, "recording.timing.set", RecordingTimingParams{ID: empty.Operation.ID, Level: "0", Direction: "both"}, "empty-off")
	if s.Snapshot().Recording.Timing != nil {
		t.Fatal("empty capture cannot turn timing off")
	}
	if zero, err := parseThresholdLevel("0e-999 A", "A"); err != nil || zero != 0 {
		t.Fatal("explicit zero level refused", err)
	}
	if tiny, err := parseThresholdLevel("1e-320 mA", "A"); err != nil || tiny == 0 {
		t.Fatal("representable tiny level lost", tiny, err)
	}
}
