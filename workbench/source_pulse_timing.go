package workbench

import (
	"encoding/json"
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

// PulseTiming describes the configured waveform, not a measured live period.
type PulseTiming struct {
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
	return PulseTiming{ID: id, Frequency: frequency, Period: period, Duty: duty, FrequencyText: strconv.FormatFloat(frequency, 'g', -1, 64) + " Hz", DutyText: strconv.FormatFloat(duty, 'g', -1, 64) + " %"}
}

func (s *Service) pulseTimingsLocked() []PulseTiming {
	out := []PulseTiming{}
	for _, p := range s.graph.Components {
		if p.Kind == "finite-pulse-source" && p.Pulse != nil {
			out = append(out, pulseTiming(p.ID, *p.Pulse))
		}
	}
	return out
}

func (s *Service) setPulseTimingLocked(raw json.RawMessage) error {
	var p PulseTimingParams
	if err := params(raw, &p, "id", "duty_percent"); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	_, hasFrequency := fields["frequency_hz"]
	_, hasPeriod := fields["period_s"]
	if hasFrequency == hasPeriod {
		return fmt.Errorf("choose exactly one frequency_hz or period_s, plus duty_percent")
	}
	duty, err := parsePulseQuantity(p.Duty, false)
	if err != nil || duty <= 0 || duty >= 100 {
		return fmt.Errorf("duty_percent requires a finite percentage strictly between 0 and 100")
	}
	period := p.Period
	frequency := 0.
	if hasFrequency {
		frequency, err = parsePulseQuantity(p.Frequency, true)
		if err != nil || frequency < .05 || frequency > 1000 {
			return fmt.Errorf("frequency_hz requires 0.05–1000 Hz; phases and retained rise/fall must also fit")
		}
		period = 1 / frequency
	}
	if !finite(period) || period < .001 || period > 20 {
		return fmt.Errorf("period_s requires 1 ms–20 s; each high/low phase must also fit within 10 s")
	}
	part, exists := s.graph.Component(p.ID)
	if !exists {
		return &actionError{"not-found", "Finite pulse source not found"}
	}
	if part.Kind != "finite-pulse-source" || part.Pulse == nil {
		return fmt.Errorf("choose a finite pulse source from pulse_timings")
	}
	current := pulseTiming(part.ID, *part.Pulse)
	if hasFrequency && frequency == current.Frequency {
		// Preserve the actual configured period across reciprocal rounding.
		period = current.Period
	}
	if period == current.Period && duty == current.Duty {
		return nil
	}
	next := *part.Pulse
	next.HighSeconds = period * (duty / 100)
	next.LowSeconds = period - next.HighSeconds
	if err := next.Validate(); err != nil {
		return fmt.Errorf("timing cannot fit the retained rise %.9g s / fall %.9g s and phase limits: %w", next.RiseSeconds, next.FallSeconds, err)
	}
	return s.setPulseLocked(PulseParams{ID: part.ID, Waveform: next})
}
