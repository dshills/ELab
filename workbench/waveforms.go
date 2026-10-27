package workbench

import (
	"fmt"
	"math"
)

const SineSourceModelID = "elab.ideal-sine-source.v1"
const FinitePulseModelID = "elab.finite-pulse-source.v1"
const WaveformAnalysis = "Waveform network transient"
const WaveformEngineID = "elab.network-waveforms-mna-sdirk2.v1"

type WaveformModel struct {
	ID                string  `json:"id"`
	SineModelID       string  `json:"sine_model_id"`
	PulseModelID      string  `json:"pulse_model_id"`
	MinimumTransition float64 `json:"minimum_transition_s"`
	MaximumFrequency  float64 `json:"maximum_frequency_hz"`
	MaximumMagnitude  float64 `json:"maximum_magnitude_v"`
	Limitations       string  `json:"limitations"`
}

func SourceWaveformModel() WaveformModel {
	return WaveformModel{"elab.source-waveforms.v1", SineSourceModelID, FinitePulseModelID, 1e-6, 1000, 12, "Ideal grounded voltage sources with per-component periodic sine or linear pulse ramps. No output resistance, current limit, battery discharge, noise or physical generator identity. Finite pulse high/low phase lengths include rise/fall respectively. Output sampling can alias; internal integration resolves waveform features within its work budget."}
}

type SineWaveform struct {
	Amplitude float64 `json:"amplitude_v"`
	Offset    float64 `json:"offset_v"`
	Frequency float64 `json:"frequency_hz"`
	Phase     float64 `json:"phase_degrees"`
}
type PulseWaveform struct {
	HighVoltage float64 `json:"high_v"`
	LowVoltage  float64 `json:"low_v"`
	HighSeconds float64 `json:"high_s"`
	LowSeconds  float64 `json:"low_s"`
	RiseSeconds float64 `json:"rise_s"`
	FallSeconds float64 `json:"fall_s"`
}

func DefaultSineWaveform() SineWaveform   { return SineWaveform{2.5, 0, 10, 0} }
func DefaultPulseWaveform() PulseWaveform { return PulseWaveform{5, 0, .05, .05, .001, .001} }
func (p SineWaveform) Validate() error {
	if !finite(p.Amplitude) || !finite(p.Offset) || p.Amplitude < 0 || math.Abs(p.Offset)+p.Amplitude > 12 {
		return fmt.Errorf("sine amplitude must be nonnegative; offset ± amplitude must remain within ±12 V")
	}
	if !finite(p.Frequency) || p.Frequency < .1 || p.Frequency > 1000 || !finite(p.Phase) || math.Abs(p.Phase) > 360 {
		return fmt.Errorf("sine frequency must be 0.1–1000 Hz and phase within ±360 degrees")
	}
	return nil
}
func (p PulseWaveform) Validate() error {
	for _, v := range []float64{p.HighVoltage, p.LowVoltage, p.HighSeconds, p.LowSeconds, p.RiseSeconds, p.FallSeconds} {
		if !finite(v) {
			return fmt.Errorf("pulse parameters must be finite")
		}
	}
	if p.LowVoltage > p.HighVoltage || math.Abs(p.LowVoltage) > 12 || math.Abs(p.HighVoltage) > 12 {
		return fmt.Errorf("pulse levels must be ordered low ≤ high and within ±12 V")
	}
	if p.HighSeconds < 1e-6 || p.LowSeconds < 1e-6 || p.HighSeconds > 10 || p.LowSeconds > 10 || p.HighSeconds+p.LowSeconds < .001 || p.RiseSeconds < 1e-6 || p.FallSeconds < 1e-6 || p.RiseSeconds > p.HighSeconds || p.FallSeconds > p.LowSeconds {
		return fmt.Errorf("pulse phase lengths must be 1 µs–10 s, period at least 1 ms; rise/fall must be 1 µs or more and fit their high/low phase")
	}
	return nil
}

type SineParams struct {
	ID       string       `json:"id"`
	Waveform SineWaveform `json:"waveform"`
}
type PulseParams struct {
	ID       string        `json:"id"`
	Waveform PulseWaveform `json:"waveform"`
}

func (s *Service) setSineLocked(p SineParams) error {
	if err := p.Waveform.Validate(); err != nil {
		return err
	}
	for i := range s.graph.Components {
		c := &s.graph.Components[i]
		if c.ID == p.ID {
			if c.Kind != "sine-source" {
				return fmt.Errorf("choose a sine source")
			}
			v := p.Waveform
			c.Sine = &v
			c.Value = v.Amplitude
			return nil
		}
	}
	return fmt.Errorf("choose a sine source")
}
func (s *Service) setPulseLocked(p PulseParams) error {
	if err := p.Waveform.Validate(); err != nil {
		return err
	}
	for i := range s.graph.Components {
		c := &s.graph.Components[i]
		if c.ID == p.ID {
			if c.Kind != "finite-pulse-source" {
				return fmt.Errorf("choose a finite pulse source")
			}
			v := p.Waveform
			c.Pulse = &v
			c.Value = v.HighVoltage
			return nil
		}
	}
	return fmt.Errorf("choose a finite pulse source")
}
func IsWaveformSource(kind string) bool {
	return kind == "sine-source" || kind == "finite-pulse-source"
}
func IsVoltageSource(kind string) bool {
	return kind == "battery" || kind == "dc-source" || kind == "pulse-source" || IsWaveformSource(kind)
}
func sourceVoltageAt(p Component, t *Transient, at float64, primary bool) float64 {
	mode := "auto"
	if primary {
		mode = t.Mode
	}
	if p.Sine != nil {
		w := p.Sine
		if mode == "high" {
			return w.Offset + w.Amplitude
		}
		if mode == "low" {
			return w.Offset - w.Amplitude
		}
		phase := math.Mod(at*w.Frequency+w.Phase/360, 1)
		return w.Offset + w.Amplitude*math.Sin(2*math.Pi*phase)
	}
	if p.Pulse != nil {
		w := p.Pulse
		if mode == "high" {
			return w.HighVoltage
		}
		if mode == "low" {
			return w.LowVoltage
		}
		period := w.HighSeconds + w.LowSeconds
		phase := math.Mod(at, period)
		epsilon := period * 1e-12
		if phase >= period-epsilon {
			phase = 0
		}
		if phase < w.RiseSeconds {
			return w.LowVoltage + (w.HighVoltage-w.LowVoltage)*phase/w.RiseSeconds
		}
		if phase < w.HighSeconds {
			return w.HighVoltage
		}
		if phase < w.HighSeconds+w.FallSeconds {
			return w.HighVoltage - (w.HighVoltage-w.LowVoltage)*(phase-w.HighSeconds)/w.FallSeconds
		}
		return w.LowVoltage
	}
	if mode == "low" {
		return 0
	}
	if mode == "high" {
		return p.Value
	}
	if p.Kind == "pulse-source" && !pulseHigh(t.Settings, at) {
		return 0
	}
	return p.Value
}

// Corners belong to individual sources, so different periods cannot be folded
// into the legacy global square-wave clock.
func nextSourceCorner(p Component, t *Transient, at float64, primary bool) float64 {
	if primary && t.Mode != "auto" {
		return math.Inf(1)
	}
	if p.Kind == "pulse-source" {
		return nextPulseEdge(t.Settings, at)
	}
	if p.Pulse == nil || p.Pulse.HighVoltage == p.Pulse.LowVoltage {
		return math.Inf(1)
	}
	w := p.Pulse
	period := w.HighSeconds + w.LowSeconds
	cycle := math.Floor(at / period)
	epsilon := period * 1e-12
	for _, offset := range []float64{w.RiseSeconds, w.HighSeconds, w.HighSeconds + w.FallSeconds, period} {
		corner := cycle*period + offset
		if corner > at+epsilon {
			return corner
		}
	}
	return (cycle+1)*period + w.RiseSeconds
}
func (n *mixedNetwork) nextSourceCorner(t *Transient, at float64) float64 {
	next := math.Inf(1)
	for _, p := range n.waveformComponents() {
		next = math.Min(next, nextSourceCorner(p, t, at, p.ID == n.c.Source.ID && !IsCurrentDevice(p.Kind)))
	}
	return next
}
func (n *mixedNetwork) maximumWaveformStep(t *Transient, at float64) float64 {
	limit := math.Inf(1)
	for _, p := range n.waveformComponents() {
		if p.ID == n.c.Source.ID && !IsCurrentDevice(p.Kind) && t.Mode != "auto" {
			continue
		}
		if p.Sine != nil && p.Sine.Amplitude > 0 {
			limit = math.Min(limit, 1/(64*p.Sine.Frequency))
		}
		if p.Pulse != nil && p.Pulse.HighVoltage != p.Pulse.LowVoltage {
			w := p.Pulse
			phase := math.Mod(at, w.HighSeconds+w.LowSeconds)
			if phase < w.RiseSeconds {
				limit = math.Min(limit, w.RiseSeconds/8)
			} else if phase >= w.HighSeconds && phase < w.HighSeconds+w.FallSeconds {
				limit = math.Min(limit, w.FallSeconds/8)
			}
		}
	}
	return limit
}
func waveformRecommendedStep(p Component) float64 {
	return math.Max(minimumTransientSampleStep, waveformTargetStep(p))
}
