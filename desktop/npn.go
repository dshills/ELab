package main

import (
	"fmt"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) npnControls(state workbench.Snapshot) []ui.Node {
	if state.Analysis != workbench.NPNAnalysis && state.Analysis != workbench.NPNTransientAnalysis {
		return nil
	}
	c, _, d := (workbench.Graph{Components: state.Components, Wires: state.Wires}).Compile()
	if d != nil {
		return nil
	}
	nodes := []ui.Node{section("npn-drive-title", "Base drive · "+c.Input.ID), quiet("npn-drive-value", fmt.Sprintf("%.3g V through %s · %.3g kΩ", c.Input.Value, c.BaseResistor.ID, c.BaseResistor.Value/1000))}
	if c.Analysis == workbench.NPNTransientAnalysis {
		nodes = append(nodes, quiet("npn-pulse-model", "Pulse-driven · instantaneous switching.\nUse Auto / High / Low below the scope."))
	} else if c.Input.ID != c.Source.ID {
		nodes = append(nodes, ui.Node{ID: "npn-drive", Kind: ui.Row, Style: ui.Style{Gap: ui.Some(float32(4))}, Children: []ui.Node{
			s.action("npn-off", "0 V", "component.set", workbench.ValueParams{ID: c.Input.ID, Value: "0 V"}),
			s.action("npn-active", "0.8 V", "component.set", workbench.ValueParams{ID: c.Input.ID, Value: "0.8 V"}),
			s.action("npn-on", "5 V", "component.set", workbench.ValueParams{ID: c.Input.ID, Value: "5 V"}),
		}}, quiet("npn-drive-help", "Edits the source value · undoable"))
	} else {
		nodes = append(nodes, quiet("npn-shared-source", "Supply and base drive share a source. Select it to edit both together."))
	}
	if r := state.NPN; r != nil && !r.Stale {
		nodes = append(nodes, quiet("npn-junctions", fmt.Sprintf("VBE %.3f V · IB %.2f µA\nIE %.3f mA · power %.3f mW", r.VBE, r.BaseCurrent*1e6, r.EmitterCurrent*1000, r.TransistorPower*1000)), quiet("npn-explanation", r.Explanation))
	}
	return nodes
}
