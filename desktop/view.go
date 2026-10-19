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

// Keep each inspector section in a stable accessibility group. A long flat
// child list can lose its leaf controls in external AX traversal when request
// guidance is inserted. Groups also give readers meaningful section boundaries;
// their zero padding and existing gap preserve the rail's layout.
func inspectorGroup(id, name string, children []ui.Node) []ui.Node {
	if len(children) == 0 {
		return nil
	}
	return []ui.Node{{ID: "inspector-group-" + id, Kind: ui.Column, Accessibility: ui.Accessibility{Name: name}, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(12))}, Children: children}}
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

type actionContext struct {
	revision, generation uint64
}

func (s *screen) action(id, text, command string, p any) ui.Node {
	context := s.actionContext
	if context == nil {
		state := s.service.Snapshot()
		context = &actionContext{state.CommandRevision, state.Document.Generation}
	}
	return buttonStyle(ui.Node{ID: id, Kind: ui.Button, Text: text, Accessibility: ui.Accessibility{Name: text}, Activate: func() { s.invokeAtCommandRevision(command, p, context.revision, context.generation) }}, false)
}
func (s *screen) view() ui.Node {
	state := s.service.Snapshot()
	previousContext := s.actionContext
	s.actionContext = &actionContext{state.CommandRevision, state.Document.Generation}
	defer func() { s.actionContext = previousContext }()
	colors := paletteForTheme(state.Theme)
	s.nativePalette.Store(&colors)
	if state.Closed {
		s.drawing.pointer.invalidateFrame()
		s.drawing.pointer.setEnabled(false)
		s.nativePointerPane = ""
		root := s.shutdownView(state)
		applyPalette(&root, colors)
		return root
	}
	if s.requestHelp != nil && state.CommandRevision != s.requestHelpRevision {
		s.requestHelp = nil
		s.errorMessage = ""
	}
	pane := "schematic"
	if sweepVisible(state) {
		pane = ""
		if state.DCSweep.Result != nil {
			pane = "sweep"
		}
	}
	if previewVisible(state) || s.fileBusy || s.pendingCommand != "" {
		pane = ""
	}
	if pane != s.nativePointerPane {
		s.drawing.pointer.invalidateFrame()
		s.nativePointerPane = pane
	}
	s.drawing.pointer.setEnabled(pane != "")
	if sourcePaneHidden(state) {
		s.drawing.dragID = ""
		s.drawing.dragging = false
	}
	s.observeSchematicViewport(*s.drawing.View, state)
	s.drawing.View.Center = canvas.Point{X: state.View.Center.X, Y: state.View.Center.Y}
	s.drawing.View.Zoom = state.View.Zoom
	drawRevision := state.Revision
	if state.Analysis == "RC transient" {
		drawRevision = state.CommandRevision
	}
	if drawRevision != s.lastDrawRevision || state.Analysis != s.lastDrawAnalysis || colors != s.lastPalette {
		s.renderRevision++
		s.lastDrawRevision = drawRevision
		s.lastDrawAnalysis = state.Analysis
		s.lastPalette = colors
	}
	s.drawing.Revision = s.renderRevision
	toolbar := ui.Node{ID: "toolbar", Kind: ui.Toolbar, Style: chromeStyle(), Children: []ui.Node{
		s.commandMenu("file-menu", "File", [][2]string{{"project.clear", "New schematic"}, {"file.open", "Open…"}, {"file.save", "Save"}, {"file.save-as", "Save as…"}, {"app.close", "Close window"}}),
		s.button("file.save", "Save"),
		s.commandMenu("examples-menu", "Examples", [][2]string{{"example.reset", "LED circuit"}, {"example.split-supply", "Split-rail DC supply"}, {"example.rc", "RC charge / discharge"}, {"example.rc-high-pass", "RC high-pass filter"}, {"example.rl", "Coil buildup / decay"}, {"example.rlc", "RLC ringing"}, {"example.flyback", "Coil / flyback diode"}, {"example.npn", "Transistor LED"}, {"example.switch-npn", "Manual transistor switch"}, {"example.npn-pulse", "Blinking LED"}, {"example.flasher", "Capacitor-timed flasher"}, {"example.led-fade", "LED fade"}, {"example.nmos", "MOSFET LED switch"}, {"example.pmos", "High-side MOSFET LED"}, {"example.cmos", "CMOS inverter"}, {"example.opamp-follower", "Op-amp follower"}, {"example.opamp-inverting", "Inverting amplifier"}, {"example.rectifier", "Rectifier / smoothing"}}),
		s.button("edit.undo", "Undo"), s.button("edit.redo", "Redo"), s.themeMenu(state.Theme),
	}}
	// Reserve the complete action row even when both history labels include
	// "agent"/"other". Otherwise the toolbar clips Theme while header space
	// remains unused beside the status controls.
	toolbar.Style.MinSize = ui.Some(ui.Size{Width: 560, Height: 48})
	for i := range toolbar.Children {
		toolbar.Children[i].Style.Padding = ui.Some(ui.Insets{Top: 7, Bottom: 7, Left: 9, Right: 9})
	}
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
		if state.Status == "updating" {
			status = "Initializing · paused"
		}
	}
	if state.StepWork != nil && (state.StepWork.Status == "queued" || state.StepWork.Status == "running" || state.StepWork.Status == "canceling") {
		status = fmt.Sprintf("Stepping · %.0f%% · committed readings paused", 100*state.StepWork.Progress)
	}
	if state.Result != nil && state.Result.Stale || state.NPN != nil && state.NPN.Stale || state.Mixed != nil && state.Mixed.Stale {
		status += " · stale reading"
	}
	statusNode := label("status", status)
	statusNode.Style.Foreground = ui.Some(accent)
	statusNode.Style.Typography = ui.Some(ui.Typography{Size: 13, Weight: 600})
	statusNode.Style.MaxSize = ui.Some(ui.Size{Width: 180})

	palette := s.componentPalette(state)
	palette = append(palette, s.placementControls(state)...)
	palette = append(palette, s.namedWireControls(state)...)
	previewButton := s.outputPreviewAction(state, "print-preview", "Print preview")
	previewButton.Disabled = state.PrintPreview != nil && previewActive(state.PrintPreview)
	palette = append(palette, previewButton)
	palette = append(palette, ui.Node{ID: "export-svg", Kind: ui.Button, Text: "Export SVG", Accessibility: ui.Accessibility{Name: "Export schematic SVG"}, Activate: func() { s.beginFile("export.schematic.svg", true, workbench.SaveParams{}, nil) }})
	palette = append(palette, ui.Node{ID: "export-pdf", Kind: ui.Button, Text: "Export PDF", Accessibility: ui.Accessibility{Name: "Export printable schematic PDF"}, Activate: func() { s.beginFile("export.schematic.pdf", true, workbench.PDFExportParams{}, nil) }})
	palette = append(palette, ui.Node{ID: "export-png", Kind: ui.Button, Text: "Export PNG", Accessibility: ui.Accessibility{Name: "Export schematic image PNG"}, Activate: func() { s.beginFile("export.schematic.png", true, workbench.PNGExportParams{}, nil) }})
	palette = append(palette, s.exampleGuidance(state)...)
	palette = append(palette, section("getting-started-title", "Your first circuit"), quiet("step-place", "1  Place parts and ground"), quiet("step-wire", "2  Wire the terminal dots"), quiet("step-observe", "3  Watch the live measurements"))
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
	previous := s.action("selection-previous", "Back", "selection.navigate", workbench.NavigateParams{Direction: "previous", Center: true})
	previous.Accessibility.Name = "Previous schematic object"
	next := s.action("selection-next", "Next", "selection.navigate", workbench.NavigateParams{Direction: "next", Center: true})
	next.Accessibility.Name = "Next schematic object"
	previous.Disabled, next.Disabled = len(objectOptions) == 1, len(objectOptions) == 1
	center := s.action("selection-center", "Center", "view.selection.center", workbench.CenterSelectionParams{ID: state.Selected})
	center.Accessibility.Name = "Center selection"
	center.Disabled = state.Selected == ""
	inspector = append(inspector, ui.Node{ID: "selection-navigation", Kind: ui.Row, Children: []ui.Node{previous, next, center}})

	graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
	part, hasPart := graph.Component(state.Selected)
	if hasPart {
		inspector = append(inspector, s.componentInspector(state, graph, part)...)
	} else if state.Selected == "" {
		inspector = append(inspector, heading("nothing-selected", "Explore your circuit"), quiet("inspection-help", "Select a component to edit its value, rotate it, or see its terminals."))
	}
	for _, w := range state.Wires {
		if w.ID == state.Selected {
			inspector = append(inspector, heading("wire-title", w.ID+" · Wire"), quiet("wire-inspector", w.From+" → "+w.To), s.action("wire-insert-junction", "Insert junction on this wire", "wire.junction.insert", workbench.JunctionInsertParams{WireIDs: []string{w.ID}}))
			for _, reading := range state.Display.Wires {
				if reading.ID != w.ID {
					continue
				}
				text := "Wire current unavailable · " + reading.CurrentReason
				if reading.Current != nil {
					text = fmt.Sprintf("Conventional %.6g mA · %s → %s\nDerived by current conservation; %s", *reading.Current*1000, reading.From, reading.To, reading.State)
				}
				inspector = append(inspector, quiet("wire-current-reading", text))
			}
		}
	}
	if state.Selected != "" {
		n := s.action("delete-object", "Delete selected", "object.delete", workbench.IDParams{ID: state.Selected})
		n.Style.Foreground = ui.Some(color.NRGBA{R: 168, G: 65, B: 68, A: 255})
		inspector = append(inspector, n)
	}
	inspector = inspectorGroup("object", "Object inspector", inspector)
	circuitControls := append(s.lcSamplingControls(state), s.npnControls(state)...)
	if workbench.IsNetworkAnalysis(state.Analysis) {
		circuitControls = append(circuitControls, section("mixed-title", "Circuit network"), quiet("mixed-help", "Bounded network simulation reports device currents, capacitor state and conservation residuals."))
		if r := state.Mixed; r != nil {
			for _, m := range r.MOSFETs {
				circuitControls = append(circuitControls, quiet("mixed-"+m.ID, fmt.Sprintf("%s · %s\nVGS %.3f V · VDS %.3f V\nID %.4g mA · IG %.3g A", m.ID, m.State, m.VGS, m.VDS, m.DrainCurrent*1000, m.GateCurrent)))
			}
			for _, q := range r.Transistors {
				circuitControls = append(circuitControls, quiet("mixed-"+q.TransistorID, fmt.Sprintf("%s · %s\nVBE %.3f V · VCE %.3f V", q.TransistorID, q.State, q.VBE, q.VCE)))
			}
			for _, c := range r.Capacitors {
				circuitControls = append(circuitControls, quiet("mixed-"+c.ID, fmt.Sprintf("%s %.3f V · %.3f µJ", c.ID, c.Voltage, c.Energy*1e6)))
			}
		}
	}
	circuitControls = append(circuitControls, quiet("circuit-capacity", fmt.Sprintf("%d/%d simulated parts · %d/%d drawing objects\n%d/%d nodes (including ground)", state.SimulatedComponents, state.Limits.SimulatedComponents, len(state.Components), state.Limits.DrawingObjects, len(state.Nets), state.Limits.ElectricalNodes)), quiet("selection-navigation-help", "Canvas: Option + ← / →\nHome / End with Option: first / last"))
	inspector = append(inspector, inspectorGroup("circuit", "Circuit information", circuitControls)...)
	inspector = append(inspector, inspectorGroup("measurements", "Live measurements", s.measurementInspector(state))...)
	if workbench.IsTransientAnalysis(state.Analysis) || state.Transient != nil {
		inspector = append(inspector, inspectorGroup("probes", "Scope probes", s.probeInspector(state))...)
	}
	inspector = append(inspector, inspectorGroup("pinned", "Pinned readings", s.pinnedReadingControls(state))...)
	inspector = append(inspector, inspectorGroup("display", "Circuit display", s.displayControls(state))...)
	inspector = append(inspector, inspectorGroup("appearance", "Appearance", s.appearanceControls(state))...)
	if !previewVisible(state) {
		inspector = append(inspector, inspectorGroup("output", "Schematic output", s.schematicPageControls(state))...)
	}
	inspector = append(inspector, inspectorGroup("guidance", "Circuit guidance", s.diagnosticControls(state))...)
	inspector = append(inspector, inspectorGroup("project-work", "File progress", s.projectWorkControls(state))...)
	inspector = append(inspector, inspectorGroup("input-work", "Input progress", s.inputWorkControls(state))...)

	if workbench.IsTransientAnalysis(state.Analysis) || state.Transient != nil {
		inspector = append(inspector, inspectorGroup("transient", "Live simulation", s.transientInspector(state))...)
	}
	inspector = append(inspector, inspectorGroup("recovery", "Recovery", s.recoveryInspector(state))...)
	inspector = append(inspector, inspectorGroup("verification", "Recorded verification", s.verificationInspector(state))...)
	if !sweepVisible(state) {
		inspector = append(inspector, inspectorGroup("sweep", "DC sweep", s.sweepInspector(state))...)
	}
	tools := ui.Node{ID: "tools", Kind: ui.Toolbar, Grow: true, Style: chromeStyle(), Children: []ui.Node{}}
	for _, tool := range []string{"select", "wire", "pan"} {
		n := s.action("tool-"+tool, strings.ToUpper(tool[:1])+tool[1:], "drawing.tool.set", workbench.ToolParams{Tool: tool})
		tools.Children = append(tools.Children, buttonStyle(n, state.Drawing.Tool == tool))
	}
	fit := buttonStyle(s.button("view.fit", "Fit"), state.Viewport.Following)
	fit.Accessibility.Help = "Fit the complete schematic and follow workspace resizing. Pan, zoom or center an object to return to manual positioning."
	tools.Children = append(tools.Children, s.button("drawing.cancel", "Esc"), ui.Node{ID: "tool-spacer", Kind: ui.Row, Grow: true, MinWidth: 1}, s.action("sweep-open", "DC sweep", "sweep.visibility.set", workbench.SweepVisibilityParams{Visible: true}), fit)
	for i := range tools.Children {
		if tools.Children[i].Kind == ui.Button {
			tools.Children[i].Style.Padding = ui.Some(ui.Insets{Top: 9, Bottom: 9, Left: 9, Right: 9})
		}
	}
	hints := "Click a part, terminal or wire to inspect · drag parts/readings · scroll to zoom"
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
	message := ""
	for _, candidate := range state.Recovery.Candidates {
		if !candidate.ActiveSession || candidate.OwnerStatus == workbench.RecoveryOwnerUnknown {
			message = "Recovery available · choose a snapshot in the Inspector to restore a copy."
			break
		}
	}
	if state.Recovery.Error != "" {
		message = "Autosave needs attention · " + state.Recovery.Error
		if !state.Recovery.Enabled {
			message = "Autosave is off · last recovery error: " + state.Recovery.Error
		}
	}
	if s.fileMessage != "" && !(state.Document.Dirty && strings.HasPrefix(s.fileMessage, "Saved ")) {
		message = s.fileMessage
	}
	if s.fileBusy {
		message = "Working on your schematic file…"
	}
	if len(state.DiagnosticPresentation.Items) > 0 {
		message = state.DiagnosticPresentation.Items[0].Summary
	}
	if state.Viewport.FitFailure != nil {
		message = "Enlarge the schematic workspace or pan/zoom to inspect this circuit."
	}
	if s.errorMessage != "" {
		message = s.errorMessage
		if s.requestHelp != nil && s.requestHelp.Details == s.errorMessage {
			message = s.requestHelp.Summary
		}
	}
	messages := quiet("messages", message)
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
			if len(r.MOSFETs) > 0 {
				powerTitle = r.MOSFETs[0].ID + " region"
				power = r.MOSFETs[0].State
			}
			if len(r.Transistors) == 0 && len(r.MOSFETs) == 0 {
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
			if len(r.Capacitors) == 0 && len(r.LEDs) == 0 && len(r.Potentiometers) > 0 {
				p := r.Potentiometers[0]
				voltageTitle = p.ID + " wiper / ground"
				voltage = fmt.Sprintf("%.6g V", p.WiperVoltage)
				currentTitle = p.ID + " pin 1 current"
				current = fmt.Sprintf("%.6g mA", p.Terminal1Current*1000)
			}
			if len(r.Capacitors) == 0 && len(r.LEDs) == 0 && len(r.Resistors) == 0 && len(r.Potentiometers) == 0 && len(r.Diodes) > 0 {
				d := r.Diodes[0]
				voltageTitle = d.ID + " voltage"
				voltage = fmt.Sprintf("%.6g V", d.Voltage)
				currentTitle = d.ID + " current"
				current = fmt.Sprintf("%.6g mA", d.Current*1000)
			}

			if len(r.Inductors) > 0 {
				coil := r.Inductors[0]
				currentTitle = coil.ID + " current"
				current = fmt.Sprintf("%.5g mA", coil.Current*1000)
				if len(r.Capacitors) == 0 {
					voltageTitle = coil.ID + " voltage"
					voltage = fmt.Sprintf("%.5g V", coil.Voltage)
				}
				energy := 0.
				for _, coil := range r.Inductors {
					energy += coil.Energy
				}
				for _, capacitor := range r.Capacitors {
					energy += capacitor.Energy
				}
				powerTitle = "L + C stored energy"
				power = fmt.Sprintf("%.3g µJ", energy*1e6)
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
	// Native max sizes constrain content, while padding consumes extra space.
	// Bound descendants too: a long selector or button must not widen the rail.
	if previewVisible(state) {
		exports := []ui.Node{}
		for _, node := range palette {
			if node.ID == "export-svg" || node.ID == "export-pdf" || node.ID == "export-png" {
				exports = append(exports, node)
			}
		}
		palette = append(s.printingControls(state), s.schematicPageControls(state)...)
		palette = append(palette, exports...)
	} else if sweepVisible(state) {
		palette = s.sweepInspector(state)
	}
	constrainRail(palette, 152)
	constrainRail(inspector, 248)
	filename := "Untitled"
	if state.Document.Path != "" {
		filename = filepath.Base(state.Document.Path)
	}
	if state.Document.Dirty {
		filename += " · Unsaved"
	}
	schematicChildren := []ui.Node{tools, {ID: "schematic", Kind: ui.Extension, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 80})}, Custom: s.drawing}, quiet("schematic-help", hints)}
	if message != "" {
		schematicChildren = append(schematicChildren, messages)
	}
	if previewVisible(state) {
		schematicChildren = append(s.printPreviewPanel(state), messages)
	} else if sweepVisible(state) {
		schematicChildren = append(s.sweepPanel(state), messages)
		s.previewPixels = previewPixels{}
	} else {
		s.previewPixels = previewPixels{}
	}
	root := ui.Node{ID: "root", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(12)), Gap: ui.Some(float32(10)), Background: ui.Some(surface)}, Children: []ui.Node{
		{ID: "header", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 60}), Padding: ui.Some(ui.UniformInsets(12)), Gap: ui.Some(float32(8)), Align: ui.Some(ui.AlignCenter), Background: ui.Some(paper)}, Children: []ui.Node{heading("title", "ELab"), toolbar, statusNode, s.button(pause, pauseText)}},
		{ID: "workspace", Kind: ui.Row, Grow: true, Style: ui.Style{Gap: ui.Some(float32(10))}, Children: []ui.Node{
			{ID: "parts-scroll", Kind: ui.Scroll, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 200}), MaxSize: ui.Some(ui.Size{Width: 200}), Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper)}, Children: []ui.Node{panel("palette", 152, palette...)}},
			{ID: "schematic-panel", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(12)), Gap: ui.Some(float32(8)), Background: ui.Some(paper)}, Children: schematicChildren},
			{ID: "sidebar", Kind: ui.Scroll, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 296}), MaxSize: ui.Some(ui.Size{Width: 296}), Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper)}, Children: []ui.Node{panel("sidebar-content", 248, inspector...)}},
		}},
		{ID: "readings", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 72}), Gap: ui.Some(float32(10)), Align: ui.Some(ui.AlignCenter)}, Children: []ui.Node{metric("current", currentTitle, current), metric("voltage", voltageTitle, voltage), metric("power", powerTitle, power), panel("simulation-info", 232, section("simulation-title", "Live simulation"), quiet("reading-status", readStatus))}},
		{ID: "footer-row", Kind: ui.Row, Grow: true, Style: ui.Style{MaxSize: ui.Some(ui.Size{Height: 20}), Gap: ui.Some(float32(8))}, Children: []ui.Node{quiet("footer", fmt.Sprintf("%s · %d components · %d wires · last change: %s", filename, len(state.Components), len(state.Wires), state.LastActor)), {ID: "footer-spacer", Kind: ui.Row, Grow: true, MinWidth: 1}, quiet("zoom", fmt.Sprintf("%.0f%%", state.View.Zoom*100)), quiet("grid-info", "10-unit snap · local workspace")}},
		s.filePrompt(),
	}}
	if sourcePaneHidden(state) {
		children := make([]ui.Node, 0, len(root.Children))
		for _, child := range root.Children {
			if child.ID != "readings" {
				children = append(children, child)
			}
		}
		root.Children = children
	}
	if scopeAvailable(state) && !sourcePaneHidden(state) {
		if state.ScopeVisible {
			root.Children[1] = s.workspaceDock(root.Children[1], s.scopePanel(state), state)
		} else {
			index := len(root.Children) - 3
			children := append([]ui.Node{}, root.Children[:index]...)
			children = append(children, s.scopePanel(state))
			children = append(children, root.Children[index:]...)
			root.Children = children
		}
	}
	if s.keyboardFocus.ID != "" {
		if _, ok := ui.FocusTarget(root, s.keyboardFocus.ID); ok {
			root.Focus = s.keyboardFocus
		}
	}
	s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
	s.platformSelectors(&root, state.CommandRevision)
	applyPalette(&root, colors)
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
