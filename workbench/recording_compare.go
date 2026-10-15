package workbench

import (
	"encoding/json"
	"fmt"
)

// Comparison is session/view state; its inputs always name retained captures.
// An empty reference clears the configured comparison (set command only).
type RecordingCompareParams struct {
	ShowOverlay    bool    `json:"show_overlay,omitempty"`
	ShowStatistics bool    `json:"show_statistics,omitempty"`
	ID             string  `json:"id"`
	ReferenceID    string  `json:"reference_id"`
	Time           float64 `json:"time_s"`
}

// Output schema uses numeric SI rather than command unit strings.
type RecordingCompareSettings RecordingCompareParams

type ComparisonSample struct {
	ID       string  `json:"id"`
	OriginID string  `json:"origin_id"`
	Analysis string  `json:"analysis"`
	Engine   string  `json:"engine"`
	Duration float64 `json:"duration_s"`
	Step     float64 `json:"sample_step_s"`
	Count    int     `json:"sample_count"`
	Index    int     `json:"sample_index"`
	Time     float64 `json:"time_s"`
	Offset   float64 `json:"offset_from_requested_s"`
}
type ComparisonReading struct {
	Probe       Probe           `json:"probe"`
	A           float64         `json:"a"`
	B           float64         `json:"b"`
	Delta       float64         `json:"b_minus_a"`
	StatisticsA ScopeStatistics `json:"whole_record_a"`
	StatisticsB ScopeStatistics `json:"whole_record_b"`
}
type RecordingComparison struct {
	Settings   RecordingCompareSettings `json:"settings"`
	Available  bool                     `json:"available"`
	Reason     string                   `json:"reason,omitempty"`
	A          ComparisonSample         `json:"a"`
	B          ComparisonSample         `json:"b"`
	Readings   []ComparisonReading      `json:"readings"`
	UnmatchedA []Probe                  `json:"unmatched_a"`
	UnmatchedB []Probe                  `json:"unmatched_b"`
}

func (s *Service) compareRecordingsLocked(p RecordingCompareParams) (*RecordingComparison, error) {
	if !finite(p.Time) || p.Time < 0 || p.Time > 10 {
		return nil, fmt.Errorf("comparison time_s must be within 0–10 seconds")
	}
	if p.ID == p.ReferenceID {
		return nil, fmt.Errorf("choose two different retained recordings")
	}
	records := [2]*verificationRecord{}
	samples := [2]ComparisonSample{}
	for i, id := range []string{p.ID, p.ReferenceID} {
		op := s.operations[id]
		if op == nil || op.record == nil {
			return nil, &actionError{"not-ready", "Both comparison IDs must name retained completed recordings"}
		}
		record := op.record
		if p.Time > record.metadata.Duration {
			return nil, fmt.Errorf("comparison time_s must be within both captured durations (maximum %.9g s for %s)", record.metadata.Duration, id)
		}
		index := nearestRecordingIndex(record.rows, p.Time)
		m := record.metadata
		records[i] = record
		samples[i] = ComparisonSample{ID: id, OriginID: m.OperationID, Analysis: m.Analysis, Engine: m.Engine, Duration: m.Duration, Step: m.Settings.StepSeconds, Count: len(record.rows), Index: index, Time: record.rows[index].Time, Offset: record.rows[index].Time - p.Time}
	}
	out := &RecordingComparison{Settings: RecordingCompareSettings(p), Available: true, A: samples[0], B: samples[1], Readings: []ComparisonReading{}, UnmatchedA: []Probe{}, UnmatchedB: []Probe{}}
	matched := map[int]bool{}
	for i, a := range records[0].metadata.Probes {
		found := -1
		for j, b := range records[1].metadata.Probes {
			// Labels are presentation. Polarity/reference and units are electrical
			// identity; equal display names alone never establish a comparable reading.
			if sameComparisonProbe(a, b) {
				found = j
				break
			}
		}
		if found < 0 {
			out.UnmatchedA = append(out.UnmatchedA, a)
			continue
		}
		matched[found] = true
		av, bv := records[0].rows[out.A.Index].Values[i], records[1].rows[out.B.Index].Values[found]
		out.Readings = append(out.Readings, ComparisonReading{Probe: a, A: av, B: bv, Delta: bv - av, StatisticsA: records[0].metadata.Statistics[i], StatisticsB: records[1].metadata.Statistics[found]})
	}
	for j, b := range records[1].metadata.Probes {
		if !matched[j] {
			out.UnmatchedB = append(out.UnmatchedB, b)
		}
	}
	if len(out.Readings) == 0 {
		out.Available = false
		out.Reason = "No captured probes share the same ID, units and terminal references"
	}
	return out, nil
}

func sameComparisonProbe(a, b Probe) bool {
	return a.ID == b.ID && a.Unit == b.Unit && a.Positive == b.Positive && a.Reference == b.Reference
}
func (s *Service) setRecordingComparisonLocked(raw json.RawMessage) error {
	var p RecordingCompareParams
	if err := params(raw, &p, "id", "reference_id", "time_s"); err != nil {
		return err
	}
	if p.ID != s.selectedOperation {
		return &actionError{"conflict", "Selected recording changed; refresh before configuring comparison"}
	}
	if _, err := s.selectedRecordLocked(); err != nil {
		return err
	}
	if p.ReferenceID == "" {
		if !finite(p.Time) || p.Time < 0 || p.Time > 10 {
			return fmt.Errorf("comparison time_s must be within 0–10 seconds")
		}
		s.recording.comparison = nil
		return nil
	}
	if _, err := s.compareRecordingsLocked(p); err != nil {
		return err
	}
	s.recording.comparison = &p
	return nil
}
func (s *Service) recordingComparisonQueryLocked(raw json.RawMessage) (*RecordingComparison, error) {
	normalized, err := normalizeTimeInput("recording.compare", raw)
	if err != nil {
		return nil, err
	}
	var p RecordingCompareParams
	if err := params(normalized, &p, "id", "reference_id", "time_s"); err != nil {
		return nil, err
	}
	return s.compareRecordingsLocked(p)
}
func (s *Service) recordingComparisonViewLocked() *RecordingComparison {
	if s.recording.comparison == nil {
		return nil
	}
	p := *s.recording.comparison
	out, err := s.compareRecordingsLocked(p)
	if err == nil {
		return out
	}
	return &RecordingComparison{Settings: RecordingCompareSettings(p), Reason: err.Error(), Readings: []ComparisonReading{}, UnmatchedA: []Probe{}, UnmatchedB: []Probe{}}
}
func cloneRecordingComparison(in *RecordingComparison) *RecordingComparison {
	if in == nil {
		return nil
	}
	out := *in
	out.Readings = append([]ComparisonReading{}, in.Readings...)
	out.UnmatchedA = append([]Probe{}, in.UnmatchedA...)
	out.UnmatchedB = append([]Probe{}, in.UnmatchedB...)
	return &out
}

func addRecordingComparisonDetails(id string, schema map[string]any) bool {
	if id != "recording.compare" && id != "recording.compare.set" {
		return false
	}
	time := inputProperty(schema, "time_s")
	time["minimum"], time["maximum"] = 0, 10
	time["x-normalized-si-range"] = map[string]float64{"minimum": 0, "maximum": 10}
	schema["x-relations"] = []map[string]any{{"operator": "within-both-captured-durations", "fields": []string{"id", "reference_id", "time_s"}, "except": "empty reference clears set command"}}
	schema["x-inventory"] = map[string]string{"ids": "snapshot.operations with status completed"}
	inputProperty(schema, "show_overlay")["description"] = "Set-only viewer preference: plot matched reference channels on the selected recording's visible interval, at original absolute captured times. No alignment/extrapolation. Read snapshot.recording.reference_overlay while viewing recorded data. Query echoes the preference without changing a viewer."
	schema["x-max-reference-points-per-trace"] = 2*RecordingPlotBuckets + 2
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "captured-record-comparison", Preconditions: []string{"Distinct retained completed IDs; exact probe IDs, units and terminal references establish matching readings, not circuit/model equivalence", "time_s within both durations; no clamping or extrapolation", "Set command ID must match current selection; empty reference clears; selecting any operation clears comparison"}, Effects: []string{"Exact nearest stored rows; later/right-hand ties; no interpolation, solver work or live/history/project changes", "B minus A uses independently selected actual sample times; whole-record sample statistics retain each capture's resolution and duration", "Query is pure; set updates only session viewer configuration, including optional statistics display", "Optional show_overlay publishes bounded matching B traces on A visible absolute time interval; no alignment or extrapolation; plot cursors remain on A"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed input, same IDs or time outside either duration"}, {"not-ready", "admission", "Either capture is unavailable or incomplete"}, {"conflict", "admission", "Set command names a changed selected recording"}}, Completion: &commandCompletionDetail{Mode: "bounded immediate in-memory comparison; no worker or file IO", Observe: []string{"recording.compare", "project.inspect"}}}
	schema["examples"] = []any{RecordingCompareParams{ID: "verify-2", ReferenceID: "verify-1", Time: .01}}
	return true
}
