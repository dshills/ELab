package main

import "github.com/dshills/vellum/ui"

func chromeStyle() ui.Style {
	return ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(6)), Align: ui.Some(ui.AlignCenter), Background: ui.Some(paper), Grow: ui.Some(float32(1)), MinSize: ui.Some(ui.Size{Width: 320, Height: 48}), MaxSize: ui.Some(ui.Size{Height: 48})}
}

func desktopTheme() ui.Theme {
	kinds := map[ui.Kind]ui.Style{}
	for _, kind := range []ui.Kind{ui.TextField, ui.NumberField, ui.MultilineText, ui.Selector, ui.MenuButton, ui.Button, ui.Disclosure, ui.Toolbar, ui.StatusBar, ui.FormGroup} {
		kinds[kind] = ui.Style{Foreground: ui.Some(ink), Background: ui.Some(surface)}
	}
	kinds[ui.Label] = ui.Style{Padding: ui.Some(ui.UniformInsets(0))}
	return ui.Theme{Default: ui.Style{Typography: ui.Some(ui.Typography{Size: 13}), Foreground: ui.Some(ink)}, Kinds: kinds}
}

// Menu choices reuse the mounted command buttons' enablement and callbacks,
// including file replacement checks and revision guarded history actions.
func (s *screen) commandMenu(id, text string, entries [][2]string) ui.Node {
	actions := make(map[string]ui.Node, len(entries))
	options := make([]ui.Option, 0, len(entries))
	for _, entry := range entries {
		action := s.button(entry[0], entry[1])
		actions[entry[0]] = action
		options = append(options, ui.Option{ID: entry[0], Label: entry[1], Disabled: action.Disabled})
	}
	node := buttonStyle(ui.Node{ID: id, Kind: ui.MenuButton, Text: text, Options: options, Accessibility: ui.Accessibility{Name: text}, Change: func(e ui.ChangeEvent) {
		if action, ok := actions[e.SelectedID]; ok && !action.Disabled {
			action.Activate()
		}
	}}, false)
	return node
}

func constrainRail(nodes []ui.Node, width float32) {
	for i := range nodes {
		node := &nodes[i]
		padding := float32(0)
		switch node.Kind {
		case ui.Button, ui.MenuButton, ui.Selector, ui.TextField, ui.NumberField:
			insets := ui.Insets{Top: 8, Bottom: 8, Left: 10, Right: 10}
			node.Style.Padding = ui.Some(insets)
			padding = insets.Left + insets.Right
		case ui.Label, ui.Row, ui.Column:
			node.Style.Padding = ui.Some(ui.UniformInsets(0))
		}
		maximum := ui.Size{}
		if node.Style.MaxSize.Set {
			maximum = node.Style.MaxSize.Value
		}
		maximum.Width = width - padding
		if node.Kind == ui.Label {
			maximum.Width -= 12
		}
		node.Style.MaxSize = ui.Some(maximum)
		if node.Style.MinSize.Set {
			minimum := node.Style.MinSize.Value
			minimum.Width = min(minimum.Width, maximum.Width)
			node.Style.MinSize = ui.Some(minimum)
		}
		node.MinWidth = min(node.MinWidth, maximum.Width)
		childWidth := maximum.Width
		if node.Kind == ui.Row && len(node.Children) > 0 {
			gap := float32(8)
			if node.Style.Gap.Set {
				gap = node.Style.Gap.Value
			}
			childWidth = max(24, (width-gap*float32(len(node.Children)-1))/float32(len(node.Children)))
		}
		constrainRail(node.Children, childWidth)
	}
}

// findAction lets the native smoke harness exercise menu entries through their
// real selection callback after actions move out of standalone buttons.
func findAction(root ui.Node, id string) (ui.Node, bool) {
	if node, ok := findNode(root, id); ok {
		return node, true
	}
	choices := menuChoices{Options: root.Options, Change: root.Change}
	if native, ok := root.Custom.(menuChoices); ok {
		choices = native
	}
	if choices.Change != nil {
		for _, option := range choices.Options {
			if option.ID == id {
				return ui.Node{ID: id, Kind: ui.Button, Disabled: root.Disabled || option.Disabled, Activate: func() { choices.Change(ui.ChangeEvent{ID: root.ID, Kind: ui.MenuButton, SelectedID: id}) }}, true
			}
		}
	}
	for _, child := range root.Children {
		if node, ok := findAction(child, id); ok {
			return node, true
		}
	}
	return ui.Node{}, false
}
