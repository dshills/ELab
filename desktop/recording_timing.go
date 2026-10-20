package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) recordingTimingControls(state workbench.Snapshot) []ui.Node {
	s.pruneRecordedTimingRejections(state)
	r := state.Recording
	options := []ui.Option{{ID: "__none", Label: "Timing off"}}
	for _, probe := range r.Scope.Available {
		options = append(options, ui.Option{ID: probe.ID, Label: probe.Label + " · " + probe.Unit})
	}
	selected := "__none"
	if r.Timing != nil {
		selected = r.Timing.Settings.ProbeID
	}
	nodes := []ui.Node{section("recorded-timing-title", "Threshold timing"), ui.Node{ID: "recorded-timing-probe", Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Captured threshold timing probe"}, Change: func(e ui.ChangeEvent) {
		p := workbench.RecordingTimingParams{ID: r.ID, Level: "0", Direction: "both"}
		if e.SelectedID != "__none" {
			p.ProbeID = e.SelectedID
			p.End = r.Duration
		}
		s.invoke("recording.timing.set", p)
	}}}
	if r.Timing == nil {
		return nodes
	}
	result := r.Timing
	cfg := workbench.RecordingTimingParams(result.Settings)
	nodes = append(nodes, quiet("recorded-timing-help", "Saved-row brackets; no interpolation.\nTiming interval is separate from plot."), quiet("recorded-timing-level-label", "Threshold · "+result.Probe.Unit), ui.Node{ID: "recorded-timing-level", Kind: ui.TextField, Text: s.recordedTimingEntryText(cfg, "level"), Accessibility: ui.Accessibility{Name: "Threshold level in " + result.Probe.Unit}, Change: func(e ui.ChangeEvent) {
		s.changeRecordedTimingField(state, "level", e.Text)
	}}, ui.Node{ID: "recorded-timing-direction", Kind: ui.Selector, Options: []ui.Option{{ID: "both", Label: "Both directions"}, {ID: "rising", Label: "Rising arrivals"}, {ID: "falling", Label: "Falling arrivals"}}, SelectedID: cfg.Direction, Accessibility: ui.Accessibility{Name: "Threshold arrival direction"}, Change: func(e ui.ChangeEvent) {
		s.changeRecordedTimingDirection(state, e.SelectedID)
	}})
	nodes = append(nodes, quiet("recorded-timing-second-level-label", "Other threshold · "+result.Probe.Unit+" (optional)"), ui.Node{ID: "recorded-timing-second-level", Kind: ui.TextField, Text: s.recordedTimingEntryText(cfg, "second_level"), Accessibility: ui.Accessibility{Name: "Other transition threshold in " + result.Probe.Unit}, Change: func(e ui.ChangeEvent) {
		s.changeRecordedTimingField(state, "second_level", e.Text)
	}})
	for _, field := range []struct {
		name  string
		value float64
	}{{"start", cfg.Start}, {"end", cfg.End}} {
		field := field
		entry := timeField("recorded-timing-"+field.name, "Threshold interval "+field.name, field.value, false, func(text string) {
			s.changeRecordedTimingField(state, field.name+"_s", text)
		})
		entry.Text = s.recordedTimingEntryText(cfg, field.name+"_s")
		nodes = append(nodes, quiet("recorded-timing-"+field.name+"-label", "Timing "+field.name+" · time"), entry)
	}
	nodes = append(nodes, quiet("recorded-timing-count", fmt.Sprintf("%d arrivals · %d saved rows", result.Total, result.WindowSamples)))
	for _, spacing := range []struct {
		name  string
		value *workbench.TimingSpacing
	}{{"Rising", result.RisingSpacing}, {"Falling", result.FallingSpacing}} {
		if spacing.value != nil {
			nodes = append(nodes, quiet("recorded-timing-spacing-"+spacing.name, fmt.Sprintf("First %s spacing\n%.6g → %.6g ms", spacing.name, spacing.value.Minimum*1000, spacing.value.Maximum*1000)))
		}
	}
	nodes = append(nodes, recordingPulseReadouts(result)...)
	nodes = append(nodes, recordingTransitionReadouts(result)...)
	if result.Total == 0 {
		nodes = append(nodes, quiet("recorded-timing-none", "No sampled threshold arrival.\nCheck level, direction and interval."))
	}
	for i, crossing := range result.Crossings {
		nodes = append(nodes, quiet(fmt.Sprintf("recorded-timing-edge-%d", i), fmt.Sprintf("%s · rows %d → %d\n%.6g → %.6g ms\n%.5g → %.5g %s", crossing.Direction, crossing.Before.Index, crossing.After.Index, crossing.Before.Time*1000, crossing.After.Time*1000, crossing.Before.Value, crossing.After.Value, result.Probe.Unit)))
	}
	if cfg.Offset > 0 {
		nodes = append(nodes, s.recordedTimingPageAction(state, "previous"))
	}
	if result.NextOffset < result.Total {
		nodes = append(nodes, s.recordedTimingPageAction(state, "next"))
	}
	return nodes
}

func recordingPulseReadouts(result *workbench.RecordingTiming) []ui.Node {
	if result.HighPulse == nil && result.LowPulse == nil && result.Cycle == nil {
		return nil
	}
	nodes := []ui.Node{quiet("recorded-pulse-help", "High time spans rising → falling arrivals. Summaries use both directions across the timing interval, regardless of the arrival filter.")}
	for _, pulse := range []struct {
		id, label string
		value     *workbench.TimingSpacing
	}{{"high", "First high time", result.HighPulse}, {"low", "First low time", result.LowPulse}} {
		if pulse.value != nil {
			nodes = append(nodes, quiet("recorded-pulse-"+pulse.id, fmt.Sprintf("%s\n%.6g → %.6g ms", pulse.label, pulse.value.Minimum*1000, pulse.value.Maximum*1000)))
		}
	}
	cycle := result.Cycle
	if cycle == nil {
		return append(nodes, quiet("recorded-pulse-cycle-missing", "Duty cycle needs three consecutive rising → falling → rising arrivals."))
	}
	nodes = append(nodes, quiet("recorded-pulse-period", fmt.Sprintf("First rising cycle\n%.6g → %.6g ms", cycle.Period.Minimum*1000, cycle.Period.Maximum*1000)))
	if cycle.Duty == nil {
		nodes = append(nodes, quiet("recorded-pulse-duty-unavailable", cycle.Reason))
	} else {
		nodes = append(nodes, quiet("recorded-pulse-duty", fmt.Sprintf("Duty cycle · first rising cycle\n%.6g → %.6g %% high\nSample bounds; periodicity is not verified.", cycle.Duty.Minimum*100, cycle.Duty.Maximum*100)))
	}
	return nodes
}

func recordingTransitionReadouts(result *workbench.RecordingTiming) []ui.Node {
	tr := result.Transitions
	if tr == nil {
		return nil
	}
	nodes := []ui.Node{quiet("recorded-transition-levels", fmt.Sprintf("Transition levels · %.6g → %.6g %s\nSaved brackets; no interpolation.", tr.Lower, tr.Upper, result.Probe.Unit))}
	for _, entry := range []struct {
		id, label string
		value     *workbench.TimingSpacing
	}{
		{"rise", "First rise · low → high", tr.Rise}, {"fall", "First fall · high → low", tr.Fall},
	} {
		text := entry.label + "\nNo complete sampled traversal in interval."
		if v := entry.value; v != nil {
			text = fmt.Sprintf("%s\n%.6g → %.6g ms\nStart rows %d → %d; end rows %d → %d", entry.label, v.Minimum*1000, v.Maximum*1000, v.First.Before.Index, v.First.After.Index, v.Second.Before.Index, v.Second.After.Index)
		}
		nodes = append(nodes, quiet("recorded-transition-"+entry.id, text))
	}
	return nodes
}
