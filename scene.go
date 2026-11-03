package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/dshills/elab/workbench"
)

type scenePart struct {
	workbench.Component
	Labels    []sceneLabel `json:"labels"`
	Segments  []string     `json:"segments,omitempty"`
	Lit       [7]bool      `json:"lit"`
	Paths     []string     `json:"paths"`
	Pins      []scenePin   `json:"pins"`
	Bounds    [4]float64   `json:"bounds"`
	ValueText string       `json:"value_text"`
}
type sceneLabel struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Text string  `json:"text"`
}
type scenePin struct {
	Ref  string  `json:"ref"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}
type sceneWire struct {
	workbench.Wire
	Points []workbench.Point `json:"points"`
}
type scene struct {
	Parts   []scenePart               `json:"parts"`
	Wires   []sceneWire               `json:"wires"`
	Bounds  [4]float64                `json:"bounds"`
	Markers []workbench.CurrentMarker `json:"markers"`
}

func makeScene(state workbench.Snapshot) scene {
	graph := (workbench.Graph{Components: state.Components, Wires: state.Wires}).VisibleGraph()
	out := scene{Parts: []scenePart{}, Wires: []sceneWire{}, Bounds: [4]float64{-280, -180, 280, 180}}
	if len(graph.Components) > 0 {
		out.Bounds = [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	}
	extend := func(x, y float64) {
		out.Bounds[0] = math.Min(out.Bounds[0], x)
		out.Bounds[1] = math.Min(out.Bounds[1], y)
		out.Bounds[2] = math.Max(out.Bounds[2], x)
		out.Bounds[3] = math.Max(out.Bounds[3], y)
	}
	for _, p := range graph.Components {
		left, right, top, bottom := workbench.BodyBounds(p)
		item := scenePart{Component: p, Paths: symbolPaths(p), Pins: []scenePin{}, Bounds: [4]float64{left, top, right, bottom}}
		if workbench.IsControlKind(p.Kind) && state.Mixed != nil && !state.Mixed.Stale {
			for _, reading := range state.Mixed.Controls {
				if reading.ID == p.ID {
					item.Paths = pathLines(workbench.ControlSymbolPaths(p.Kind, reading.Memory))
				}
			}
		}
		for _, label := range symbolLabels(p) {
			item.Labels = append(item.Labels, sceneLabel{label.At.X, label.At.Y, label.Text})
		}
		if p.Kind == "seven-segment" {
			item.Segments = pathLines(workbench.DisplaySegmentPaths())
			if state.Mixed != nil && !state.Mixed.Stale {
				for _, reading := range state.Mixed.Displays {
					if reading.ID == p.ID {
						item.Lit = reading.Lit
					}
				}
			}
		}
		if p.Unit != "" {
			item.ValueText = workbench.FormatValue(p.Value, p.Unit)
		} else if p.Label != "" {
			item.ValueText = p.Label
		}
		for _, pin := range p.Terminals {
			if at, ok := workbench.TerminalPosition(p, pin); ok {
				item.Pins = append(item.Pins, scenePin{p.ID + "." + pin, pin, at.X, at.Y})
				extend(at.X, at.Y)
			}
		}
		extend(left-65, top-55)
		extend(right+100, bottom+50)
		out.Parts = append(out.Parts, item)
	}
	for _, w := range graph.Wires {
		points := workbench.WirePath(graph, w)
		out.Wires = append(out.Wires, sceneWire{w, points})
		for _, p := range points {
			extend(p.X, p.Y)
		}
	}
	for _, r := range state.PinnedReadings {
		extend(r.Definition.Position.X-8, r.Definition.Position.Y-24)
		extend(r.Definition.Position.X+190, r.Definition.Position.Y+31)
	}
	out.Markers = workbench.MakeDisplayFrame(graph, state.Display, 0).Markers
	return out
}
func polyline(points []workbench.Point) string {
	var d strings.Builder
	for i, p := range points {
		if i == 0 {
			fmt.Fprintf(&d, "M%g %g", p.X, p.Y)
		} else {
			fmt.Fprintf(&d, "L%g %g", p.X, p.Y)
		}
	}
	return d.String()
}
func pathLines(paths [][]workbench.Point) []string {
	out := []string{}
	for _, p := range paths {
		out = append(out, polyline(p))
	}
	return out
}
func symbolPaths(p workbench.Component) []string {
	switch p.Kind {
	case "resistor":
		return []string{"M-55 0H-34L-28 -10L-17 10L-6 -10L6 10L17 -10L28 10L34 0H55"}
	case "capacitor":
		return []string{"M0 -32V-7M-23 -7H23M-23 7H23M0 7V32"}
	case "inductor":
		return []string{polyline(workbench.InductorPolyline())}
	case "ground":
		return []string{"M0 -20V0M-20 0H20M-12 6H12M-5 12H5"}
	case "junction":
		return []string{"M-3 0A3 3 0 1 0 3 0A3 3 0 1 0 -3 0"}
	case "net-label":
		return []string{"M0 0H10V-7L18 0L10 7V0"}
	case "switch":
		if p.SwitchClosed != nil && *p.SwitchClosed {
			return []string{"M-55 0H-25L25 0H55"}
		}
		return []string{"M-55 0H-25L19 -25M25 0H55"}
	case "dc-source", "pulse-source", "sine-source", "finite-pulse-source":
		shape := []string{"M0 -34V-22M0 22V34M-22 0A22 22 0 1 0 22 0A22 22 0 1 0 -22 0"}
		if p.Kind == "sine-source" {
			return append(shape, "M-15 0C-8 -19 -2 -19 0 0C2 19 8 19 15 0")
		}
		if p.Kind == "pulse-source" || p.Kind == "finite-pulse-source" {
			return append(shape, "M-14 9V-8H0V9H14V-8")
		}
		return append(shape, "M-6 -9H6M0 -15V-3M-6 10H6")
	case "led", "diode", "zener", "schottky":
		shape := []string{"M0 -32V-12M-12 -12H12L0 10ZM-12 11H12M0 11V32"}
		if p.Kind == "led" {
			shape = append(shape, "M18 -5L28 -15M22 7L32 -3M23 -15H28V-10M27 -3H32V2")
		}
		if p.Kind == "zener" {
			shape = append(shape, "M-12 5V11M12 11V17")
		}
		return shape
	case "nmos", "pmos":
		shape := []string{"M-40 0H-16M-16 -20V20M-7 -20V-12M-7 -4V4M-7 12V20M-7 -16H25V-35M-7 16H25V35M-7 0H25V16"}
		if p.Kind == "pmos" {
			return append(shape, "M-2 -5L7 0L-2 5")
		}
		return append(shape, "M4 -5L-5 0L4 5")
	case "npn", "pnp":
		shape := []string{"M-25 0A25 25 0 1 0 25 0A25 25 0 1 0 -25 0", "M-40 0H-10M-10 -14V14M-10 -10L25 -35M-10 10L25 35"}
		if p.Kind == "pnp" {
			return append(shape, "M12 -18L0 -17L5 -28")
		}
		return append(shape, "M15 31L25 35L21 25")
	case "opamp":
		return pathLines(workbench.OpAmpSymbolPaths())
	case "njfet", "pjfet":
		return pathLines(workbench.JFETSymbolPaths(p.Kind))
	case "current-source", "electronic-load":
		return pathLines(workbench.CurrentSymbolPaths(p.Kind))
	case "hbridge":
		return pathLines(workbench.HBridgeSymbolPaths())
	case "timer555":
		return pathLines(workbench.Timer555SymbolPaths())
	case "dc-motor":
		return pathLines(workbench.MotorSymbolPaths())
	case "battery":
		return pathLines(workbench.BatterySymbolPaths())
	case "transformer":
		reversed := p.Power != nil && p.Power.Transformer != nil && p.Power.Transformer.Reversed
		return pathLines(workbench.TransformerSymbolPaths(reversed))
	case "optocoupler":
		return pathLines(workbench.OptocouplerSymbolPaths())
	case "linear-regulator", "ldo-regulator":
		return pathLines(workbench.RegulatorSymbolPaths(p.Kind))
	case "photodiode":
		return pathLines(workbench.PhotodiodeSymbolPaths())
	case "scr", "triac":
		return pathLines(workbench.ThyristorSymbolPaths(p.Kind))
	case "circuit-block":
		return pathLines(workbench.BlockSymbolPaths(p))
	}
	if workbench.IsLogicKind(p.Kind) {
		return pathLines(workbench.LogicSymbolPaths(p.Kind))
	}
	switch p.Kind {
	case "speaker", "pwm-controller", "shift-register", "piezo", "active-buzzer":
		return pathLines(workbench.ExtendedSymbolPaths(p.Kind))
	case "d-flipflop", "jk-flipflop", "t-flipflop", "binary-counter", "decade-counter":
		return pathLines(workbench.SequentialSymbolPaths(p.Kind))
	case "bcd-decoder", "seven-segment":
		return pathLines(workbench.DisplaySymbolPaths(p.Kind))
	case "thermistor", "ldr":
		return pathLines(workbench.SensorSymbolPaths(p.Kind))
	case "relay", "comparator", "pushbutton", "bouncing-pushbutton":
		return pathLines(workbench.ControlSymbolPaths(p.Kind, workbench.ControlMemory{}))
	}
	if p.Kind == "potentiometer" {
		return []string{"M-55 0H-34L-28 -10L-17 10L-6 -10L6 10L17 -10L28 10L34 0H55M0 45V14L-5 22M0 14L5 22"}
	}
	return []string{"M-30 -25H30V25H-30ZM-55 0H-30M30 0H55"}
}

func symbolLabels(p workbench.Component) []workbench.OpAmpSymbolLabel {
	switch {
	case p.Kind == "timer555":
		return workbench.Timer555SymbolLabels(p)
	case p.Kind == "hbridge":
		return workbench.HBridgeSymbolLabels(p)
	case p.Kind == "opamp":
		return workbench.OpAmpSupplyLabels(p)
	case workbench.IsLogicKind(p.Kind):
		return workbench.LogicSymbolLabels(p)
	case workbench.IsSequentialKind(p.Kind):
		return workbench.SequentialSymbolLabels(p)
	case workbench.IsDisplayKind(p.Kind):
		return workbench.DisplaySymbolLabels(p)
	case workbench.IsControlKind(p.Kind):
		return workbench.ControlSymbolLabels(p)
	case workbench.IsRegulatorKind(p.Kind):
		return workbench.RegulatorSymbolLabels(p)
	case p.Kind == "speaker" || p.Kind == "pwm-controller" || p.Kind == "shift-register" || p.Kind == "piezo" || p.Kind == "active-buzzer":
		return workbench.ExtendedSymbolLabels(p)
	}
	return nil
}
