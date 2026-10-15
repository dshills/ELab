package workbench

import (
	"math"
	"sort"
)

func nearestRecordingIndex(rows []VerificationRow, at float64) int {
	i := sort.Search(len(rows), func(i int) bool { return rows[i].Time >= at })
	if i == len(rows) {
		return i - 1
	}
	if i > 0 && at-rows[i-1].Time < rows[i].Time-at {
		i--
	}
	for i+1 < len(rows) && rows[i+1].Time == rows[i].Time {
		i++
	}
	return i
}
func (s *Service) recordingViewLocked() *RecordingView {
	operation := s.operations[s.selectedOperation]
	if operation == nil || operation.record == nil {
		return nil
	}
	record := operation.record
	m := record.metadata
	index := min(max(s.recording.index, 0), len(record.rows)-1)
	out := &RecordingView{ID: operation.summary.ID, OriginID: m.OperationID, Imported: operation.summary.Kind != "verify", Viewing: s.recording.viewing, Playing: s.recording.playing, Speed: s.recording.speed, Index: index, Time: record.rows[index].Time, Duration: m.Duration, Step: m.Settings.StepSeconds, Total: len(record.rows), Readings: []VerificationReading{}}
	scope := Scope{Window: ScopeWindow{Mode: "fixed", End: m.Duration}, WindowValid: true, Available: append([]Probe{}, m.Probes...), Selected: []string{}, Traces: []ScopeTrace{}, Statistics: []ScopeStatistics{}, Readings: []CursorReading{}, ActiveCursor: s.recording.activeCursor}
	if scope.ActiveCursor == "" {
		scope.ActiveCursor = "a"
	}
	if s.recording.window != nil {
		scope.Window.Start = s.recording.window.Start
		scope.Window.End = s.recording.window.End
	}
	ids := s.recording.probes
	if ids == nil {
		ids = []string{}
		for _, probe := range m.Probes {
			ids = append(ids, probe.ID)
		}
	}
	scope.Selected = append([]string{}, ids...)
	for _, row := range record.rows {
		if scopeTimeInside(row.Time, scope.Window.Start, scope.Window.End) {
			out.WindowSamples++
		}
	}
	if out.WindowSamples == 0 {
		scope.WindowValid = false
		scope.WindowReason = "No exact recorded samples in this interval"
	}
	for _, id := range ids {
		for i, probe := range m.Probes {
			if probe.ID != id {
				continue
			}
			points := []ScopePoint{}
			for _, row := range record.rows {
				if scopeTimeInside(row.Time, scope.Window.Start, scope.Window.End) {
					points = append(points, ScopePoint{Time: row.Time, Value: row.Values[i]})
				}
			}
			stat := scopeStatistics(points, id)
			scope.Statistics = append(scope.Statistics, stat)
			reduced := reduceRecordedPoints(points)
			if len(reduced) < len(points) {
				out.PlotReduced = true
			}
			scope.Traces = append(scope.Traces, ScopeTrace{Probe: probe, Points: reduced})
			out.Readings = append(out.Readings, VerificationReading{Probe: probe, Value: record.rows[index].Values[i], Minimum: stat.Minimum, Maximum: stat.Maximum})
		}
	}
	cursor := func(requested *float64) *ScopeCursor {
		if requested == nil {
			return nil
		}
		at := record.rows[nearestRecordingIndex(record.rows, *requested)].Time
		c := &ScopeCursor{Requested: *requested, Time: at, Valid: scopeTimeInside(at, scope.Window.Start, scope.Window.End)}
		if !c.Valid {
			c.Reason = "Recorded cursor is outside the visible interval"
		}
		return c
	}
	scope.A = cursor(s.recording.a)
	scope.B = cursor(s.recording.b)
	if scope.A != nil && scope.B != nil && scope.A.Valid && scope.B.Valid {
		delta := scope.B.Time - scope.A.Time
		scope.DeltaSeconds = &delta
		if delta != 0 {
			frequency := 1 / math.Abs(delta)
			if finite(frequency) {
				scope.Frequency = &frequency
			}
		}
		a, b := nearestRecordingIndex(record.rows, scope.A.Time), nearestRecordingIndex(record.rows, scope.B.Time)
		for _, id := range ids {
			for i, p := range m.Probes {
				if p.ID == id {
					scope.Readings = append(scope.Readings, CursorReading{ProbeID: id, A: record.rows[a].Values[i], B: record.rows[b].Values[i], Delta: record.rows[b].Values[i] - record.rows[a].Values[i]})
				}
			}
		}
	}
	out.Scope = scope
	out.Comparison = s.recordingComparisonViewLocked()
	return out
}

// Preserve first/last and each bucket's min/max in original order. Exact rows,
// cursor lookup and statistics are never reduced or interpolated.
func reduceRecordedPoints(points []ScopePoint) []ScopePoint {
	if len(points) <= 2*RecordingPlotBuckets+2 {
		return points
	}
	indices := map[int]bool{0: true, len(points) - 1: true}
	for bucket := 0; bucket < RecordingPlotBuckets; bucket++ {
		lo, hi := bucket*len(points)/RecordingPlotBuckets, (bucket+1)*len(points)/RecordingPlotBuckets
		low, high := lo, lo
		for i := lo; i < hi; i++ {
			if points[i].Value < points[low].Value {
				low = i
			}
			if points[i].Value > points[high].Value {
				high = i
			}
		}
		indices[low] = true
		indices[high] = true
	}
	ordered := make([]int, 0, len(indices))
	for i := range indices {
		ordered = append(ordered, i)
	}
	sort.Ints(ordered)
	out := make([]ScopePoint, 0, len(ordered))
	for _, i := range ordered {
		out = append(out, points[i])
	}
	return out
}
func cloneRecordingView(view *RecordingView) *RecordingView {
	if view == nil {
		return nil
	}
	out := *view
	out.Scope = cloneScope(view.Scope)
	out.Comparison = cloneRecordingComparison(view.Comparison)
	out.Readings = append([]VerificationReading{}, view.Readings...)
	return &out
}
