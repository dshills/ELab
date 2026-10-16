package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) potentiometerControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	var nodes []ui.Node
	position := *part.WiperPosition
	if activeInputWork(state) && state.InputWork.Wiper != nil && state.InputWork.Wiper.ID == part.ID {
		position = state.InputWork.Wiper.Position
	}
	change := func(e ui.ChangeEvent) {
		// The native numeric surface is float32. Recommitting its displayed
		// value must not quantize a more precise stored fraction or add history.
		if e.Number == float32(position*100) {
			return
		}
		s.invoke("potentiometer.wiper.set", workbench.WiperParams{ID: part.ID, Position: float64(e.Number) / 100})
	}
	nodes = append(nodes, section("potentiometer-title", "Linear potentiometer"), quiet("wiper-label", fmt.Sprintf("Wiper %.4g%% · 0 toward pin 2 / 100 toward pin 1", position*100)), ui.Node{ID: "pot-wiper", Kind: ui.Slider, Number: float32(position * 100), Numeric: ui.Numeric{Min: 0, Max: 100, Step: 1}, Accessibility: ui.Accessibility{Name: "Potentiometer wiper percent"}, Change: change}, ui.Node{ID: "pot-wiper-exact", Kind: ui.NumberField, Number: float32(position * 100), Numeric: ui.Numeric{Min: 0, Max: 100, Step: .1}, Accessibility: ui.Accessibility{Name: "Exact potentiometer wiper percent"}, Change: change}, quiet("pot-actuation-help", workbench.PartActuationHint(part.Kind)), quiet("pot-model", "Two linear track segments; 1 Ω minimum each. Total resistance is preserved. No contact noise, taper, heating or ratings."))
	if state.Mixed != nil {
		for _, r := range state.Mixed.Potentiometers {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				} else if activeInputWork(state) && state.InputWork.Wiper != nil && state.InputWork.Wiper.ID == part.ID {
					status = "Committed · wiper change pending"
				}
				nodes = append(nodes, quiet("pot-reading", fmt.Sprintf("%s · wiper %.6g V / ground\nPin 1 %.6g mA · pin 2 %.6g mA\nWiper %.6g mA into track · %.6g mW", status, r.WiperVoltage, r.Terminal1Current*1000, r.Terminal2Current*1000, r.WiperCurrent*1000, r.Power*1000)))
			}
		}
	}
	return nodes
}

func (s *screen) switchControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	var nodes []ui.Node
	closed := part.SwitchClosed != nil && *part.SwitchClosed
	text := "Close switch"
	stateText := "Open"
	if closed {
		text = "Open switch"
		stateText = "Closed"
	}
	for _, input := range state.SwitchStates {
		if input.ID == part.ID {
			if input.AppliedClosed == nil {
				stateText += " · startup; no fresh applied reading"
			} else {
				applied := "open"
				if *input.AppliedClosed {
					applied = "closed"
				}
				stateText += " · applied " + applied
				if input.Pending {
					stateText += " · pending next step"
				}
			}
		}
	}

	nodes = append(nodes, section("switch-title", "SPST switch"), quiet("switch-state", stateText+" · 1 Ω on / 1 GΩ off"), s.action("switch-actuate", text, "switch.set", workbench.SwitchParams{ID: part.ID, Closed: !closed}), quiet("switch-actuation-help", workbench.PartActuationHint(part.Kind)), quiet("switch-limits", "Finite leakage, no bounce/arcing or physical rating."))
	if state.Mixed != nil && !state.Mixed.Stale {
		for _, branch := range state.Mixed.Switches {
			if branch.ID == part.ID {
				nodes = append(nodes, quiet("switch-reading", fmt.Sprintf("%.6g V · %.6g mA\n%.6g mW", branch.Voltage, branch.Current*1000, branch.Power*1000)))
			}
		}
	}
	return nodes
}
