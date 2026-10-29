package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) extendedControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	nodes := []ui.Node{}
	field := func(id, label string, value, min, max, step float64, change func(float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label}, Change: func(e ui.ChangeEvent) {
			if e.Number != float32(value) {
				value := float64(e.Number)
				if p.Kind == "speaker" || workbench.IsBipolarKind(p.Kind) {
					if parsed, err := strconv.ParseFloat(strconv.FormatFloat(value, 'g', -1, 32), 64); err == nil {
						value = parsed
					}
				}
				change(value)
			}
		}})
	}
	if workbench.IsBipolarKind(p.Kind) {
		enabled := p.Device != nil && p.Device.Bipolar != nil
		c := workbench.DefaultTransistorProfile()
		if enabled {
			c = *p.Device.Bipolar
		}
		apply := func(v workbench.TransistorProfile, on bool) {
			s.invokeAtCommandRevision("transistor.profile.set", workbench.TransistorProfileParams{ID: p.ID, Enabled: on, Parameters: v}, state.CommandRevision, state.Document.Generation)
		}
		nodes = append(nodes, section("transistor-profile-title", "Transistor model"), ui.Node{ID: "transistor-profile-enabled", Kind: ui.Checkbox, Text: "Custom transistor model", Checked: enabled, Accessibility: ui.Accessibility{Name: "Custom transistor model", Help: "Use configurable educational transistor parameters instead of the generic profile."}, Change: func(e ui.ChangeEvent) {
			if e.Checked != enabled {
				apply(c, e.Checked)
			}
		}})
		if enabled {
			field("transistor-beta", "Forward gain · β", c.ForwardBeta, 1, 1000, 1, func(v float64) { n := c; n.ForwardBeta = v; apply(n, true) })
			field("transistor-reverse-beta", "Reverse gain", c.ReverseBeta, .1, 100, .1, func(v float64) { n := c; n.ReverseBeta = v; apply(n, true) })
			field("transistor-is", "Transport IS · fA", c.SaturationCurrent*1e15, .001, 1e5, .001, func(v float64) { n := c; n.SaturationCurrent = decimalControlSI(v, -15); apply(n, true) })
			nodes = append(nodes, ui.Node{ID: "transistor-early-enabled", Kind: ui.Checkbox, Text: "Include Early effect", Checked: c.EarlyVoltage > 0, Accessibility: ui.Accessibility{Name: "Include Early effect", Help: "Include the transistor Early effect using the configured Early voltage."}, Change: func(e ui.ChangeEvent) {
				if e.Checked == (c.EarlyVoltage > 0) {
					return
				}
				n := c
				n.EarlyVoltage = 0
				if e.Checked {
					n.EarlyVoltage = 100
				}
				apply(n, true)
			}})
			if c.EarlyVoltage > 0 {
				field("transistor-early", "Early voltage · V", c.EarlyVoltage, 1, 1000, 1, func(v float64) { n := c; n.EarlyVoltage = v; apply(n, true) })
			}
			field("transistor-cbe", "Base–emitter · pF", c.CBE*1e12, 0, 1e6, 1, func(v float64) { n := c; n.CBE = decimalControlSI(v, -12); apply(n, true) })
			field("transistor-cbc", "Base–collector · pF", c.CBC*1e12, 0, 1e6, 1, func(v float64) { n := c; n.CBC = decimalControlSI(v, -12); apply(n, true) })
			nodes = append(nodes, quiet("transistor-profile-help", "Fixed 27 °C · constant junction capacitance\nBiased startup · uses existing storage budget\nEducational parameters; no manufacturer identity."))
		}
	}
	if p.Kind == "speaker" {
		if p.Device == nil || p.Device.Speaker == nil {
			return append(nodes, quiet("speaker-missing-config", "Speaker configuration is unavailable"))
		}
		c := *p.Device.Speaker
		apply := func(v workbench.SpeakerConfig) {
			s.invokeAtCommandRevision("speaker.configure", workbench.SpeakerParams{ID: p.ID, Parameters: v}, state.CommandRevision, state.Document.Generation)
		}
		nodes = append(nodes, section("speaker-title", "Loudspeaker load"), quiet("speaker-help", "Passive coil and resonance equivalent\nPreview is explicit; no acoustic calibration."))
		field("speaker-coil-r", "Coil resistance · Ω", c.Resistance, 1, 100, .5, func(v float64) { n := c; n.Resistance = v; apply(n) })
		field("speaker-coil-l", "Coil inductance · mH", c.Inductance*1000, .01, 100, .1, func(v float64) { n := c; n.Inductance = decimalControlSI(v, -3); apply(n) })
		field("speaker-fs", "Resonance · Hz", c.Resonance, 20, 1000, 10, func(v float64) { n := c; n.Resonance = v; apply(n) })
		field("speaker-motional-r", "Motional resistance · Ω", c.MotionalResistance, 1, 1000, 1, func(v float64) { n := c; n.MotionalResistance = v; apply(n) })
		field("speaker-motional-l", "Motional inductance · mH", c.MotionalInductance*1000, .1, 1000, 1, func(v float64) { n := c; n.MotionalInductance = decimalControlSI(v, -3); apply(n) })
		if state.Mixed != nil {
			for _, r := range state.Mixed.Speakers {
				if r.ID == p.ID {
					nodes = append(nodes, quiet("speaker-reading", fmt.Sprintf("%s · %.5g V · %.5g mA\nSigned terminal power %.5g mW\nCoil heat %.5g · motional loss %.5g mW\nStored energy %.5g µJ", measurementState(state.Mixed.Stale), r.Voltage, r.Current*1000, r.Power*1000, r.CoilHeat*1000, r.MotionalHeat*1000, r.Energy*1e6)))
				}
			}
		}
		nodes = append(nodes, quiet("speaker-preview-help", "Terminal-voltage illustration · peak normalized\nSignals below 1 µV stay silent; no autoplay."))
		nodes = append(nodes, s.audioControls(state, p)...)
	}
	if p.Kind == "pwm-controller" {
		c := *p.Device.PWM
		apply := func(v workbench.PWMConfig) {
			s.invokeAtCommandRevision("pwm.configure", workbench.PWMParams{ID: p.ID, Parameters: v}, state.CommandRevision, state.Document.Generation)
		}
		nodes = append(nodes, section("pwm-title", "PWM controller"), quiet("pwm-help", "Powered clock · 50 Ω output · finite edges\nFrequency and duty edits restart the experiment."))
		field("pwm-frequency", "Frequency · Hz", c.Frequency, 10, 1000, 10, func(v float64) { next := c; next.Frequency = v; apply(next) })
		field("pwm-duty", "Duty · %", c.Duty*100, 10, 90, 1, func(v float64) { next := c; next.Duty = v / 100; apply(next) })
	}
	if p.Kind == "opamp" {
		enabled := p.Device != nil && p.Device.Dynamics != nil
		c := workbench.OpAmpDynamics{GBW: 1000, Slew: 1000}
		if enabled {
			c = *p.Device.Dynamics
		}
		apply := func(v workbench.OpAmpDynamics, on bool) {
			s.invokeAtCommandRevision("opamp.dynamics.set", workbench.DynamicsParams{ID: p.ID, Enabled: on, Parameters: v}, state.CommandRevision, state.Document.Generation)
		}
		nodes = append(nodes, section("opamp-dynamics-title", "Frequency response"), ui.Node{ID: "opamp-dynamics-enabled", Kind: ui.Checkbox, Text: "Bandwidth / slew limits", Checked: enabled, Accessibility: ui.Accessibility{Name: "Bandwidth / slew limits", Help: "Include the configured gain-bandwidth and slew-rate limits."}, Change: func(e ui.ChangeEvent) {
			if e.Checked != enabled {
				apply(c, e.Checked)
			}
		}})
		if enabled {
			field("opamp-gbw", "Gain-bandwidth · Hz", c.GBW, 100, 1e6, 100, func(v float64) { next := c; next.GBW = v; apply(next, true) })
			field("opamp-slew", "Slew limit · V/s", c.Slew, 100, 1e7, 100, func(v float64) { next := c; next.Slew = v; apply(next, true) })
			nodes = append(nodes, quiet("opamp-dynamics-help", "Single dominant pole · explicit compensation energy\nSaved settings; edits restart simulation."))
		}
	}
	if p.Kind == "shift-register" {
		nodes = append(nodes, section("shift-title", "Shift and latch"), quiet("shift-help", "DATA sampled on rising CLK. Rising LATCH publishes the byte. RESET_N low clears the shift chain; OE_N low enables Q0–Q7. SERIAL always follows the last shift bit."))
		if state.Mixed != nil {
			for _, r := range state.Mixed.Shifts {
				if r.ID == p.ID {
					nodes = append(nodes, quiet("shift-reading", fmt.Sprintf("Shift 0x%02X · latch 0x%02X\nParallel outputs enabled: %t", r.Shifted, r.Latched, r.Enabled)))
				}
			}
		}
	}
	if p.Kind == "piezo" || p.Kind == "active-buzzer" {
		c := *p.Device.Sound
		apply := func(v workbench.SoundConfig) {
			s.invokeAtCommandRevision("sound.configure", workbench.SoundParams{ID: p.ID, Parameters: v}, state.CommandRevision, state.Document.Generation)
		}
		nodes = append(nodes, section("sound-title", "Sounder"))
		if p.Kind == "piezo" {
			field("sound-capacitance", "Piezo capacitance · nF", c.Capacitance*1e9, 1, 1000, 1, func(v float64) { next := c; next.Capacitance = v / 1e9; apply(next) })
		} else {
			field("sound-frequency", "Buzzer tone · Hz", c.Frequency, 100, 4000, 100, func(v float64) { next := c; next.Frequency = v; apply(next) })
		}
		nodes = append(nodes, quiet("sound-help", "Electrical simulation is silent. Preview is illustrative audio, not a sound-pressure prediction."))
		nodes = append(nodes, s.audioControls(state, p)...)
	}

	if state.Mixed != nil && workbench.IsBipolarKind(p.Kind) && p.Device != nil && p.Device.Bipolar != nil {
		for _, q := range state.Mixed.Transistors {
			if q.TransistorID == p.ID {
				nodes = append(nodes, quiet("transistor-profile-reading", fmt.Sprintf("%s · junction energy %.5g nJ\nBE charging %.5g · BC charging %.5g µA\nConductive heat %.5g mW", measurementState(state.Mixed.Stale), q.JunctionEnergy*1e9, q.JunctionBECurrent*1e6, q.JunctionBCCurrent*1e6, q.ConductiveHeat*1000)))
			}
		}
	}
	if state.Mixed != nil {
		status := "Live"
		if state.Mixed.Stale {
			status = "Stale"
		}
		for _, r := range state.Mixed.PWM {
			if r.ID == p.ID {
				nodes = append(nodes, quiet("pwm-reading", fmt.Sprintf("%s · output %.5g V · drive %.3g%%\nController loss %.5g mW", status, r.Voltage, r.Phase*100, r.Power*1000)))
			}
		}
		for _, r := range state.Mixed.Dynamics {
			if r.ID == p.ID {
				nodes = append(nodes, quiet("opamp-pole-reading", fmt.Sprintf("%s · compensation %.5g V\nStored %.5g nJ · rate %.5g V/s", status, r.Voltage, r.Energy*1e9, r.Current/1e-9)))
			}
		}
		for _, r := range state.Mixed.Sounds {
			if r.ID == p.ID {
				nodes = append(nodes, quiet("sound-reading", fmt.Sprintf("%s · %.5g V · %.5g mA\nStored %.5g nJ · loss %.5g mW", status, r.Voltage, r.Current*1000, r.Energy*1e9, r.Power*1000)))
			}
		}
	}
	return nodes
}

func decimalControlSI(value float64, exponent int) float64 {
	v, err := strconv.ParseFloat(strconv.FormatFloat(value, 'f', -1, 64)+"e"+strconv.Itoa(exponent), 64)
	if err != nil {
		return math.NaN()
	}
	return v
}

func measurementState(stale bool) string {
	if stale {
		return "Stale"
	}
	return "Live"
}
