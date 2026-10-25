package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) extendedControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	nodes := []ui.Node{}
	field := func(id, label string, value, min, max, step float64, change func(float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label}, Change: func(e ui.ChangeEvent) {
			if e.Number != float32(value) {
				change(float64(e.Number))
			}
		}})
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
		nodes = append(nodes, section("opamp-dynamics-title", "Frequency response"), ui.Node{ID: "opamp-dynamics-enabled", Kind: ui.Checkbox, Text: "Enable bandwidth and slew limits", Checked: enabled, Accessibility: ui.Accessibility{Name: "Enable op-amp dynamics"}, Change: func(e ui.ChangeEvent) {
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
