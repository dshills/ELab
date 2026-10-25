package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

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
		match := p.Kind == "optocoupler" && (probe.ID == "v:"+p.ID+".output" || probe.ID == "i:"+p.ID+".anode" || probe.ID == "i:"+p.ID+".collector") || workbench.IsSensorKind(p.Kind) && strings.HasPrefix(probe.ID, "v:"+p.ID+".") || workbench.IsRegulatorKind(p.Kind) && strings.HasPrefix(probe.ID, "v:"+p.ID+".") || workbench.IsControlKind(p.Kind) && strings.HasPrefix(probe.ID, "v:"+p.ID+".") || workbench.IsDisplayKind(p.Kind) && (strings.HasPrefix(probe.ID, "v:"+p.ID+".") || strings.HasPrefix(probe.ID, "i:"+p.ID+".")) || workbench.IsSequentialKind(p.Kind) && strings.HasPrefix(probe.ID, "v:"+p.ID+".") || workbench.IsLogicKind(p.Kind) && (probe.ID == "v:"+p.ID+".a" || probe.ID == "v:"+p.ID+".b") || p.Kind == "timer555" && (probe.ID == "v:"+p.ID+".threshold" || probe.ID == "v:"+p.ID+".trigger" || probe.ID == "v:"+p.ID+".control") || p.Kind == "opamp" && (probe.ID == "v:"+p.ID+".input-difference" || probe.ID == "v:"+p.ID+".supply") || workbench.IsMOSFETKind(p.Kind) && (probe.ID == "v:"+p.ID+".gate-source" || probe.ID == "i:"+p.ID+".drain") || p.Kind == "potentiometer" && (probe.ID == "v:"+p.ID+".wiper" || probe.ID == "i:"+p.ID+".wiper") || probe.ID == "v:"+p.ID || probe.ID == "i:"+p.ID || workbench.IsBipolarKind(p.Kind) && (probe.ID == "v:"+p.ID+".collector" || probe.ID == "v:"+p.ID+".base-emitter" || probe.ID == "i:"+p.ID+".collector") || workbench.IsConnectivityPrimitive(p.Kind) && probe.ID == "v:"+p.ID+".node" || p.Kind == "ground" && probe.ID == "v:"+p.ID+".reference"
		if match {
			text := "Plot voltage"
			if probe.Unit == "A" {
				text = "Plot current"
			}
			if p.Kind == "optocoupler" {
				switch probe.ID {
				case "v:" + p.ID:
					text = "Plot input LED voltage"
				case "v:" + p.ID + ".output":
					text = "Plot collector-emitter voltage"
				case "i:" + p.ID + ".anode":
					text = "Plot input LED current"
				case "i:" + p.ID + ".collector":
					text = "Plot collector current"
				}
			}
			if workbench.IsSensorKind(p.Kind) {
				if strings.HasPrefix(probe.ID, "v:"+p.ID+".") {
					text = "Plot pin " + strings.TrimPrefix(probe.ID, "v:"+p.ID+".") + " voltage"
				} else if probe.Unit == "V" {
					text = "Plot sensor voltage"
				} else {
					text = "Plot sensor current"
				}
			}
			if workbench.IsControlKind(p.Kind) || workbench.IsRegulatorKind(p.Kind) {
				switch probe.ID {
				case "v:" + p.ID:
					text = "Plot output voltage"
					if p.Kind == "relay" {
						text = "Plot coil voltage"
					}
					if workbench.IsButtonKind(p.Kind) {
						text = "Plot button voltage"
					}
				case "i:" + p.ID:
					text = "Plot delivered output current"
					if p.Kind == "relay" {
						text = "Plot coil current"
					}
					if workbench.IsButtonKind(p.Kind) {
						text = "Plot button current"
					}
				case "v:" + p.ID + ".input-difference":
					text = "Plot input difference"
				default:
					text = "Plot " + strings.TrimPrefix(probe.ID, "v:"+p.ID+".") + " voltage"
				}
			}
			if workbench.IsDisplayKind(p.Kind) {
				pin := strings.TrimPrefix(strings.TrimPrefix(probe.ID, "v:"+p.ID+"."), "i:"+p.ID+".")
				text = "Plot " + pin + " voltage"
				if probe.Unit == "A" {
					text = "Plot " + pin + " current"
				}
			}
			if p.Kind == "timer555" || workbench.IsLogicKind(p.Kind) || workbench.IsSequentialKind(p.Kind) {
				switch probe.ID {
				case "v:" + p.ID:
					text = "Plot output voltage"
				case "i:" + p.ID:
					text = "Plot delivered output current"
				default:
					text = "Plot " + strings.TrimPrefix(probe.ID, "v:"+p.ID+".") + " voltage"
				}
			}
			if p.Kind == "opamp" {
				switch probe.ID {
				case "v:" + p.ID:
					text = "Plot output voltage"
				case "v:" + p.ID + ".input-difference":
					text = "Plot input difference"
				case "v:" + p.ID + ".supply":
					text = "Plot supply voltage"
				default:
					text = "Plot delivered output current"
				}
			}
			if workbench.IsMOSFETKind(p.Kind) {
				switch probe.ID {
				case "v:" + p.ID:
					text = "Plot drain-source voltage"
				case "v:" + p.ID + ".gate-source":
					text = "Plot gate-source voltage"
				default:
					text = "Plot drain current"
				}
			}
			if workbench.IsBipolarKind(p.Kind) {
				switch probe.ID {
				case "v:" + p.ID:
					text = "Plot VCE"
				case "v:" + p.ID + ".base-emitter":
					text = "Plot VBE"
				case "v:" + p.ID + ".collector":
					text = "Plot collector voltage"
				default:
					text = "Plot collector current"
				}
			}
			n := s.action("plot-"+probe.ID, text, "scope.probes.set", workbench.ProbeParams{IDs: withProbe(state.Scope.Selected, probe.ID)})
			n.Disabled = state.Transient == nil
			nodes = append(nodes, n)
		}
	}
	if p.Kind == "opamp" {
		// Keep the four measurements near the live reading in a short inspector.
		// Full action names remain available to accessibility clients.
		byID := map[string]ui.Node{}
		for _, n := range nodes {
			n.Accessibility.Name = n.Text
			n.Grow = true
			byID[n.ID] = n
		}
		nodes = nil
		for row, pair := range [][2]struct{ suffix, label string }{
			{{"v:" + p.ID, "Output V"}, {"i:" + p.ID, "Output I"}},
			{{"v:" + p.ID + ".input-difference", "Input ΔV"}, {"v:" + p.ID + ".supply", "Supply V"}},
		} {
			children := []ui.Node{}
			for _, item := range pair {
				if n, ok := byID["plot-"+item.suffix]; ok {
					n.Text = item.label
					children = append(children, n)
				}
			}
			if len(children) > 0 {
				nodes = append(nodes, ui.Node{ID: fmt.Sprintf("opamp-plot-row-%d", row), Kind: ui.Row, Children: children})
			}
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
	nodes := []ui.Node{section("scope-window-title", "Measure an interval")}
	live := s.intervalResetAction(state, "scope-live-window", "Follow live history", workbench.ScopeWindow{Mode: "live"}, "")
	nodes = append(nodes, buttonStyle(live, sc.Window.Mode != "fixed"))
	lo, hi := 0., 0.
	if t != nil && len(t.Samples) > 0 {
		lo = t.Samples[0].Time
		hi = t.Sample.Time
	}
	latest := s.intervalResetAction(state, "scope-latest-window", "Last 200 ms", workbench.ScopeWindow{Mode: "fixed", Start: math.Max(lo, hi-.2), End: hi}, "")
	latest.Disabled = t == nil || t.Stale || hi <= lo
	nodes = append(nodes, latest)
	for _, side := range []string{"start", "end"} {
		side := side
		value := sc.Window.Start
		if side == "end" {
			value = sc.Window.End
		}
		field := timeField("window-"+side, "Scope interval "+side, value, t == nil || t.Stale, func(text string) {
			s.changeIntervalTime(state, side, text, "")
		})
		s.applyIntervalTimeRejection(state, side, "", &field)
		nodes = append(nodes, quiet("window-label-"+side, side+" · time"), field)
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
	owner := fmt.Sprintf("live-run:%d", state.RunID)
	for i := range nodes {
		scopeOwnedControlState(&nodes[i], owner)
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
