package main

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

var (
	ink     = color.NRGBA{R: 30, G: 44, B: 58, A: 255}
	muted   = color.NRGBA{R: 102, G: 118, B: 133, A: 255}
	accent  = color.NRGBA{R: 18, G: 117, B: 113, A: 255}
	paper   = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	surface = color.NRGBA{R: 243, G: 246, B: 249, A: 255}
)

func label(id, text string) ui.Node { return ui.Node{ID: id, Kind: ui.Label, Text: text} }
func heading(id, text string) ui.Node {
	n := label(id, text)
	n.Style.Typography = ui.Some(ui.Typography{Size: 18, Weight: 600})
	return n
}
func quiet(id, text string) ui.Node {
	n := label(id, text)
	n.Style.Foreground = ui.Some(muted)
	n.Style.Typography = ui.Some(ui.Typography{Size: 12})
	return n
}
func section(id, text string) ui.Node {
	n := quiet(id, strings.ToUpper(text))
	n.Style.Typography = ui.Some(ui.Typography{Size: 11, Weight: 600})
	return n
}
func panel(id string, width float32, children ...ui.Node) ui.Node {
	return ui.Node{ID: id, Kind: ui.Column, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: width}), MaxSize: ui.Some(ui.Size{Width: width}), Padding: ui.Some(ui.UniformInsets(16)), Gap: ui.Some(float32(12)), Background: ui.Some(paper)}, Children: children}
}
func buttonStyle(n ui.Node, selected bool) ui.Node {
	n.Style.Foreground = ui.Some(ink)
	n.Style.Background = ui.Some(surface)
	n.Style.Typography = ui.Some(ui.Typography{Size: 13, Weight: 500})
	n.Style.Padding = ui.Some(ui.Insets{Top: 9, Bottom: 9, Left: 14, Right: 14})
	if selected {
		n.Style.Background = ui.Some(accent)
		n.Style.Foreground = ui.Some(paper)
	}
	return n
}
func (s *screen) action(id, text, command string, p any) ui.Node {
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Activate: func() { s.invoke(command, p) }}, false)
}
func (s *screen) view() ui.Node {
	state := s.service.Snapshot()
	s.drawing.pointer.setEnabled(!s.fileBusy && s.pendingCommand == "")
	s.drawing.View.Center = canvas.Point{X: state.View.Center.X, Y: state.View.Center.Y}
	s.drawing.View.Zoom = state.View.Zoom
	drawRevision := state.Revision
	if state.Analysis == "RC transient" {
		drawRevision = state.CommandRevision
	}
	if drawRevision != s.lastDrawRevision || state.Analysis != s.lastDrawAnalysis {
		s.renderRevision++
		s.lastDrawRevision = drawRevision
		s.lastDrawAnalysis = state.Analysis
	}
	s.drawing.Revision = s.renderRevision
	toolbar := ui.Node{ID: "toolbar", Kind: ui.Row, Style: ui.Style{Gap: ui.Some(float32(6)), Align: ui.Some(ui.AlignCenter)}, Children: []ui.Node{
		s.button("project.clear", "New"), s.button("file.open", "Open"), s.button("file.save", "Save"), s.button("file.save-as", "Save As"), s.button("example.reset", "LED"), s.button("example.rc", "RC"), s.button("example.npn", "NPN"), s.button("example.npn-pulse", "Blink"), s.button("example.flasher", "Flasher"), s.button("edit.undo", "Undo"), s.button("edit.redo", "Redo"),
	}}
	pause, pauseText := "simulation.pause", "Pause"
	if state.Paused {
		pause, pauseText = "simulation.resume", "Resume"
	}
	status := "Building · live simulation ready"
	switch state.Status {
	case "running":
		status = "● Live · DC simulation"
		if workbench.IsTransientAnalysis(state.Analysis) {
			status = "● Live · " + state.Analysis
		}
	case "paused":
		status = "Ⅱ Paused"
	case "error":
		status = "Check circuit"
	case "unsupported":
		status = "Circuit not supported"
	case "updating":
		status = "Solving…"
	}
	if state.Paused {
		status = "Ⅱ Paused"
	}
	if state.Result != nil && state.Result.Stale || state.NPN != nil && state.NPN.Stale || state.Mixed != nil && state.Mixed.Stale {
		status += " · stale reading"
	}
	statusNode := label("status", status)
	statusNode.Style.Foreground = ui.Some(accent)
	statusNode.Style.Typography = ui.Some(ui.Typography{Size: 13, Weight: 600})

	palette := []ui.Node{section("parts-title", "Components"), quiet("placement-help", "Choose a part, then click the sheet.")}
	names := map[string]string{"dc-source": "+  DC source", "resistor": "∿  Resistor", "led": "▷  Red LED", "ground": "⊥  Ground", "capacitor": "Ⅱ  Capacitor", "pulse-source": "▥  Pulse source", "npn": "↗  NPN transistor", "junction": "●  Junction", "net-label": "→  Net label", "switch": "/  SPST switch", "diode": "▷  Silicon diode"}
	palette = append(palette, ui.Node{ID: "part-search", Kind: ui.TextField, Text: state.PartSearch, Accessibility: ui.Accessibility{Name: "Search components"}, Change: func(e ui.ChangeEvent) { s.invoke("parts.search.set", workbench.SearchParams{Query: e.Text}) }})
	for _, part := range state.MatchingParts {
		n := s.action("part-"+part.Kind, names[part.Kind], "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: part.Kind})
		n = buttonStyle(n, state.Drawing.Tool == "place" && state.Drawing.Kind == part.Kind)
		n.Style.MinSize = ui.Some(ui.Size{Width: 112, Height: 22})
		palette = append(palette, n)
	}
	palette = append(palette, ui.Node{ID: "export-svg", Kind: ui.Button, Text: "Export SVG", Accessibility: ui.Accessibility{Name: "Export schematic SVG"}, Activate: func() { s.beginFile("export.schematic.svg", true, workbench.SaveParams{}, nil) }})
	palette = append(palette, section("getting-started-title", "Your first circuit"), quiet("step-place", "1  Place parts and ground"), quiet("step-wire", "2  Wire the terminal dots"), quiet("step-observe", "3  Watch the live measurements"))
	placement := []ui.Node{}
	if state.Drawing.Tool == "place" {
		if state.Drawing.Kind == "net-label" {
			placement = append(placement, quiet("place-label-help", "Matching labels connect their attached nets. Leave blank for a unique name."), ui.Node{ID: "place-label", Kind: ui.TextField, Text: state.Drawing.Value, Accessibility: ui.Accessibility{Name: "New net label"}, Change: func(e ui.ChangeEvent) { s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text}) }})
		}
		position := state.Drawing.Position
		placement = append(placement, section("position-title", "Exact placement"), quiet("position-labels", "X / Y · snaps to 10 units"), ui.Node{ID: "position-row", Kind: ui.Row, Children: []ui.Node{
			{ID: "position-x", Kind: ui.NumberField, Number: float32(position.X), Numeric: ui.Numeric{Min: -10000, Max: 10000, Step: 10}, Accessibility: ui.Accessibility{Name: "Placement X"}, Change: func(e ui.ChangeEvent) {
				s.invoke("drawing.position.set", workbench.PositionParams{Position: workbench.Point{X: float64(e.Number), Y: position.Y}})
			}},
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
	palette = append(palette, quiet("undo-name", "Undo: "+state.UndoName), quiet("redo-name", "Redo: "+state.RedoName))
	palette = append(palette, section("shortcuts-title", "Shortcuts"), quiet("shortcut-undo", "⌘ Z · Undo"), quiet("shortcut-delete", "Delete · Remove selection"))

	objectOptions := []ui.Option{{ID: "__none", Label: "Choose an object…"}}
	for _, c := range state.Components {
		objectOptions = append(objectOptions, ui.Option{ID: c.ID, Label: c.ID + " · " + c.Name})
	}
	for _, w := range state.Wires {
		objectOptions = append(objectOptions, ui.Option{ID: w.ID, Label: w.ID + " · wire"})
	}
	selected := state.Selected
	if selected == "" {
		selected = "__none"
	}
	inspector := []ui.Node{section("inspector-title", "Inspector"), {ID: "selection", Kind: ui.Selector, Options: objectOptions, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Selected object"}, Change: func(e ui.ChangeEvent) {
		id := e.SelectedID
		if id == "__none" {
			id = ""
		}
		s.invoke("component.select", workbench.IDParams{ID: id})
	}}}
	inspector = append(inspector, placement...)
	inspector = append(inspector, s.npnControls(state)...)
	if workbench.IsNetworkAnalysis(state.Analysis) {
		inspector = append(inspector, section("mixed-title", "Circuit network"), quiet("mixed-help", "Bounded network simulation reports device currents, capacitor state and conservation residuals."))
		if r := state.Mixed; r != nil {
			for _, q := range r.Transistors {
				inspector = append(inspector, quiet("mixed-"+q.TransistorID, fmt.Sprintf("%s · %s\nVBE %.3f V · VCE %.3f V", q.TransistorID, q.State, q.VBE, q.VCE)))
			}
			for _, c := range r.Capacitors {
				inspector = append(inspector, quiet("mixed-"+c.ID, fmt.Sprintf("%s %.3f V · %.3f µJ", c.ID, c.Voltage, c.Energy*1e6)))
			}
		}
	}
	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	part, hasPart := graph.Component(state.Selected)
	if hasPart {
		inspector = append(inspector, s.partPlotControls(state, part)...)
		inspector = append(inspector, heading("inspector", part.ID+" · "+part.Name), quiet("part-terminals", "Terminals: "+strings.Join(part.Terminals, ", ")))
		if part.Kind == "switch" {
			closed := part.SwitchClosed != nil && *part.SwitchClosed
			text := "Close switch"
			stateText := "Open"
			if closed {
				text = "Open switch"
				stateText = "Closed"
			}
			inspector = append(inspector, section("switch-title", "SPST switch"), quiet("switch-state", stateText+" · 1 Ω on / 1 GΩ off"), s.action("switch-actuate", text, "switch.set", workbench.SwitchParams{ID: part.ID, Closed: !closed}), quiet("switch-limits", "Finite leakage, no bounce/arcing or physical rating. Live actuation preserves capacitor voltage and time."))
			if state.Mixed != nil && !state.Mixed.Stale {
				for _, branch := range state.Mixed.Switches {
					if branch.ID == part.ID {
						inspector = append(inspector, quiet("switch-reading", fmt.Sprintf("%.6g V · %.6g mA\n%.6g mW", branch.Voltage, branch.Current*1000, branch.Power*1000)))
					}
				}
			}
		}
		if part.Kind == "net-label" {
			inspector = append(inspector, section("label-title", "Net label · exact matching"), ui.Node{ID: "value-entry", Kind: ui.TextField, Text: state.Drawing.Value, Accessibility: ui.Accessibility{Name: "Net label name"}, Change: func(e ui.ChangeEvent) { s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text}) }}, s.action("apply-value", "Apply label", "component.set", workbench.ValueParams{ID: part.ID, Value: state.Drawing.Value}), quiet("label-connectivity-help", "Names are case-sensitive. GND is a name; connect it to a ground symbol to establish the reference."))
		}
		if workbench.IsConnectivityPrimitive(part.Kind) {
			for _, net := range state.Nets {
				for _, ref := range net.Terminals {
					if ref == part.ID+".node" {
						inspector = append(inspector, quiet("primitive-net", net.ID+"\n"+strings.Join(net.Terminals, " · ")))
					}
				}
			}
		}
		if part.Unit != "" {
			inspector = append(inspector, section("value-title", "Value · "+part.Unit), ui.Node{ID: "value-entry", Kind: ui.TextField, Text: state.Drawing.Value, Accessibility: ui.Accessibility{Name: "Component value", Help: "Use a value such as 1k, 2.2k ohm, or 5 V"}, Change: func(e ui.ChangeEvent) { s.invoke("editor.value.set", workbench.DraftParams{Value: e.Text}) }}, buttonStyle(s.action("apply-value", "Apply value", "component.set", workbench.ValueParams{ID: part.ID, Value: state.Drawing.Value}), true))
			if part.Kind == "resistor" {
				inspector = append(inspector, ui.Node{ID: "presets", Kind: ui.Row, Children: []ui.Node{s.action("r-1k", "1 kΩ", "component.set", workbench.ValueParams{ID: part.ID, Value: "1k"}), s.action("r-2k", "2 kΩ", "component.set", workbench.ValueParams{ID: part.ID, Value: "2k"})}})
			}
			if part.Kind == "capacitor" {
				initial := state.TransientSettings.InitialVoltage
				if c, _, d := graph.Compile(); d == nil && c.Analysis == "RC transient" && c.TerminalNet[part.ID+".1"] != c.Output {
					initial = -initial
				}
				if part.InitialVoltage != nil {
					initial = *part.InitialVoltage
				}
				inspector = append(inspector, quiet("cap-initial-label", "Initial voltage · pin 1 minus pin 2"), ui.Node{ID: "cap-initial", Kind: ui.NumberField, Number: float32(initial), Numeric: ui.Numeric{Min: -12, Max: 12, Step: .01}, Accessibility: ui.Accessibility{Name: "Capacitor initial voltage"}, Change: func(e ui.ChangeEvent) {
					s.invoke("capacitor.initial.set", workbench.ValueParams{ID: part.ID, Value: fmt.Sprintf("%.9g V", e.Number)})
				}}, s.action("cap-initial-default", "Use experiment initial voltage", "capacitor.initial.set", workbench.ValueParams{ID: part.ID, Value: "default"}))
				inspector = append(inspector, ui.Node{ID: "cap-presets", Kind: ui.Row, Children: []ui.Node{s.action("c-1u", "1 µF", "component.set", workbench.ValueParams{ID: part.ID, Value: "1uF"}), s.action("c-100u", "100 µF", "component.set", workbench.ValueParams{ID: part.ID, Value: "100uF"})}})
			}
		} else if part.Kind == "npn" {
			inspector = append(inspector, quiet("npn-model", "Generic NPN · βF 100 / βR 1\nIS 10 fA · 27°C\nInstantaneous junction response.\nNo heating or switching delay."))
		} else if part.Kind == "diode" {
			inspector = append(inspector, quiet("diode-model", "Generic silicon diode · 27°C\nIS 1 nA · n 1.7 · Rs 1 Ω\nReverse leakage included. No breakdown, charge storage or ratings."))
		} else if part.Kind == "led" {
			inspector = append(inspector, quiet("led-model", "Generic red LED\nBrightness is illustrative."))
		}
		if r := state.Mixed; r != nil {
			for _, branches := range [][]workbench.BranchResult{r.LEDs, r.Diodes, r.Resistors, r.Sources} {
				for _, b := range branches {
					if b.ID == part.ID {
						status := "Live"
						if r.Stale {
							status = "Stale"
						}
						inspector = append(inspector, quiet("mixed-selected-reading", fmt.Sprintf("%s · %.6g mA · %.6g V\nPower %.6g mW", status, b.Current*1000, b.Voltage, b.Power*1000)))
					}
				}
			}
		}
		inspector = append(inspector, quiet("part-position", fmt.Sprintf("Position %.0f, %.0f · rotation %d°", part.Position.X, part.Position.Y, part.Rotation)), s.action("rotate-part", "Rotate 90°", "component.rotate", workbench.IDParams{ID: part.ID}), s.action("duplicate-part", "Duplicate", "component.duplicate", workbench.DuplicateParams{ID: part.ID}))
	} else if state.Selected == "" {
		inspector = append(inspector, heading("nothing-selected", "Explore your circuit"), quiet("inspection-help", "Select a component to edit its value, rotate it, or see its terminals."))
	}
	for _, w := range state.Wires {
		if w.ID == state.Selected {
			inspector = append(inspector, heading("wire-title", w.ID+" · Wire"), quiet("wire-inspector", w.From+" → "+w.To), s.action("wire-insert-junction", "Insert junction on this wire", "wire.junction.insert", workbench.JunctionInsertParams{WireIDs: []string{w.ID}}))
		}
	}
	if state.Selected != "" {
		n := s.action("delete-object", "Delete selected", "object.delete", workbench.IDParams{ID: state.Selected})
		n.Style.Foreground = ui.Some(color.NRGBA{R: 168, G: 65, B: 68, A: 255})
		inspector = append(inspector, n)
	}
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
		inspector = append(inspector, section("wires-title", "Connect by name"), quiet("wire-help", "Click two terminal dots on the sheet, or choose them here."), selector("wire-from", "Wire from terminal", from, func(v string) { s.invoke("wire.terminals.set", workbench.TerminalsParams{From: v, To: to}) }), selector("wire-to", "Wire to terminal", to, func(v string) { s.invoke("wire.terminals.set", workbench.TerminalsParams{From: from, To: v}) }), s.action("connect-terminals", "Connect wire", "wire.connect", workbench.ConnectParams{From: from, To: to}))
		inspector[len(inspector)-1].Disabled = from == "" || to == ""
	}
	if workbench.IsTransientAnalysis(state.Analysis) || state.Transient != nil {
		inspector = append(inspector, s.transientInspector(state)...)
	}
	inspector = append(inspector, s.recoveryInspector(state)...)
	inspector = append(inspector, s.verificationInspector(state)...)
	tools := ui.Node{ID: "tools", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 36}), Align: ui.Some(ui.AlignCenter), Gap: ui.Some(float32(6))}, Children: []ui.Node{}}
	for _, tool := range []string{"select", "wire", "pan"} {
		n := s.action("tool-"+tool, strings.ToUpper(tool[:1])+tool[1:], "drawing.tool.set", workbench.ToolParams{Tool: tool})
		tools.Children = append(tools.Children, buttonStyle(n, state.Drawing.Tool == tool))
	}
	tools.Children = append(tools.Children, s.button("drawing.cancel", "Esc"), ui.Node{ID: "tool-spacer", Kind: ui.Row, Grow: true, MinWidth: 1}, s.button("view.fit", "Fit"), label("zoom", fmt.Sprintf("%.0f%%", state.View.Zoom*100)))
	hints := "Click to inspect · drag to move · scroll to zoom"
	switch state.Drawing.Tool {
	case "wire":
		hints = "Click two terminal dots to connect them · Esc cancels"
	case "pan":
		hints = "Drag the sheet to pan · scroll to zoom"
	case "place":
		hints = "Click to place " + state.Drawing.Kind + " · Esc cancels"
	}
	if state.Drawing.WireStart != "" {
		hints = "Wire from " + state.Drawing.WireStart + " · click its destination"
	}
	message := "Ready to experiment. Values update the simulation automatically."
	for _, candidate := range state.Recovery.Candidates {
		if !candidate.ActiveSession {
			message = "Recovery available · choose a snapshot in the Inspector to restore a copy."
			break
		}
	}
	if state.Recovery.Error != "" {
		message = "Autosave needs attention · " + state.Recovery.Error
	}
	if s.fileMessage != "" && !(state.Document.Dirty && strings.HasPrefix(s.fileMessage, "Saved ")) {
		message = s.fileMessage
	}
	if s.fileBusy {
		message = "Working on your schematic file…"
	}
	if len(state.Diagnostics) > 0 {
		message = state.Diagnostics[0].Message
	}
	if s.errorMessage != "" {
		message = s.errorMessage
	}
	messages := quiet("messages", message)
	messages.Style.MinSize = ui.Some(ui.Size{Height: 40})
	messages.Style.MaxSize = ui.Some(ui.Size{Height: 40})
	if s.errorMessage != "" {
		messages.Style.Foreground = ui.Some(color.NRGBA{R: 168, G: 65, B: 68, A: 255})
	}
	current, voltage, power := "—", "—", "—"
	readStatus := "Complete the circuit to start"
	if state.Result != nil {
		r := state.Result
		current = fmt.Sprintf("%.3f mA", r.Current*1000)
		voltage = fmt.Sprintf("%.3f V", r.LEDVoltage)
		power = fmt.Sprintf("%.2f mW", r.LEDPower*1000)
		readStatus = r.LEDID + " · DC operating point"
		if r.Stale {
			readStatus = "Previous reading · stale"
		} else if state.Paused {
			readStatus = "Paused reading"
		}
	}
	currentTitle, voltageTitle, powerTitle := "LED current", "LED voltage", "LED power"
	if workbench.IsTransientAnalysis(state.Analysis) || state.Transient != nil {
		currentTitle, voltageTitle, powerTitle = "RC current", "Capacitor voltage", "Stored energy"
		current, voltage, power = "—", "—", "—"
		if t := state.Transient; t != nil {
			current = fmt.Sprintf("%.3f mA", t.Sample.Current*1000)
			voltage = fmt.Sprintf("%.3f V", t.Sample.Voltage)
			power = fmt.Sprintf("%.3g µJ", t.Sample.Energy*1e6)
			readStatus = fmt.Sprintf("%s · t = %.4g s", t.CapacitorID, t.Sample.Time)
			if t.Stale {
				readStatus = "Previous trace · stale"
			} else if state.Paused {
				readStatus += " · paused"
			}
		}
	}
	if state.Analysis == workbench.NPNAnalysis || state.Analysis == workbench.NPNTransientAnalysis || state.NPN != nil {
		currentTitle, voltageTitle, powerTitle = "Collector / LED current", "Collector voltage · VCE", "Transistor state"
		current, voltage, power = "—", "—", "—"
		if r := state.NPN; r != nil {
			current = fmt.Sprintf("%.3f mA", r.CollectorCurrent*1000)
			voltage = fmt.Sprintf("%.3f V", r.VCE)
			power = transistorStateLabel(r.State)
			readStatus = r.TransistorID + " · DC operating point"
			if t := state.Transient; t != nil && workbench.IsNPNTransientEngine(t.Engine) {
				readStatus = fmt.Sprintf("%s · t = %.4g s", r.TransistorID, t.Sample.Time)
			}
			if r.Stale {
				readStatus = "Previous reading · stale"
			} else if state.Paused {
				readStatus = "Paused reading"
			}
		}
	}
	if state.Analysis == workbench.MixedAnalysis || state.Mixed != nil {
		currentTitle, voltageTitle, powerTitle = "LED current", "Timing capacitor voltage", "Transistor state"
		current, voltage, power = "—", "—", "—"
		if r := state.Mixed; r != nil {
			if len(r.LEDs) > 0 {
				current = fmt.Sprintf("%.3f mA", r.LEDs[0].Current*1000)
				currentTitle = r.LEDs[0].ID + " current"
			}
			if len(r.Capacitors) > 0 {
				voltage = fmt.Sprintf("%.3f V", r.Capacitors[0].Voltage)
				voltageTitle = r.Capacitors[0].ID + " voltage"
			}
			if len(r.Transistors) > 0 {
				power = transistorStateLabel(r.Transistors[0].State)
			}
			readStatus = fmt.Sprintf("Network · t = %.4g s", r.Time)
			if len(r.Transistors) == 0 {
				powerTitle = "Power residual"
				power = fmt.Sprintf("%.3g nW", r.PowerResidual*1e9)
			}
			if len(r.Capacitors) == 0 && len(r.LEDs) == 0 && len(r.Resistors) > 0 {
				voltageTitle = r.Resistors[0].ID + " voltage"
				voltage = fmt.Sprintf("%.5g V", r.Resistors[0].Voltage)
			}
			if len(r.Capacitors) == 0 && len(r.LEDs) > 0 {
				voltageTitle = r.LEDs[0].ID + " voltage"
				voltage = fmt.Sprintf("%.4g V", r.LEDs[0].Voltage)
			}
			if len(r.LEDs) == 0 && len(r.Resistors) > 0 {
				currentTitle = r.Resistors[0].ID + " current"
				current = fmt.Sprintf("%.5g mA", r.Resistors[0].Current*1000)
			}
			if r.Stale {
				readStatus = "Previous result · stale"
			} else if state.Paused {
				readStatus += " · paused"
			}
		}
	}
	metric := func(id, title, value string) ui.Node {
		n := heading(id+"-value", value)
		n.Style.Typography = ui.Some(ui.Typography{Size: 24, Weight: 500})
		return ui.Node{ID: id, Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(14)), Gap: ui.Some(float32(4)), Background: ui.Some(paper)}, Children: []ui.Node{section(id+"-title", title), n}}
	}
	// Width constraints also limit text wrapping; both rails remain independently scrollable.
	for i := range palette {
		if palette[i].Kind == ui.Label {
			palette[i].Style.MaxSize = ui.Some(ui.Size{Width: 152})
		}
	}
	for i := range inspector {
		if inspector[i].Kind == ui.Label {
			inspector[i].Style.MaxSize = ui.Some(ui.Size{Width: 216})
		}
	}
	filename := "Untitled"
	if state.Document.Path != "" {
		filename = filepath.Base(state.Document.Path)
	}
	if state.Document.Dirty {
		filename += " · Unsaved"
	}
	root := ui.Node{ID: "root", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(12)), Gap: ui.Some(float32(10)), Background: ui.Some(surface)}, Children: []ui.Node{
		{ID: "header", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 60}), Padding: ui.Some(ui.UniformInsets(12)), Gap: ui.Some(float32(16)), Align: ui.Some(ui.AlignCenter), Background: ui.Some(paper)}, Children: []ui.Node{heading("title", "ELab"), toolbar, {ID: "header-spacer", Kind: ui.Row, Grow: true, MinWidth: 1}, statusNode, s.button(pause, pauseText)}},
		{ID: "workspace", Kind: ui.Row, Grow: true, Style: ui.Style{Gap: ui.Some(float32(10))}, Children: []ui.Node{
			{ID: "parts-scroll", Kind: ui.Scroll, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 200}), MaxSize: ui.Some(ui.Size{Width: 200}), Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper)}, Children: []ui.Node{panel("palette", 152, palette...)}},
			{ID: "schematic-panel", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(12)), Gap: ui.Some(float32(8)), Background: ui.Some(paper)}, Children: []ui.Node{tools, {ID: "schematic", Kind: ui.Extension, Grow: true, Custom: s.drawing}, quiet("schematic-help", hints), messages}},
			{ID: "sidebar", Kind: ui.Scroll, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 264}), MaxSize: ui.Some(ui.Size{Width: 264}), Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper)}, Children: []ui.Node{panel("sidebar-content", 216, inspector...)}},
		}},
		{ID: "readings", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 72}), Gap: ui.Some(float32(10)), Align: ui.Some(ui.AlignCenter)}, Children: []ui.Node{metric("current", currentTitle, current), metric("voltage", voltageTitle, voltage), metric("power", powerTitle, power), panel("simulation-info", 232, section("simulation-title", "Live simulation"), quiet("reading-status", readStatus))}},
		{ID: "footer-row", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 20})}, Children: []ui.Node{quiet("footer", fmt.Sprintf("%s · %d components · %d wires · last change: %s", filename, len(state.Components), len(state.Wires), state.LastActor)), {ID: "footer-spacer", Kind: ui.Row, Grow: true, MinWidth: 1}, quiet("grid-info", "10-unit snap · local workspace")}},
		s.filePrompt(),
	}}
	if state.Recording != nil && state.Recording.Viewing || workbench.IsTransientAnalysis(state.Analysis) || state.Transient != nil {
		index := len(root.Children) - 3
		children := append([]ui.Node{}, root.Children[:index]...)
		children = append(children, s.scopePanel(state))
		children = append(children, root.Children[index:]...)
		root.Children = children
	}
	return root
}

func transistorStateLabel(state string) string {
	switch state {
	case "cutoff":
		return "Cutoff"
	case "active":
		return "Active"
	case "saturation":
		return "Saturation"
	default:
		return state
	}
}
