package main

import (
	"image/color"
	"strings"

	"github.com/dshills/vellum/ui"
)

const toolbarHeight float32 = 40

func toolbarDivider(id string) ui.Node {
	return ui.Node{ID: "toolbar-divider-" + id, Kind: ui.Row, Style: ui.Style{
		MinSize: ui.Some(ui.Size{Width: 1, Height: 18}), MaxSize: ui.Some(ui.Size{Width: 1, Height: 18}),
		Padding: ui.Some(ui.UniformInsets(0)),
	}}
}

// Apply only to toolbar descendants after palette resolution. Inspector tabs
// and the drawing-mode group retain their own presentation. Existing control
// identities, callback captures, enablement and editing state are untouched.
func polishToolbars(node *ui.Node, p appPalette, inToolbar bool) {
	if node.ID == "inspector-navigation" || node.ID == "drawing-mode-tabs" {
		return
	}
	if node.Kind == ui.Toolbar {
		inToolbar = true
		minimum, maximum := node.Style.MinSize.Value, node.Style.MaxSize.Value
		minimum.Height, maximum.Height = toolbarHeight, toolbarHeight
		node.Style.MinSize, node.Style.MaxSize = ui.Some(minimum), ui.Some(maximum)
		padding := node.Style.Padding.Value
		// Keep spacing outside constrained rows: a padded toolbar can let its
		// next sibling cover controls or indicators even with valid AX bounds.
		padding.Top, padding.Bottom = 0, 0
		node.Style.Padding = ui.Some(padding)
		node.Style.Gap = ui.Some(float32(8))
		node.Style.Align = ui.Some(ui.AlignCenter)
		node.Style.Background = ui.Some(p.Paper)
	}
	if inToolbar {
		if strings.HasPrefix(node.ID, "toolbar-divider-") {
			node.Style.Background = ui.Some(p.Grid)
		}
		switch node.Kind {
		case ui.Button, ui.MenuButton, ui.Selector:
			polishToolbarButton(node, p)
		case ui.TextField, ui.NumberField:
			toolbarControlSize(node, 8)
			node.Style.Foreground, node.Style.Background = ui.Some(p.Ink), ui.Some(p.Surface)
			node.Style.Border = ui.Some(ui.Border{Width: 1, Color: toolbarOutline(p)})
			node.Style.CornerRadius = ui.Some(p.ControlRadius)
		}
	}
	for i := range node.Children {
		polishToolbars(&node.Children[i], p, inToolbar)
	}
}

func polishToolbarButton(node *ui.Node, p appPalette) {
	selected := node.Style.Background.Set && node.Style.Background.Value == p.Accent
	if node.Kind == ui.Button || node.Kind == ui.MenuButton {
		node.Appearance = ui.ButtonTool
	}
	toolbarControlSize(node, 10)
	if node.Icon != "" && strings.TrimSpace(node.Text) == "" && node.Kind == ui.Button {
		// Icon actions use the same height as text actions, in a square target.
		node.Style.Padding = ui.Some(ui.UniformInsets(4))
		minimum, maximum := node.Style.MinSize.Value, node.Style.MaxSize.Value
		minimum.Width, maximum.Width = 20, 20
		node.Style.MinSize, node.Style.MaxSize = ui.Some(minimum), ui.Some(maximum)
	}
	if node.ID == "simulation.pause" || node.ID == "simulation.resume" {
		minimum, maximum := node.Style.MinSize.Value, node.Style.MaxSize.Value
		minimum.Width, maximum.Width = 54, 54
		node.Style.MinSize, node.Style.MaxSize = ui.Some(minimum), ui.Some(maximum)
	}
	node.Style.Foreground, node.Style.Background = ui.Some(p.Ink), ui.Some(p.Paper)
	outline := toolbarOutline(p)
	if selected {
		node.Style.Foreground, node.Style.Background = ui.Some(p.Accent), ui.Some(p.Surface)
		outline = p.Accent
		if !strings.Contains(node.Accessibility.Help, "Selected") {
			node.Accessibility.Help = strings.TrimSpace(node.Accessibility.Help + " Selected.")
		}
	}
	if node.Disabled {
		node.Style.Foreground = ui.Some(p.Muted)
	}
	node.Style.Border = ui.Some(ui.Border{Width: 1, Color: outline})
	node.Style.CornerRadius = ui.Some(p.ControlRadius)
	node.Style.TextAlign = ui.Some(ui.AlignCenter)
	workspaceTabWeight(node, selected)
}

func toolbarControlSize(node *ui.Node, horizontalPadding float32) {
	minimum, maximum := node.Style.MinSize.Value, node.Style.MaxSize.Value
	minimum.Width = max(minimum.Width, node.MinWidth)
	minimum.Height, maximum.Height = 20, 20
	node.Style.MinSize, node.Style.MaxSize = ui.Some(minimum), ui.Some(maximum)
	node.Style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: horizontalPadding, Right: horizontalPadding})
}

func toolbarOutline(p appPalette) color.NRGBA {
	if p.Outline.A != 0 {
		return p.Outline
	}
	return p.Grid
}
