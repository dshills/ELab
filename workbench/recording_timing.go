package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const MaxTimingPage = 100

type RecordingTimingParams struct {
	ID          string  `json:"id"`
	ProbeID     string  `json:"probe_id"`
	Level       string  `json:"level"`
	SecondLevel string  `json:"second_level,omitempty"`
	Direction   string  `json:"direction"`
	Start       float64 `json:"start_s"`
	End         float64 `json:"end_s"`
	Offset      int     `json:"offset,omitempty"`
	Limit       int     `json:"limit,omitempty"`
}

// Separate numeric output times from command-only unit-time input schemas.
type RecordingTimingSettings RecordingTimingParams

type TimingSample struct {
	Index int     `json:"index"`
	Time  float64 `json:"time_s"`
	Value float64 `json:"value"`
}
type ThresholdCrossing struct {
	Direction string       `json:"direction"`
	Before    TimingSample `json:"before"`
	After     TimingSample `json:"after"`
}
type TimingSpacing struct {
	First   ThresholdCrossing `json:"first"`
	Second  ThresholdCrossing `json:"second"`
	Minimum float64           `json:"minimum_s"`
	Maximum float64           `json:"maximum_s"`
}
type RecordingTiming struct {
	Settings       RecordingTimingSettings `json:"settings"`
	OriginID       string                  `json:"origin_id"`
	Engine         string                  `json:"engine"`
	Probe          Probe                   `json:"probe"`
	Level          float64                 `json:"level_si"`
	Duration       float64                 `json:"duration_s"`
	SampleCount    int                     `json:"sample_count"`
	WindowSamples  int                     `json:"window_sample_count"`
	Total          int                     `json:"total_crossings"`
	NextOffset     int                     `json:"next_offset"`
	Crossings      []ThresholdCrossing     `json:"crossings"`
	RisingSpacing  *TimingSpacing          `json:"first_rising_spacing,omitempty"`
	FallingSpacing *TimingSpacing          `json:"first_falling_spacing,omitempty"`
	HighPulse      *TimingSpacing          `json:"first_high_pulse,omitempty"`
	LowPulse       *TimingSpacing          `json:"first_low_pulse,omitempty"`
	Cycle          *TimingCycle            `json:"first_cycle,omitempty"`
	Transitions    *TimingTransitions      `json:"transitions,omitempty"`
}

func parseThresholdLevel(text, unit string) (float64, error) {
	if len(text) == 0 || len(text) > 128 {
		return 0, fmt.Errorf("threshold level requires 1–128 bytes")
	}
	value := strings.TrimSpace(text)
	scale := 1.
	// Units must match the captured probe. Bare numbers are signed SI.
	if strings.HasSuffix(value, unit) {
		value = strings.TrimSpace(strings.TrimSuffix(value, unit))
		for _, prefix := range []struct {
			name  string
			scale float64
		}{{"m", 1e-3}, {"u", 1e-6}, {"µ", 1e-6}, {"μ", 1e-6}} {
			if strings.HasSuffix(value, prefix.name) {
				value = strings.TrimSpace(strings.TrimSuffix(value, prefix.name))
				scale = prefix.scale
				break
			}
		}
	}
	mantissa := value
	if i := strings.IndexAny(mantissa, "eE"); i >= 0 {
		mantissa = mantissa[:i]
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || number == 0 && strings.ContainsAny(mantissa, "123456789") || !finite(number) || !finite(number*scale) || math.Abs(number*scale) > 1e12 || number != 0 && number*scale == 0 {
		return 0, fmt.Errorf("enter a finite threshold in %s, optionally with m/u/µ/μ prefix; magnitude at most 1e12 SI", unit)
	}
	return number * scale, nil
}
func (s *Service) measureRecordingTimingLocked(p RecordingTimingParams) (*RecordingTiming, error) {
	op := s.operations[p.ID]
	if op == nil || op.record == nil {
		return nil, &actionError{"not-ready", "Choose a retained completed recording for threshold timing"}
	}
	record := op.record
	if !finite(p.Start) || !finite(p.End) || p.Start < 0 || p.Start > p.End || p.End > record.metadata.Duration {
		return nil, fmt.Errorf("timing interval requires 0 ≤ start_s ≤ end_s ≤ captured duration")
	}
	if p.Direction != "both" && p.Direction != "rising" && p.Direction != "falling" {
		return nil, fmt.Errorf("timing direction must be rising, falling or both")
	}
	if p.Limit == 0 {
		p.Limit = 8
	}
	if p.Limit < 1 || p.Limit > MaxTimingPage || p.Offset < 0 {
		return nil, fmt.Errorf("timing pages require nonnegative offset and limit 1–100")
	}
	index := -1
	for i, probe := range record.metadata.Probes {
		if probe.ID == p.ProbeID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, &actionError{"not-found", "Timing probe was not captured in this recording"}
	}
	probe := record.metadata.Probes[index]
	level, err := parseThresholdLevel(p.Level, probe.Unit)
	if err != nil {
		return nil, err
	}
	out := &RecordingTiming{Settings: RecordingTimingSettings(p), OriginID: record.metadata.OperationID, Engine: record.metadata.Engine, Probe: probe, Level: level, Duration: record.metadata.Duration, SampleCount: len(record.rows), Crossings: []ThresholdCrossing{}}
	if p.SecondLevel != "" {
		second, err := parseThresholdLevel(p.SecondLevel, probe.Unit)
		if err != nil {
			return nil, err
		}
		if second == level {
			return nil, fmt.Errorf("transition levels must be distinct in SI units")
		}
		out.Transitions = &TimingTransitions{Lower: math.Min(level, second), Upper: math.Max(level, second)}
	}
	var transitions timingTransitionCollector
	first := map[string]ThresholdCrossing{}
	var pulses timingPulseCollector
	for i, row := range record.rows {
		// Strict interval membership: include only brackets whose two stored rows
		// are inside the requested interval; never fabricate a boundary sample.
		if row.Time >= p.Start && row.Time <= p.End {
			out.WindowSamples++
		}
		if i == 0 {
			continue
		}
		previous := record.rows[i-1]
		if previous.Time < p.Start || row.Time > p.End {
			continue
		}
		if out.Transitions != nil {
			transitions.observe(out.Transitions, sampledArrival(i, previous, row, index, out.Transitions.Lower), sampledArrival(i, previous, row, index, out.Transitions.Upper))
		}
		arrival := sampledArrival(i, previous, row, index, level)
		if arrival == nil {
			continue
		}
		crossing := *arrival
		direction := crossing.Direction
		// Pulse/cycle summaries use both directions across the whole interval;
		// direction and paging continue to filter the arrival list and spacing.
		pulses.observe(out, crossing)
		if p.Direction != "both" && p.Direction != direction {
			continue
		}
		if old, exists := first[direction]; !exists {
			first[direction] = crossing
		} else {
			target := &out.RisingSpacing
			if direction == "falling" {
				target = &out.FallingSpacing
			}
			if *target == nil {
				*target = &TimingSpacing{First: old, Second: crossing, Minimum: math.Max(0, crossing.Before.Time-old.After.Time), Maximum: crossing.After.Time - old.Before.Time}
			}
		}
		if out.Total >= p.Offset && len(out.Crossings) < p.Limit {
			out.Crossings = append(out.Crossings, crossing)
		}
		out.Total++
	}
	if p.Offset > out.Total {
		return nil, fmt.Errorf("timing offset exceeds crossing count %d", out.Total)
	}
	out.NextOffset = p.Offset + len(out.Crossings)
	return out, nil
}
func (s *Service) recordingTimingQueryLocked(raw json.RawMessage) (*RecordingTiming, error) {
	normalized, err := normalizeTimeInput("recording.timing", raw)
	if err != nil {
		return nil, err
	}
	var p RecordingTimingParams
	if err := params(normalized, &p, "id", "probe_id", "level", "direction", "start_s", "end_s"); err != nil {
		return nil, err
	}
	return s.measureRecordingTimingLocked(p)
}
func (s *Service) setRecordingTimingLocked(raw json.RawMessage) error {
	var p RecordingTimingParams
	if err := params(raw, &p, "id", "probe_id", "level", "direction", "start_s", "end_s"); err != nil {
		return err
	}
	if p.ID != s.selectedOperation {
		return &actionError{"conflict", "Selected recording changed; refresh before measuring timing"}
	}
	if _, err := s.selectedRecordLocked(); err != nil {
		return err
	}
	if p.ProbeID == "" {
		if p.SecondLevel != "" || p.Level != "0" || p.Direction != "both" || p.Start != 0 || p.End != 0 || p.Offset != 0 || p.Limit != 0 {
			return fmt.Errorf("clear timing with empty second_level, level 0, both directions and zero interval/page fields")
		}

		s.recording.timing = nil
		return nil
	}
	result, err := s.measureRecordingTimingLocked(p)
	if err != nil {
		return err
	}
	s.recording.timing = result
	return nil
}
func cloneRecordingTiming(in *RecordingTiming) *RecordingTiming {
	if in == nil {
		return nil
	}
	out := *in
	out.Crossings = append([]ThresholdCrossing{}, in.Crossings...)
	out.RisingSpacing = cloneTimingSpacing(in.RisingSpacing)
	out.FallingSpacing = cloneTimingSpacing(in.FallingSpacing)
	out.HighPulse = cloneTimingSpacing(in.HighPulse)
	out.LowPulse = cloneTimingSpacing(in.LowPulse)
	out.Cycle = cloneTimingCycle(in.Cycle)
	out.Transitions = cloneTimingTransitions(in.Transitions)
	return &out
}

func addRecordingTimingDetails(id string, schema map[string]any) bool {
	if id != "recording.timing" && id != "recording.timing.set" {
		return false
	}
	inputProperty(schema, "direction")["enum"] = []string{"both", "rising", "falling"}
	level := inputProperty(schema, "level")
	level["minLength"], level["maxLength"], level["x-max-utf8-bytes"] = 1, 128, 128
	level["description"] = "Signed SI text matching captured probe units, optionally m/u/µ/μ prefix with explicit unit (for example 2.5 V or 1 mA); finite magnitude at most 1e12."
	second := inputProperty(schema, "second_level")
	second["maxLength"], second["x-max-utf8-bytes"] = 128, 128
	second["description"] = "Optional second level in captured probe units; omitted/empty disables transition timing. Distinct from level after SI normalization. Both levels are ordered low/high; first complete observed low-to-high and high-to-low traversals use saved brackets, independently of arrival filtering/pages. Returning to the starting threshold aborts an unfinished traversal."
	inputProperty(schema, "offset")["minimum"] = 0
	inputProperty(schema, "limit")["minimum"], inputProperty(schema, "limit")["maximum"], inputProperty(schema, "limit")["default"] = 0, MaxTimingPage, 8
	inputProperty(schema, "limit")["description"] = "Omit or zero for eight arrivals; otherwise 1–100."
	for _, name := range []string{"start_s", "end_s"} {
		inputProperty(schema, name)["x-normalized-si-range"] = map[string]float64{"minimum": 0, "maximum": 10}
	}
	schema["x-relations"] = []map[string]any{{"operator": "0<=start_s<=end_s<=captured duration", "fields": []string{"start_s", "end_s", "id"}}, {"operator": "offset<=total sampled arrivals", "fields": []string{"offset"}}}
	schema["x-inventory"] = map[string]string{"ids": "snapshot.operations with completed status", "probe_ids": "operation.inspect metadata.probes (captured, not live)"}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "sampled-threshold-timing", Preconditions: []string{"Retained completed capture and a captured probe; DC/empty intervals can yield zero arrivals", "Set ID must match selected recording; empty probe clears only with empty second_level, level 0, both directions and zero interval/page fields"}, Effects: []string{"Rising: before < level and after >= level; falling: before > level and after <= level; equal plateaus do not repeatedly trigger", "Only adjacent exact saved rows wholly inside strict interval; duplicates retain edge order; no interpolation, resampling or hidden-transition guarantee", "First same-direction spacing bounds subtract observed sample brackets; not a certification of periodic behavior", "First high/low widths use consecutive opposite observed arrivals; first cycle requires consecutive rising/falling/rising. Pulse/cycle summaries use both directions regardless of list filter/page", "Cycle high_fraction bounds retain the shared rising-time dependence: (fall.before-rise.after)/(next.after-rise.after) through (fall.after-rise.before)/(next.before-rise.before). Omitted if minimum cycle duration is not positive; reason explains why", "Optional second_level enables sampled low/high transition timing. Levels must differ after SI normalization; unfinished traversal resets on an opposite arrival at its starting threshold. No monotonicity or hidden-transition guarantee", "Query is pure; set caches a bounded derived page as session/view state; selection/reopen clears; no live/history/project/solver changes"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Invalid level/unit/direction/interval/page or missing required fields"}, {"not-ready", "admission", "Capture missing or incomplete"}, {"not-found", "admission", "Probe not captured"}, {"conflict", "admission", "Set ID differs from selected capture"}}, Completion: &commandCompletionDetail{Mode: "bounded in-memory scan of at most 4096 saved rows; no worker/file IO", Observe: []string{"recording.timing", "project.inspect"}}}
	schema["examples"] = []any{RecordingTimingParams{ID: "verify-1", ProbeID: "v:C1", Level: "2.5 V", Direction: "rising", End: .02}, RecordingTimingParams{ID: "verify-1", ProbeID: "v:C1", Level: "500 mV", SecondLevel: "4.5 V", Direction: "both", End: .04}}
	if id == "recording.timing.set" {
		schema["examples"] = append(schema["examples"].([]any), RecordingTimingParams{ID: "verify-1", Level: "0", Direction: "both"})
	}
	return true
}
