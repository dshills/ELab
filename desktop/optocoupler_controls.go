package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) optocouplerControls(state workbench.Snapshot, part workbench.Component) []ui.Node {
	ctr := part.Power.Optocoupler.CTR
	change := func(e ui.ChangeEvent) {
		if e.Number == float32(ctr*100) {
			return
		}
		v := float64(e.Number) / 100
		if e.Number == 1 {
			v = .01
		}
		s.invoke("optocoupler.configure", workbench.OptocouplerParams{ID: part.ID, CTR: v})
	}
	nodes := []ui.Node{section("opto-title", "Optocoupler"), quiet("opto-config-help", "Saved CTR setting restarts a running circuit. Paused readings wait for Restart."), quiet("opto-ctr-label", "Current-transfer ratio · %"), {ID: "opto-ctr", Kind: ui.NumberField, Number: float32(ctr * 100), Numeric: ui.Numeric{Min: 1, Max: 500, Step: 1}, Accessibility: ui.Accessibility{Name: "Optocoupler current-transfer ratio percent"}, Change: change}, {ID: "opto-ctr-slider", Kind: ui.Slider, Number: float32(ctr * 100), Numeric: ui.Numeric{Min: 1, Max: 500, Step: 1}, Accessibility: ui.Accessibility{Name: "Optocoupler CTR percent slider"}, Change: change}, quiet("opto-isolation-help", "Separate electrical ports; shared ground symbols are simulation references. No isolation-voltage rating.")}
	if state.Mixed != nil {
		for _, r := range state.Mixed.Optocouplers {
			if r.ID == part.ID {
				status := "Live"
				if state.Mixed.Stale {
					status = "Stale"
				}
				nodes = append(nodes, section("opto-reading-title", status+" optical transfer"), quiet("opto-state", r.State), quiet("opto-input", fmt.Sprintf("LED %.5g V · %.5g mA", r.InputVoltage, r.InputCurrent*1000)), quiet("opto-output", fmt.Sprintf("Output %.5g V · %.5g mA", r.OutputVoltage, r.OutputCurrent*1000)), quiet("opto-effective", fmt.Sprintf("Effective CTR %.5g%% · available %.5g mA", r.EffectiveCTR*100, r.AvailableCurrent*1000)), quiet("opto-power", fmt.Sprintf("Input %.5g mW · output %.5g mW", r.InputPower*1000, r.OutputPower*1000)))
			}
		}
	}
	return nodes
}
