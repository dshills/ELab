package workbench

// RecordingOverlay is a bounded view of captured reference samples, never a
// simulation or a resampled/aligned experiment. Its axis follows selected A.
type RecordingOverlay struct {
	ID            string  `json:"id"`
	OriginID      string  `json:"origin_id"`
	Analysis      string  `json:"analysis"`
	Engine        string  `json:"engine"`
	TimeBasis     string  `json:"time_basis"`
	Duration      float64 `json:"duration_s"`
	Step          float64 `json:"sample_step_s"`
	WindowSamples int     `json:"window_sample_count"`
	PlotReduced   bool    `json:"plot_reduced"`
	Scope         Scope   `json:"scope"`
}

func (s *Service) recordingOverlayLocked(view *RecordingView) *RecordingOverlay {
	id := view.Comparison.Settings.ReferenceID
	out := &RecordingOverlay{ID: id, TimeBasis: "captured-seconds-from-startup", Scope: Scope{Window: view.Scope.Window, Available: []Probe{}, Selected: []string{}, Traces: []ScopeTrace{}, Statistics: []ScopeStatistics{}, Readings: []CursorReading{}}}
	op := s.operations[id]
	if op == nil || op.record == nil {
		out.Scope.WindowReason = "Reference recording is no longer available"
		return out
	}
	record := op.record
	m := record.metadata
	out.OriginID, out.Analysis, out.Engine, out.Duration, out.Step = m.OperationID, m.Analysis, m.Engine, m.Duration, m.Settings.StepSeconds
	out.Scope.Available = append([]Probe{}, m.Probes...)
	for _, row := range record.rows {
		if scopeWindowTimeInside(row.Time, out.Scope.Window.Start, out.Scope.Window.End) {
			out.WindowSamples++
		}
	}
	// Follow selected A's channel order and identity. A display label alone does
	// not identify a comparable polarity/reference; differently wired graphs may
	// still intentionally share the same terminal/probe identity.
	for _, trace := range view.Scope.Traces {
		for i, probe := range m.Probes {
			if !sameComparisonProbe(trace.Probe, probe) {
				continue
			}
			points := []ScopePoint{}
			for _, row := range record.rows {
				if scopeWindowTimeInside(row.Time, out.Scope.Window.Start, out.Scope.Window.End) {
					points = append(points, ScopePoint{Time: row.Time, Value: row.Values[i]})
				}
			}
			reduced := reduceRecordedPoints(points)
			out.PlotReduced = out.PlotReduced || len(reduced) < len(points)
			out.Scope.Selected = append(out.Scope.Selected, probe.ID)
			out.Scope.Traces = append(out.Scope.Traces, ScopeTrace{Probe: probe, Points: reduced})
		}
	}
	switch {
	case len(out.Scope.Selected) == 0:
		out.Scope.WindowReason = "No selected channels have matching captured reference probes"
	case out.WindowSamples == 0:
		out.Scope.WindowReason = "No exact reference samples in the selected interval"
	default:
		out.Scope.WindowValid = true
	}
	return out
}

func cloneRecordingOverlay(in *RecordingOverlay) *RecordingOverlay {
	if in == nil {
		return nil
	}
	out := *in
	out.Scope = cloneScope(in.Scope)
	return &out
}
