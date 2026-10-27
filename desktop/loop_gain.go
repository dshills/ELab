package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func loopMarginReadouts(m *workbench.LoopMargins) []ui.Node {
	nodes := []ui.Node{section("loop-margins", "Loop margin estimates"), quiet("loop-method", "Controlled-source return ratio\nSampled crossings; no stability verdict")}
	for k, c := range m.Unity {
		nodes = append(nodes, quiet(fmt.Sprintf("loop-phase-%d", k), fmt.Sprintf("Unity %.6g Hz · phase margin %.4g°", c.Frequency, c.Margin)))
	}
	for k, c := range m.Phase {
		nodes = append(nodes, quiet(fmt.Sprintf("loop-gain-%d", k), fmt.Sprintf("Phase %.6g Hz · gain margin %.4g dB", c.Frequency, c.Margin)))
	}
	if m.UnityReason != "" {
		nodes = append(nodes, quiet("loop-unity-unavailable", m.UnityReason))
	}
	if m.PhaseReason != "" {
		nodes = append(nodes, quiet("loop-phase-unavailable", m.PhaseReason))
	}
	return nodes
}
