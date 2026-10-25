package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) componentPalette(state workbench.Snapshot) []ui.Node {
	options := make([]ui.Option, 0, len(state.MatchingParts))
	selected := ""
	for _, part := range state.MatchingParts {
		options = append(options, ui.Option{ID: part.Kind, Label: componentLabel(part.Kind, part.Name)})
		if state.Drawing.Tool == "place" && state.Drawing.Kind == part.Kind {
			selected = part.Kind
		}
	}
	categories := make([]ui.Option, 0, len(state.PartCategories))
	for _, category := range state.PartCategories {
		categories = append(categories, ui.Option{ID: category.ID, Label: fmt.Sprintf("%s · %d", category.Name, category.Count)})
	}
	help := fmt.Sprintf("%d matches · arrows choose · Tab to position", len(options))
	if len(options) == 1 {
		help = "1 match · Return chooses · Tab to position"
	}
	if len(options) == 0 {
		help = "No matches. Try a part name or alias."
	}
	if strings.TrimSpace(state.PartSearch) != "" {
		help += " · all categories"
	}
	search := s.componentSearch(state)
	return []ui.Node{
		section("parts-title", "Components"),
		{ID: "part-category", Kind: ui.Selector, Options: categories, SelectedID: state.PartCategory, Accessibility: ui.Accessibility{Name: "Component category", Help: "Common holds six everyday parts. Choosing a category clears search."}, Change: func(e ui.ChangeEvent) {
			s.invoke("parts.category.set", workbench.PartCategoryParams{Category: e.SelectedID})
		}},
		quiet("part-search-label", "Search all components"),
		search,
		{ID: "part-list", Kind: ui.ListBox, Options: options, SelectedID: selected, Disabled: len(options) == 0, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 120, Height: float32(min(320, max(1, len(options))*40+4))}), MaxSize: ui.Some(ui.Size{Height: 320})}, Accessibility: ui.Accessibility{Name: "Matching components", Help: "Use Up/Down, Home/End, page keys or an initial letter to choose a part. Return or Space chooses the focused row. Tab moves to exact placement."}, Change: func(e ui.ChangeEvent) {
			for _, option := range options {
				if option.ID == e.SelectedID {
					s.invoke("drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: e.SelectedID})
					return
				}
			}
			s.errorMessage = "Choose a component from the matching list."
		}},
		quiet("placement-help", help),
	}
}

func (s *screen) componentSearch(state workbench.Snapshot) ui.Node {
	search := &s.partSearch
	search.ID, search.Label, search.Text = "part-search-box", "Search components", state.PartSearch
	search.Help = "Search all categories by name or hobbyist alias. Return focuses the matching component list."
	search.Placeholder = "Name or alias"
	search.Style = ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(4)), Align: ui.Some(ui.AlignCenter)}
	search.Change = func(e ui.SearchEvent) {
		result := s.invoke("parts.search.set", workbench.SearchParams{Query: e.Text})
		if result.Error == nil {
			target := "part-search"
			if len(result.Result.(workbench.Snapshot).MatchingParts) > 0 {
				target = "part-list"
			}
			s.requestKeyboardFocus(target)
		}
	}
	node, err := search.View()
	if err != nil {
		panic(err) // Static component configuration; never derived from user input.
	}
	// Keep ELab's existing keyboard/automation target while using the public
	// SearchBox composition. These are ordinary leaf IDs, not new commands.
	node.Children[1].ID = "part-search"
	node.Children[2].ID = "part-search-clear"
	clear := node.Children[2].Activate
	context := actionContext{state.CommandRevision, state.Document.Generation}
	node.Children[2].Activate = func() {
		previous, previousAccepted := s.fieldContext, s.fieldAccepted
		s.fieldContext = &context
		s.fieldAccepted = nil
		defer func() { s.fieldContext, s.fieldAccepted = previous, previousAccepted }()
		clear()
		if s.fieldAccepted != nil {
			s.requestKeyboardFocus("part-search")
		}
	}
	return node
}

func componentLabel(kind, name string) string {
	if kind == "pulse-source" {
		return "Ideal square"
	}
	if kind == "finite-pulse-source" {
		return "Finite pulse"
	}
	return name
}

func componentName(kind string) string {
	for _, part := range workbench.Library() {
		if part.Kind == kind {
			return componentLabel(kind, part.Name)
		}
	}
	return kind
}
