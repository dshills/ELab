package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Rehome the original plot controls; never replace a measured result with
// decorative/mock waveforms. The advanced disclosure retains every editor.
func (s *screen) nativeInstrumentBody(state workbench.Snapshot, body ui.Node, width float32) (ui.Node, []ui.Node) {
	if !state.ScopeVisible {
		return body, nil
	}
	if state.Recording != nil && state.Recording.Viewing && state.ScopeView != "spectrum" {
		return body, nil
	}
	var plot, footer, empty ui.Node
	var advanced, header []ui.Node
	for _, n := range body.Children {
		switch n.ID {
		case "scope-toolbar", "scope-empty-controls", "spectrum-dock-controls":
			var rest []ui.Node
			for _, c := range n.Children {
				switch c.ID {
				case "scope-visibility", "transient-step", "simulation.restart":
					header = append(header, c)
				default:
					rest = append(rest, c)
				}
			}
			if len(rest) > 0 {
				n.Children = rest
				advanced = append(advanced, n)
			}
		case "rc-scope", "spectrum-plot", "sweep-plot":
			plot = n
		case "scope-trace-row":
			footer = n
		case "spectrum-dock-summary":
			footer = ui.Node{ID: "native-spectrum-footer", Kind: ui.Row, Children: []ui.Node{n}}
		case "instrument-empty", "sweep-no-record":
			empty = n
		default:
			advanced = append(advanced, n)
		}
	}
	if body.ID == "native-sweep-panel" {
		header = append(header, s.scopeVisibilityButton(state))
	}
	if state.ScopeView == "spectrum" && state.Transient != nil {
		for _, row := range s.scopePanel(state).Children {
			if row.ID == "scope-toolbar" {
				for _, action := range row.Children {
					if action.ID == "transient-step" || action.ID == "simulation.restart" {
						header = append(header, action)
					}
				}
			}
		}
	}
	for i := range header {
		n := &header[i]
		w := float32(75)
		switch n.ID {
		case "transient-step":
			w = 100
		case "simulation.restart":
			w = 48
		}
		if width < 800 {
			if n.ID == "scope-visibility" {
				n.Accessibility.Name = n.Text
				n.Text = ""
				n.Icon = ui.SymbolDown
				w = 20
			}
			if n.ID == "transient-step" || n.ID == "simulation.restart" {
				n.Accessibility.Name = n.Text
				n.Text, n.Icon = "", ui.SymbolRight
				if n.ID == "simulation.restart" {
					n.Icon = ui.SymbolUndo
				}
				w = 20
			}
		}
		n.MinWidth, n.MinHeight = 0, 0
		n.Style.MinSize = ui.Some(ui.Size{Width: w, Height: 20})
		n.Style.MaxSize = n.Style.MinSize
		// A fixed outer slot absorbs native disabled/focus inset changes.
		// Let the managed button fill that slot so both its bounds and its
		// neighbors remain stationary while the simulation is running.
		action := *n
		action.Style.MinSize = ui.Some(ui.Size{Height: 20})
		action.Style.MaxSize = ui.Some(ui.Size{})
		action.Style.GrowY = ui.Some(float32(1))
		action.Style.GrowX = ui.Some(float32(1))
		header[i] = ui.Node{ID: "native-dock-slot-" + action.ID, Kind: ui.Column, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: w + 26, Height: 36}), MaxSize: ui.Some(ui.Size{Width: w + 26, Height: 36}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0))}, Children: []ui.Node{action}}
	}
	if plot.ID == "" && len(advanced) == 0 {
		body.Children = []ui.Node{empty}
		return body, header
	}
	// Fit the actual canvas to the dock. Cursor/settings controls no longer take
	// the plot's space until the user explicitly opens them.
	plot.Style.MinSize = ui.Some(ui.Size{Width: 240, Height: 120})
	plot.Style.MaxSize = ui.Some(ui.Size{})
	plot.Style.GrowX, plot.Style.GrowY = ui.Some(float32(1)), ui.Some(float32(1))
	plot.Style.Background = ui.Some(paper)
	plot.Style.Border = ui.Some(ui.Border{Width: 1, Color: nativeHairlineToken})
	plot.Style.CornerRadius = ui.Some(float32(8))
	if footer.ID != "" {
		footer.Style = ui.Style{MinSize: ui.Some(ui.Size{Height: 30}), MaxSize: ui.Some(ui.Size{Height: 30}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(8)), Align: ui.Some(ui.AlignCenter)}
		for i := range footer.Children {
			if footer.Children[i].Kind == ui.Label {
				footer.Children[i].Style.MinSize = ui.Some(ui.Size{Width: max(140, width-400), Height: 20})
				footer.Children[i].Style.MaxSize = ui.Some(ui.Size{Width: max(140, width-400), Height: 20})
			}
		}
	}
	for i := range advanced {
		advanced[i] = nativeInstrumentRows(advanced[i], max(220, width-72))
		nativeClampInstrumentText(&advanced[i], max(220, width-72))
	}
	// The settings are the existing shared Live/Spectrum disclosure state, with
	// a distinct node ID so inspector and dock can coexist without duplicate IDs.
	section := "live"
	if state.ScopeView == "spectrum" {
		section = "spectrum"
	}
	details := s.detailsSection(state, section, "Instrument controls & cursors", advanced)
	details.ID = "workspace-details-instrument"
	children := []ui.Node{}
	if empty.ID != "" {
		children = append(children, empty)
	}
	if plot.ID != "" {
		children = append(children, ui.Node{ID: "native-plot-card", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper), CornerRadius: ui.Some(float32(8)), Border: ui.Some(ui.Border{Width: 1, Color: nativeHairlineToken})}, Children: []ui.Node{plot}})
	}
	if footer.ID != "" {
		children = append(children, footer)
	}
	if state.ScopeView != "spectrum" && state.Scope.DeltaSeconds != nil {
		dt := *state.Scope.DeltaSeconds
		frequency := "—"
		if state.Scope.Frequency != nil {
			frequency = fmt.Sprintf("%.4g", *state.Scope.Frequency)
		}
		children = append(children, quiet("native-cursor-summary", fmt.Sprintf("Δ %.2f ms · %s Hz", dt*1000, frequency)))
	}
	if len(advanced) > 0 {
		children = append(children, details)
	}
	body.Children = children
	body.Style.Padding = ui.Some(ui.Insets{Left: 12, Right: 12, Top: 4, Bottom: 4})
	body.Style.MaxSize = ui.Some(ui.Size{})
	return body, header
}

func nativeSegmentedTabs(n *ui.Node, p appPalette) {
	selected := strings.Contains(n.Accessibility.Help, "Selected") || n.Style.Foreground.Value == p.Accent
	if strings.HasPrefix(n.ID, "inspector-tab-indicator-") {
		n.Style.MinSize = ui.Some(ui.Size{})
		n.Style.MaxSize = ui.Some(ui.Size{})
		n.Style.Background = ui.Some(p.Surface)
	}
	if strings.HasPrefix(n.ID, "inspector-tab-cell-") {
		n.Style.MinSize = ui.Some(ui.Size{Width: 84, Height: 28})
		n.Style.MaxSize = ui.Some(ui.Size{Width: 84, Height: 28})
		n.Style.Gap = ui.Some(float32(0))
		n.Style.Padding = ui.Some(ui.UniformInsets(0))
		if len(n.Children) > 1 {
			n.Children = n.Children[:1]
		}
	}
	if strings.HasPrefix(n.ID, "inspector-section-") || strings.HasPrefix(n.ID, "scope-view-") {
		n.Style.Foreground, n.Style.Background = ui.Some(p.Muted), ui.Some(nativeSegmentColor(p))
		n.Style.Border = ui.Some(ui.Border{})
		n.Style.CornerRadius = ui.Some(float32(5))
		n.Style.MinSize, n.Style.MaxSize = ui.Some(ui.Size{Width: 62, Height: 18}), ui.Some(ui.Size{Width: 62, Height: 18})
		n.Style.Padding = ui.Some(ui.Insets{Top: 3, Bottom: 3, Left: 8, Right: 8})
		if selected {
			n.Style.Foreground, n.Style.Background = ui.Some(p.Ink), ui.Some(p.Paper)
			n.Style.Border = ui.Some(ui.Border{Width: 1, Color: nativeHairlineToken})
		}
	}
}

// Instrument editors must stay editable: a toolbar may overflow buttons into a
// menu, but moving a text field there makes it unavailable. Keep original
// controls/callbacks in bounded rows, and keep each caption beside its editor.
func nativeInstrumentRows(n ui.Node, width float32) ui.Node {
	if n.Kind != ui.Toolbar {
		return n
	}
	var rows []ui.Node
	row := ui.Node{ID: n.ID + "-row-0", Kind: ui.Row, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(8)), Align: ui.Some(ui.AlignCenter)}}
	used := float32(0)
	for i := 0; i < len(n.Children); i++ {
		group := n.Children[i]
		estimate := nativeInstrumentControlWidth(group)
		if group.Kind == ui.Button {
			action := group
			action.MinWidth, action.MinHeight = 0, 0
			action.Style.MinSize, action.Style.MaxSize = ui.Some(ui.Size{}), ui.Some(ui.Size{})
			action.Style.GrowX, action.Style.GrowY = ui.Some(float32(1)), ui.Some(float32(1))
			group = ui.Node{ID: action.ID + "-stable-slot", Kind: ui.Column, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: estimate, Height: 36}), MaxSize: ui.Some(ui.Size{Width: estimate, Height: 36}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0))}, Children: []ui.Node{action}}
		}
		if group.Kind == ui.Label && i+1 < len(n.Children) {
			next := n.Children[i+1]
			if next.Kind == ui.TextField || strings.HasSuffix(next.ID, "-slot") {
				group = ui.Node{ID: group.ID + "-pair", Kind: ui.Row, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(8)), Align: ui.Some(ui.AlignCenter)}, Children: []ui.Node{group, next}}
				estimate += 8 + nativeInstrumentControlWidth(next)
				i++
			}
		}
		if used > 0 && used+8+estimate > width {
			rows = append(rows, row)
			row.Children = nil
			row.ID = fmt.Sprintf("%s-row-%d", n.ID, len(rows))
			used = 0
		}
		row.Children = append(row.Children, group)
		used += estimate + 8
	}
	if len(row.Children) > 0 {
		rows = append(rows, row)
	}
	n.Kind = ui.Column
	n.Children = rows
	n.Style = ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(6))}
	return n
}
func nativeInstrumentControlWidth(n ui.Node) float32 {
	if n.Style.MinSize.Set && n.Style.MinSize.Value.Width > 0 {
		return n.Style.MinSize.Value.Width + 8
	}
	return max(40, float32(len([]rune(n.Text)))*7+32)
}

func nativeClampInstrumentText(n *ui.Node, width float32) {
	if n.Kind == ui.Label {
		minimum, maximum := n.Style.MinSize.Value, n.Style.MaxSize.Value
		if maximum.Width <= 0 || maximum.Width > width {
			maximum.Width = width
		}
		minimum.Width = min(minimum.Width, maximum.Width)
		n.Style.MinSize, n.Style.MaxSize = ui.Some(minimum), ui.Some(maximum)
	}
	for i := range n.Children {
		nativeClampInstrumentText(&n.Children[i], width)
	}
}
