package main

// The experimental Native direction composes existing managed controls. Leaf
// identities, guarded callbacks, private editors and all electrical commands are
// retained; only their placement and appearance change.
import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func nativeFixed(n ui.Node, width, height float32) ui.Node {
	n.MinWidth, n.MinHeight = 0, 0
	n.Style.MinSize = ui.Some(ui.Size{Width: width, Height: height})
	n.Style.MaxSize = ui.Some(ui.Size{Width: width, Height: height})
	n.Style.GrowX, n.Style.GrowY = ui.Some(float32(0)), ui.Some(float32(0))
	return n
}
func nativeIcon(n ui.Node, symbol ui.Symbol) ui.Node {
	if n.Accessibility.Name == "" {
		n.Accessibility.Name = n.Text
	}
	n.Icon, n.Text = symbol, ""
	return nativeFixed(n, 20, 20)
}
func nativeSpace(id string) ui.Node {
	return ui.Node{ID: id, Kind: ui.Row, Grow: true, Style: ui.Style{GrowY: ui.Some(float32(0))}}
}

func (s *screen) nativeToolbar(state workbench.Snapshot, width float32) ui.Node {
	actions := []ui.Node{}
	if !state.WorkspacePanels.PartsVisible {
		actions = append(actions, nativeIcon(s.panelToggle(state, false), ui.SymbolSidebar))
	}
	actions = append(actions, nativeIcon(s.button("edit.undo", "Undo"), ui.SymbolUndo), nativeIcon(s.button("edit.redo", "Redo"), ui.SymbolRedo), s.nativeExamplesMenu(state))
	title := s.documentTitle(state)
	title.Style.MinSize = ui.Some(ui.Size{Width: 80, Height: 34})
	// The title column absorbs all remaining space. Its labels have their own
	// width limits so a change between Live and Paused cannot move actions.
	title.Style.MaxSize = ui.Some(ui.Size{Height: 34})
	title.Children[0].Style.MaxSize = ui.Some(ui.Size{Width: max(80, min(420, width-420)), Height: 18})
	subtitle := quiet("document-subtitle", nativeRunSubtitle(state))
	subtitle.Style.Typography = ui.Some(ui.Typography{Size: 11})
	subtitle.Style.MaxSize = ui.Some(ui.Size{Width: max(80, width-420), Height: 16})
	title.Children[1] = subtitle
	title.Style.TextAlign = ui.Some(ui.AlignCenter)
	title.Style.Align = ui.Some(ui.AlignCenter)
	for i := range title.Children {
		title.Children[i].Style.TextAlign = ui.Some(ui.AlignCenter)
	}
	actions = append(actions, title)
	sweep := s.action("sweep-open", "Sweeps", "sweep.visibility.set", workbench.SweepVisibilityParams{Visible: true})
	sweepActivate := sweep.Activate
	sweep.Activate = func() {
		sweepActivate()
		if s.errorMessage == "" {
			s.nativeSweepOpen = true
		}
	}
	fit := s.action("view.fit", "Fit", "view.fit", workbench.FitParams{Width: s.drawing.View.Width, Height: s.drawing.View.Height, FollowViewport: true})
	pause, label := "simulation.pause", "Pause"
	if state.Paused {
		pause, label = "simulation.resume", "Resume"
	}
	primary := s.button(pause, label)
	if state.DCSweep != nil {
		for _, op := range state.DCSweep.Operations {
			if op.Status == "running" || op.Status == "queued" || op.Status == "canceling" {
				primary = s.action("native-stop-sweep", "Stop sweep", "sweep.cancel", workbench.IDParams{ID: op.ID})
				primary.Disabled = op.Status == "canceling"
				break
			}
		}
	}
	primary.Style.Background, primary.Style.Foreground = ui.Some(accent), ui.Some(paper)
	share := s.nativeExportMenu(state)
	inspector := nativeIcon(s.panelToggle(state, true), ui.SymbolSidebar)
	if width < 640 {
		// Keep every toolbar command visible beside open drawers. The title
		// yields space first and icon actions retain full accessible names.
		actions[len(actions)-2] = nativeIcon(actions[len(actions)-2], ui.SymbolFolder)
		sweep = nativeIcon(sweep, ui.SymbolSearch)
		title = actions[len(actions)-1]
		title.Style.MinSize = ui.Some(ui.Size{Width: 24, Height: 34})
		for i := range title.Children {
			title.Children[i].Style.MaxSize = ui.Some(ui.Size{Width: max(24, width-460), Height: 16})
		}
		actions[len(actions)-1] = title
	}
	primaryWidth := float32(84)
	if primary.ID == "native-stop-sweep" {
		primaryWidth = 112
	}
	primary.Style.MinSize = ui.Some(ui.Size{Height: 20})
	primary.Style.MaxSize = ui.Some(ui.Size{})
	primary.Style.GrowX = ui.Some(float32(1))
	primarySlot := ui.Node{ID: "native-primary-slot", Kind: ui.Column, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: primaryWidth, Height: 34}), MaxSize: ui.Some(ui.Size{Width: primaryWidth, Height: 34}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0))}, Children: []ui.Node{primary}}
	actions = append(actions, sweep, fit, primarySlot, share, inspector)
	gap := float32(8)
	if width < 640 {
		gap = 4
	}
	return ui.Node{ID: "header", Kind: ui.Row, Style: ui.Style{GrowX: ui.Some(float32(1)), GrowY: ui.Some(float32(0)), MinSize: ui.Some(ui.Size{Height: 52}), MaxSize: ui.Some(ui.Size{Height: 52}), Padding: ui.Some(ui.Insets{Left: 12, Right: 12}), Gap: ui.Some(gap), Align: ui.Some(ui.AlignCenter), Background: ui.Some(surface)}, Children: actions}
}
func nativeRunSubtitle(state workbench.Snapshot) string {
	status := "Live"
	if state.Paused {
		status = "Paused"
	}
	if state.Transient != nil {
		return fmt.Sprintf("%s · %s · t = %.3f s · run %d", status, state.Analysis, state.Transient.Sample.Time, state.RunID)
	}
	return status + " · " + state.Analysis
}
func (s *screen) nativeExamplesMenu(state workbench.Snapshot) ui.Node {
	browse := s.examplesButton(state)
	actions := map[string]ui.Node{"browse": browse}
	options := []ui.Option{}
	for _, exampleID := range []string{"rc", "led", "speaker-amplifier", "rectifier", "555-flasher", "rlc"} {
		for _, info := range workbench.ExampleCatalog() {
			if info.ID == exampleID {
				id := "example." + exampleID
				target := info.ID
				actions[id] = ui.Node{ID: id, Kind: ui.Button, Disabled: s.fileBusy || s.pendingCommand != "" || state.Closed, Activate: func() { s.openBrowserExample(target) }}
				options = append(options, ui.Option{ID: id, Label: info.Title, Checkable: true, Checked: state.Example != nil && state.Example.Info.ID == info.ID})
				break
			}
		}
	}
	options = append(options, ui.Option{ID: "browse", Label: "Browse all examples…"})
	for _, pair := range [][2]string{{"project.clear", "New schematic"}, {"file.open", "Open…"}, {"file.save", "Save"}, {"file.save-as", "Save as…"}, {"app.close", "Close window"}} {
		actions[pair[0]] = s.button(pair[0], pair[1])
		options = append(options, ui.Option{ID: pair[0], Label: pair[1]})
	}
	return ui.Node{ID: "examples-menu", Kind: ui.MenuButton, Text: "Examples", Options: options, Accessibility: ui.Accessibility{Name: "Examples and file actions"}, Change: func(e ui.ChangeEvent) {
		if n, ok := actions[e.SelectedID]; ok && !n.Disabled {
			n.Activate()
		}
	}}
}
func (s *screen) nativeExportMenu(state workbench.Snapshot) ui.Node {
	entries := map[string]func(){"pdf": func() { s.beginFile("export.schematic.pdf", true, workbench.PDFExportParams{}, nil) }, "png": func() { s.beginFile("export.schematic.png", true, workbench.PNGExportParams{}, nil) }, "svg": func() { s.beginFile("export.schematic.svg", true, workbench.SaveParams{}, nil) }}
	return ui.Node{ID: "native-share", Kind: ui.MenuButton, Icon: ui.SymbolShare, Disabled: s.fileBusy || state.Closed, Accessibility: ui.Accessibility{Name: "Export schematic", Help: "Export a local PDF, PNG or SVG copy. ELab documents are local files."}, Options: []ui.Option{{ID: "pdf", Label: "Schematic PDF…"}, {ID: "png", Label: "Schematic PNG…"}, {ID: "svg", Label: "Schematic SVG…"}}, Change: func(e ui.ChangeEvent) {
		if fn := entries[e.SelectedID]; fn != nil {
			fn()
		}
	}}
}
func (s *screen) nativeParts(state workbench.Snapshot, nodes []ui.Node, width float32) ui.Node {
	var content, footer []ui.Node
	var search, category ui.Node
	for _, n := range nodes {
		if strings.HasPrefix(n.ID, railLabelBoundsPrefix) && len(n.Children) == 1 {
			n = n.Children[0]
		}
		switch n.ID {
		case "parts-title", "part-search-label":
			continue
		case "part-search-box":
			search = n
		case "part-category":
			category = n
		case "wire-by-name":
			old := n.Activate
			n.Activate = func() {
				old()
				if s.errorMessage == "" {
					s.nativeConnectOpen = true
				}
			}
			n.Text = "Connect by name…"
			n.Style.MinSize = ui.Some(ui.Size{Width: 196, Height: 20})
			n.Style.MaxSize = ui.Some(ui.Size{Width: 196, Height: 20})
			footer = append(footer, n)
		default:
			content = append(content, n)
		}
	}
	if search.ID != "" {
		content = append([]ui.Node{search, {ID: "native-parts-scope", Kind: ui.Row, Children: []ui.Node{category, quiet("native-match-count", fmt.Sprintf("%d matches", len(state.MatchingParts)))}}}, content...)
	}
	footer = append(footer, quiet("footer", fmt.Sprintf("%d components · %d wires", len(state.Components), len(state.Wires))))
	constrainRail(content, width-24)
	constrainRail(footer, width-24)
	toggle := nativeIcon(s.panelToggle(state, false), ui.SymbolSidebar)
	header := ui.Node{ID: "parts-drawer-header", Kind: ui.Row, Style: ui.Style{GrowX: ui.Some(float32(1)), MinSize: ui.Some(ui.Size{Width: width - 24, Height: 52}), MaxSize: ui.Some(ui.Size{Width: width - 24, Height: 52}), Padding: ui.Some(ui.Insets{Left: 12, Right: 12}), Align: ui.Some(ui.AlignCenter)}, Children: []ui.Node{nativeSpace("parts-title-space"), toggle}}
	return ui.Node{ID: "native-parts", Kind: ui.Column, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: width}), MaxSize: ui.Some(ui.Size{Width: width}), GrowY: ui.Some(float32(1)), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0)), Background: ui.Some(nativeSidebarToken)}, Children: []ui.Node{header, {ID: "parts-scroll", Kind: ui.Scroll, Grow: true, Style: ui.Style{Padding: ui.Some(ui.Insets{Left: 12, Right: 12}), Background: ui.Some(nativeSidebarToken)}, Children: []ui.Node{{ID: "palette", Kind: ui.Column, Style: ui.Style{Gap: ui.Some(float32(8)), Padding: ui.Some(ui.UniformInsets(0))}, Children: content}}}, {ID: "native-parts-footer", Kind: ui.Column, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(12)), Gap: ui.Some(float32(8)), Background: ui.Some(nativeSidebarToken)}, Children: footer}}}
}
func (s *screen) nativeZoom(state workbench.Snapshot) ui.Node {
	values := []float64{.25, .33, .5, .67, .75, .9, 1, 1.1, 1.25, 1.5, 1.75, 2, 2.5, 3, 4}
	lo, hi := values[0], values[len(values)-1]
	for _, v := range values {
		if v < state.View.Zoom-1e-8 {
			lo = v
		}
		if v > state.View.Zoom+1e-8 {
			hi = v
			break
		}
	}
	minus := s.action("native-zoom-out", "Zoom out", "view.zoom", map[string]any{"factor": lo / state.View.Zoom})
	plus := s.action("native-zoom-in", "Zoom in", "view.zoom", map[string]any{"factor": hi / state.View.Zoom})
	minus.Disabled = state.View.Zoom <= .25
	plus.Disabled = state.View.Zoom >= 4
	return ui.Node{ID: "native-zoom", Kind: ui.Row, Style: ui.Style{Gap: ui.Some(float32(4)), Padding: ui.Some(ui.UniformInsets(4)), Background: ui.Some(paper), CornerRadius: ui.Some(float32(9)), Border: ui.Some(ui.Border{Width: 1, Color: nativeHairlineToken})}, Children: []ui.Node{nativeIcon(minus, ui.SymbolRemove), nativeFixed(quiet("zoom", fmt.Sprintf("%.0f%%", state.View.Zoom*100)), 36, 20), nativeIcon(plus, ui.SymbolAdd)}}
}
func (s *screen) nativeShell(state workbench.Snapshot, total float32, palette, inspector []ui.Node, tools, messages ui.Node, overview []ui.Node) ui.Node {
	left, right := sidePanelWidths(total, state.WorkspacePanels)
	centerWidth := total - left - right
	if left > 0 {
		centerWidth -= sideDividerWidth
	}
	if right > 0 {
		centerWidth -= sideDividerWidth
	}
	centerWidth = max(360, centerWidth)
	var wireFields []ui.Node
	remaining := make([]ui.Node, 0, len(palette))
	for _, n := range palette {
		if strings.HasPrefix(n.ID, "wire-") && n.ID != "wire-by-name" || n.ID == "wires-title" || n.ID == "connect-terminals" {
			wireFields = append(wireFields, n)
		} else {
			remaining = append(remaining, n)
		}
	}
	palette = remaining
	tools = s.nativeToolPill(state, tools)
	if centerWidth < 700 {
		w := nativeToolPillWidth(true, len(tools.Children) > 4)
		tools.Style.MinSize = ui.Some(ui.Size{Width: w, Height: 34})
		tools.Style.MaxSize = tools.Style.MinSize
		for i := range tools.Children {
			if tools.Children[i].ID == "schematic-help" {
				tools.Children[i].Text = ""
			}
		}
	}
	sheetChildren := []ui.Node{{ID: "schematic", Kind: ui.Extension, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 80})}, Custom: s.drawing}, tools, s.nativeZoom(state)}
	if messages.Text != "" || len(messages.Children) > 0 {
		sheetChildren = append(sheetChildren, s.nativeSheetNotice(state, messages, centerWidth))
	}
	sheet := ui.Node{ID: "schematic-panel", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0)), Background: ui.Some(paper)}, Children: sheetChildren}
	middle := sheet
	if !previewVisible(state) {
		s.previewPixels = previewPixels{}
	}
	if previewVisible(state) {
		middle = ui.Node{ID: "native-print-preview", Kind: ui.Column, Grow: true, Children: s.printPreviewPanel(state)}
	} else if state.ScopeVisible {
		middle = s.workspaceDock(sheet, s.instrumentPanel(state, centerWidth), state)
	} else {
		middle = ui.Node{ID: "native-collapsed-dock", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0))}, Children: []ui.Node{sheet, s.instrumentPanel(state, centerWidth)}}
	}
	center := ui.Node{ID: "native-center", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0))}, Children: []ui.Node{s.nativeToolbar(state, centerWidth), middle}}
	// Keep each old reading available in Inspect, rather than erasing it with the
	// removed bottom strip. Existing Pin/Plot controls remain independent.
	if state.InspectorSection == "component" && len(state.Components) > 0 && state.MeasurementInspection.Target == nil {
		inspector = append(inspector, inspectorGroup("circuit-readings", "Circuit overview", overview)...)
	}
	partsRail, inspectorRail := ui.Node{}, ui.Node{}
	if left > 0 {
		partsRail = s.nativeParts(state, palette, left)
	}
	if right > 0 {
		if s.nativeSweepOpen {
			var kept []ui.Node
			for _, g := range inspector {
				if g.ID != "inspector-group-sweep" {
					kept = append(kept, g)
				}
			}
			inspector = kept
		}
		constrainRail(inspector, right-40)
		inspectorRail = s.inspectorRail(state, inspector, right)
	}
	// Reuse the qualified, origin-guarded drawer dividers without their old
	// extra header row. Native header buttons provide the same toggles.
	panels := s.sidePanels(state, partsRail, center, inspectorRail, total, left, right)
	children := []ui.Node{panels.Children[1], s.filePrompt()}
	if s.nativeSweepOpen && state.DCSweep != nil {
		fields := s.sweepInspector(state)
		constrainRail(fields, 264)
		pop, err := (ui.Popover{ID: "native-sweep-popover", AnchorID: "sweep-open", Label: "Sweep", Open: true, Dismiss: func() { s.nativeSweepOpen = false; s.app.Invalidate("main") }, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 300}), MaxSize: ui.Some(ui.Size{Width: 300}), Padding: ui.Some(ui.UniformInsets(12))}, Children: []ui.Node{{ID: "native-sweep-fields", Kind: ui.Scroll, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 264, Height: 360}), MaxSize: ui.Some(ui.Size{Width: 264, Height: 460})}, Children: fields}}}).View()
		if err == nil {
			children = append(children, pop)
		}
	}

	if len(wireFields) > 0 {
		constrainRail(wireFields, 248)
		cancel := ui.Node{ID: "native-connect-cancel", Kind: ui.Button, Text: "Cancel"}
		cancel.Activate = func() { s.nativeConnectOpen = false; s.app.Invalidate("main") }
		wireFields = append(wireFields, cancel)
		pop, _ := (ui.Popover{ID: "native-connect-popover", AnchorID: "wire-by-name", Label: "Connect by name", Open: s.nativeConnectOpen && state.Drawing.Tool == "wire", Dismiss: func() { s.nativeConnectOpen = false; s.app.Invalidate("main") }, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(14)), MinSize: ui.Some(ui.Size{Width: 280}), MaxSize: ui.Some(ui.Size{Width: 280})}, Children: wireFields}).View()
		children = append(children, pop)
	}
	return ui.Node{ID: "root", Kind: ui.Row, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(1)), Background: ui.Some(surface)}, Children: children}
}
func (s *screen) nativeToolPill(state workbench.Snapshot, old ui.Node) ui.Node {
	var nodes []ui.Node
	if group, ok := findNode(old, "drawing-mode-tabs"); ok {
		for _, n := range group.Children {
			n.Accessibility.Name = n.Text
			if n.ID != "tool-wire" {
				n.Text = ""
			}
			n.Icon = ui.SymbolRight
			nodes = append(nodes, n)
		}
	}
	if state.Drawing.Tool == "wire" || state.Drawing.Tool == "place" {
		if cancel, ok := findNode(old, "drawing.cancel"); ok {
			nodes = append(nodes, cancel)
		}
	}
	hint := "R rotate · W wire · Delete remove"
	if state.Drawing.Tool == "wire" {
		hint = "Click two terminal dots · Escape cancels"
	}
	if state.Drawing.Tool == "pan" {
		hint = "Drag to pan · scroll to zoom"
	}
	nodes = append(nodes, quiet("schematic-help", hint))
	width := nativeToolPillWidth(false, len(nodes) > 4)
	return ui.Node{ID: "tools", Kind: ui.Toolbar, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: width, Height: 34}), MaxSize: ui.Some(ui.Size{Width: width, Height: 34}), Padding: ui.Some(ui.UniformInsets(4)), Gap: ui.Some(float32(6)), CornerRadius: ui.Some(float32(13)), Background: ui.Some(paper), Align: ui.Some(ui.AlignCenter), Border: ui.Some(ui.Border{Width: 1, Color: nativeHairlineToken}), GrowX: ui.Some(float32(0)), GrowY: ui.Some(float32(0))}, Children: nodes}
}

func (s *screen) nativeSheetNotice(state workbench.Snapshot, original ui.Node, width float32) ui.Node {
	message, ok := findNode(original, "messages")
	if !ok || s.errorMessage != "" || state.Recording != nil && state.Recording.Viewing {
		return original
	}
	for _, warning := range state.SourceWarnings {
		if message.Text != warning.Message {
			continue
		}
		for _, advice := range state.SourceSampling {
			if advice.ID != warning.ObjectID || !advice.CanFit {
				continue
			}
			message.Text = fmt.Sprintf("%s’s %s sample step may hide waveform detail.", advice.ID, nativeQuantity(state.TransientSettings.StepSeconds, "s"))
			if width < 700 {
				message.Accessibility.Name = message.Text
				message.Text = fmt.Sprintf("%s · %s sample step", advice.ID, nativeQuantity(state.TransientSettings.StepSeconds, "s"))
			}
			message.Style.Typography = ui.Some(ui.Typography{Size: 11})
			message.Style.MaxSize = ui.Some(ui.Size{Height: 20})
			apply := s.action("native-fit-sampling", "Use "+nativeQuantity(advice.RecommendedStep, "s"), "simulation.sampling.fit", struct{}{})
			target := "part-search"
			if !state.WorkspacePanels.PartsVisible {
				target = "simulation.pause"
				if state.Paused {
					target = "simulation.resume"
				}
			}
			apply = s.focusAfterAction(apply, target)
			return ui.Node{ID: "native-sampling-notice", Kind: ui.Row, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(6)), Gap: ui.Some(float32(8)), Align: ui.Some(ui.AlignCenter), CornerRadius: ui.Some(float32(14)), Background: ui.Some(paper), Border: ui.Some(ui.Border{Width: 1, Color: nativeHairlineToken})}, Children: []ui.Node{message, apply}}
		}
	}
	return original
}
