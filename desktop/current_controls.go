package main

import (
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) currentControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	if !workbench.IsCurrentDevice(p.Kind) {
		return nil
	}
	if p.Device == nil || (p.Kind == "current-source" && p.Device.Current == nil) || (p.Kind == "electronic-load" && p.Device.Load == nil) {
		return []ui.Node{quiet("current-config-unavailable", "Current-device configuration is unavailable.")}
	}
	nodes := []ui.Node{section("current-model", "Current control"), quiet("current-model-id", workbench.CurrentDeviceModelID)}
	c, l := workbench.DefaultCurrentConfig(), workbench.DefaultLoadConfig()
	if p.Kind == "current-source" {
		c = *p.Device.Current
	} else {
		l = *p.Device.Load
	}
	apply := func() {
		if p.Kind == "current-source" {
			s.invokeAtCommandRevision("current-source.configure", workbench.CurrentParams{ID: p.ID, Parameters: c}, state.CommandRevision, state.Document.Generation)
		} else {
			s.invokeAtCommandRevision("load.configure", workbench.ElectronicLoadParams{ID: p.ID, Parameters: l}, state.CommandRevision, state.Document.Generation)
		}
	}
	field := func(id, label string, value, min, max, step float64, set func(float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(value) {
				return
			}
			parsed, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
			if err != nil {
				return
			}
			set(parsed)
			apply()
		}})
	}
	timing := c.Timing
	if p.Kind == "electronic-load" {
		timing = l.Timing
	}
	if p.Kind == "current-source" {
		field("current-level", "High / DC current · mA", c.Current*1000, 0, 1000, 1, func(v float64) { c.Current = decimalControlSI(v, -3) })
		field("current-low", "Low current · mA", c.Low*1000, 0, 1000, 1, func(v float64) { c.Low = decimalControlSI(v, -3) })
		field("current-compliance", "Compliance · V", c.Compliance, 1, 24, .1, func(v float64) { c.Compliance = v })
	} else {
		nodes = append(nodes, ui.Node{ID: "load-power", Kind: ui.Checkbox, Text: "Constant power (W)", Checked: l.Mode == "cp", Accessibility: ui.Accessibility{Name: "Constant power (W)", Help: "Checked uses constant power in watts; unchecked uses constant current in amperes."}, Change: func(e ui.ChangeEvent) {
			if e.Checked == (l.Mode == "cp") {
				return
			}
			l.Mode = "cc"
			if e.Checked {
				l.Mode = "cp"
			}
			l.Level, l.Low = .01, 0
			apply()
		}})
		unit, max := "A", 1.
		if l.Mode == "cp" {
			unit, max = "W", 24
		}
		field("load-level", "High / DC load · "+unit, l.Level, 0, max, .01, func(v float64) { l.Level = v })
		field("load-low", "Low load · "+unit, l.Low, 0, max, .01, func(v float64) { l.Low = v })
		field("load-min-voltage", "Minimum operating voltage · V", l.MinimumVoltage, .1, 12, .1, func(v float64) { l.MinimumVoltage = v })
		field("load-max-current", "Maximum current · A", l.MaximumCurrent, .001, 1, .01, func(v float64) { l.MaximumCurrent = v })
	}
	setTiming := func(t workbench.CurrentTiming) { c.Timing = t; l.Timing = t }
	nodes = append(nodes, ui.Node{ID: "current-pulsed", Kind: ui.Checkbox, Text: "Pulse between levels", Checked: timing.Pulsed, Accessibility: ui.Accessibility{Name: "Pulse between levels", Help: "Pulse between the configured high and low current or load levels."}, Change: func(e ui.ChangeEvent) {
		if e.Checked == timing.Pulsed {
			return
		}
		t := timing
		t.Pulsed = e.Checked
		setTiming(t)
		apply()
	}})
	if timing.Pulsed {
		for _, f := range []struct {
			id, label string
			value     float64
			set       func(*workbench.CurrentTiming, float64)
		}{{"current-high-time", "High phase · ms", timing.HighSeconds, func(t *workbench.CurrentTiming, v float64) { t.HighSeconds = v }}, {"current-low-time", "Low phase · ms", timing.LowSeconds, func(t *workbench.CurrentTiming, v float64) { t.LowSeconds = v }}, {"current-rise-time", "Rise · ms", timing.RiseSeconds, func(t *workbench.CurrentTiming, v float64) { t.RiseSeconds = v }}, {"current-fall-time", "Fall · ms", timing.FallSeconds, func(t *workbench.CurrentTiming, v float64) { t.FallSeconds = v }}} {
			field(f.id, f.label, f.value*1000, .001, 10000, .1, func(v float64) { t := timing; f.set(&t, decimalControlSI(v, -3)); setTiming(t) })
		}
	}
	nodes = append(nodes, quiet("current-help", "1 GΩ terminal bleed. Source reduces current in the last 1 V before compliance; load folds to a linear sink below minimum voltage. No thermal or protection model."))
	if state.Mixed != nil {
		group := state.Mixed.Sources
		if p.Kind == "electronic-load" {
			group = state.Mixed.Loads
		}
		for _, r := range group {
			if r.ID == p.ID {
				nodes = append(nodes, quiet("current-reading", fmt.Sprintf("%s · %.4g V · %.4g mA\n%.4g mW", measurementState(state.Mixed.Stale), r.Voltage, r.Current*1000, r.Power*1000)))
			}
		}
	}
	return nodes
}
