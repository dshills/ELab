package main

import (
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Focus continuation is native presentation after a successful user action.
// Domain edits still use the same guarded commands as all agent adapters.
func (s *screen) focusAfterAction(node ui.Node, target string) ui.Node {
	activate := node.Activate
	node.Activate = func() {
		activate()
		if s.errorMessage == "" {
			s.requestKeyboardFocus(target)
		}
	}
	return node
}

func (s *screen) placementControls(state workbench.Snapshot) []ui.Node {
	placement := []ui.Node{}
	if state.Drawing.Tool == "place" {
		if state.Drawing.Kind == "net-label" {
			placement = append(placement, quiet("place-label-help", "Matching labels connect their attached nets. Leave blank for a unique name."), ui.Node{ID: "place-label", Kind: ui.TextField, Text: state.Drawing.Value, Accessibility: ui.Accessibility{Name: "New net label"}, Change: func(e ui.ChangeEvent) { s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text}) }})
		}
		position := state.Drawing.Position
		placement = append(placement, section("position-title", "Place "+componentName(state.Drawing.Kind)), quiet("position-labels", "X / Y · snaps to 10 units"), ui.Node{ID: "position-row", Kind: ui.Column, Children: []ui.Node{
			quiet("position-x-label", "X"),
			{ID: "position-x", Kind: ui.NumberField, Number: float32(position.X), Numeric: ui.Numeric{Min: -10000, Max: 10000, Step: 10}, Accessibility: ui.Accessibility{Name: "Placement X"}, Change: func(e ui.ChangeEvent) {
				s.invoke("drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: float64(e.Number), Y: position.Y}})
			}},
			quiet("position-y-label", "Y"),
			{ID: "position-y", Kind: ui.NumberField, Number: float32(position.Y), Numeric: ui.Numeric{Min: -10000, Max: 10000, Step: 10}, Accessibility: ui.Accessibility{Name: "Placement Y"}, Change: func(e ui.ChangeEvent) {
				s.invoke("drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: position.X, Y: float64(e.Number)}})
			}},
		}}, s.action("place-at-position", "Place at X / Y", "component.place", workbench.PlaceParams{Kind: state.Drawing.Kind, Position: position, Value: func() string {
			if state.Drawing.Kind == "net-label" {
				return state.Drawing.Value
			}
			return ""
		}()}))
	}
	for i := range placement {
		if placement[i].ID == "place-at-position" {
			placement[i] = s.focusAfterAction(placement[i], "part-search")
		}
	}
	if state.Drawing.Tool == "place" && state.Drawing.Kind == "junction" {
		graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
		point := state.Drawing.Position
		ids := []string{}
		for _, wire := range state.Wires {
			if wire.Connected && workbench.WireDistance(graph, wire, point) <= workbench.GridSize/2+1e-9 {
				ids = append(ids, wire.ID)
			}
		}
		placement = append(placement, quiet("junction-targets", "Explicit junction targets: "+strings.Join(ids, ", ")))
		join := s.action("junction-join-routes", "Join named routes at X / Y", "wire.junction.insert", workbench.JunctionInsertParams{WireIDs: ids, Position: &point})
		join.Disabled = len(ids) == 0 || len(ids) > 8
		placement = append(placement, join, quiet("junction-help", "Crossings stay separate until this explicit join. Place an empty junction to connect terminal dots manually."))
	}

	return placement
}

func (s *screen) namedWireControls(state workbench.Snapshot) []ui.Node {
	nodes := []ui.Node{s.focusAfterAction(s.action("wire-by-name", "Connect by name", "drawing.tool.set", workbench.ToolParams{Tool: "wire"}), "wire-from")}
	terminals := 0
	for _, part := range state.Components {
		terminals += len(part.Terminals)
	}
	nodes[0].Disabled = terminals < 2
	if state.Drawing.Tool == "wire" {
		terminalOptions := []ui.Option{{ID: "__none", Label: "Choose a terminal…"}}
		for _, c := range state.Components {
			for _, t := range c.Terminals {
				ref := c.ID + "." + t
				terminalOptions = append(terminalOptions, ui.Option{ID: ref, Label: ref})
			}
		}
		from, to := state.Drawing.From, state.Drawing.To
		selector := func(id, name, value string, change func(string)) ui.Node {
			if value == "" {
				value = "__none"
			}
			return ui.Node{ID: id, Kind: ui.Selector, Options: terminalOptions, SelectedID: value, Accessibility: ui.Accessibility{Name: name}, Change: func(e ui.ChangeEvent) {
				v := e.SelectedID
				if v == "__none" {
					v = ""
				}
				change(v)
			}}
		}
		nodes = append(nodes, section("wires-title", "Wire terminals"), quiet("wire-help", "Click two terminal dots on the sheet, or choose them here."), quiet("wire-from-label", "From"), selector("wire-from", "Wire from terminal", from, func(v string) { s.invoke("wire.terminals.set", workbench.TerminalsParams{From: v, To: to}) }), quiet("wire-to-label", "To"), selector("wire-to", "Wire to terminal", to, func(v string) { s.invoke("wire.terminals.set", workbench.TerminalsParams{From: from, To: v}) }), s.action("connect-terminals", "Connect wire", "wire.connect", workbench.ConnectParams{From: from, To: to}))
		nodes[len(nodes)-1].Disabled = from == "" || to == ""
	}
	if state.Drawing.Tool == "wire" {
		nodes[len(nodes)-1] = s.focusAfterAction(nodes[len(nodes)-1], "wire-from")
	}
	return nodes
}

func (s *screen) requestKeyboardFocus(id string) { s.keyboardFocus.Sequence++; s.keyboardFocus.ID = id }
