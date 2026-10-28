package workbench

import (
	"encoding/json"
	"fmt"
	"math"
)

// DrawingPreview is a noncommitting plan. Commit its command with the returned
// command/document guards; another actor can invalidate a previously valid plan.
type DrawingPreview struct {
	CommandRevision    uint64          `json:"command_revision"`
	DocumentGeneration uint64          `json:"document_generation"`
	Tool               string          `json:"tool"`
	Position           Point           `json:"position"`
	Valid              bool            `json:"valid"`
	Message            string          `json:"message"`
	Terminal           string          `json:"terminal,omitempty"`
	Component          *Component      `json:"component,omitempty"`
	Path               []Point         `json:"path"`
	Command            string          `json:"command,omitempty"`
	Params             json.RawMessage `json:"params,omitempty"`
}

// NearestTerminal uses screen-sized acquisition radii at the current zoom.
// Equidistant pins require a named choice rather than component-order guessing.
func NearestTerminal(graph Graph, position Point, zoom float64) (string, bool) {
	if zoom <= 0 || math.IsNaN(zoom) || math.IsInf(zoom, 0) {
		zoom = 1
	}
	return TerminalAt(graph, position, math.Max(6, 10/zoom))
}

// TerminalAt returns the closest pin within a world-coordinate radius.
func TerminalAt(graph Graph, position Point, radius float64) (string, bool) {
	if !finite(position.X) || !finite(position.Y) || !finite(radius) || radius < 0 {
		return "", false
	}
	graph = graph.VisibleGraph()
	best, distance, ambiguous := "", radius, false
	for _, part := range graph.Components {
		for _, pin := range part.Terminals {
			at, _ := TerminalPosition(part, pin)
			d := math.Hypot(at.X-position.X, at.Y-position.Y)
			if d > distance+1e-9 {
				continue
			}
			if best != "" && math.Abs(d-distance) <= 1e-9 {
				ambiguous = true
				continue
			}
			best, distance, ambiguous = part.ID+"."+pin, d, false
		}
	}
	if ambiguous {
		return "", true
	}
	return best, false
}

// PreviewDrawing shares real command admission on a disposable graph. It never
// starts a simulation, reserves IDs, changes selection or creates history.
func PreviewDrawing(state Snapshot, position Point) DrawingPreview {
	out := DrawingPreview{CommandRevision: state.CommandRevision, DocumentGeneration: state.Document.Generation, Tool: state.Drawing.Tool, Position: position, Path: []Point{}}
	graph := Graph{Components: state.Components, Wires: state.Wires}
	stage := &Service{graph: graph.Clone(), drawing: state.Drawing, usedIDs: map[string]bool{}}
	plan := func(command string, input any) {
		raw, err := json.Marshal(input)
		if err == nil {
			err = stage.applyLocked(command, raw)
		}
		if err != nil {
			out.Message = err.Error()
			return
		}
		out.Valid, out.Command, out.Params = true, command, raw
	}
	if state.Closed {
		out.Message = "The document is closed."
		return out
	}
	if _, err := snapPoint(position); err != nil {
		out.Message = err.Error()
		return out
	}
	switch state.Drawing.Tool {
	case "place":
		out.Position, _ = snapPoint(position)
		value := ""
		if state.Drawing.Kind == "net-label" {
			value = state.Drawing.Value
		}
		// Retain a ghost even when this valid kind's destination is occupied.
		if ghost, err := MakeComponent("Preview", state.Drawing.Kind, position, value); err == nil {
			out.Component = &ghost
		}
		if state.Drawing.Kind == "junction" {
			ids := []string{}
			for _, wire := range graph.Wires {
				if wire.Connected && WireDistance(graph, wire, out.Position) <= GridSize/2+1e-9 {
					ids = append(ids, wire.ID)
				}
			}
			if len(ids) > 1 {
				out.Message = "Several wires cross here. Use the named junction controls to choose which routes to join."
				return out
			}
			if len(ids) == 1 {
				plan("wire.junction.insert", JunctionInsertParams{WireIDs: ids, Position: &out.Position})
				if out.Valid {
					out.Message = "Insert a junction on " + ids[0]
				}
				return out
			}
		}
		plan("component.place", PlaceParams{Kind: state.Drawing.Kind, Position: out.Position, Value: value})
		if out.Valid {
			out.Message = fmt.Sprintf("Place %s at %.0f, %.0f", state.Drawing.Kind, out.Position.X, out.Position.Y)
		}
	case "wire":
		ref, ambiguous := NearestTerminal(graph, position, state.View.Zoom)
		out.Terminal = ref
		if start, pin, ok := graph.Terminal(state.Drawing.WireStart); ok {
			at, _ := TerminalPosition(start, pin)
			out.Path = []Point{at, {X: position.X, Y: at.Y}, position}
		}
		if ambiguous {
			out.Message = "Pins overlap here. Zoom in or choose a named terminal."
			return out
		}
		if ref == "" {
			out.Message = "Point at a terminal dot; crossings alone do not connect."
			return out
		}
		part, pin, _ := graph.Terminal(ref)
		out.Position, _ = TerminalPosition(part, pin)
		if state.Drawing.WireStart == "" {
			plan("wire.begin", map[string]string{"terminal": ref})
			if out.Valid {
				out.Message = "Start a wire at " + ref
			}
		} else {
			plan("wire.connect", ConnectParams{From: state.Drawing.WireStart, To: ref})
			out.Path = WirePath(graph, Wire{From: state.Drawing.WireStart, To: ref, Connected: true})
			if out.Valid {
				out.Message = "Connect " + state.Drawing.WireStart + " to " + ref
			}
		}
	default:
		out.Message = "Choose a component or the Wire tool to preview an edit."
	}
	return out
}
