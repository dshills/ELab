package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) scopeVisibilityButton(state workbench.Snapshot) ui.Node {
	text := "Show scope"
	if state.ScopeVisible {
		text = "Hide scope"
	}
	n := s.action("scope-visibility", text, "scope.visibility.set", workbench.ScopeVisibilityParams{Visible: !state.ScopeVisible})
	n.Accessibility.Help = "Expand or collapse the dock without changing simulation or recorded measurements"
	return n
}

// Keep the instruments discoverable without inventing a trace for a DC result.
// This is explanatory UI only: opening a tab never changes the circuit or starts
// a recording. Recorded analysis remains usable independently of the live graph.
func instrumentEmptyNotice(state workbench.Snapshot, recorded bool) ui.Node {
	help := "There are no live voltage or current samples over time. A DC operating point alone has no waveform. Open a transient example such as RC charging or an LED flasher, or open a recording in Simulate → Recorded runs."
	if len(state.Components) == 0 {
		help = "Build a circuit or open a transient example such as RC charging or an LED flasher. Scope and Spectrum need samples over time; a DC operating point alone has no waveform. You can also open a recording in Simulate → Recorded runs."
	} else if workbench.IsTransientAnalysis(state.Analysis) {
		help = "Waiting for a valid transient run. Check the circuit diagnostics, then run the simulation to collect samples over time. You can also open a recording in Simulate → Recorded runs."
	}
	if recorded {
		help = "Choose a captured run and probe, then Analyze. To capture or open time-based data, use Simulate → Recorded runs. A DC operating point alone has no waveform; live analysis needs a transient circuit."
	}
	return ui.Node{ID: "instrument-empty", Kind: ui.Column, Style: ui.Style{Gap: ui.Some(float32(8))}, Children: []ui.Node{
		section("instrument-empty-title", "No time-based data"),
		quiet("instrument-empty-help", help),
	}}
}
func (s *screen) collapsedScope(state workbench.Snapshot) ui.Node {
	caption := "Scope collapsed · no live time-based data"
	if record := state.Recording; record != nil && record.Viewing {
		caption = fmt.Sprintf("Recorded scope collapsed · sample %d · %.4g s", record.Index, record.Time)
	} else if state.Transient != nil {
		status := "live"
		if state.Paused {
			status = "paused"
		}
		if state.Transient.Stale {
			status = "stale"
		}
		caption = fmt.Sprintf("Scope collapsed · %.4g s · %s", state.Transient.Sample.Time, status)
	}
	style := chromeStyle()
	style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 10, Right: 10})
	return ui.Node{ID: "scope-panel", Kind: ui.Toolbar, Style: style, Children: []ui.Node{
		s.action("scope-expand", "Show scope", "scope.visibility.set", workbench.ScopeVisibilityParams{Visible: true}), quiet("scope-collapsed-caption", caption),
	}}
}
