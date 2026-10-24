package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) batteryControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	b := *part.Power.Battery
	apply := func(next workbench.BatteryConfig) {
		s.invoke("battery.configure", workbench.BatteryParams{ID: part.ID, CapacityAh: next.CapacityAh, Resistance: next.Resistance, Charge: next.Charge})
	}
	charge := func(e ui.ChangeEvent) {
		if e.Number == float32(b.Charge*100) {
			return
		}
		v := b
		v.Charge = float64(e.Number) / 100
		apply(v)
	}
	nodes := []ui.Node{section("battery-startup-title", "Battery startup"), quiet("battery-startup-help", "These saved settings restart a running circuit. Paused readings stay stale until Restart. Live charge is shown separately below."), quiet("battery-charge-label", fmt.Sprintf("Startup charge · %.6g%%", b.Charge*100)), {ID: "battery-charge-slider", Kind: ui.Slider, Number: float32(b.Charge * 100), Numeric: ui.Numeric{Min: 0, Max: 100, Step: 1}, Accessibility: ui.Accessibility{Name: "Battery startup charge percent"}, Change: charge}, {ID: "battery-charge-exact", Kind: ui.NumberField, Number: float32(b.Charge * 100), Numeric: ui.Numeric{Min: 0, Max: 100, Step: 1}, Accessibility: ui.Accessibility{Name: "Exact battery startup charge percent"}, Change: charge}, quiet("battery-capacity-title", "Capacity · mAh"), {ID: "battery-capacity", Kind: ui.NumberField, Number: float32(b.CapacityAh * 1000), Numeric: ui.Numeric{Min: .001, Max: 100000, Step: .001}, Accessibility: ui.Accessibility{Name: "Battery capacity in mAh"}, Change: func(e ui.ChangeEvent) {
		if e.Number == float32(b.CapacityAh*1000) {
			return
		}
		v := b
		v.CapacityAh = float64(e.Number) / 1000
		apply(v)
	}}, quiet("battery-esr-title", "Internal resistance · Ω"), {ID: "battery-esr", Kind: ui.NumberField, Number: float32(b.Resistance), Numeric: ui.Numeric{Min: .01, Max: 1000, Step: .01}, Accessibility: ui.Accessibility{Name: "Battery internal resistance in ohms"}, Change: func(e ui.ChangeEvent) {
		if e.Number == float32(b.Resistance) {
			return
		}
		v := b
		v.Resistance = float64(e.Number)
		if e.Number == float32(.01) {
			v.Resistance = .01
		}
		apply(v)
	}}}
	if state.Mixed != nil {
		for _, r := range state.Mixed.Batteries {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				nodes = append(nodes, section("battery-reading-title", status+" battery"), quiet("battery-state", fmt.Sprintf("%.5g%% · %s", r.Charge*100, r.State)), quiet("battery-voltage", fmt.Sprintf("OCV %.5g V · terminals %.5g V", r.OpenCircuitVoltage, r.Voltage)), quiet("battery-current", fmt.Sprintf("Delivered %.5g mA · loss %.5g mW", r.Current*1000, r.InternalHeat*1000)), quiet("battery-energy", fmt.Sprintf("Remaining %.5g mAh · %.5g J", r.RemainingAh*1000, r.Energy)))
			}
		}
	}
	return nodes
}
