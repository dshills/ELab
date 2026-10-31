package main

import (
	"image/color"
	"strings"

	"cogentcore.org/core/icons"
	"github.com/dshills/vellum/ui"
)

// Experimental design tokens. Distinct sentinels are mapped only at the view
// boundary, keeping system/dark and independent windows isolated.
var nativeSidebarToken = color.NRGBA{R: 236, G: 236, B: 239, A: 255}
var nativeSegmentToken = color.NRGBA{R: 227, G: 227, B: 231, A: 255}
var nativeHairlineToken = color.NRGBA{A: 31}

func polishNativeDesign(n *ui.Node, p appPalette) {
	for i := range n.Children {
		polishNativeDesign(&n.Children[i], p)
	}
	sidebar := nativeSidebarToken
	hairline := nativeHairlineColor(p)
	if p.Paper.R < 80 {
		sidebar = color.NRGBA{R: 38, G: 38, B: 42, A: 255}
	}
	if n.Style.Background.Set && n.Style.Background.Value == nativeSegmentToken {
		n.Style.Background = ui.Some(nativeSegmentColor(p))
	}
	if n.Style.Background.Set && n.Style.Background.Value == nativeSidebarToken {
		n.Style.Background = ui.Some(sidebar)
	}
	if n.Style.Border.Set && n.Style.Border.Value.Color == nativeHairlineToken {
		b := n.Style.Border.Value
		b.Color = hairline
		n.Style.Border = ui.Some(b)
	}
	if n.ID == "diagnostic-details" {
		n.Text = "Circuit checks"
	}
	if n.ID == "setting-label-speed" {
		n.Text = "Playback"
	}

	switch n.ID {
	case "header", "inspector-rail", "inspector-navigation", "instrument-panel", "instrument-tabs", "scope-panel", "spectrum-panel", "sidebar-content":
		n.Style.Background = ui.Some(p.Surface)
	case "parts-scroll", "palette", "native-parts-footer", "parts-drawer-header":
		n.Style.Background = ui.Some(sidebar)
	}
	if n.Kind == ui.Button || n.Kind == ui.MenuButton || n.Kind == ui.Selector || n.Kind == ui.TextField || n.Kind == ui.NumberField {
		n.Style.CornerRadius = ui.Some(float32(7))
		n.Style.Typography = ui.Some(ui.Typography{Size: 13})
		selected := n.Style.Background.Set && n.Style.Background.Value == p.Accent || n.Style.Border.Set && n.Style.Border.Value.Color == p.Accent
		outline := hairline
		if selected {
			n.Style.Foreground = ui.Some(p.Accent)
			outline = p.Accent
		}
		n.Style.Border = ui.Some(ui.Border{Width: 1, Color: outline})
		n.Style.Background = ui.Some(p.Paper)
		n.Style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4, Left: 9, Right: 9})
		if n.ID == "simulation.pause" || n.ID == "simulation.resume" || n.ID == "native-stop-sweep" || strings.HasPrefix(n.ID, "tool-") && n.Accessibility.Help == "Selected drawing mode." {
			n.Style.Background, n.Style.Foreground = ui.Some(p.Accent), ui.Some(color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			n.Style.Border = ui.Some(ui.Border{})
			if n.ID == "simulation.pause" || n.ID == "simulation.resume" || n.ID == "native-stop-sweep" {
				n.Appearance = ui.ButtonPrimary
			}
		}
		if n.Icon != "" && n.Text == "" {
			n.Style.Padding = ui.Some(ui.UniformInsets(4))
		}
	}
	if n.ID == "tools" {
		minimum, maximum := n.Style.MinSize.Value, n.Style.MaxSize.Value
		minimum.Height, maximum.Height = 34, 34
		n.Style.MinSize = ui.Some(minimum)
		n.Style.MaxSize = ui.Some(maximum)
		n.Style.Padding = ui.Some(ui.UniformInsets(4))
		n.Style.Gap = ui.Some(float32(6))
	}
	if n.ID == "header" {
		for i := range n.Children {
			c := &n.Children[i]
			if c.Kind == ui.Button || c.Kind == ui.MenuButton {
				// Keep text actions as tall as the icon actions even when the
				// platform theme uses a more compact intrinsic text height.
				if c.Text != "" {
					minimum := c.Style.MinSize.Value
					minimum.Height = max(minimum.Height, 20)
					c.Style.MinSize = ui.Some(minimum)
				}
				c.Style.Border = ui.Some(ui.Border{})
				if c.ID != "simulation.pause" && c.ID != "simulation.resume" && c.ID != "native-stop-sweep" {
					c.Style.Background = ui.Some(p.Surface)
				}
			}
		}
	}

	if strings.HasPrefix(n.ID, "tool-") && n.Kind == ui.Button {
		n.MinWidth, n.MinHeight = 0, 0
		width := float32(24)
		if n.ID == "tool-wire" {
			width = 64 // icon plus a visible mode label
		}
		n.Style.MinSize = ui.Some(ui.Size{Width: width, Height: 24})
		n.Style.MaxSize = n.Style.MinSize
		n.Style.Padding = ui.Some(ui.UniformInsets(4))
		n.Style.Border = ui.Some(ui.Border{})
	}
	if n.ID == "native-zoom-in" || n.ID == "native-zoom-out" {
		n.MinWidth, n.MinHeight = 0, 0
		n.Style.MinSize, n.Style.MaxSize = ui.Some(ui.Size{Width: 16, Height: 16}), ui.Some(ui.Size{Width: 16, Height: 16})
		n.Style.Padding = ui.Some(ui.UniformInsets(2))
	}
	if n.ID == "schematic-help" {
		n.Style.MinSize = ui.Some(ui.Size{Height: 16})
		n.Style.MaxSize = ui.Some(ui.Size{Height: 16})
		n.Style.Typography = ui.Some(ui.Typography{Size: 11})
		n.Style.GrowY = ui.Some(float32(0))
	}
	if n.ID == "operation-notice-area" || n.ID == "operation-notice" {
		n.Style.Background = ui.Some(p.Paper)
		n.Style.CornerRadius = ui.Some(float32(12))
		if n.ID == "operation-notice-area" {
			n.Style.Padding = ui.Some(ui.UniformInsets(8))
			n.Style.Border = ui.Some(ui.Border{Width: 1, Color: hairline})
		}
	}
	if strings.HasPrefix(n.ID, "measurement-value-") {
		n.Style.MinSize = ui.Some(ui.Size{Width: n.Style.MaxSize.Value.Width, Height: 40})
		n.Style.MaxSize = ui.Some(ui.Size{Width: n.Style.MaxSize.Value.Width, Height: 40})
	}
	if n.ID == "inspector" && n.Kind == ui.Label {
		n.Style.Typography = ui.Some(ui.Typography{Size: 17, Weight: 600})
	}
	if n.ID == "workspace-details-measurement" {
		n.Text = "More measurements"
	}
	if n.ID == "workspace-details-pins" {
		n.Text = "Manage pinned readings"
	}
	nativeCompactForm(n, p)
	nativeSegmentedTabs(n, p)
	symbol := icons.None
	switch n.ID {
	case "tool-select":
		symbol = icons.ArrowSelectorTool
	case "tool-wire":
		symbol = icons.Cable
	case "tool-pan":
		symbol = icons.PanTool
	case "parts-toggle", "inspector-toggle":
		symbol = icons.DockToRight
	case "edit.undo":
		symbol = icons.Undo
	case "edit.redo":
		symbol = icons.Redo
	case "native-share":
		symbol = icons.IosShare
	case "examples-menu":
		if n.Text == "" {
			symbol = icons.FolderOpen
		}
	case "sweep-open":
		if n.Text == "" {
			symbol = icons.Timeline
		}
	}
	if symbol != icons.None {
		n.Icon = ""
		n.IconImage = nativeArtwork(symbol, n.Style.Foreground.Value)
	}
	if n.ID == "inspector-navigation" {
		n.Kind = ui.Toolbar
		n.Style.Background = ui.Some(nativeSegmentColor(p))
		n.Style.CornerRadius = ui.Some(float32(7))
		n.Style.Padding = ui.Some(ui.UniformInsets(2))
		n.Style.Gap = ui.Some(float32(0))
		n.Style.MinSize = ui.Some(ui.Size{Width: 252, Height: 24})
		n.Style.MaxSize = ui.Some(ui.Size{Width: 252, Height: 24})
	}
	if n.ID == "inspector-navigation-top-space" {
		n.Style.MinSize = ui.Some(ui.Size{Height: 10})
		n.Style.MaxSize = ui.Some(ui.Size{Height: 10})
	}

}

func nativeSegmentColor(p appPalette) color.NRGBA {
	if p.Paper.R < 80 {
		return color.NRGBA{R: 53, G: 53, B: 58, A: 255}
	}
	return nativeSegmentToken
}

// Label/editor rows reuse the original managed editors and their private drafts.
// Only allocation changes; the editor's origin and guarded callbacks are intact.
func nativeCompactForm(n *ui.Node, p appPalette) {
	if !strings.HasPrefix(n.ID, "inspector-field-") || len(n.Children) != 2 {
		return
	}
	width := n.Style.MaxSize.Value.Width
	if width < 180 {
		return
	}
	n.Kind = ui.Row
	n.Style.Background = ui.Some(p.Surface)
	n.Style.Gap = ui.Some(float32(8))
	n.Style.Align = ui.Some(ui.AlignCenter)
	caption := &n.Children[0]
	if strings.HasPrefix(caption.ID, railLabelBoundsPrefix) && len(caption.Children) == 1 {
		caption.MinWidth = 70
		caption.Style.MinSize = ui.Some(ui.Size{Width: 70})
		caption.Style.MaxSize = caption.Style.MinSize
		caption = &caption.Children[0]
	}
	caption.MinWidth = 70
	caption.Style.MinSize = ui.Some(ui.Size{Width: 70})
	caption.Style.MaxSize = ui.Some(ui.Size{Width: 70})
	caption.Style.Typography = ui.Some(ui.Typography{Size: 12})
	caption.Style.Foreground = ui.Some(p.Muted)
	editor := &n.Children[1]
	w := max(40, width-98)
	editor.MinWidth = 0
	editor.Style.MinSize = ui.Some(ui.Size{Width: w, Height: 20})
	editor.Style.MaxSize = editor.Style.MinSize
}

func nativeHairlineColor(p appPalette) color.NRGBA {
	if p.Paper.R < 80 {
		return color.NRGBA{R: 64, G: 64, B: 70, A: 255}
	}
	return color.NRGBA{R: 215, G: 215, B: 220, A: 255}
}
