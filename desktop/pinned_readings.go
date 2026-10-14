package main

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) pinnedReadingControls(state workbench.Snapshot) []ui.Node {
	options := []ui.Option{{ID: "__none", Label: "Pin a reading…"}}
	pinned := map[string]bool{}
	for _, reading := range state.PinnedReadings {
		pinned[reading.Definition.ID] = true
	}
	for _, probe := range state.Scope.Available {
		if !pinned[probe.ID] {
			options = append(options, ui.Option{ID: probe.ID, Label: probe.Label})
		}
	}
	nodes := []ui.Node{section("pinned-readings-title", "Pinned live readings"), quiet("pinned-readings-help", "Pin up to eight voltage/current readings, then drag a card on the sheet or edit X / Y. Readings always follow the live circuit."),
		{ID: "pin-reading-probe", Kind: ui.Selector, Options: options, SelectedID: "__none", Disabled: len(state.PinnedReadings) >= workbench.MaxPinnedReadings, Accessibility: ui.Accessibility{Name: "Pin a live voltage or current reading"}, Change: func(e ui.ChangeEvent) {
			if e.SelectedID == "__none" {
				return
			}
			position := defaultPinPosition(state, e.SelectedID)
			s.invoke("measurement.pin.add", workbench.PinSpec{ID: e.SelectedID, Position: position})
		}}}
	for _, reading := range state.PinnedReadings {
		pin := reading.Definition
		nodes = append(nodes, quiet("pinned-value-"+pin.ID, reading.Display), quiet("pinned-reference-"+pin.ID, reading.ReferenceDisplay))
		if reading.Reason != "" {
			nodes = append(nodes, quiet("pinned-reason-"+pin.ID, reading.Reason))
		}
		for _, axis := range []string{"X", "Y"} {
			value := pin.Position.X
			if axis == "Y" {
				value = pin.Position.Y
			}
			nodes = append(nodes, ui.Node{ID: "pin-" + axis + "-" + pin.ID, Kind: ui.NumberField, Number: float32(value), Numeric: ui.Numeric{Min: -10000, Max: 10000, Step: 10}, Accessibility: ui.Accessibility{Name: fmt.Sprintf("%s pinned reading %s position", pin.ID, axis)}, Change: func(e ui.ChangeEvent) {
				position := pin.Position
				if axis == "X" {
					position.X = float64(e.Number)
				} else {
					position.Y = float64(e.Number)
				}
				s.invoke("measurement.pin.move", workbench.MoveParams{ID: pin.ID, Position: position, PreviousPosition: &pin.Position})
			}})
		}
		nodes = append(nodes, s.action("pin-remove-"+pin.ID, "Remove pinned reading", "measurement.pin.remove", workbench.IDParams{ID: pin.ID}))
	}
	return nodes
}

func hitPinnedReading(state workbench.Snapshot, at workbench.Point) string {
	for i := len(state.PinnedReadings) - 1; i >= 0; i-- {
		r := state.PinnedReadings[i]
		min, max := workbench.PinnedReadingBounds(r)
		if at.X >= min.X && at.X <= max.X && at.Y >= min.Y && at.Y <= max.Y {
			return r.Definition.ID
		}
	}
	return ""
}

func defaultPinPosition(state workbench.Snapshot, id string) workbench.Point {
	position := workbench.Point{X: state.View.Center.X, Y: state.View.Center.Y}
	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	for _, probe := range state.Scope.Available {
		if probe.ID == id {
			if part, _, ok := graph.Terminal(probe.Positive); ok {
				position = workbench.Point{X: part.Position.X + 80, Y: part.Position.Y + 80}
			}
		}
	}
	position.X = math.Max(-10000, math.Min(10000, position.X))
	position.Y = math.Max(-10000, math.Min(10000, position.Y))
	return position
}
