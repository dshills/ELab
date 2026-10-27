package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

const StepResponseMethod = "elab.recorded-linear-step-response.v1"

type StepResponseParams struct {
	ID        string  `json:"id"`
	ProbeID   string  `json:"probe_id"`
	Start     float64 `json:"start_s"`
	End       float64 `json:"end_s"`
	Initial   float64 `json:"initial_value"`
	Final     float64 `json:"final_value"`
	Tolerance float64 `json:"tolerance_fraction"`
	Hold      float64 `json:"minimum_hold_s"`
}
type StepResponseSettings StepResponseParams
type StepResponse struct {
	Settings  StepResponseSettings `json:"settings"`
	OriginID  string               `json:"origin_id"`
	Engine    string               `json:"engine"`
	Method    string               `json:"method"`
	Probe     Probe                `json:"probe"`
	Overshoot float64              `json:"overshoot_percent"`
	PeakTime  float64              `json:"peak_time_s"`
	PeakValue float64              `json:"peak_value"`
	Settling  *float64             `json:"settling_delay_s,omitempty"`
	Held      float64              `json:"observed_hold_s"`
	Reason    string               `json:"settling_reason"`
	Samples   int                  `json:"samples"`
}

func cloneStepResponse(in *StepResponse) *StepResponse {
	if in == nil {
		return nil
	}
	out := *in
	if in.Settling != nil {
		v := *in.Settling
		out.Settling = &v
	}
	return &out
}
func (s *Service) measureStepResponseLocked(p StepResponseParams) (*StepResponse, error) {
	op := s.operations[p.ID]
	if op == nil || op.record == nil {
		return nil, &actionError{"not-ready", "Choose a retained completed recording"}
	}
	record := op.record
	for _, v := range []float64{p.Start, p.End, p.Initial, p.Final, p.Tolerance, p.Hold} {
		if !finite(v) {
			return nil, fmt.Errorf("step response settings must be finite")
		}
	}
	if p.Start < 0 || p.End <= p.Start || p.End > record.metadata.Duration || p.Tolerance < .0001 || p.Tolerance > .5 || p.Hold <= 0 || p.Hold > p.End-p.Start || math.Abs(p.Initial) > 1e6 || math.Abs(p.Final) > 1e6 || p.Initial == p.Final {
		return nil, fmt.Errorf("require a covered positive interval, distinct initial/final references within ±1e6, tolerance 0.0001–0.5 and 0 < hold ≤ interval")
	}
	index := -1
	for k, probe := range record.metadata.Probes {
		if probe.ID == p.ProbeID {
			index = k
		}
	}
	if index < 0 {
		return nil, &actionError{"not-found", "Step-response probe was not captured"}
	}
	rows := record.rows
	if len(rows) < 2 || p.Start < rows[0].Time || p.End > rows[len(rows)-1].Time {
		return nil, &actionError{"unavailable", "Step interval must be entirely covered by saved timestamps"}
	}
	type point struct{ at, value float64 }
	points := []point{}
	// Clip positive-time segments and retain duplicate-time left/right limits.
	appendPoint := func(q point) {
		if len(points) == 0 || points[len(points)-1] != q {
			points = append(points, q)
		}
	}
	for k := 1; k < len(rows); k++ {
		a, b := rows[k-1], rows[k]
		if b.Time < p.Start || a.Time > p.End {
			continue
		}
		if b.Time == a.Time {
			if a.Time >= p.Start && a.Time <= p.End {
				appendPoint(point{a.Time, a.Values[index]})
				appendPoint(point{b.Time, b.Values[index]})
			}
			continue
		}
		lo, hi := math.Max(a.Time, p.Start), math.Min(b.Time, p.End)
		if hi < lo {
			continue
		}
		interp := func(at float64) float64 {
			f := (at - a.Time) / (b.Time - a.Time)
			return (1-f)*a.Values[index] + f*b.Values[index]
		}
		appendPoint(point{lo, interp(lo)})
		appendPoint(point{hi, interp(hi)})
	}
	if len(points) < 2 {
		return nil, &actionError{"unavailable", "No saved segments cover this step"}
	}
	out := &StepResponse{Settings: StepResponseSettings(p), OriginID: record.metadata.OperationID, Engine: record.metadata.Engine, Method: StepResponseMethod, Probe: record.metadata.Probes[index], Samples: len(points)}
	delta := p.Final - p.Initial
	progress := func(v float64) float64 { return (v - p.Initial) / delta }
	inside := func(v float64) bool { return math.Abs(progress(v)-1) <= p.Tolerance }
	peak := math.Inf(-1)
	entry := p.Start
	for k, q := range points {
		y := progress(q.value)
		if !finite(y) {
			return nil, fmt.Errorf("step response normalization exceeds numeric range")
		}
		if y > peak {
			peak = y
			out.PeakTime, out.PeakValue = q.at, q.value
		}
		if k == 0 {
			continue
		}
		a := points[k-1]
		if !inside(q.value) {
			entry = q.at
			continue
		}
		if !inside(a.value) { // Last linear arrival at the final-value band, including instantaneous jumps.
			if q.at == a.at {
				entry = q.at
			} else {
				bound := 1 - p.Tolerance
				if progress(a.value) > 1+p.Tolerance {
					bound = 1 + p.Tolerance
				}
				fraction := (bound - progress(a.value)) / (y - progress(a.value))
				entry = a.at + fraction*(q.at-a.at)
			}
		}
	}
	out.Overshoot = 100 * math.Max(0, peak-1)
	if !finite(out.Overshoot) {
		return nil, fmt.Errorf("step overshoot exceeds numeric range")
	}
	if !inside(points[len(points)-1].value) {
		out.Reason = "Final saved value is outside the supplied final-value band"
	} else {
		out.Held = p.End - entry
		if out.Held >= p.Hold {
			delay := entry - p.Start
			out.Settling = &delay
		} else {
			out.Reason = "Capture does not show the requested continuous hold after the last band arrival"
		}
	}
	return out, nil
}
func (s *Service) stepResponseQueryLocked(raw json.RawMessage) (*StepResponse, error) {
	normalized, err := normalizeTimeInput("recording.step-response", raw)
	if err != nil {
		return nil, err
	}
	var p StepResponseParams
	if err = params(normalized, &p, "id", "probe_id", "start_s", "end_s", "initial_value", "final_value", "tolerance_fraction", "minimum_hold_s"); err != nil {
		return nil, err
	}
	return s.measureStepResponseLocked(p)
}
func (s *Service) setStepResponseLocked(raw json.RawMessage) error {
	var p StepResponseParams
	if err := params(raw, &p, "id", "probe_id", "start_s", "end_s", "initial_value", "final_value", "tolerance_fraction", "minimum_hold_s"); err != nil {
		return err
	}
	if p.ID != s.selectedOperation {
		return &actionError{"conflict", "Selected recording changed; refresh before measuring"}
	}
	if _, err := s.selectedRecordLocked(); err != nil {
		return err
	}
	if p.ProbeID == "" {
		zero := p
		zero.ID = ""
		if zero != (StepResponseParams{}) {
			return fmt.Errorf("disable step measurements with empty probe and all zero settings")
		}
		s.recording.stepResponse = nil
		return nil
	}
	result, err := s.measureStepResponseLocked(p)
	if err != nil {
		return err
	}
	s.recording.stepResponse = result
	return nil
}
