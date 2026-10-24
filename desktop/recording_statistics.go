package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) recordingStatisticsControls(state workbench.Snapshot) []ui.Node {
	s.pruneRecordedStatisticsRejections(state)
	r := state.Recording
	options := []ui.Option{{ID: "__none", Label: "Measurements off"}}
	for _, probe := range r.Scope.Available {
		options = append(options, ui.Option{ID: probe.ID, Label: probe.Label + " · " + probe.Unit})
	}
	selected := "__none"
	if r.Statistics != nil {
		selected = r.Statistics.Settings.ProbeID
	}
	nodes := []ui.Node{section("recorded-statistics-title", "Interval measurements"), {ID: "recorded-statistics-probe", Kind: ui.Selector, Options: options, SelectedID: selected, Disabled: r.Duration == 0, Accessibility: ui.Accessibility{Name: "Captured interval measurement probe"}, Change: func(e ui.ChangeEvent) {
		p := workbench.RecordingStatisticsParams{ID: r.ID}
		if e.SelectedID != "__none" {
			p.ProbeID, p.End = e.SelectedID, r.LastSampleTime
			if r.Statistics != nil {
				p.Start, p.End = r.Statistics.Settings.Start, r.Statistics.Settings.End
			}
		}
		if response := s.invoke("recording.statistics.set", p); response.Error == nil {
			s.pruneRecordedStatisticsRejections(response.Result.(workbench.Snapshot))
		}
	}}}
	if r.Duration == 0 {
		return append(nodes, quiet("recorded-statistics-unavailable", "A DC point has no time interval to average."))
	}
	if r.Statistics == nil {
		return nodes
	}
	result := r.Statistics
	cfg := workbench.RecordingStatisticsParams(result.Settings)
	nodes = append(nodes, quiet("recorded-statistics-help", "Time weighted · linear between saved rows.\nIndependent of plot interval; no unseen peaks or steady-state guarantee."))
	for _, field := range []struct {
		name  string
		value float64
	}{{"start", cfg.Start}, {"end", cfg.End}} {
		field := field
		entry := timeField("recorded-statistics-"+field.name, "Measurement interval "+field.name, field.value, false, func(text string) {
			s.changeRecordedStatisticsTime(state, field.name, text)
		})
		if rejected, ok := s.recordedStatisticsRejected[field.name]; ok {
			entry.Text = rejected.text
		}
		nodes = append(nodes, quiet("recorded-statistics-"+field.name+"-label", "Measurement "+field.name+" · time"), entry)
	}
	if len(s.recordedStatisticsRejected) != 0 {
		nodes = append(nodes, quiet("recorded-statistics-pending", "Showing the last accepted measurement.\nCorrect the interval to update these results."))
	}
	nodes = append(nodes, quiet("recorded-statistics-values", fmt.Sprintf("%s · %.6g → %.6g ms\nAverage %.6g %s\nRMS %.6g %s\nAC ripple RMS %.6g %s\nPeak-to-peak %.6g %s", result.Probe.Label, cfg.Start*1000, cfg.End*1000, result.Mean, result.Probe.Unit, result.RMS, result.Probe.Unit, result.ACRMS, result.Probe.Unit, result.PeakToPeak, result.Probe.Unit)), quiet("recorded-statistics-extrema", fmt.Sprintf("Min %.6g · max %.6g %s\n%d positive-time segments · %.6g ms", result.Minimum, result.Maximum, result.Probe.Unit, result.Segments, result.Duration*1000)))
	return nodes
}
