package main

import (
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) passiveControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	if !workbench.IsPassiveStorageKind(p.Kind) {
		return nil
	}
	enabled := p.Device != nil && p.Device.Passive != nil
	cfg := workbench.PassiveConfig{Version: workbench.PassiveLossModelID}
	if p.Kind == "capacitor" {
		cfg.ESR, cfg.Leakage = 1, 1e6
	} else {
		cfg.Winding = 1
	}
	if enabled {
		cfg = *p.Device.Passive
	}
	apply := func(c workbench.PassiveConfig, on bool) {
		s.invokeAtCommandRevision("passive.configure", workbench.PassiveParams{ID: p.ID, Enabled: on, Parameters: c}, state.CommandRevision, state.Document.Generation)
	}
	nodes := []ui.Node{section("passive-title", "Component losses"), {ID: "passive-enabled", Kind: ui.Checkbox, Text: "Include electrical losses", Checked: enabled, Accessibility: ui.Accessibility{Name: "Include electrical losses", Help: "Include capacitor ESR and leakage, or inductor winding resistance."}, Change: func(e ui.ChangeEvent) {
		if e.Checked != enabled {
			apply(cfg, e.Checked)
		}
	}}}
	if !enabled {
		return append(nodes, quiet("passive-help", "Ideal model · enable to include electrical loss."))
	}
	field := func(id, label string, value, min, max, step float64, exponent int, change func(*workbench.PassiveConfig, float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label, Help: "Saved electrical setting; restarts the experiment. Use the leakage checkbox to disable leakage; zero ESR/winding resistance restores the ideal path."}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(value) {
				return
			}
			next := cfg
			parsed, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
			if err != nil {
				return
			}
			change(&next, decimalControlSI(parsed, exponent))
			apply(next, true)
		}})
	}
	if p.Kind == "capacitor" {
		field("passive-esr", "Series ESR · Ω (0 off)", cfg.ESR, 0, 1e5, .1, 0, func(c *workbench.PassiveConfig, v float64) { c.ESR = v })
		nodes = append(nodes, ui.Node{ID: "passive-leak-enabled", Kind: ui.Checkbox, Text: "Include leakage", Checked: cfg.Leakage > 0, Accessibility: ui.Accessibility{Name: "Include leakage", Help: "Include capacitor leakage across the terminals."}, Change: func(e ui.ChangeEvent) {
			if e.Checked == (cfg.Leakage > 0) {
				return
			}
			next := cfg
			next.Leakage = 0
			if e.Checked {
				next.Leakage = 1e6
			}
			apply(next, true)
		}})
		if cfg.Leakage > 0 {
			field("passive-leak", "Leakage resistance · MΩ", cfg.Leakage/1e6, .001, 1e6, .1, 6, func(c *workbench.PassiveConfig, v float64) { c.Leakage = v })
		}
		nodes = append(nodes, quiet("passive-help", "ESR is in series with storage. Leakage is across the terminals. Initial voltage refers to stored charge."))
	} else {
		field("passive-winding", "Winding resistance · Ω (0 off)", cfg.Winding, 0, 1e5, .1, 0, func(c *workbench.PassiveConfig, v float64) { c.Winding = v })
	}
	if state.Mixed != nil {
		var loss *workbench.PassiveReading
		for _, c := range state.Mixed.Capacitors {
			if c.ID == p.ID {
				loss = c.Loss
			}
		}
		for _, c := range state.Mixed.Inductors {
			if c.ID == p.ID {
				loss = c.Loss
			}
		}
		if loss != nil {
			text := fmt.Sprintf("%s · inductive %.4g V\nCurrent %.4g mA\nWinding heat %.4g mW", measurementState(state.Mixed.Stale), loss.StoredVoltage, loss.StorageCurrent*1000, loss.WindingHeat*1000)
			if p.Kind == "capacitor" {
				text = fmt.Sprintf("%s · stored %.4g V\nCharging %.4g mA\nLeakage %.4g µA\nESR %.4g / leakage %.4g mW", measurementState(state.Mixed.Stale), loss.StoredVoltage, loss.StorageCurrent*1000, loss.LeakageCurrent*1e6, loss.ESRHeat*1000, loss.LeakageHeat*1000)
			}
			nodes = append(nodes, quiet("passive-reading", text))
		}
	}
	return nodes
}
