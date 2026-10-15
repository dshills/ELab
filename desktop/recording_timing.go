package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) recordingTimingControls(state workbench.Snapshot) []ui.Node {
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
	nodes = append(nodes, quiet("recorded-timing-help", "Saved-row brackets; no interpolation.\nTiming interval is separate from plot."), quiet("recorded-timing-level-label", "Threshold · "+result.Probe.Unit), ui.Node{ID: "recorded-timing-level", Kind: ui.TextField, Text: cfg.Level, Accessibility: ui.Accessibility{Name: "Threshold level in " + result.Probe.Unit}, Change: func(e ui.ChangeEvent) {
		next := cfg
		next.Level = e.Text
		next.Offset = 0
		s.invoke("recording.timing.set", next)
	}}, ui.Node{ID: "recorded-timing-direction", Kind: ui.Selector, Options: []ui.Option{{ID: "both", Label: "Both directions"}, {ID: "rising", Label: "Rising arrivals"}, {ID: "falling", Label: "Falling arrivals"}}, SelectedID: cfg.Direction, Accessibility: ui.Accessibility{Name: "Threshold arrival direction"}, Change: func(e ui.ChangeEvent) {
		next := cfg
		next.Direction = e.SelectedID
		next.Offset = 0
		s.invoke("recording.timing.set", next)
	}})
	for _, field := range []struct {
		name  string
		value float64
	}{{"start", cfg.Start}, {"end", cfg.End}} {
		field := field
		nodes = append(nodes, quiet("recorded-timing-"+field.name+"-label", "Timing "+field.name+" · time"), timeField("recorded-timing-"+field.name, "Threshold interval "+field.name, field.value, false, func(text string) {
			p := map[string]any{"id": cfg.ID, "probe_id": cfg.ProbeID, "level": cfg.Level, "direction": cfg.Direction, "start_s": cfg.Start, "end_s": cfg.End, "limit": cfg.Limit, "offset": 0}
			p[field.name+"_s"] = text
			s.invokeAtCommandRevision("recording.timing.set", p, state.CommandRevision)
		}))
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
	if result.Total == 0 {
		nodes = append(nodes, quiet("recorded-timing-none", "No sampled threshold arrival.\nCheck level, direction and interval."))
	}
	for i, crossing := range result.Crossings {
		nodes = append(nodes, quiet(fmt.Sprintf("recorded-timing-edge-%d", i), fmt.Sprintf("%s · rows %d → %d\n%.6g → %.6g ms\n%.5g → %.5g %s", crossing.Direction, crossing.Before.Index, crossing.After.Index, crossing.Before.Time*1000, crossing.After.Time*1000, crossing.Before.Value, crossing.After.Value, result.Probe.Unit)))
	}
	if cfg.Offset > 0 {
		previous := cfg
		previous.Offset = max(0, cfg.Offset-cfg.Limit)
		nodes = append(nodes, s.action("recorded-timing-previous", "Previous arrivals", "recording.timing.set", previous))
	}
	if result.NextOffset < result.Total {
		next := cfg
		next.Offset = result.NextOffset
		nodes = append(nodes, s.action("recorded-timing-next", "Next arrivals", "recording.timing.set", next))
	}
	return nodes
}
