package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) measurementInspector(state workbench.Snapshot) []ui.Node {
	inspection := state.MeasurementInspection
	nodes := []ui.Node{section("measurement-title", "Measure"), quiet("measurement-help", "Choose a source or select it on the schematic. Plot adds a scope trace; Pin adds a reading to the sheet.")}
	options := []ui.Option{{ID: "__none", Label: "Follow schematic selection"}}
	add := func(kind, id, name string) { options = append(options, ui.Option{ID: kind + "|" + id, Label: name}) }
	for _, part := range state.Components {
		add("object", part.ID, part.ID+" · component")
		for _, terminal := range part.Terminals {
			ref := part.ID + "." + terminal
			add("terminal", ref, ref+" · terminal / ground")
		}
	}
	for _, wire := range state.Wires {
		add("object", wire.ID, wire.ID+" · wire net")
	}
	for _, net := range state.Nets {
		name := net.ID
		if len(net.Labels) > 0 {
			name += " · " + strings.Join(net.Labels, ", ")
		}
		add("net", net.ID, fmt.Sprintf("%s · %d terminals", name, len(net.Terminals)))
	}
	for _, probe := range state.Scope.Available {
		add("probe", probe.ID, probe.Label+" · "+probe.Unit)
	}
	selected := "__none"
	// Object selection and explicit object measurement share the same service
	// target. Show the exact accepted target rather than guessing its origin.
	if inspection.Target != nil {
		selected = inspection.Target.Kind + "|" + inspection.Target.ID
		found := false
		for _, o := range options {
			found = found || o.ID == selected
		}
		if !found {
			options = append(options, ui.Option{ID: selected, Label: inspection.Target.ID + " · unavailable"})
		}
	}
	nodes = append(nodes, ui.Node{ID: "measurement-source", Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Measurement source", Help: "Component, terminal, wire, net or voltage/current probe. Follow schematic selection restores automatic inspection."}, Change: func(e ui.ChangeEvent) {
		p := workbench.MeasurementSelectParams{}
		if e.SelectedID != "__none" {
			kind, id, ok := strings.Cut(e.SelectedID, "|")
			if !ok {
				return
			}
			p = workbench.MeasurementSelectParams{Kind: kind, ID: id}
		}
		s.invoke("measurement.select", p)
	}})
	if inspection.Target == nil {
		return append(nodes, quiet("measurement-empty", "Select a component, terminal or wire to inspect. Wires show their net voltage; current cues require a uniquely determined wire current."))
	}
	caption := inspection.Target.ID
	if inspection.ResultKind == "dc" {
		caption += " · steady-state calculation"
	} else if inspection.ResultKind == "transient" && len(inspection.Readings) > 0 {
		caption += fmt.Sprintf(" · sample %.5g ms", inspection.Readings[0].Time*1000)
	}
	nodes = append(nodes, quiet("measurement-caption", caption))
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
	var more []ui.Node
	for index, reading := range inspection.Readings {
		var row []ui.Node
		id := reading.Probe.ID
		row = append(row, quiet("measurement-value-"+id, reading.Display), quiet("measurement-reference-"+id, reading.ReferenceDisplay))
		if reading.Reason != "" {
			row = append(row, quiet("measurement-status-"+id, reading.Reason))
		}
		plot := s.action("measurement-plot-"+id, "Plot live", "scope.probe.plot", workbench.IDParams{ID: id})
		plot.Disabled = state.Transient == nil || !inspection.Available
		pin := s.action("measurement-pin-"+id, "Pin", "measurement.pin.add", workbench.PinSpec{ID: id, Position: defaultPinPosition(state, id)})
		pin.Disabled = pinned[id] || len(state.PinnedReadings) >= workbench.MaxPinnedReadings || !inspection.Available
		row = append(row, ui.Node{ID: "measurement-actions-" + id, Kind: ui.Row, Children: []ui.Node{plot, pin}})
		if index == 0 {
			nodes = append(nodes, row...)
		} else {
			more = append(more, row...)
		}
	}
	nodes = append(nodes, s.detailsSection(state, "measurement", "More measurements & circuit overview", more))
	return append(nodes, s.action("measurement-clear", "Inspect selected object", "measurement.select", workbench.MeasurementSelectParams{}))
}
