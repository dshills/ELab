package main

import "github.com/dshills/vellum/ui"

// Vellum reserves a two-point focus border for managed actions, including when
// they are unfocused. Keep ELab's existing outer sizes by taking the additional
// allowance from explicit padding rather than clipping constrained toolbars.
func reserveManagedFocusOutline(node *ui.Node) {
	if (node.Kind == ui.Button || node.Kind == ui.MenuButton || node.Kind == ui.Selector || node.Kind == ui.Checkbox) && node.Style.Border.Set && node.Style.Padding.Set {
		allowance := max(float32(0), 2-node.Style.Border.Value.Width)
		padding := node.Style.Padding.Value
		padding.Top = max(0, padding.Top-allowance)
		padding.Bottom = max(0, padding.Bottom-allowance)
		padding.Left = max(0, padding.Left-allowance)
		padding.Right = max(0, padding.Right-allowance)
		node.Style.Padding = ui.Some(padding)
	}
	for i := range node.Children {
		reserveManagedFocusOutline(&node.Children[i])
	}
}
