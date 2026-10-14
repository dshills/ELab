package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) measurementInspector(state workbench.Snapshot) []ui.Node {
	inspection := state.MeasurementInspection
	nodes := []ui.Node{section("measurement-title", "Live measurements"), quiet("measurement-help", "Plot adds a trace, replacing the last if six are shown. Pin keeps a reading on the sheet.")}
	selectTarget := func(kind, name string, options []ui.Option) ui.Node {
		selected := "__none"
		if inspection.Target != nil && inspection.Target.Kind == kind {
			selected = inspection.Target.ID
			found := false
			for _, option := range options {
				if option.ID == selected {
					found = true
				}
			}
			if !found {
				options = append(options, ui.Option{ID: selected, Label: selected + " · unavailable"})
			}
		}
		return ui.Node{ID: "inspect-" + kind, Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: name}, Change: func(e ui.ChangeEvent) {
			p := workbench.MeasurementSelectParams{Kind: kind, ID: e.SelectedID}
			if e.SelectedID == "__none" {
				p = workbench.MeasurementSelectParams{}
			}
			s.invoke("measurement.select", p)
		}}
	}
	terminals := []ui.Option{{ID: "__none", Label: "Measure a terminal…"}}
	for _, part := range state.Components {
		for _, terminal := range part.Terminals {
			ref := part.ID + "." + terminal
			terminals = append(terminals, ui.Option{ID: ref, Label: ref + " / ground"})
		}
	}
	nets := []ui.Option{{ID: "__none", Label: "Measure a net…"}}
	for _, net := range state.Nets {
		name := net.ID
		if len(net.Labels) > 0 {
			name += " · " + strings.Join(net.Labels, ", ")
		}
		nets = append(nets, ui.Option{ID: net.ID, Label: fmt.Sprintf("%s · %d terminals", name, len(net.Terminals))})
	}
	probes := []ui.Option{{ID: "__none", Label: "Measure voltage/current…"}}
	for _, probe := range state.Scope.Available {
		probes = append(probes, ui.Option{ID: probe.ID, Label: probe.Label + " · " + probe.Unit})
	}
	nodes = append(nodes, selectTarget("terminal", "Inspect named terminal voltage", terminals), selectTarget("net", "Inspect named electrical net", nets), selectTarget("probe", "Inspect voltage or current probe", probes))
	if inspection.Target == nil {
		return append(nodes, quiet("measurement-empty", "Select a component, terminal or wire to inspect. Wires show their net voltage; ideal-wire current is not inferred."))
	}
	caption := inspection.Target.ID
	if inspection.ResultKind == "dc" {
		caption += " · steady-state calculation"
	} else if inspection.ResultKind == "transient" && len(inspection.Readings) > 0 {
		caption += fmt.Sprintf(" · sample %.5g ms", inspection.Readings[0].Time*1000)
	}
	nodes = append(nodes, quiet("measurement-source", caption))
	if inspection.Reason != "" {
		nodes = append(nodes, quiet("measurement-reason", inspection.Reason))
	}
	if len(inspection.Wires) > 0 {
		nodes = append(nodes, quiet("measurement-net-members", fmt.Sprintf("%d connected terminals · %d wire routes highlighted", len(inspection.Terminals), len(inspection.Wires))))
	}
	pinned := map[string]bool{}
	for _, pin := range state.PinnedReadings {
		pinned[pin.Definition.ID] = true
	}
	for _, reading := range inspection.Readings {
		id := reading.Probe.ID
		nodes = append(nodes, quiet("measurement-value-"+id, reading.Display), quiet("measurement-reference-"+id, reading.ReferenceDisplay))
		if reading.Reason != "" {
			nodes = append(nodes, quiet("measurement-status-"+id, reading.Reason))
		}
		plot := s.action("measurement-plot-"+id, "Plot", "scope.probes.set", workbench.ProbeParams{IDs: withProbe(state.Scope.Selected, id)})
		plot.Disabled = state.Transient == nil || !inspection.Available
		pin := s.action("measurement-pin-"+id, "Pin", "measurement.pin.add", workbench.PinSpec{ID: id, Position: defaultPinPosition(state, id)})
		pin.Disabled = pinned[id] || len(state.PinnedReadings) >= workbench.MaxPinnedReadings || !inspection.Available
		nodes = append(nodes, ui.Node{ID: "measurement-actions-" + id, Kind: ui.Row, Children: []ui.Node{plot, pin}})
	}
	return append(nodes, s.action("measurement-clear", "Inspect selected object", "measurement.select", workbench.MeasurementSelectParams{}))
}
