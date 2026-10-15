package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) recordingComparisonControls(state workbench.Snapshot) []ui.Node {
	r := state.Recording
	options := []ui.Option{{ID: "__none", Label: "No comparison"}}
	for _, op := range state.Operations {
		if op.ID != r.ID && op.Status == "completed" {
			options = append(options, ui.Option{ID: op.ID, Label: op.ID + " · " + op.Kind})
		}
	}
	selected, at := "__none", 0.
	if r.Comparison != nil {
		selected, at = r.Comparison.Settings.ReferenceID, r.Comparison.Settings.Time
		found := false
		for _, option := range options {
			if option.ID == selected {
				found = true
			}
		}
		if !found {
			options = append(options, ui.Option{ID: selected, Label: "Recording unavailable", Disabled: true})
		}
	}
	nodes := []ui.Node{section("recorded-compare-title", "Compare captures"), quiet("recorded-compare-help", "A: selected · B: reference\nSaved samples; no interpolation."), ui.Node{ID: "recorded-compare-reference", Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Reference recording B"}, Change: func(e ui.ChangeEvent) {
		reference := e.SelectedID
		if reference == "__none" {
			reference = ""
		}
		// Start at a time every completed capture contains. Different durations
		// must not make selecting a shorter reference impossible.
		s.invoke("recording.compare.set", workbench.RecordingCompareParams{ID: r.ID, ReferenceID: reference, Time: 0})
	}}}
	if r.Comparison == nil {
		return nodes
	}
	comparison := r.Comparison
	nodes = append(nodes, quiet("recorded-compare-time-label", "Compare at · time"), timeField("recorded-compare-time", "Requested comparison time", at, false, func(text string) {
		s.invokeAtCommandRevision("recording.compare.set", map[string]any{"id": r.ID, "reference_id": comparison.Settings.ReferenceID, "time_s": text, "show_statistics": comparison.Settings.ShowStatistics}, state.CommandRevision)
	}))
	if !comparison.Available {
		return append(nodes, quiet("recorded-compare-unavailable", comparison.Reason))
	}
	nodes = append(nodes, ui.Node{ID: "recorded-compare-statistics", Kind: ui.Checkbox, Text: "Whole-run statistics", Checked: comparison.Settings.ShowStatistics, Accessibility: ui.Accessibility{Name: "Show whole-run comparison statistics"}, Change: func(e ui.ChangeEvent) {
		next := workbench.RecordingCompareParams(comparison.Settings)
		next.ShowStatistics = e.Checked
		s.invoke("recording.compare.set", next)
	}})
	a, b := comparison.A, comparison.B
	nodes = append(nodes, quiet("recorded-compare-times", fmt.Sprintf("A %s · %.5g ms\nB %s · %.5g ms", a.ID, a.Time*1000, b.ID, b.Time*1000)), quiet("recorded-compare-offsets", fmt.Sprintf("Sample offsets A %.5g / B %.5g ms", a.Offset*1000, b.Offset*1000)), quiet("recorded-compare-stats-help", "Statistics cover each whole run.\nSample mean/RMS depend on sampling.\nMatched terminals may be wired differently."))
	for _, reading := range comparison.Readings {
		scale, unit := 1., reading.Probe.Unit
		if unit == "A" {
			scale, unit = 1000, "mA"
		}
		caption := fmt.Sprintf("%s\nA %.5g · B %.5g %s\nB − A %.5g %s", reading.Probe.Label, reading.A*scale, reading.B*scale, unit, reading.Delta*scale, unit)
		sa, sb := reading.StatisticsA, reading.StatisticsB
		if comparison.Settings.ShowStatistics {
			caption += fmt.Sprintf("\nA range %.5g → %.5g %s\nB range %.5g → %.5g %s\nA mean/RMS %.5g / %.5g\nB mean/RMS %.5g / %.5g %s\nSamples A %d / B %d", sa.Minimum*scale, sa.Maximum*scale, unit, sb.Minimum*scale, sb.Maximum*scale, unit, sa.Mean*scale, sa.RMS*scale, sb.Mean*scale, sb.RMS*scale, unit, sa.Count, sb.Count)
		}
		nodes = append(nodes, quiet("recorded-compare-reading-"+reading.Probe.ID, caption))
	}
	if len(comparison.UnmatchedA)+len(comparison.UnmatchedB) > 0 {
		nodes = append(nodes, quiet("recorded-compare-unmatched", fmt.Sprintf("Unmatched probes: A %d · B %d", len(comparison.UnmatchedA), len(comparison.UnmatchedB))))
	}
	// Different durations/rates are useful experiment information, never an
	// implicit claim that the two nearest rows share an identical timestamp.
	if a.Time != b.Time || a.Step != b.Step {
		nodes = append(nodes, quiet("recorded-compare-resolution", "Different sample times or rates.\nDelta uses the rows shown above."))
	}
	return nodes
}
