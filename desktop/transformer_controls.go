package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) transformerControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	c := *part.Power.Transformer
	apply := func(next workbench.TransformerConfig) {
		s.invoke("transformer.configure", workbench.TransformerParams{ID: part.ID, Parameters: next})
	}
	nodes := []ui.Node{section("transformer-settings-title", "Coupled windings"), quiet("transformer-settings-help", "Saved settings restart a running circuit. Paused readings wait for Restart."), quiet("transformer-ratio-help", "Turns ratio is secondary / primary; Ls = Lp × ratio².")}
	for _, field := range []struct {
		id, label             string
		value, min, max, step float64
		set                   func(*workbench.TransformerConfig, float64)
	}{
		{"lp", "Primary inductance · H", c.Inductance, 1e-6, 100, .001, func(v *workbench.TransformerConfig, n float64) { v.Inductance = n }},
		{"ratio", "Secondary / primary turns", c.Ratio, .1, 10, .01, func(v *workbench.TransformerConfig, n float64) { v.Ratio = n }},
		{"coupling", "Coupling · 0–0.9999", c.Coupling, 0, .9999, .001, func(v *workbench.TransformerConfig, n float64) { v.Coupling = n }},
		{"rp", "Primary resistance · Ω", c.PrimaryResistance, .01, 1000, .01, func(v *workbench.TransformerConfig, n float64) { v.PrimaryResistance = n }},
		{"rs", "Secondary resistance · Ω", c.SecondaryResistance, .01, 1000, .01, func(v *workbench.TransformerConfig, n float64) { v.SecondaryResistance = n }},
	} {
		nodes = append(nodes, quiet("transformer-"+field.id+"-label", field.label), ui.Node{ID: "transformer-" + field.id, Kind: ui.NumberField, Number: float32(field.value), ControlStateKey: fmt.Sprintf("%d/%s", state.Document.Generation, part.ID), Numeric: ui.Numeric{Min: float32(field.min), Max: float32(field.max), Step: float32(field.step)}, Accessibility: ui.Accessibility{Name: field.label}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(field.value) {
				return
			}
			value := float64(e.Number)
			if e.Number == float32(field.min) {
				value = field.min
			}
			if e.Number == float32(field.max) {
				value = field.max
			}
			next := c
			field.set(&next, value)
			apply(next)
		}})
	}
	nodes = append(nodes, ui.Node{ID: "transformer-reverse", Kind: ui.Checkbox, Text: "Reverse secondary dot", Checked: c.Reversed, Accessibility: ui.Accessibility{Name: "Reverse secondary dot", Help: "Reverse the secondary winding polarity relative to the primary dot."}, Change: func(e ui.ChangeEvent) { next := c; next.Reversed = e.Checked; apply(next) }}, quiet("transformer-derived", fmt.Sprintf("Secondary inductance %.6g H", c.Inductance*c.Ratio*c.Ratio)), quiet("transformer-assumptions", "Linear core · zero startup currents · no saturation or isolation rating. Exact coupling 1 is unsupported."))
	if state.Mixed != nil {
		for _, r := range state.Mixed.Transformers {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				nodes = append(nodes, section("transformer-readings-title", status+" windings"), quiet("transformer-voltages", fmt.Sprintf("Primary %.6g V · secondary %.6g V", r.PrimaryVoltage, r.SecondaryVoltage)), quiet("transformer-currents", fmt.Sprintf("Into pin 1: %.6g / %.6g mA", r.PrimaryCurrent*1000, r.SecondaryCurrent*1000)), quiet("transformer-flux", fmt.Sprintf("Flux linkage %.5g / %.5g Wb·turn", r.PrimaryFluxLinkage, r.SecondaryFluxLinkage)), quiet("transformer-energy", fmt.Sprintf("Magnetic %.6g mJ · heat %.6g mW", r.Energy*1000, r.WindingLoss*1000)), quiet("transformer-power", fmt.Sprintf("Primary %.6g mW · secondary delivered %.6g mW", r.PrimaryPower*1000, -r.SecondaryPower*1000)))
			}
		}
	}
	return nodes
}
