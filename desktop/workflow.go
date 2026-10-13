package main

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func withProbe(ids []string, id string) []string {
	out := append([]string{}, ids...)
	for _, old := range out {
		if old == id {
			return out
		}
	}
	if len(out) == workbench.MaxScopeProbes {
		out[len(out)-1] = id
	} else {
		out = append(out, id)
	}
	return out
}
func (s *screen) partPlotControls(state workbench.Snapshot, p workbench.Component) []ui.Node {
	nodes := []ui.Node{}
	for _, probe := range state.Scope.Available {
		match := probe.ID == "v:"+p.ID || probe.ID == "i:"+p.ID || p.Kind == "npn" && (probe.ID == "v:"+p.ID+".collector" || probe.ID == "i:"+p.ID+".collector") || workbench.IsConnectivityPrimitive(p.Kind) && probe.ID == "v:"+p.ID+".node" || p.Kind == "ground" && probe.ID == "v:"+p.ID+".reference"
		if match {
			text := "Plot voltage"
			if probe.Unit == "A" {
				text = "Plot current"
			}
			n := s.action("plot-"+probe.ID, text, "scope.probes.set", workbench.ProbeParams{IDs: withProbe(state.Scope.Selected, probe.ID)})
			n.Disabled = state.Transient == nil
			nodes = append(nodes, n)
		}
	}
	if len(nodes) > 0 {
		nodes = append(nodes, quiet("plot-help", "Adds a trace; replaces the last when six are shown."))
	}
	return nodes
}
func (s *screen) scopeWindowControls(state workbench.Snapshot) []ui.Node {
	sc := state.Scope
	t := state.Transient
	nodes := []ui.Node{section("scope-window-title", "Measure an interval"), {ID: "export-csv", Kind: ui.Button, Text: "Export scope CSV", Accessibility: ui.Accessibility{Name: "Export scope CSV"}, Disabled: t == nil || t.Stale || !sc.WindowValid, Activate: func() { s.beginFile("export.scope.csv", true, workbench.SaveParams{}, nil) }}}
	live := s.action("scope-live-window", "Follow live history", "scope.window.set", workbench.ScopeWindow{Mode: "live"})
	nodes = append(nodes, buttonStyle(live, sc.Window.Mode != "fixed"))
	lo, hi := 0., 0.
	if t != nil && len(t.Samples) > 0 {
		lo = t.Samples[0].Time
		hi = t.Sample.Time
	}
	latest := s.action("scope-latest-window", "Last 200 ms", "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: math.Max(lo, hi-.2), End: hi})
	latest.Disabled = t == nil || t.Stale || hi <= lo
	nodes = append(nodes, latest)
	for _, side := range []string{"start", "end"} {
		side := side
		value := sc.Window.Start
		if side == "end" {
			value = sc.Window.End
		}
		nodes = append(nodes, quiet("window-label-"+side, side+" · ms"), ui.Node{ID: "window-" + side, Kind: ui.NumberField, Number: float32(value * 1000), Disabled: t == nil || t.Stale, Numeric: ui.Numeric{Min: 0, Max: float32(math.Max(hi, state.TransientSettings.StepSeconds) * 1000), Step: float32(state.TransientSettings.StepSeconds * 1000)}, Accessibility: ui.Accessibility{Name: "Scope interval " + side + " in ms"}, Change: func(e ui.ChangeEvent) {
			p := sc.Window
			p.Mode = "fixed"
			if side == "start" {
				p.Start = float64(e.Number) / 1000
			} else {
				p.End = math.Min(hi, float64(e.Number)/1000)
			}
			s.invoke("scope.window.set", p)
		}})
	}
	if !sc.WindowValid {
		nodes = append(nodes, quiet("window-warning", sc.WindowReason))
	}
	nodes = append(nodes, quiet("window-summary", fmt.Sprintf("%.4g–%.4g ms · sample statistics", sc.Window.Start*1000, sc.Window.End*1000)))
	for _, stat := range sc.Statistics {
		unit, scale := "V", 1.
		for _, p := range sc.Available {
			if p.ID == stat.ProbeID && p.Unit == "A" {
				unit = "mA"
				scale = 1000
			}
		}
		nodes = append(nodes, quiet("stats-"+stat.ProbeID, fmt.Sprintf("%s · %d samples\nMin %.4g · Max %.4g %s\nMean %.4g · RMS %.4g %s", stat.ProbeID, stat.Count, stat.Minimum*scale, stat.Maximum*scale, unit, stat.Mean*scale, stat.RMS*scale, unit)))
	}
	// A common flasher edit is a single named transaction for both capacitors.
	caps := []workbench.Component{}
	for _, p := range state.Components {
		if p.Kind == "capacitor" {
			caps = append(caps, p)
		}
	}
	if len(caps) == 2 {
		for _, value := range []string{"10uF", "20uF"} {
			edits := []workbench.BatchEdit{}
			for _, p := range caps {
				raw, _ := json.Marshal(workbench.ValueParams{ID: p.ID, Value: value})
				edits = append(edits, workbench.BatchEdit{Command: "component.set", Params: raw})
			}
			nodes = append(nodes, s.action("cap-pair-"+value, "Both capacitors · "+value, "edit.batch", workbench.BatchParams{Name: "Set both capacitors to " + value, Edits: edits}))
		}
	}
	return nodes
}
