package workbench

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Quantity text preserves exact decimal commits instead of crossing a native
// float32 number field. Period input also accepts the shared time-unit grammar.
type PulseTimingParams struct {
	ID        string  `json:"id"`
	Frequency string  `json:"frequency_hz,omitempty"`
	Period    float64 `json:"period_s,omitempty"`
	Duty      string  `json:"duty_percent"`
}

type IdealPulseTimingParams PulseTimingParams

// PulseTiming describes the configured waveform, not a measured live period.
type PulseTiming struct {
	SourceKind    string  `json:"source_kind"`
	Shared        bool    `json:"shared"`
	ID            string  `json:"id"`
	Frequency     float64 `json:"frequency_hz"`
	Period        float64 `json:"period_s"`
	Duty          float64 `json:"duty_percent"`
	FrequencyText string  `json:"frequency_text"`
	DutyText      string  `json:"duty_text"`
}

func pulseTiming(id string, waveform PulseWaveform) PulseTiming {
	period := waveform.HighSeconds + waveform.LowSeconds
	frequency, duty := 1/period, 100*waveform.HighSeconds/period
	return PulseTiming{SourceKind: "finite-pulse-source", ID: id, Frequency: frequency, Period: period, Duty: duty, FrequencyText: strconv.FormatFloat(frequency, 'g', -1, 64) + " Hz", DutyText: strconv.FormatFloat(duty, 'g', -1, 64) + " %"}
}

func (s *Service) pulseTimingsLocked() []PulseTiming {
	out := []PulseTiming{}
	for _, p := range s.graph.Components {
		if p.Kind == "finite-pulse-source" && p.Pulse != nil {
			out = append(out, pulseTiming(p.ID, *p.Pulse))
		} else if p.Kind == "pulse-source" {
			clock := pulseTiming(p.ID, PulseWaveform{HighSeconds: s.transientSettings.HighSeconds, LowSeconds: s.transientSettings.LowSeconds})
			clock.SourceKind, clock.Shared = "pulse-source", true
			out = append(out, clock)
		}
	}
	return out
}

type pulseTimingRequest struct {
	ID          string
	Frequency   float64
	Period      float64
	Duty        float64
	ByFrequency bool
}

func parsePulseTimingRequest(raw json.RawMessage, minimumPeriod, maximumFrequency float64) (pulseTimingRequest, error) {
	fail := func(message string) (pulseTimingRequest, error) {
		return pulseTimingRequest{}, errors.New(message)
	}
	var p PulseTimingParams
	if err := params(raw, &p, "id", "duty_percent"); err != nil {
		return pulseTimingRequest{}, err
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	_, hasFrequency := fields["frequency_hz"]
	_, hasPeriod := fields["period_s"]
	if hasFrequency == hasPeriod {
		return fail("choose exactly one frequency_hz or period_s, plus duty_percent")
	}
	duty, err := parsePulseQuantity(p.Duty, false)
	if err != nil || duty <= 0 || duty >= 100 {
		return fail("duty_percent requires a finite percentage strictly between 0 and 100")
	}
	period := p.Period
	frequency := 0.
	if hasFrequency {
		frequency, err = parsePulseQuantity(p.Frequency, true)
		if err != nil || frequency < .05 || frequency > maximumFrequency {
			return fail(fmt.Sprintf("frequency_hz requires 0.05–%.9g Hz; source phase/sampling limits must also hold", maximumFrequency))
		}
		period = 1 / frequency
	}
	if !finite(period) || period < minimumPeriod || period > 20 {
		return fail(fmt.Sprintf("period_s requires %.9g–20 s; each high/low phase must also fit within 10 s", minimumPeriod))
	}
	return pulseTimingRequest{ID: p.ID, Frequency: frequency, Period: period, Duty: duty, ByFrequency: hasFrequency}, nil
}

func (s *Service) setPulseTimingLocked(raw json.RawMessage) error {
	p, err := parsePulseTimingRequest(raw, .001, 1000)
	if err != nil {
		return err
	}
	part, exists := s.graph.Component(p.ID)
	if !exists {
		return &actionError{"not-found", "Finite pulse source not found"}
	}
	if part.Kind != "finite-pulse-source" || part.Pulse == nil {
		return fmt.Errorf("choose a finite pulse source from pulse_timings")
	}
	current := pulseTiming(part.ID, *part.Pulse)
	period := p.Period
	if p.ByFrequency && p.Frequency == current.Frequency {
		// Preserve the actual configured period across reciprocal rounding.
		period = current.Period
	}
	if period == current.Period && p.Duty == current.Duty {
		return nil
	}
	next := *part.Pulse
	next.HighSeconds = period * (p.Duty / 100)
	next.LowSeconds = period - next.HighSeconds
	if err := next.Validate(); err != nil {
		return fmt.Errorf("timing cannot fit the retained rise %.9g s / fall %.9g s and phase limits: %w", next.RiseSeconds, next.FallSeconds, err)
	}
	return s.setPulseLocked(PulseParams{ID: part.ID, Waveform: next})
}

func (s *Service) setIdealPulseTimingLocked(raw json.RawMessage) error {
	p, err := parsePulseTimingRequest(raw, 2e-6, 500000)
	if err != nil {
		return err
	}
	part, exists := s.graph.Component(p.ID)
	if !exists {
		return &actionError{"not-found", "Ideal square source not found"}
	}
	if part.Kind != "pulse-source" {
		return fmt.Errorf("choose an ideal square source with shared timing")
	}
	next := s.transientSettings
	current := pulseTiming(part.ID, PulseWaveform{HighSeconds: next.HighSeconds, LowSeconds: next.LowSeconds})
	period := p.Period
	if p.ByFrequency && p.Frequency == current.Frequency {
		period = current.Period
	}
	if period != current.Period || p.Duty != current.Duty {
		next.HighSeconds = period * (p.Duty / 100)
		next.LowSeconds = period - next.HighSeconds
	}
	if err := next.validateForGraph(s.graph, true); err != nil {
		return fmt.Errorf("shared pulse timing must fit the current sample step and phase limits; sampling/playback are not changed: %w", err)
	}
	s.transientSettings = next
	return nil
}
