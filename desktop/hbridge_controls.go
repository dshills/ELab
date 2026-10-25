package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) hbridgeControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	nodes := []ui.Node{section("hbridge-title", "Motor driver"), quiet("hbridge-controls-help", "Wire IN1 / IN2 for direction and ENABLE for PWM. Inputs: low ≤0.8 V, high ≥2 V relative to GND."), quiet("hbridge-truth-table", "Enabled: 10 forward · 01 reverse\n00 brake low · 11 brake high\nENABLE low: coast; flyback clamps remain active."), quiet("hbridge-live-help", "Use connected switches or sources to change modes. Motor speed and winding current remain continuous.")}
	if state.Mixed != nil {
		for _, r := range state.Mixed.HBridges {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				nodes = append(nodes, section("hbridge-reading-title", status+" bridge"), quiet("hbridge-state", r.State), quiet("hbridge-inputs", fmt.Sprintf("IN1 %s · IN2 %s · EN %s", r.Input1, r.Input2, r.Enable)), quiet("hbridge-voltage", fmt.Sprintf("OUT1 − OUT2 %.6g V · VM %.6g V", r.OutputVoltage, r.SupplyVoltage)), quiet("hbridge-current", fmt.Sprintf("Into VM %.6g A · negative returns energy", r.PinCurrents[5])), quiet("hbridge-loss", fmt.Sprintf("Driver dissipation %.6g W", r.Power)), quiet("hbridge-clamps", fmt.Sprintf("Clamp currents A: %.4g / %.4g / %.4g / %.4g", r.ClampCurrents[0], r.ClampCurrents[1], r.ClampCurrents[2], r.ClampCurrents[3])))
			}
		}
	}
	return nodes
}
