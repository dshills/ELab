package main

import (
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) waveformControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	nodes := []ui.Node{}
	if p.Sine != nil {
		cfg := *p.Sine
		nodes = append(nodes, section("sine-title", "Sine waveform"), quiet("sine-range", fmt.Sprintf("%.5g to %.5g V · %.5g Hz", cfg.Offset-cfg.Amplitude, cfg.Offset+cfg.Amplitude, cfg.Frequency)))
		for _, field := range []struct {
			id, label, name, unit, help string
			value                       float64
		}{
			{"amplitude", "Amplitude · V", "amplitude_v", "V", "Enter V, mV or uV. Amplitude plus absolute offset must stay within 12 V.", cfg.Amplitude},
			{"offset", "Offset · V", "offset_v", "V", "Enter V, mV or uV. Offset plus/minus amplitude must stay within ±12 V.", cfg.Offset},
			{"frequency", "Frequency · Hz", "frequency_hz", "Hz", "Enter Hz, kHz or mHz; 0.1–1000 Hz. Output sampling stays fixed.", cfg.Frequency},
			{"phase", "Phase · degrees", "phase_degrees", "deg", "Enter degrees using deg or °; −360 to 360.", cfg.Phase},
		} {
			nodes = append(nodes, quiet("sine-label-"+field.id, field.label), ui.Node{ID: "sine-" + field.id, Kind: ui.TextField, Text: strconv.FormatFloat(field.value, 'g', -1, 64) + " " + field.unit, MinWidth: 90, Accessibility: ui.Accessibility{Name: field.label, Help: field.help}, Change: func(e ui.ChangeEvent) {
				s.invokeAtCommandRevision("source.sine.set", map[string]any{"id": p.ID, "waveform": textFieldParams(cfg, field.name, e.Text)}, state.CommandRevision)
			}})
		}
	}
	if p.Pulse != nil {
		cfg := *p.Pulse
		nodes = append(nodes, section("pulse-title", "Finite pulse waveform"), quiet("pulse-period", fmt.Sprintf("%.5g Hz · %.4g%% high phase", 1/(cfg.HighSeconds+cfg.LowSeconds), 100*cfg.HighSeconds/(cfg.HighSeconds+cfg.LowSeconds))), quiet("pulse-phase-help", "Rise is included in the high phase; fall is included in the low phase. Each source has its own timing."))
		nodes = append(nodes, s.pulseClockControls(state, p)...)
		for _, field := range []struct {
			id, label, name string
			value           float64
			time            bool
		}{
			{"low-v", "Low level · V", "low_v", cfg.LowVoltage, false},
			{"high-v", "High level · V", "high_v", cfg.HighVoltage, false},
			{"high", "High phase", "high_s", cfg.HighSeconds, true},
			{"low", "Low phase", "low_s", cfg.LowSeconds, true},
			{"rise", "Rise time", "rise_s", cfg.RiseSeconds, true},
			{"fall", "Fall time", "fall_s", cfg.FallSeconds, true},
		} {
			commit := func(text string) {
				s.invokeAtCommandRevision("source.pulse.set", map[string]any{"id": p.ID, "waveform": textFieldParams(cfg, field.name, text)}, state.CommandRevision)
			}
			node := ui.Node{ID: "pulse-" + field.id, Kind: ui.TextField, Text: strconv.FormatFloat(field.value, 'g', -1, 64) + " V", MinWidth: 90, Accessibility: ui.Accessibility{Name: field.label, Help: "Enter V, mV or uV. Levels must be ordered low ≤ high within ±12 V."}, Change: func(e ui.ChangeEvent) { commit(e.Text) }}
			if field.time {
				node = timeField("pulse-"+field.id, field.label, field.value, false, commit)
			}
			nodes = append(nodes, quiet("pulse-label-"+field.id, field.label), node)
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
