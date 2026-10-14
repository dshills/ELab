package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func probeColorName(p workbench.Probe) string {
	if p.Unit == "A" {
		if strings.Contains(p.ID, "2") {
			return "Rose"
		}
		return "Coral"
	}
	if p.Reference != "ground" && strings.HasPrefix(p.Positive, "C") {
		return "Purple"
	}
	if strings.Contains(p.Positive, "positive") {
		return "Amber"
	}
	if strings.Contains(p.ID, "2") {
		return "Blue"
	}
	return "Teal"
}
func scopeProbeColors(traces []workbench.ScopeTrace) []string {
	names := []string{}
	used := map[string]bool{}
	for _, trace := range traces {
		name := probeColorName(trace.Probe)
		if used[name] {
			for _, candidate := range []string{"Teal", "Blue", "Coral", "Rose", "Purple", "Amber"} {
				if !used[candidate] {
					name = candidate
					break
				}
			}
		}
		used[name] = true
		names = append(names, name)
	}
	return names
}
func probeColor(name string) color.NRGBA {
	switch name {
	case "Coral":
		return color.NRGBA{R: 220, G: 85, B: 75, A: 255}
	case "Rose":
		return color.NRGBA{R: 184, G: 56, B: 118, A: 255}
	case "Purple":
		return color.NRGBA{R: 126, G: 94, B: 180, A: 255}
	case "Amber":
		return color.NRGBA{R: 201, G: 137, B: 36, A: 255}
	case "Blue":
		return color.NRGBA{R: 67, G: 118, B: 201, A: 255}
	default:
		return accent
	}
}
func (s *screen) probeInspector(state workbench.Snapshot) []ui.Node {
	nodes := []ui.Node{section("probes-title", "Scope probes"), quiet("probes-help", "Choose up to six traces. Voltages reference ground or the named second pin; currents follow the named direction.")}
	for _, missing := range state.Scope.Unavailable {
		nodes = append(nodes, quiet("probe-unavailable-"+missing.ID, missing.ID+" · "+missing.Unit+" · unavailable\n"+missing.Reason), s.action("remove-probe-"+missing.ID, "Remove "+missing.ID, "scope.probe.remove", workbench.IDParams{ID: missing.ID}))
	}

	options := []ui.Option{{ID: "__none", Label: "Hidden"}}
	for _, p := range state.Scope.Available {
		options = append(options, ui.Option{ID: p.ID, Label: p.Label + " · " + p.Unit})
	}
	// Fixed slots keep changing a selection simple and allow an empty scope.
	for i := 0; i < workbench.MaxScopeProbes; i++ {
		i := i
		value := "__none"
		if i < len(state.Scope.Selected) {
			value = state.Scope.Selected[i]
		}
		nodes = append(nodes, ui.Node{ID: fmt.Sprintf("probe-%d", i), Kind: ui.Selector, Options: options, SelectedID: value, Accessibility: ui.Accessibility{Name: fmt.Sprintf("Scope probe %d", i+1)}, Change: func(e ui.ChangeEvent) {
			ids := append([]string{}, state.Scope.Selected...)
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
			s.invoke("scope.probes.set", workbench.ProbeParams{IDs: ids})
		}})
	}
	for i, trace := range state.Scope.Traces {
		p := trace.Probe
		nodes = append(nodes, s.action("inspect-trace-"+p.ID, "Inspect "+p.Label, "scope.trace.select", workbench.IDParams{ID: p.ID}), quiet("probe-direction-"+p.ID, scopeProbeColors(state.Scope.Traces)[i]+" · "+p.Positive+" → "+p.Reference))
	}
	if state.TraceFocus != nil {
		text := "Inspecting " + state.TraceFocus.ProbeID
		if state.TraceFocus.Unavailable {
			text += " · " + state.TraceFocus.Reason
		}
		nodes = append(nodes, quiet("trace-focus-status", text), s.action("trace-focus-clear", "Clear trace highlight", "scope.trace.select", workbench.IDParams{}))
	}
	for _, r := range state.Scope.Readings {
		unit := "V"
		scale := 1.
		for _, p := range state.Scope.Available {
			if p.ID == r.ProbeID && p.Unit == "A" {
				unit = "mA"
				scale = 1000
			}
		}
		nodes = append(nodes, quiet("cursor-reading-"+r.ProbeID, fmt.Sprintf("%s · A %.4g · B %.4g · Δ %.4g %s", r.ProbeID, r.A*scale, r.B*scale, r.Delta*scale, unit)))
	}
	nodes = append(nodes, s.scopeWindowControls(state)...)
	return nodes
}
func (s *screen) cursorControls(state workbench.Snapshot) ui.Node {
	nodes := []ui.Node{}
	t := state.Transient
	for _, cursor := range []struct {
		id    string
		value *workbench.ScopeCursor
	}{{"a", state.Scope.A}, {"b", state.Scope.B}} {
		cursor := cursor
		button := s.action("cursor-select-"+cursor.id, strings.ToUpper(cursor.id), "scope.cursor.select", workbench.CursorSelectParams{Cursor: cursor.id})
		nodes = append(nodes, buttonStyle(button, state.Scope.ActiveCursor == cursor.id))
		value := 0.
		if t != nil && len(t.Samples) > 0 {
			value = t.Samples[0].Time
		}
		if cursor.value != nil {
			value = cursor.value.Requested
		}
		nodes = append(nodes, timeField("cursor-time-"+cursor.id, "Cursor "+strings.ToUpper(cursor.id)+" time", value, t == nil || t.Stale, func(text string) {
			s.invokeAtCommandRevision("scope.cursor.set", map[string]any{"cursor": cursor.id, "time_s": text}, state.CommandRevision)
		}))
		clear := s.action("cursor-clear-"+cursor.id, "×", "scope.cursor.clear", workbench.CursorSelectParams{Cursor: cursor.id})
		clear.Disabled = cursor.value == nil
		nodes = append(nodes, clear)
	}
	caption := "ms · click / drag plot to place selected cursor"
	if state.Scope.DeltaSeconds != nil {
		caption = fmt.Sprintf("A %.4g · B %.4g ms · Δt %.4g ms", state.Scope.A.Time*1000, state.Scope.B.Time*1000, *state.Scope.DeltaSeconds*1000)
		if state.Scope.Frequency != nil {
			caption += fmt.Sprintf(" · 1/|Δt| %.4g Hz", *state.Scope.Frequency)
		}
	}
	for _, c := range []*workbench.ScopeCursor{state.Scope.A, state.Scope.B} {
		if c != nil && !c.Valid {
			caption += " · " + c.Reason
		}
	}
	nodes = append(nodes, quiet("cursor-measurement", caption))
	// The inspector contains per-trace readings; the scope toolbar stays compact.
	return ui.Node{ID: "scope-cursors", Kind: ui.Toolbar, Style: chromeStyle(), Children: nodes}
}
func (c *scopeCanvas) placeCursor(px float64) {
	if c.screen == nil || c.View == nil || c.screen.fileBusy || c.screen.pendingCommand != "" {
		return
	}
	state := c.screen.service.Snapshot()
	if !state.ScopeVisible {
		return
	}
	if record := state.Recording; record != nil && record.Viewing {
		left, right := 58., c.View.Width-78
		if right <= left {
			return
		}
		fraction := math.Max(0, math.Min(1, (px-left)/(right-left)))
		at := record.Scope.Window.Start + fraction*(record.Scope.Window.End-record.Scope.Window.Start)
		c.screen.invoke("recording.cursor.set", workbench.CursorParams{Cursor: record.Scope.ActiveCursor, Time: at})
		return
	}
	t := state.Transient
	if t == nil || t.Stale || len(t.Samples) == 0 {
		return
	}
	left, right := 58., c.View.Width-78
	if right <= left {
		return
	}
	fraction := math.Max(0, math.Min(1, (px-left)/(right-left)))
	first := state.Scope.Window.Start
	at := first + fraction*(state.Scope.Window.End-first)
	c.screen.invoke("scope.cursor.set", workbench.CursorParams{Cursor: state.Scope.ActiveCursor, Time: at})
}
