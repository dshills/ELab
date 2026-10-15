package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) waveformControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	nodes := []ui.Node{}
	if p.Sine != nil {
		cfg := *p.Sine
		nodes = append(nodes, section("sine-title", "Sine waveform"), quiet("sine-range", fmt.Sprintf("%.5g to %.5g V · %.5g Hz", cfg.Offset-cfg.Amplitude, cfg.Offset+cfg.Amplitude, cfg.Frequency)))
		for _, field := range []struct {
			id, label             string
			value, min, max, step float64
			set                   func(*workbench.SineWaveform, float64)
		}{
			{"amplitude", "Amplitude · V", cfg.Amplitude, 0, 12, .01, func(w *workbench.SineWaveform, v float64) { w.Amplitude = v }},
			{"offset", "Offset · V", cfg.Offset, -12, 12, .01, func(w *workbench.SineWaveform, v float64) { w.Offset = v }},
			{"frequency", "Frequency · Hz", cfg.Frequency, .1, 1000, .1, func(w *workbench.SineWaveform, v float64) { w.Frequency = v }},
			{"phase", "Phase · degrees", cfg.Phase, -360, 360, 1, func(w *workbench.SineWaveform, v float64) { w.Phase = v }},
		} {
			field := field
			nodes = append(nodes, quiet("sine-label-"+field.id, field.label), ui.Node{ID: "sine-" + field.id, Kind: ui.NumberField, Number: float32(field.value), Numeric: ui.Numeric{Min: float32(field.min), Max: float32(field.max), Step: float32(field.step)}, Accessibility: ui.Accessibility{Name: field.label}, Change: func(e ui.ChangeEvent) {
				next := cfg
				field.set(&next, float64(e.Number))
				s.invoke("source.sine.set", workbench.SineParams{ID: p.ID, Waveform: next})
			}})
		}
	}
	if p.Pulse != nil {
		cfg := *p.Pulse
		nodes = append(nodes, section("pulse-title", "Finite pulse waveform"), quiet("pulse-period", fmt.Sprintf("%.5g Hz · %.4g%% high phase", 1/(cfg.HighSeconds+cfg.LowSeconds), 100*cfg.HighSeconds/(cfg.HighSeconds+cfg.LowSeconds))), quiet("pulse-phase-help", "Rise is included in the high phase; fall is included in the low phase. Each source has its own timing."))
		nodes = append(nodes, s.pulseClockControls(state, p)...)
		for _, field := range []struct {
			id, label                    string
			value, scale, min, max, step float64
			set                          func(*workbench.PulseWaveform, float64)
		}{
			{"low-v", "Low level · V", cfg.LowVoltage, 1, -12, 12, .01, func(w *workbench.PulseWaveform, v float64) { w.LowVoltage = v }},
			{"high-v", "High level · V", cfg.HighVoltage, 1, -12, 12, .01, func(w *workbench.PulseWaveform, v float64) { w.HighVoltage = v }},
			{"high", "High phase", cfg.HighSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.HighSeconds = v }},
			{"low", "Low phase", cfg.LowSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.LowSeconds = v }},
			{"rise", "Rise time", cfg.RiseSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.RiseSeconds = v }},
			{"fall", "Fall time", cfg.FallSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.FallSeconds = v }},
		} {
			field := field
			if field.scale == 1000 {
				name := field.id + "_s"
				nodes = append(nodes, quiet("pulse-label-"+field.id, field.label), timeField("pulse-"+field.id, field.label, field.value, false, func(text string) {
					s.invokeAtCommandRevision("source.pulse.set", map[string]any{"id": p.ID, "waveform": timeParams(cfg, name, text)}, state.CommandRevision)
				}))
				continue
			}
			nodes = append(nodes, quiet("pulse-label-"+field.id, field.label), ui.Node{ID: "pulse-" + field.id, Kind: ui.NumberField, Number: float32(field.value * field.scale), Numeric: ui.Numeric{Min: float32(field.min), Max: float32(field.max), Step: float32(field.step)}, Accessibility: ui.Accessibility{Name: field.label}, Change: func(e ui.ChangeEvent) {
				next := cfg
				field.set(&next, float64(e.Number)/field.scale)
				s.invoke("source.pulse.set", workbench.PulseParams{ID: p.ID, Waveform: next})
			}})
		}
	}
	if workbench.IsWaveformSource(p.Kind) {
		nodes = append(nodes, quiet("waveform-source-model", "Ideal voltage source · no current limit, output resistance or physical generator rating."))
		for _, warning := range state.SourceWarnings {
			if warning.ObjectID == p.ID {
				nodes = append(nodes, quiet("source-sampling-warning", warning.Message), s.action("fit-waveform-sampling", "Fit waveform sampling", "simulation.sampling.fit", struct{}{}))
			}
		}
	}
	if p.Kind == "pulse-source" {
		nodes = append(nodes, section("pulse-title", "Shared ideal-square clock"), quiet("legacy-pulse-model", "Instantaneous edges. Timing changes affect every ideal-square source; sample step and playback stay fixed."))
		nodes = append(nodes, s.pulseClockControls(state, p)...)
	}
	return nodes
}

func (s *screen) pulseClockControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	nodes := []ui.Node{}
	for _, clock := range state.PulseTimings {
		if clock.ID != p.ID {
			continue
		}
		nodes = append(nodes, quiet("pulse-frequency-label", "Frequency · Hz"), ui.Node{ID: "pulse-frequency", Kind: ui.TextField, Text: clock.FrequencyText, MinWidth: 90, Accessibility: ui.Accessibility{Name: "Pulse source frequency", Help: "Enter Hz, kHz or mHz; retained transition or shared sampling limits must fit."}, Change: func(e ui.ChangeEvent) {
			s.invokePulseClockAtRevision(clock.Shared, workbench.PulseTimingParams{ID: p.ID, Frequency: e.Text, Duty: clock.DutyText}, state.CommandRevision)
		}})
		nodes = append(nodes, quiet("pulse-period-time-label", "Period"), timeField("pulse-period-time", "Pulse source period", clock.Period, false, func(text string) {
			s.invokePulseClockAtRevision(clock.Shared, map[string]any{"id": p.ID, "period_s": text, "duty_percent": clock.DutyText}, state.CommandRevision)
		}))
		nodes = append(nodes, quiet("pulse-duty-label", "High phase · %"), ui.Node{ID: "pulse-duty", Kind: ui.TextField, Text: clock.DutyText, MinWidth: 90, Accessibility: ui.Accessibility{Name: "Pulse source high phase percentage", Help: "Enter a percentage between 0 and 100; the high phase includes any finite rise."}, Change: func(e ui.ChangeEvent) {
			s.invokePulseClockAtRevision(clock.Shared, workbench.PulseTimingParams{ID: p.ID, Period: clock.Period, Duty: e.Text}, state.CommandRevision)
		}})
		break
	}
	return nodes
}

func (s *screen) invokePulseClockAtRevision(shared bool, params any, revision uint64) {
	if shared {
		s.invokeAtCommandRevision("simulation.pulse.timing.set", params, revision)
	} else {
		s.invokeAtCommandRevision("source.pulse.timing.set", params, revision)
	}
}
