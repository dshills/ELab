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
		for _, field := range []struct {
			id, label                    string
			value, scale, min, max, step float64
			set                          func(*workbench.PulseWaveform, float64)
		}{
			{"low-v", "Low level · V", cfg.LowVoltage, 1, -12, 12, .01, func(w *workbench.PulseWaveform, v float64) { w.LowVoltage = v }},
			{"high-v", "High level · V", cfg.HighVoltage, 1, -12, 12, .01, func(w *workbench.PulseWaveform, v float64) { w.HighVoltage = v }},
			{"high", "High phase · ms", cfg.HighSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.HighSeconds = v }},
			{"low", "Low phase · ms", cfg.LowSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.LowSeconds = v }},
			{"rise", "Rise · ms", cfg.RiseSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.RiseSeconds = v }},
			{"fall", "Fall · ms", cfg.FallSeconds, 1000, .001, 10000, .001, func(w *workbench.PulseWaveform, v float64) { w.FallSeconds = v }},
		} {
			field := field
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
		nodes = append(nodes, quiet("legacy-pulse-model", "Ideal square: instantaneous edges and shared experiment timing. Finite pulse sources have independent rise/fall settings."))
	}
	return nodes
}
