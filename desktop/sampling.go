package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) lcSamplingControls(state workbench.Snapshot) []ui.Node {
	advice := state.LCSampling
	if advice == nil || !advice.Unresolved {
		return nil
	}
	text := fmt.Sprintf("%s / %s reference: ≈%.4g µs\nSuggested step: ≈%.4g µs", advice.InductorID, advice.CapacitorID, advice.ReferencePeriod*1e6, advice.RecommendedStep*1e6)
	if advice.MinimumLimited {
		text = "This LC scale is faster than the 1 µs scope limit. Fitting cannot fully resolve it."
	}
	nodes := []ui.Node{section("lc-sampling-title", "Live LC sampling"), quiet("lc-sampling-summary", text)}
	if advice.CanFit {
		target := "part-search"
		if workbench.IsTransientAnalysis(state.Analysis) || state.Transient != nil {
			target = "setting-step"
		}
		nodes = append(nodes, s.focusAfterAction(s.action("fit-circuit-sampling", "Fit sampling", "simulation.sampling.fit", struct{}{}), target))
		nodes = append(nodes, quiet("lc-sampling-restart", "Running experiments restart. If paused, use Restart or Resume to apply."))
	}
	note := "Value-based reference, not measured timing. Connections, damping and active parts can change it. Compare finer steps."
	if advice.MinimumLimited {
		note = "Value-based reference, not measured timing. Larger L/C values may give a slower experiment that fits the scope's resolution."
	}
	nodes = append(nodes, quiet("lc-sampling-note", note))
	if state.Recording != nil && state.Recording.Viewing {
		nodes = append(nodes, quiet("lc-sampling-recorded-note", "Live sampling only. This saved capture keeps its original samples."))
	}
	return nodes
}
