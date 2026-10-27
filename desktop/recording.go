package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) recordingInspector(state workbench.Snapshot) []ui.Node {
	r := state.Recording
	if r == nil || !r.Viewing {
		return nil
	}
	origin := "Captured verification"
	if r.Imported {
		origin = "Imported recorded data"
	}
	nodes := []ui.Node{section("recorded-inspector-title", "Recorded measurements"), quiet("recorded-origin", origin+" · "+r.OriginID), quiet("recorded-measure-help", "Measurements use exact saved samples. Playback and cursors do not change the live circuit.")}
	for _, reading := range r.Readings {
		scale, unit := 1., reading.Probe.Unit
		if unit == "A" {
			scale, unit = 1000, "mA"
		}
		nodes = append(nodes, quiet("recorded-current-"+reading.Probe.ID, fmt.Sprintf("%s · sample %d\n%.5g %s", reading.Probe.Label, r.Index, reading.Value*scale, unit)))
	}
	options := []ui.Option{{ID: "__none", Label: "Hidden"}}
	for _, probe := range r.Scope.Available {
		options = append(options, ui.Option{ID: probe.ID, Label: probe.Label + " · " + probe.Unit})
	}
	for i := 0; i < workbench.MaxScopeProbes; i++ {
		i := i
		selected := "__none"
		if i < len(r.Scope.Selected) {
			selected = r.Scope.Selected[i]
		}
		nodes = append(nodes, ui.Node{ID: fmt.Sprintf("recorded-probe-%d", i), Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: fmt.Sprintf("Recorded probe %d", i+1)}, Change: func(e ui.ChangeEvent) {
			ids := append([]string{}, r.Scope.Selected...)
			if i < len(ids) {
				ids = append(ids[:i], ids[i+1:]...)
			}
			if e.SelectedID != "__none" {
				for _, id := range ids {
					if id == e.SelectedID {
						return
					}
				}
				at := min(i, len(ids))
				ids = append(ids, "")
				copy(ids[at+1:], ids[at:])
				ids[at] = e.SelectedID
			}
			s.invoke("recording.probes.set", workbench.ProbeParams{IDs: ids})
		}})
	}
	nodes = append(nodes, s.intervalResetAction(state, "recorded-whole-window", "Show complete recording", workbench.RecordingWindowParams{End: r.LastSampleTime}, r.ID))
	for _, side := range []string{"start", "end"} {
		side := side
		value := r.Scope.Window.Start
		if side == "end" {
			value = r.Scope.Window.End
		}
		field := timeField("recorded-window-"+side, "Recorded interval "+side, value, r.Duration == 0, func(text string) {
			s.changeIntervalTime(state, side, text, r.ID)
		})
		s.applyIntervalTimeRejection(state, side, r.ID, &field)
		nodes = append(nodes, quiet("recorded-window-label-"+side, "Recorded "+side+" · time"), field)
	}

	if !r.Scope.WindowValid {
		nodes = append(nodes, quiet("recorded-no-interval-samples", r.Scope.WindowReason))
	}
	units := map[string]string{}
	for _, probe := range r.Scope.Available {
		units[probe.ID] = probe.Unit
	}
	for _, stat := range r.Scope.Statistics {
		if stat.Count == 0 {
			continue
		}
		nodes = append(nodes, quiet("recorded-stat-"+stat.ProbeID, fmt.Sprintf("%s · %d exact samples\nMin %.5g · Max %.5g %s\nMean %.5g · RMS %.5g %s", stat.ProbeID, stat.Count, stat.Minimum, stat.Maximum, units[stat.ProbeID], stat.Mean, stat.RMS, units[stat.ProbeID])))
	}
	for _, reading := range r.Scope.Readings {
		nodes = append(nodes, quiet("recorded-cursor-reading-"+reading.ProbeID, fmt.Sprintf("%s · A %.5g · B %.5g\nΔ %.5g %s", reading.ProbeID, reading.A, reading.B, reading.Delta, units[reading.ProbeID])))
	}

	nodes = append(nodes, s.recordingStatisticsControls(state)...)
	nodes = append(nodes, s.stepResponseControls(state)...)
	nodes = append(nodes, s.recordingTimingControls(state)...)
	nodes = append(nodes, s.recordingComparisonControls(state)...)

	// Source inspection uses the same read-only metadata command as agents.
	raw, _ := json.Marshal(workbench.IDParams{ID: r.ID})
	response := s.service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "ui-recorded-source", Command: "operation.inspect", Params: raw}, "user")
	if response.Error == nil {
		metadata := response.Result.(workbench.OperationView).Metadata
		if metadata != nil {
			nodes = append(nodes, section("recorded-source-title", "Captured source"), quiet("recorded-source-settings", fmt.Sprintf("%s\n%s · %s\nSample step %.5g ms · startup %.5g V", metadata.Analysis, metadata.Engine, metadata.InputMode, metadata.Settings.StepSeconds*1000, metadata.Settings.InitialVoltage)))
			for _, part := range metadata.Graph.Components {
				value := ""
				if part.Unit != "" {
					value = fmt.Sprintf(" · %.5g %s", part.Value, part.Unit)
				}
				if part.InitialVoltage != nil {
					value += fmt.Sprintf(" · startup %.5g V", *part.InitialVoltage)
				}
				nodes = append(nodes, quiet("recorded-source-"+part.ID, part.ID+" · "+part.Name+value))
			}
			wires := []string{}
			for _, wire := range metadata.Graph.Wires {
				status := ""
				if !wire.Connected {
					status = " · disconnected"
				}
				wires = append(wires, wire.From+" → "+wire.To+status)
			}
			nodes = append(nodes, quiet("recorded-source-wires", strings.Join(wires, "\n")))
		}
	}
	for i := range nodes {
		owner := "recording:" + r.ID
		if r.Statistics != nil && strings.HasPrefix(nodes[i].ID, "recorded-statistics-") {
			owner += ":statistics:" + r.Statistics.Settings.ProbeID
		}
		if r.Comparison != nil && strings.HasPrefix(nodes[i].ID, "recorded-compare-") {
			owner += ":comparison:" + r.Comparison.Settings.ReferenceID
		}
		if r.StepResponse != nil && strings.HasPrefix(nodes[i].ID, "step-response-") {
			owner += ":step-response:" + r.StepResponse.Settings.ProbeID
		}
		if r.Timing != nil && strings.HasPrefix(nodes[i].ID, "recorded-timing-") {
			// Thresholds and interval drafts belong to this captured channel,
			// even when another channel has identical accepted settings.
			owner += ":timing:" + r.Timing.Settings.ProbeID
		}
		scopeOwnedControlState(&nodes[i], owner)
	}
	return nodes
}
func (s *screen) recordingPanel(state workbench.Snapshot) ui.Node {
	s.pruneRecordedPlaybackRejections(state)
	r := state.Recording
	status := "paused"
	if r.Playing {
		status = "playing"
	}
	caption := fmt.Sprintf("%s · sample %d / %d · t %.5g s", r.ID, r.Index, r.Total-1, r.Time)
	controls := []ui.Node{s.scopeVisibilityButton(state), section("recorded-scope-title", "Recorded scope"), quiet("recorded-scope-caption", caption), s.action("recorded-back-live", "View live", "recording.view.set", workbench.RecordingViewParams{Mode: "live"})}
	for _, direction := range []string{"previous", "next"} {
		direction := direction
		n := s.recordedPlaybackAction(state, direction)
		controls = append(controls, n)
	}
	speed := r.Speed
	if speed == 0 {
		speed = .1
	}
	play := s.recordedPlaybackAction(state, "")
	pause := s.action("recorded-pause", "Pause", "recording.pause", struct{}{})
	pause.Disabled = !r.Playing
	controls = append(controls, play, pause)
	// DC captures have one row and no editable index range. Vellum numeric
	// fields require a positive span, so expose the fixed position as text.
	indexControl := quiet("recorded-index", fmt.Sprint(r.Index))
	indexControl.MinWidth = 90
	indexControl.Accessibility.Name = "Recorded sample index"
	if r.Total > 1 {
		numeric := ui.Numeric{Min: 0, Max: float32(r.Total - 1), Step: 1}
		index := float32(r.Index)
		if rejected, ok := s.recordedPlaybackRejected["recorded-index"]; ok && numeric.Validate(rejected.number) == nil {
			index = rejected.number
		}
		indexControl = ui.Node{ID: "recorded-index", Kind: ui.NumberField, Number: index, MinWidth: 90, Numeric: numeric, Accessibility: ui.Accessibility{Name: "Recorded sample index"}, Change: func(e ui.ChangeEvent) {
			// Preserve the entry for shared integer validation, as with agent
			// requests. Truncating here silently selected a different sample.
			s.changeRecordedPlaybackField(state, "recorded-index", e)
		}}
	}
	seek := timeField("recorded-time", "Recorded seek time", r.Time, false, func(text string) {
		s.changeRecordedPlaybackField(state, "recorded-time", ui.ChangeEvent{Text: text})
	})
	// Keep rejected input visible until correction or owner/value replacement.
	// Otherwise a fresh edit back to the accepted value looks unchanged to the
	// SDK settlement barrier, leaving the previous rejection stuck in place.
	if rejected, ok := s.recordedPlaybackRejected["recorded-time"]; ok {
		seek.Text = rejected.text
	}
	position := []ui.Node{quiet("recorded-status", status), indexControl, seek, quiet("recorded-speed-label", "sim s / wall s"), {ID: "recorded-speed", Kind: ui.NumberField, Number: float32(speed), MinWidth: 80, Numeric: ui.Numeric{Min: .001, Max: 10, Step: .001}, Accessibility: ui.Accessibility{Name: "Recorded playback speed"}, Change: func(e ui.ChangeEvent) {
		s.changeRecordedPlaybackField(state, "recorded-speed", e)
	}}}
	legend := []string{}
	if r.ReferenceOverlay != nil {
		legend = append(legend, "A solid · B dashed")
	}
	for i, trace := range r.Scope.Traces {
		legend = append(legend, scopeProbeColors(r.Scope.Traces)[i]+": "+trace.Probe.Label)
	}
	policy := fmt.Sprintf("%d exact interval samples", r.WindowSamples)
	if r.PlotReduced {
		policy += " · plot reduced with bucket extrema preserved"
	}
	s.scope.Revision = s.scopeAppearance.next(state.Revision, s.colorsForTheme(state.Theme))
	rowStyle := chromeStyle()
	panel := ui.Node{ID: "scope-panel", Kind: ui.Column, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 328}), Padding: ui.Some(ui.UniformInsets(10)), Gap: ui.Some(float32(4)), Background: ui.Some(paper)}, Children: []ui.Node{{ID: "recorded-toolbar", Kind: ui.Toolbar, Children: controls, Style: rowStyle}, {ID: "recorded-position", Kind: ui.Toolbar, Children: position, Style: rowStyle}, {ID: "rc-scope", Kind: ui.Extension, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 120}), MaxSize: ui.Some(ui.Size{Height: 120})}, Custom: s.scope}, quiet("recorded-legend", strings.Join(legend, " · ")+" · "+policy), s.recordingCursors(state)}}
	scopeOwnedControlState(&panel, "recording:"+r.ID)
	return panel
}
func (s *screen) recordingCursors(state workbench.Snapshot) ui.Node {
	r := state.Recording
	nodes := []ui.Node{}
	for _, c := range []struct {
		id    string
		value *workbench.ScopeCursor
	}{{"a", r.Scope.A}, {"b", r.Scope.B}} {
		c := c
		nodes = append(nodes, buttonStyle(s.recordedCursorAction(state, "recorded-cursor-select-"+c.id, strings.ToUpper(c.id), "recording.cursor.select", c.id), r.Scope.ActiveCursor == c.id))
		at := 0.
		if c.value != nil {
			at = c.value.Requested
		}
		field := timeField("recorded-cursor-"+c.id, "Recorded cursor "+strings.ToUpper(c.id), at, false, func(text string) {
			s.changeCursorTimeFor(state, c.id, text, r.ID)
		})
		s.applyCursorTimeRejection(state, c.id, r.ID, &field)
		nodes = append(nodes, field)
		clear := s.recordedCursorAction(state, "recorded-cursor-clear-"+c.id, "×", "recording.cursor.clear", c.id)
		clear.Disabled = clear.Disabled || c.value == nil
		nodes = append(nodes, clear)
	}
	caption := "ms · click plot for recorded cursor"
	if r.Scope.DeltaSeconds != nil {
		caption = fmt.Sprintf("Δt %.5g ms", *r.Scope.DeltaSeconds*1000)
		if r.Scope.Frequency != nil {
			caption += fmt.Sprintf(" · 1/|Δt| %.5g Hz", *r.Scope.Frequency)
		}
	}
	for _, c := range []*workbench.ScopeCursor{r.Scope.A, r.Scope.B} {
		if c != nil && !c.Valid {
			caption += " · " + c.Reason
		}
	}
	nodes = append(nodes, quiet("recorded-cursor-timing", caption))
	controls := ui.Node{ID: "recorded-cursors", Kind: ui.Toolbar, Children: nodes, Style: chromeStyle()}
	scopeOwnedControlState(&controls, "recording:"+r.ID)
	return controls
}
