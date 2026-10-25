package main

import "github.com/dshills/vellum/ui"

func layoutComponentSearch(node *ui.Node, width float32) {
	node.Style.MinSize = ui.Some(ui.Size{Width: width})
	node.Style.MaxSize = ui.Some(ui.Size{Width: width})
	for i := range node.Children {
		child := &node.Children[i]
		child.Style.Padding = ui.Some(ui.UniformInsets(0))
		switch child.ID {
		case "part-search":
			child.Style.Padding = ui.Some(ui.Insets{Top: 6, Bottom: 6, Left: 6, Right: 6})
			child.Style.MinSize = ui.Some(ui.Size{Width: width - 68, Height: 20})
			child.Style.MaxSize = child.Style.MinSize
		case "part-search-clear":
			child.Style.MinSize = ui.Some(ui.Size{Width: 24, Height: 28})
			child.Style.MaxSize = child.Style.MinSize
			child.Style.Border = ui.Some(ui.Border{})
		}
	}
}
