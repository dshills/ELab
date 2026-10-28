package main

import (
	"fmt"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) thyristorControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	if !workbench.IsThyristorKind(p.Kind) || p.Device == nil || p.Device.Thyristor == nil {
		return nil
	}
	cfg := *p.Device.Thyristor
	nodes := []ui.Node{section("thyristor-title", "Trigger and latch")}
	field := func(id, label string, value, min, max, step float64, set func(*workbench.ThyristorConfig, float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label, Help: "Saved model setting; restarts from blocking state. Holding current must be below latching current."}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(value) {
				return
			}
			v, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
			if err != nil {
				return
			}
			next := cfg
			set(&next, v)
			s.invokeAtCommandRevision("thyristor.configure", workbench.ThyristorParams{ID: p.ID, Parameters: next}, state.CommandRevision, state.Document.Generation)
		}})
	}
	field("thyristor-trigger", "Gate trigger · mA", cfg.Trigger*1000, .01, 10, .1, func(c *workbench.ThyristorConfig, v float64) { c.Trigger = decimalControlSI(v, -3) })
	field("thyristor-latching", "Latching current · mA", cfg.Latching*1000, .1, 1000, .1, func(c *workbench.ThyristorConfig, v float64) { c.Latching = decimalControlSI(v, -3) })
	field("thyristor-holding", "Holding current · mA", cfg.Holding*1000, .01, 50, .1, func(c *workbench.ThyristorConfig, v float64) { c.Holding = decimalControlSI(v, -3) })
	field("thyristor-resistance", "On resistance · Ω", cfg.Resistance, .1, 100, .1, func(c *workbench.ThyristorConfig, v float64) { c.Resistance = v })
	field("thyristor-drop", "Forward drop · V", cfg.Drop, 0, 2, .1, func(c *workbench.ThyristorConfig, v float64) { c.Drop = v })
	field("thyristor-gate-resistance", "Gate resistance · Ω", cfg.GateResistance, 10, 10000, 10, func(c *workbench.ThyristorConfig, v float64) { c.GateResistance = v })
	nodes = append(nodes, quiet("thyristor-help", "Main ±24 V / gate ±12 V model envelope. Trigger using ordinary sources or buttons. No recovery, avalanche, dv/dt, quadrant sensitivity or hardware rating."))
	if state.Mixed != nil {
		for _, r := range state.Mixed.Controls {
			if r.ID == p.ID {
				nodes = append(nodes, quiet("thyristor-reading", fmt.Sprintf("%s · %s\nMain %.5g V · %.5g mA\nGate %.5g mA · %.5g mW", measurementState(state.Mixed.Stale), r.State, r.Voltage, r.Current*1000, r.PinCurrents[2]*1000, r.Power*1000)))
			}
		}
	}
	return nodes
}
