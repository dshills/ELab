package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/command"
	"github.com/dshills/vellum/ui"
)

const exampleBrowserID = "examples-browser"

func (s *screen) registerExampleBrowserCommand() {
	if err := s.commands.Register(command.Command{ID: "examples.browse", Label: "Browse examples…", Available: func() error {
		if s.fileBusy || s.pendingCommand != "" || s.service.Snapshot().Closed {
			return fmt.Errorf("finish the current operation before browsing examples")
		}
		return nil
	}, Execute: func(context.Context) error {
		r := s.invoke("examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
		if r.Error != nil {
			return fmt.Errorf("%s", r.Error.Message)
		}
		s.requestKeyboardFocus("example-search")
		return nil
	}}); err != nil {
		panic(err)
	}
}

func (s *screen) examplesButton(state workbench.Snapshot) ui.Node {
	n := s.focusAfterAction(s.action("examples-menu", "Examples", "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true}), "example-search")
	n.Disabled = s.fileBusy || s.pendingCommand != "" || state.Closed
	n.Accessibility.Help = "Browse existing circuits by topic or search, with schematic previews and experiments."
	return n
}

// Collection/Sheet expose callbacks outside Node.Change. Bind their detached
// publication to its original command context, continuing only its own accepted
// actions. A newer agent edit must not lend an old callback a fresh revision.
func (s *screen) withExampleContext(origin *actionContext, action func()) {
	previous, accepted := s.fieldContext, s.fieldAccepted
	s.fieldContext, s.fieldAccepted = origin, nil
	defer func() { s.fieldContext, s.fieldAccepted = previous, accepted }()
	action()
	if s.fieldAccepted != nil {
		*origin = *s.fieldAccepted
	}
}

func (s *screen) openBrowserExample(id string) {
	generation := s.service.Snapshot().Document.Generation
	if s.fieldContext != nil {
		generation = s.fieldContext.generation
	}
	r := s.requestReplace("example.load", workbench.ExampleLoadParams{ID: id, ExpectedGeneration: &generation})
	if r.Error == nil {
		s.requestKeyboardFocus("part-search")
	}
}

func (s *screen) exampleSearchNode(state workbench.Snapshot) ui.Node {
	f := &s.exampleSearch
	f.ID, f.Label, f.Text = "example-search-box", "Search examples", state.ExampleBrowser.Query
	f.Placeholder = "Search all examples"
	f.Help = "Search all examples. Return shows matching circuits; arrow keys choose and Return opens."
	// Bound both the row and its editor before text shaping; a growing row
	// alone leaves the native editor at its narrow intrinsic word width.
	f.Style = ui.Style{MinSize: ui.Some(ui.Size{Width: 508}), MaxSize: ui.Some(ui.Size{Width: 508}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(6)), Align: ui.Some(ui.AlignCenter)}
	f.InputStyle = ui.Style{MinSize: ui.Some(ui.Size{Width: 440, Height: 20}), MaxSize: ui.Some(ui.Size{Width: 440, Height: 20}), Padding: ui.Some(ui.UniformInsets(6))}
	f.ClearStyle = ui.Style{MinSize: ui.Some(ui.Size{Width: 24, Height: 28}), MaxSize: ui.Some(ui.Size{Width: 24, Height: 28}), Padding: ui.Some(ui.UniformInsets(0)), Border: ui.Some(ui.Border{})}
	f.Change = func(e ui.SearchEvent) {
		r := s.invoke("examples.search.set", workbench.SearchParams{Query: e.Text})
		if r.Error == nil {
			target := "example-search"
			if len(r.Result.(workbench.Snapshot).ExampleBrowser.MatchingIDs) > 0 {
				target = "example-grid"
			}
			s.requestKeyboardFocus(target)
		}
	}
	node, err := f.View()
	if err != nil {
		panic(err)
	}
	node.Children[1].ID, node.Children[2].ID = "example-search", "example-search-clear"
	origin := actionContext{state.CommandRevision, state.Document.Generation}
	clear := node.Children[2].Activate
	node.Children[2].Activate = func() {
		s.withExampleContext(&origin, func() {
			clear()
			if s.fieldAccepted != nil {
				s.requestKeyboardFocus("example-search")
			}
		})
	}
	return node
}

func (s *screen) exampleCollection(state workbench.Snapshot) ui.Node {
	b := state.ExampleBrowser
	info := make(map[string]workbench.ExampleInfo)
	for _, e := range workbench.ExampleCatalog() {
		info[e.ID] = e
	}
	f := &s.exampleGrid
	f.ID, f.Label, f.ItemWidth = "example-grid", "Matching circuit examples", 184
	f.Items, f.SelectedIDs = nil, nil
	for _, id := range b.MatchingIDs {
		f.Items = append(f.Items, ui.CollectionItem{ID: id, Label: info[id].Title, Icon: ui.SymbolDocument, Thumbnail: s.exampleThumbnails.images[id]})
	}
	if b.SelectedID != "" {
		f.SelectedIDs = []string{b.SelectedID}
	}
	f.Style = ui.Style{MinSize: ui.Some(ui.Size{Height: 240}), MaxSize: ui.Some(ui.Size{Height: 240}), GrowX: ui.Some(float32(1)), Background: ui.Some(surface), Border: ui.Some(ui.Border{Width: 1, Color: surface}), CornerRadius: ui.Some(float32(8))}
	f.Change = func(e ui.ListSelectionEvent) { s.invoke("examples.select", workbench.IDParams{ID: e.TargetID}) }
	f.Activate = func(e ui.CollectionEvent) { s.openBrowserExample(e.ItemID) }
	node, err := f.View()
	if err != nil {
		panic(err)
	}
	node.Accessibility.Help = "Choose one example with arrow keys or a click. Return or double-click opens it; unsaved circuit changes remain protected."
	origin := actionContext{state.CommandRevision, state.Document.Generation}
	selection, activate := node.SelectionChange, node.Collection.Activate
	node.SelectionChange = func(e ui.ListSelectionEvent) { s.withExampleContext(&origin, func() { selection(e) }) }
	node.Collection.Activate = func(e ui.CollectionEvent) { s.withExampleContext(&origin, func() { activate(e) }) }
	return node
}

func (s *screen) exampleDetails(state workbench.Snapshot, width float32) ui.Node {
	b := state.ExampleBrowser
	for _, info := range workbench.ExampleCatalog() {
		if info.ID != b.SelectedID {
			continue
		}
		textWidth := width - 260
		labels := []ui.Node{heading("example-preview-title", info.Title), label("example-preview-expected", info.Expected), quiet("example-preview-try", "Try: "+info.Try), quiet("example-preview-limits", "Model: "+info.Limitation)}
		constrainRail(labels, textWidth)
		image := ui.Node{ID: "example-preview-image", Kind: ui.Image, Image: s.exampleThumbnails.images[info.ID], Fit: ui.ImageContain, Accessibility: ui.Accessibility{Name: info.Title + " schematic preview"}, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 224, Height: 148}), MaxSize: ui.Some(ui.Size{Width: 224, Height: 148}), Background: ui.Some(paper)}}
		if image.Image == nil {
			image.Kind, image.Text = ui.Label, "Preparing schematic…"
			image.Fit = ""
			if s.exampleThumbnails.errors[info.ID] != "" {
				image.Text = "Schematic preview unavailable"
			}
		}
		return ui.Node{ID: "example-preview", Kind: ui.Row, Style: ui.Style{Gap: ui.Some(float32(16)), Align: ui.Some(ui.AlignStart)}, Children: []ui.Node{image, {ID: "example-description-scroll", Kind: ui.Scroll, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: textWidth, Height: 148}), MaxSize: ui.Some(ui.Size{Width: textWidth, Height: 148}), Padding: ui.Some(ui.UniformInsets(0))}, Children: []ui.Node{{ID: "example-description", Kind: ui.Column, Style: ui.Style{Gap: ui.Some(float32(8)), Padding: ui.Some(ui.UniformInsets(0))}, Children: labels}}}}}
	}
	return ui.Node{ID: "example-empty-preview", Kind: ui.Column, Style: ui.Style{MinSize: ui.Some(ui.Size{Height: 148}), MaxSize: ui.Some(ui.Size{Height: 148})}, Children: []ui.Node{quiet("example-no-results", "No matching examples. Try another circuit or topic, or clear the search.")}}
}

func (s *screen) exampleBrowserView(state workbench.Snapshot) ui.Node {
	b := state.ExampleBrowser
	visible := b.Visible && !s.fileBusy && s.pendingCommand == ""
	sheet := &s.exampleSheet
	sheet.ID, sheet.Title, sheet.Open = exampleBrowserID, "Circuit examples", visible
	sheet.ConfirmLabel, sheet.CancelLabel, sheet.ReturnFocusID = "Open example", "Close", "examples-menu"
	sheet.ConfirmDisabled = b.SelectedID == ""
	sheet.Confirm = func() { s.openBrowserExample(b.SelectedID) }
	sheet.Cancel = func() { s.invoke("examples.visibility.set", workbench.ScopeVisibilityParams{Visible: false}) }
	sheet.Children = nil
	width := float32(760)
	sheet.Style = ui.Style{MinSize: ui.Some(ui.Size{Width: width}), MaxSize: ui.Some(ui.Size{Width: width}), Padding: ui.Some(ui.UniformInsets(16)), Gap: ui.Some(float32(10)), Background: ui.Some(paper)}
	if visible {
		s.requestExampleThumbnails()
		categories := []ui.Option{}
		for _, c := range b.Categories {
			categories = append(categories, ui.Option{ID: c.ID, Label: fmt.Sprintf("%s · %d", c.Name, c.Count)})
		}
		filter := ui.Node{ID: "example-category", Kind: ui.Selector, Options: categories, SelectedID: b.Category, Accessibility: ui.Accessibility{Name: "Example category", Help: "Choosing a category clears search. Search covers all categories."}, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 208}), MaxSize: ui.Some(ui.Size{Width: 208})}, Change: func(e ui.ChangeEvent) {
			s.invoke("examples.category.set", workbench.PartCategoryParams{Category: e.SelectedID})
		}}
		noun := "examples"
		if len(b.MatchingIDs) == 1 {
			noun = "example"
		}
		count := fmt.Sprintf("%d %s · arrows choose · Return opens", len(b.MatchingIDs), noun)
		if strings.TrimSpace(b.Query) != "" {
			count += " · searching all categories"
		}
		sheet.Children = []ui.Node{quiet("example-browser-help", "Choose a circuit, explore what it does, and make it your own."), {ID: "example-filters", Kind: ui.Row, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(12)), Align: ui.Some(ui.AlignCenter)}, Children: []ui.Node{s.exampleSearchNode(state), filter}}, quiet("example-results-count", count), s.exampleCollection(state), s.exampleDetails(state, width-32)}
	}
	node, err := sheet.View()
	if err != nil {
		panic(err)
	}
	origin := actionContext{state.CommandRevision, state.Document.Generation}
	dismiss := node.Dismiss
	node.Dismiss = func() { s.withExampleContext(&origin, dismiss) }
	var bindActions func(*ui.Node)
	bindActions = func(n *ui.Node) {
		if n.ID == ui.SheetConfirmID(exampleBrowserID) || n.ID == ui.SheetCancelID(exampleBrowserID) {
			selected := n.ID == ui.SheetConfirmID(exampleBrowserID)
			*n = buttonStyle(*n, selected)
			action := n.Activate
			n.Activate = func() { s.withExampleContext(&origin, action) }
		}
		for i := range n.Children {
			bindActions(&n.Children[i])
		}
	}
	bindActions(&node)
	return node
}
