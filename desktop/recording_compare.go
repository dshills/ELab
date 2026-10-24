package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) recordingComparisonControls(state workbench.Snapshot) []ui.Node {
	s.pruneRecordedComparisonRejection(state)
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
		next := workbench.RecordingCompareParams{ID: r.ID, ReferenceID: reference}
		if r.Comparison != nil {
			next.ShowStatistics = r.Comparison.Settings.ShowStatistics
			next.ShowOverlay = r.Comparison.Settings.ShowOverlay
		}
		if response := s.invoke("recording.compare.set", next); response.Error == nil {
			// A reference choice explicitly restarts at zero, including a
			// choice of the same reference whose accepted time is already zero.
			s.recordedComparisonRejected = nil
		}
	}}}
	if r.Comparison == nil {
		return nodes
	}
	comparison := r.Comparison
	entry := timeField("recorded-compare-time", "Requested comparison time", at, false, func(text string) {
		s.changeRecordedComparisonTime(state, text)
	})
	if rejected := s.recordedComparisonRejected; rejected != nil {
		entry.Text = rejected.text
	}
	nodes = append(nodes, quiet("recorded-compare-time-label", "Compare at · time"), entry)
	if s.recordedComparisonRejected != nil {
		nodes = append(nodes, quiet("recorded-compare-pending", "Showing the last accepted comparison.\nCorrect the time to update these results."))
	}
	nodes = append(nodes, ui.Node{ID: "recorded-compare-overlay", Kind: ui.Checkbox, Text: "Overlay reference", Checked: comparison.Settings.ShowOverlay, Accessibility: ui.Accessibility{Name: "Overlay captured reference B", Help: "A solid, B dashed; original captured seconds with shared voltage/current axes. No alignment or extrapolation. Plot cursors stay on selected A."}, Change: func(e ui.ChangeEvent) {
		next := workbench.RecordingCompareParams(comparison.Settings)
		next.ShowOverlay = e.Checked
		s.invoke("recording.compare.set", next)
	}})
	if overlay := r.ReferenceOverlay; overlay != nil {
		nodes = append(nodes, quiet("recorded-reference-identity", fmt.Sprintf("A solid · %s\nB dashed · %s\nCaptured seconds from startup\nB %.5g s · step %.5g ms", r.ID, overlay.ID, overlay.Duration, overlay.Step*1000)))
		if !overlay.Scope.WindowValid {
			nodes = append(nodes, quiet("recorded-reference-unavailable", overlay.Scope.WindowReason))
		} else if overlay.PlotReduced {
			nodes = append(nodes, quiet("recorded-reference-reduction", "Reference plot uses bucket extrema.\nExact captured rows stay unchanged."))
		}
	}
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
