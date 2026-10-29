package main

import (
	"strings"

	"github.com/dshills/vellum/ui"
)

// Arrange existing labels and fields without changing leaf identities or their
// command/editing bindings. Run before constrainRail so wrapping frames inherit
// the same width policy as the rest of the inspector.
func prepareInspectorGroups(nodes []ui.Node, validation map[string]string) {
	for i := range nodes {
		n := &nodes[i]
		if strings.HasPrefix(n.ID, "inspector-group-") && len(n.Children) > 0 {
			n.Style.Gap = ui.Some(float32(8))
			first := n.Children[0]
			if first.Kind != ui.Label || first.Style.Typography.Value.Weight < 600 {
				title := section("inspector-heading-"+strings.TrimPrefix(n.ID, "inspector-group-"), n.Accessibility.Name)
				n.Children = append([]ui.Node{title}, n.Children...)
			}
		}
		prepareInspectorGroups(n.Children, validation)
		var children []ui.Node
		for j := 0; j < len(n.Children); j++ {
			child := n.Children[j]
			// The initial section heading stays separate. Existing field captions
			// immediately before editors form stable, tightly spaced rows.
			caption := strings.Contains(child.ID, "-label") || child.ID == "value-title" || child.ID == "label-title"
			if j > 0 && child.Kind == ui.Label && caption && j+1 < len(n.Children) && inspectorEditor(n.Children[j+1]) {
				field := n.Children[j+1]
				child.Style.Typography = ui.Some(ui.Typography{Size: 12, Weight: 500})
				child.Style.Foreground = ui.Some(ink)
				children = append(children, inspectorFormField(child, field, validation[field.ID]))
				j++
			} else {
				children = append(children, child)
			}
		}
		n.Children = children
	}
}

func inspectorFormField(caption, editor ui.Node, validation string) ui.Node {
	node, err := (ui.FormField{ID: "inspector-field-" + editor.ID, Label: caption.Text, Content: editor, ValidationMessage: validation, Style: ui.Style{
		Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(4)), Border: ui.Some(ui.Border{}), Background: ui.Some(paper),
	}}).View()
	if err != nil {
		panic(err) // The caller supplies a labeled managed editor.
	}
	// Retain visible label IDs and wrapping policy. FormField supplies the
	// editor metadata; the existing label replaces its private native caption.
	node.Text = ""
	node.Children = append([]ui.Node{caption}, node.Children...)
	if validation != "" {
		errorLabel := &node.Children[len(node.Children)-1]
		errorLabel.Style.Foreground = ui.Some(resolvePalette("light", false).Danger)
		errorLabel.Style.Typography = ui.Some(ui.Typography{Size: 12})
	}
	return node
}

func inspectorEditor(n ui.Node) bool {
	switch n.Kind {
	case ui.TextField, ui.NumberField, ui.Selector, ui.Slider:
		return true
	}
	return false
}

func polishInspector(node *ui.Node, p appPalette, inInspector bool) {
	if node.ID == "simulation-tool-navigation" || node.ID == "simulation-primary-actions" {
		inInspector = true
	}
	if node.ID == "sidebar-content" {
		inInspector = true
		node.Style.Gap = ui.Some(float32(20))
	}
	if inInspector {
		if node.Kind == ui.NumberField {
			node.NumberButtons = ui.NumberButtonsStepper
		}
		switch node.Kind {
		case ui.Label:
			typeface := node.Style.Typography.Value
			if typeface.Weight >= 600 {
				typeface.Size, typeface.Weight = 13, 600
				if node.Style.Foreground.Value != p.Danger {
					node.Style.Foreground = ui.Some(p.Ink)
				}
				node.Style.Typography = ui.Some(typeface)
			}
		case ui.Button, ui.MenuButton, ui.Selector, ui.TextField, ui.NumberField:
			field := inspectorEditor(*node)
			if choices, ok := node.Custom.(menuChoices); ok && choices.Kind == ui.Selector {
				field = true
			}
			destructive := node.Style.Foreground.Value == p.Danger
			if node.Kind == ui.Button || node.Kind == ui.MenuButton {
				polishToolbarButton(node, p)
			} else {
				node.Style.Foreground = ui.Some(p.Ink)
				node.Style.Border = ui.Some(ui.Border{Width: 1, Color: toolbarOutline(p)})
				node.Style.CornerRadius = ui.Some(p.ControlRadius)
			}
			if field {
				node.Style.Background = ui.Some(p.Surface)
				node.Style.TextAlign = ui.Some(ui.AlignStart)
			}
			if destructive {
				node.Style.Foreground = ui.Some(p.Danger)
			}
			node.Style.Padding = ui.Some(ui.Insets{Top: 6, Bottom: 6, Left: 10, Right: 10})
			minimum, maximum := node.Style.MinSize.Value, node.Style.MaxSize.Value
			// constrainRail reserves padding. Also reserve the one-unit outline
			// on each side, then give every control the full remaining field width.
			outlineWidth := float32(2)
			if node.Kind == ui.TextField || node.Kind == ui.NumberField {
				// Native filled editors draw an underline, not side borders.
				outlineWidth = 0
			}
			if maximum.Width > outlineWidth {
				maximum.Width -= outlineWidth
				minimum.Width = maximum.Width
			}
			minimum.Height, maximum.Height = 20, 20
			node.Style.MinSize, node.Style.MaxSize = ui.Some(minimum), ui.Some(maximum)
		case ui.Checkbox, ui.Switch:
			node.Style.Foreground, node.Style.Background = ui.Some(p.Ink), ui.Some(p.Paper)
			node.Style.Padding = ui.Some(ui.Insets{Top: 4, Bottom: 4})
		}
	}
	for i := range node.Children {
		polishInspector(&node.Children[i], p, inInspector)
	}
}
