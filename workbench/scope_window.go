package workbench

import (
	"fmt"
	"math"
)

type ScopeWindow struct {
	Mode  string  `json:"mode"`
	Start float64 `json:"start_s"`
	End   float64 `json:"end_s"`
}
type ScopeStatistics struct {
	ProbeID string  `json:"probe_id"`
	Count   int     `json:"count"`
	Minimum float64 `json:"min"`
	Maximum float64 `json:"max"`
	Mean    float64 `json:"sample_mean"`
	RMS     float64 `json:"sample_rms"`
}

func (s *Service) setScopeWindowLocked(p ScopeWindow) error {
	if p.Mode == "live" {
		s.scopeWindow = nil
		return nil
	}
	t := s.transient
	if t == nil || t.Stale || len(t.Samples) == 0 || !validFixedScopeWindow(p, t.Samples[0].Time, t.Sample.Time) {
		return fmt.Errorf("fixed scope window requires a fresh retained interval with start_s < end_s; mode live follows the rolling history")
	}
	s.scopeWindow = &p
	return nil
}

func validFixedScopeWindow(p ScopeWindow, start, end float64) bool {
	return p.Mode == "fixed" && finite(p.Start) && finite(p.End) &&
		scopeTimeInside(p.Start, start, end) && scopeTimeInside(p.End, start, end) && p.Start < p.End
}

func scopeStatistics(points []ScopePoint, id string) ScopeStatistics {
	out := ScopeStatistics{ProbeID: id, Count: len(points)}
	if len(points) == 0 {
		return out
	}
	out.Minimum = points[0].Value
	out.Maximum = out.Minimum
	sum, square := 0., 0.
	for _, p := range points {
		out.Minimum = math.Min(out.Minimum, p.Value)
		out.Maximum = math.Max(out.Maximum, p.Value)
		sum += p.Value
		square += p.Value * p.Value
	}
	out.Mean = sum / float64(len(points))
	out.RMS = math.Sqrt(square / float64(len(points)))
	return out
}

// Inclusive decimal boundaries tolerate only floating point timestamp roundoff.
func scopeTimeInside(at, start, end float64) bool {
	epsilon := 1e-12 * math.Max(1, math.Max(math.Abs(start), math.Abs(end)))
	return at >= start-epsilon && at <= end+epsilon
}

// Visible windows must not borrow nearby rows when the legacy timestamp
// tolerance exceeds the selected span. Keep the existing boundary tolerance for
// ordinary windows, but cap it to a billionth of the visible interval.
const maxScopeWindowRoundoffFraction = 1e-9

func scopeWindowTimeInside(at, start, end float64) bool {
	epsilon := math.Min(1e-12*math.Max(1, math.Max(math.Abs(start), math.Abs(end))), math.Max(0, end-start)*maxScopeWindowRoundoffFraction)
	return at >= start-epsilon && at <= end+epsilon
}
