package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) jfetControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	if !workbench.IsJFETKind(p.Kind) || p.Device == nil || p.Device.JFET == nil {
		return nil
	}
	c := *p.Device.JFET
	nodes := []ui.Node{section("jfet-config", "Depletion JFET"), quiet("jfet-model", workbench.JFETModelID)}
	apply := func(next workbench.JFETConfig) {
		s.invokeAtCommandRevision("jfet.configure", workbench.JFETParams{ID: p.ID, Parameters: next}, state.CommandRevision, state.Document.Generation)
	}
	field := func(id, label string, value, min, max, step float64, set func(*workbench.JFETConfig, float64)) {
		nodes = append(nodes, quiet(id+"-label", label), ui.Node{ID: id, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: float32(min), Max: float32(max), Step: float32(step)}, Accessibility: ui.Accessibility{Name: label}, Change: func(e ui.ChangeEvent) {
			if e.Number == float32(value) {
				return
			}
			v, err := strconv.ParseFloat(strconv.FormatFloat(float64(e.Number), 'g', -1, 32), 64)
			if err != nil {
				return
			}
			next := c
			set(&next, v)
			apply(next)
		}})
	}
	field("jfet-idss", "IDSS at zero gate bias · mA", c.IDSS*1000, .01, 100, .1, func(c *workbench.JFETConfig, v float64) { c.IDSS = decimalControlSI(v, -3) })
	field("jfet-cutoff", "Gate cutoff magnitude · V", math.Abs(c.Cutoff), .1, 12, .1, func(c *workbench.JFETConfig, v float64) {
		c.Cutoff = -v
		if p.Kind == "pjfet" {
			c.Cutoff = v
		}
	})
	field("jfet-lambda", "Channel modulation · 1/V", c.Lambda, 0, .1, .001, func(c *workbench.JFETConfig, v float64) { c.Lambda = v })
	nodes = append(nodes, quiet("jfet-help", "Normally conducting depletion channel. Reverse gate bias reduces current; forward gate junctions conduct. Generic model, no intrinsic capacitance or breakdown."))
	if state.Mixed != nil {
		for _, q := range state.Mixed.JFETs {
			if q.ID == p.ID {
				nodes = append(nodes, quiet("jfet-reading", fmt.Sprintf("%s · %s\nVGS %.4g V · VDS %.4g V\nGate %.4g nA · drain %.4g mA\nSource %.4g mA · heat %.4g mW", measurementState(state.Mixed.Stale), q.State, q.VGS, q.VDS, q.GateCurrent*1e9, q.DrainCurrent*1000, q.SourceCurrent*1000, q.Power*1000)))
			}
		}
	}
	return nodes
}
