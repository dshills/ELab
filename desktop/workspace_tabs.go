package main

import (
	"strings"

	"github.com/dshills/vellum/ui"
)

// Keep the existing buttons and guarded callbacks. The group is presentation;
// native toolbar keyboard traversal still reaches the same stable control IDs.
func drawingModeTabs(actions []ui.Node) ui.Node {
	for i := range actions {
		actions[i].Appearance = ui.ButtonTool
		actions[i].Style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 8, Right: 8})
		actions[i].Style.MinSize = ui.Some(ui.Size{Width: 46, Height: 20})
		actions[i].Style.MaxSize = ui.Some(ui.Size{Width: 46, Height: 20})
		actions[i].Style.TextAlign = ui.Some(ui.AlignCenter)
	}
	return ui.Node{ID: "drawing-mode-tabs", Kind: ui.Row, Accessibility: ui.Accessibility{Name: "Drawing mode"}, Style: ui.Style{
		MinSize: ui.Some(ui.Size{Height: 30}), MaxSize: ui.Some(ui.Size{Height: 30}),
		Padding: ui.Some(ui.UniformInsets(2)), Gap: ui.Some(float32(0)), Align: ui.Some(ui.AlignCenter),
		Background: ui.Some(surface),
	}, Children: actions}
}

func inspectorTabCell(action ui.Node, section string, selected bool) ui.Node {
	action.Appearance = ui.ButtonTool
	action.Style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 8, Right: 8})
	action.Style.MinSize = ui.Some(ui.Size{Width: 72, Height: 20})
	action.Style.MaxSize = ui.Some(ui.Size{Width: 72, Height: 20})
	action.Style.TextAlign = ui.Some(ui.AlignCenter)
	indicator := paper
	if selected {
		indicator = accent
	}
	return ui.Node{ID: "inspector-tab-cell-" + section, Kind: ui.Column, Style: ui.Style{
		MinSize: ui.Some(ui.Size{Width: 88, Height: 34}), MaxSize: ui.Some(ui.Size{Width: 88, Height: 34}),
		Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(2)),
	}, Children: []ui.Node{action, {
		ID: "inspector-tab-indicator-" + section, Kind: ui.Row,
		Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 88, Height: 2}), MaxSize: ui.Some(ui.Size{Width: 88, Height: 2}), Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(indicator)},
	}}}
}

func applyWorkspaceTabStyle(node *ui.Node, p appPalette, selected bool) {
	switch node.ID {
	case "drawing-mode-tabs":
		outline := p.Grid
		if p.Outline.A != 0 {
			outline = p.Outline
		}
		node.Style.Border = ui.Some(ui.Border{Width: 1, Color: outline})
		node.Style.CornerRadius = ui.Some(p.ControlRadius + 2)
	case "tool-select", "tool-wire", "tool-pan":
		node.Style.Foreground, node.Style.Background = ui.Some(p.Muted), ui.Some(p.Surface)
		node.Style.Border = ui.Some(ui.Border{})
		if selected {
			outline := p.Grid
			if p.Outline.A != 0 {
				outline = p.Accent
			}
			node.Style.Foreground, node.Style.Background = ui.Some(p.Accent), ui.Some(p.Paper)
			node.Style.Border = ui.Some(ui.Border{Width: 1, Color: outline})
			// The selected outline occupies two units; keep every segment's
			// outer width identical so choosing a mode never moves its labels.
			node.Style.MinSize = ui.Some(ui.Size{Width: 44, Height: 20})
			node.Style.MaxSize = ui.Some(ui.Size{Width: 44, Height: 20})
		}
		workspaceTabWeight(node, selected)
	case "inspector-section-component", "inspector-section-simulation", "inspector-section-document", "scope-view-scope", "scope-view-spectrum":
		node.Style.Foreground, node.Style.Background = ui.Some(p.Muted), ui.Some(p.Paper)
		node.Style.Border = ui.Some(ui.Border{})
		node.Style.CornerRadius = ui.Some(float32(3))
		if selected {
			node.Style.Foreground = ui.Some(p.Accent)
		}
		workspaceTabWeight(node, selected)
	default:
		if strings.HasPrefix(node.ID, "inspector-tab-indicator-") {
			node.Style.Background = ui.Some(p.Grid)
			if selected {
				node.Style.Background = ui.Some(p.Accent)
			}
		}
	}
}

func workspaceTabWeight(node *ui.Node, selected bool) {
	typeface := node.Style.Typography.Value
	if typeface.Size == 0 {
		typeface.Size = 13
	}
	typeface.Weight = 500
	if selected {
		typeface.Weight = 600
	}
	node.Style.Typography = ui.Some(typeface)
}
