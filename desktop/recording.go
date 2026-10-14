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
	nodes = append(nodes, s.action("recorded-whole-window", "Show complete recording", "recording.window.set", workbench.RecordingWindowParams{End: r.Duration}))
	for _, side := range []string{"start", "end"} {
		side := side
		value := r.Scope.Window.Start
		if side == "end" {
			value = r.Scope.Window.End
		}
		nodes = append(nodes, quiet("recorded-window-label-"+side, "Recorded "+side+" · time"), timeField("recorded-window-"+side, "Recorded interval "+side, value, r.Duration == 0, func(text string) {
			p := workbench.RecordingWindowParams{Start: r.Scope.Window.Start, End: r.Scope.Window.End}
			s.invokeAtCommandRevision("recording.window.set", timeParams(p, side+"_s", text), state.CommandRevision)
		}))
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
	return nodes
}
func (s *screen) recordingPanel(state workbench.Snapshot) ui.Node {
	r := state.Recording
	status := "paused"
	if r.Playing {
		status = "playing"
	}
	caption := fmt.Sprintf("%s · sample %d / %d · t %.5g s", r.ID, r.Index, r.Total-1, r.Time)
	controls := []ui.Node{section("recorded-scope-title", "Recorded scope"), quiet("recorded-scope-caption", caption), s.action("recorded-back-live", "View live", "recording.view.set", workbench.RecordingViewParams{Mode: "live"})}
	for _, direction := range []string{"previous", "next"} {
		direction := direction
		n := s.action("recorded-"+direction, map[string]string{"previous": "Previous", "next": "Next"}[direction], "recording.step", workbench.RecordingStepParams{Direction: direction})
		n.Disabled = direction == "previous" && r.Index == 0 || direction == "next" && r.Index == r.Total-1
		controls = append(controls, n)
	}
	speed := r.Speed
	if speed == 0 {
		speed = .1
	}
	play := s.action("recorded-play", "Play", "recording.play", workbench.RecordingPlayParams{Speed: speed})
	play.Disabled = r.Duration == 0 || r.Playing
	pause := s.action("recorded-pause", "Pause", "recording.pause", struct{}{})
	pause.Disabled = !r.Playing
	controls = append(controls, play, pause)
	position := []ui.Node{quiet("recorded-status", status), {ID: "recorded-index", Kind: ui.NumberField, Number: float32(r.Index), MinWidth: 90, Numeric: ui.Numeric{Min: 0, Max: float32(r.Total - 1), Step: 1}, Accessibility: ui.Accessibility{Name: "Recorded sample index"}, Change: func(e ui.ChangeEvent) {
		index := int(e.Number)
		s.invoke("recording.seek", workbench.RecordingSeekParams{Index: &index})
	}}, timeField("recorded-time", "Recorded seek time", r.Time, false, func(text string) {
		s.invokeAtCommandRevision("recording.seek", map[string]any{"time_s": text}, state.CommandRevision)
	}), quiet("recorded-speed-label", "sim s / wall s"), {ID: "recorded-speed", Kind: ui.NumberField, Number: float32(speed), MinWidth: 80, Numeric: ui.Numeric{Min: .001, Max: 10, Step: .001}, Accessibility: ui.Accessibility{Name: "Recorded playback speed"}, Change: func(e ui.ChangeEvent) {
		s.invoke("recording.speed.set", workbench.RecordingPlayParams{Speed: float64(e.Number)})
	}}}
	legend := []string{}
	for i, trace := range r.Scope.Traces {
		legend = append(legend, scopeProbeColors(r.Scope.Traces)[i]+": "+trace.Probe.Label)
	}
	policy := fmt.Sprintf("%d exact interval samples", r.WindowSamples)
	if r.PlotReduced {
		policy += " · plot reduced with bucket extrema preserved"
	}
	s.scope.Revision = state.Revision
	rowStyle := ui.Style{Gap: ui.Some(float32(6)), Align: ui.Some(ui.AlignCenter), MaxSize: ui.Some(ui.Size{Height: 32})}
	return ui.Node{ID: "scope-panel", Kind: ui.Column, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 280}), Padding: ui.Some(ui.UniformInsets(10)), Gap: ui.Some(float32(4)), Background: ui.Some(paper)}, Children: []ui.Node{{ID: "recorded-toolbar", Kind: ui.Row, Children: controls, Style: rowStyle}, {ID: "recorded-position", Kind: ui.Row, Children: position, Style: rowStyle}, {ID: "rc-scope", Kind: ui.Extension, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 120}), MaxSize: ui.Some(ui.Size{Height: 120})}, Custom: s.scope}, quiet("recorded-legend", strings.Join(legend, " · ")+" · "+policy), s.recordingCursors(state)}}
}
func (s *screen) recordingCursors(state workbench.Snapshot) ui.Node {
	r := state.Recording
	nodes := []ui.Node{}
	for _, c := range []struct {
		id    string
		value *workbench.ScopeCursor
	}{{"a", r.Scope.A}, {"b", r.Scope.B}} {
		c := c
		nodes = append(nodes, buttonStyle(s.action("recorded-cursor-select-"+c.id, strings.ToUpper(c.id), "recording.cursor.select", workbench.CursorSelectParams{Cursor: c.id}), r.Scope.ActiveCursor == c.id))
		at := 0.
		if c.value != nil {
			at = c.value.Requested
		}
		nodes = append(nodes, timeField("recorded-cursor-"+c.id, "Recorded cursor "+strings.ToUpper(c.id), at, false, func(text string) {
			s.invokeAtCommandRevision("recording.cursor.set", map[string]any{"cursor": c.id, "time_s": text}, state.CommandRevision)
		}))
		clear := s.action("recorded-cursor-clear-"+c.id, "×", "recording.cursor.clear", workbench.CursorSelectParams{Cursor: c.id})
		clear.Disabled = c.value == nil
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
	return ui.Node{ID: "recorded-cursors", Kind: ui.Row, Children: nodes, Style: ui.Style{Gap: ui.Some(float32(6)), Align: ui.Some(ui.AlignCenter)}}
}
