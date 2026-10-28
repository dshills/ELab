package main

import (
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) thermalControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	if p.Kind != "resistor" && p.Kind != "diode" && !workbench.IsBipolarKind(p.Kind) {
		return nil
	}
	enabled := p.Thermal != nil
	cfg := workbench.DefaultThermalConfig(p.Kind)
	if enabled {
		cfg = *p.Thermal
	}
	apply := func(c workbench.ThermalConfig, on bool) {
		s.invokeAtCommandRevision("thermal.configure", workbench.ThermalParams{ID: p.ID, Enabled: on, Parameters: c}, state.CommandRevision, state.Document.Generation)
	}
	nodes := []ui.Node{section("thermal-title", "Temperature"), {ID: "thermal-enabled", Kind: ui.Checkbox, Text: "Include temperature effects", Checked: enabled, Accessibility: ui.Accessibility{Name: "Include temperature effects"}, Change: func(e ui.ChangeEvent) {
		if e.Checked != enabled {
			apply(cfg, e.Checked)
		}
	}}}
	if !enabled {
		return nodes
	}
	field := func(id, label string, value, min, max, step float64, set func(*workbench.ThermalConfig, float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label, Help: "Saved temperature model setting; restarts the experiment."}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(value) {
				return
			}
			v, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
			if err != nil {
				return
			}
			next := cfg
			set(&next, v)
			apply(next, true)
		}})
	}
	field("thermal-ambient", "Ambient · °C", cfg.Ambient, -40, 125, 1, func(c *workbench.ThermalConfig, v float64) { c.Ambient = v })
	if p.Kind == "resistor" {
		field("thermal-alpha", "Resistance coefficient · ppm/K", cfg.Alpha*1e6, -5000, 5000, 10, func(c *workbench.ThermalConfig, v float64) { c.Alpha = decimalControlSI(v, -6) })
	}
	nodes = append(nodes, ui.Node{ID: "thermal-self", Kind: ui.Checkbox, Text: "Include self-heating", Checked: cfg.SelfHeating, Accessibility: ui.Accessibility{Name: "Include self-heating"}, Change: func(e ui.ChangeEvent) {
		if e.Checked != cfg.SelfHeating {
			next := cfg
			next.SelfHeating = e.Checked
			apply(next, true)
		}
	}})
	if cfg.SelfHeating {
		field("thermal-resistance", "Thermal resistance · K/W", cfg.Resistance, .1, 2000, 1, func(c *workbench.ThermalConfig, v float64) { c.Resistance = v })
		field("thermal-capacitance", "Thermal capacity · J/K", cfg.Capacitance, .001, 10, .001, func(c *workbench.ThermalConfig, v float64) { c.Capacitance = v })
	}
	nodes = append(nodes, quiet("thermal-help", "Generic one-node thermal model · accepted temperature −40…150 °C. No physical device rating or heat-sink fit. Ambient sweeps use the ordinary DC sweep controls."))
	if state.Mixed != nil {
		for _, r := range state.Mixed.Thermal {
			if r.ID == p.ID {
				nodes = append(nodes, quiet("thermal-reading", fmt.Sprintf("%s · %.4g °C\nConductive heat %.4g mW\nTo ambient %.4g mW · stored %.4g mJ", measurementState(state.Mixed.Stale), r.Temperature, r.ConductiveHeat*1000, r.Transfer*1000, r.StoredHeat*1000)))
			}
		}
	}
	return nodes
}
