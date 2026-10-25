package main

import (
	"strconv"

	"github.com/dshills/vellum/ui"
)

func secondaryDrawingActions(revision uint64, actions ...ui.Node) ui.Node {
	bar := ui.WithToolbarOverflow(ui.Node{ID: "drawing-secondary-actions", Kind: ui.Toolbar,
		Accessibility: ui.Accessibility{Name: "Secondary schematic actions"}, Style: ui.Style{
			GrowX: ui.Some(float32(1)), GrowY: ui.Some(float32(0)), Padding: ui.Some(ui.UniformInsets(0)),
			MinSize: ui.Some(ui.Size{Width: 56, Height: 30}), MaxSize: ui.Some(ui.Size{Width: 144, Height: 30}), Gap: ui.Some(float32(8)),
		}, Children: actions})
	bar.Children[len(bar.Children)-1].Text = "More"
	// An open popup belongs to the command context that displayed it. Retire
	// it on any intervening edit, rather than borrowing refreshed callbacks.
	bar.Children[len(bar.Children)-1].ControlStateKey = strconv.FormatUint(revision, 10)
	return bar
}
