package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

// The interval is independent of plot reduction and the sample-statistics view.
type RecordingStatisticsParams struct {
	ID      string  `json:"id"`
	ProbeID string  `json:"probe_id"`
	Start   float64 `json:"start_s"`
	End     float64 `json:"end_s"`
}

type RecordingStatisticsSettings RecordingStatisticsParams

type RecordingStatistics struct {
	Settings    RecordingStatisticsSettings `json:"settings"`
	OriginID    string                      `json:"origin_id"`
	Engine      string                      `json:"engine"`
	Method      string                      `json:"method"`
	Probe       Probe                       `json:"probe"`
	SampleCount int                         `json:"sample_count"`
	Segments    int                         `json:"segment_count"`
	Duration    float64                     `json:"duration_s"`
	Minimum     float64                     `json:"minimum"`
	Maximum     float64                     `json:"maximum"`
	PeakToPeak  float64                     `json:"peak_to_peak"`
	Mean        float64                     `json:"time_mean"`
	RMS         float64                     `json:"time_rms"`
	ACRMS       float64                     `json:"ac_rms"`
}

type statisticsSegment struct{ duration, a, b float64 }

// Compensated sums retain small-duration contributions beside long segments.
type statisticsSum struct{ value, correction float64 }

func (s *statisticsSum) add(value float64) {
	y := value - s.correction
	next := s.value + y
	s.correction = (next - s.value) - y
	s.value = next
}

func (s *Service) measureRecordingStatisticsLocked(p RecordingStatisticsParams) (*RecordingStatistics, error) {
	op := s.operations[p.ID]
	if op == nil || op.record == nil {
		return nil, &actionError{"not-ready", "Choose a retained completed recording for interval measurements"}
	}
	record := op.record
	if !finite(p.Start) || !finite(p.End) || p.Start < 0 || p.Start >= p.End || p.End > record.metadata.Duration {
		return nil, fmt.Errorf("measurement interval requires 0 ≤ start_s < end_s ≤ captured duration")
	}
	index := -1
	for i, probe := range record.metadata.Probes {
		if probe.ID == p.ProbeID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, &actionError{"not-found", "Measurement probe was not captured in this recording"}
	}
	out := &RecordingStatistics{Settings: RecordingStatisticsSettings(p), OriginID: record.metadata.OperationID, Engine: record.metadata.Engine, Method: "piecewise-linear.v1", Probe: record.metadata.Probes[index], SampleCount: len(record.rows), Duration: p.End - p.Start}
	segments := make([]statisticsSegment, 0, len(record.rows)-1)
	for i := 1; i < len(record.rows); i++ {
		before, after := record.rows[i-1], record.rows[i]
		lo, hi := math.Max(p.Start, before.Time), math.Min(p.End, after.Time)
		// Duplicate-time edge rows preserve the left/right limits, but the
		// vertical segment has zero area and never receives a sample weight.
		if hi <= lo || after.Time <= before.Time {
			continue
		}
		interpolate := func(at float64) float64 {
			fraction := (at - before.Time) / (after.Time - before.Time)
			return (1-fraction)*before.Values[index] + fraction*after.Values[index]
		}
		a, b := interpolate(lo), interpolate(hi)
		if len(segments) == 0 {
			out.Minimum, out.Maximum = math.Min(a, b), math.Max(a, b)
		} else {
			out.Minimum = math.Min(out.Minimum, math.Min(a, b))
			out.Maximum = math.Max(out.Maximum, math.Max(a, b))
		}
		segments = append(segments, statisticsSegment{hi - lo, a, b})
	}
	if len(segments) == 0 {
		return nil, &actionError{"unavailable", "No positive-duration saved segments cover this interval"}
	}
	out.Segments, out.PeakToPeak = len(segments), out.Maximum-out.Minimum
	scale := math.Max(math.Abs(out.Minimum), math.Abs(out.Maximum))
	if scale != 0 {
		var mean, square statisticsSum
		for _, seg := range segments {
			a, b, weight := seg.a/scale, seg.b/scale, seg.duration/out.Duration
			mean.add(weight * (a/2 + b/2))
			square.add(weight * (a*a + a*b + b*b) / 3)
		}
		out.Mean = scale * mean.value
		out.RMS = scale * math.Sqrt(math.Max(0, square.value))
	}
	// Center and scale the ripple separately, before integration. Subtracting
	// RMS² - mean² or normalizing a tiny ripple by a large DC level loses it.
	if out.PeakToPeak > 0 && finite(out.PeakToPeak) {
		var offset, variance statisticsSum
		for _, seg := range segments {
			a, b := (seg.a-out.Minimum)/out.PeakToPeak, (seg.b-out.Minimum)/out.PeakToPeak
			offset.add((seg.duration / out.Duration) * (a/2 + b/2))
		}
		for _, seg := range segments {
			a, b := (seg.a-out.Minimum)/out.PeakToPeak-offset.value, (seg.b-out.Minimum)/out.PeakToPeak-offset.value
			variance.add((seg.duration / out.Duration) * (a*a + a*b + b*b) / 3)
		}
		out.ACRMS = out.PeakToPeak * math.Sqrt(math.Max(0, variance.value))
	}
	for _, value := range []float64{out.Minimum, out.Maximum, out.PeakToPeak, out.Mean, out.RMS, out.ACRMS} {
		if !finite(value) {
			return nil, fmt.Errorf("interval measurement exceeds the finite numeric range")
		}
	}
	return out, nil
}

func (s *Service) recordingStatisticsQueryLocked(raw json.RawMessage) (*RecordingStatistics, error) {
	normalized, err := normalizeTimeInput("recording.statistics", raw)
	if err != nil {
		return nil, err
	}
	var p RecordingStatisticsParams
	if err := params(normalized, &p, "id", "probe_id", "start_s", "end_s"); err != nil {
		return nil, err
	}
	return s.measureRecordingStatisticsLocked(p)
}

func (s *Service) setRecordingStatisticsLocked(raw json.RawMessage) error {
	var p RecordingStatisticsParams
	if err := params(raw, &p, "id", "probe_id", "start_s", "end_s"); err != nil {
		return err
	}
	if p.ID != s.selectedOperation {
		return &actionError{"conflict", "Selected recording changed; refresh before measuring"}
	}
	if _, err := s.selectedRecordLocked(); err != nil {
		return err
	}
	if p.ProbeID == "" {
		if p.Start != 0 || p.End != 0 {
			return fmt.Errorf("clear interval measurements with empty probe and zero interval")
		}
		s.recording.statistics = nil
		return nil
	}
	result, err := s.measureRecordingStatisticsLocked(p)
	if err != nil {
		return err
	}
	s.recording.statistics = result
	return nil
}

func cloneRecordingStatistics(in *RecordingStatistics) *RecordingStatistics {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
