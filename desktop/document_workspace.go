package main

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func documentToolbarAction(n ui.Node) ui.Node {
	symbols := map[string]ui.Symbol{"file.save": ui.SymbolSave, "edit.undo": ui.SymbolUndo, "edit.redo": ui.SymbolRedo, "theme-menu": ui.SymbolSun}
	if icon, ok := symbols[n.ID]; ok {
		n.Icon = icon
		if n.Text != "" && (n.ID == "edit.undo" || n.ID == "edit.redo") {
			n.Accessibility.Name = n.Text
		}
		// The accessible name retains the complete action and history actor.
		n.Text = ""
	}
	n.Appearance = ui.ButtonQuiet
	n.Style.CornerRadius = ui.Some(float32(5))
	return n
}

func documentName(state workbench.Snapshot) string {
	if state.Document.Path != "" {
		return filepath.Base(state.Document.Path)
	}
	if state.Example != nil && state.Example.Info.Title != "" {
		return state.Example.Info.Title
	}
	return "Untitled circuit"
}

func (s *screen) documentTitle(state workbench.Snapshot) ui.Node {
	title := heading("title", documentName(state))
	title.Style.Typography = ui.Some(ui.Typography{Size: 14, Weight: 600})
	title.Style.MaxSize = ui.Some(ui.Size{Width: 220, Height: 20})
	subtitle := "ELab"
	if state.Document.Dirty {
		subtitle = "Edited"
	}
	return ui.Node{ID: "document-title", Kind: ui.Column, Grow: true, Style: ui.Style{Gap: ui.Some(float32(1)), MinSize: ui.Some(ui.Size{Width: 160}), MaxSize: ui.Some(ui.Size{Height: 40})}, Children: []ui.Node{title, quiet("document-subtitle", subtitle)}}
}

// Group existing controls by the task they serve. Every control keeps its
// semantic command, revision guards and ID; page selection is shared with agents.
func (s *screen) organizeWorkspace(library, groups []ui.Node, state workbench.Snapshot) ([]ui.Node, []ui.Node) {
	var parts, document, help []ui.Node
	reachedOutput := false
	for _, n := range library {
		if n.ID == "print-preview" {
			reachedOutput = true
		}
		if reachedOutput {
			switch n.ID {
			case "print-preview", "export-svg", "export-pdf", "export-png":
				document = append(document, n)
			default:
				help = append(help, n)
			}
		} else {
			parts = append(parts, n)
		}
	}
	if previewVisible(state) || sweepVisible(state) {
		return library, groups
	}
	pages := map[string][]ui.Node{"component": {}, "simulation": {}, "document": {}}
	for _, group := range groups {
		if group.ID == "inspector-group-object" {
			var component, sampling []ui.Node
			for _, n := range group.Children {
				if strings.HasPrefix(n.ID, "source-sampling-") || n.ID == "fit-waveform-sampling" {
					sampling = append(sampling, n)
				} else {
					component = append(component, n)
				}
			}
			group.Children = component
			pages["simulation"] = append(pages["simulation"], inspectorGroup("source-sampling", "Source sampling", sampling)...)
		}
		section := "component"
		switch strings.TrimPrefix(group.ID, "inspector-group-") {
		case "guidance":
			// Errors belong to the current task. Moving to another page clears
			// request feedback, so an Inspect-only group is unreachable elsewhere.
			for page := range pages {
				pages[page] = append(pages[page], group)
			}
			continue
		case "circuit", "probes", "transient", "verification", "sweep":
			section = "simulation"
		case "appearance", "display", "output", "recovery", "project-work":
			section = "document"
		}
		pages[section] = append(pages[section], group)
	}
	pages["document"] = append(pages["document"], inspectorGroup("document-actions", "Share and print", document)...)
	pages["document"] = append(pages["document"], inspectorGroup("document-help", "Help and examples", help)...)
	priority := map[string]int{"source-sampling": 0, "transient": 1, "probes": 2, "verification": 3, "sweep": 4, "circuit": 5, "document-actions": 0, "output": 1, "appearance": 2, "display": 3, "recovery": 4, "project-work": 5, "document-help": 6}
	if state.InspectorSection != "component" {
		priority["guidance"] = -1
	}
	selected := pages[state.InspectorSection]
	slices.SortStableFunc(selected, func(a, b ui.Node) int {
		return priority[strings.TrimPrefix(a.ID, "inspector-group-")] - priority[strings.TrimPrefix(b.ID, "inspector-group-")]
	})
	return parts, selected
}

func (s *screen) inspectorRail(state workbench.Snapshot, groups []ui.Node) ui.Node {
	// Each document and section owns its scroll viewport. Navigation remains
	// visible outside the scroll area, with no nested native tab-page scroller.
	rail := ui.Node{ID: fmt.Sprintf("sidebar-%d-%s", state.Document.Generation, state.InspectorSection), Kind: ui.Scroll, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper)}, Children: []ui.Node{panel("sidebar-content", inspectorContentWidth, groups...)}}
	children := []ui.Node{rail}
	if !previewVisible(state) && !sweepVisible(state) {
		var actions []ui.Node
		for _, item := range []struct{ id, label string }{{"component", "Inspect"}, {"simulation", "Simulate"}, {"document", "Document"}} {
			action := s.action("inspector-section-"+item.id, item.label, "workspace.inspector.set", workbench.InspectorParams{Section: item.id})
			action = buttonStyle(action, state.InspectorSection == item.id)
			action.Accessibility.Name = item.label + " inspector"
			if state.InspectorSection == item.id {
				action.Accessibility.Help = "Selected inspector section."
			}
			actions = append(actions, inspectorTabCell(action, item.id, state.InspectorSection == item.id))
		}
		header := ui.Node{ID: "inspector-navigation", Kind: ui.Toolbar, Style: ui.Style{Background: ui.Some(paper), Padding: ui.Some(ui.Insets{Top: 14, Left: 16, Right: 16, Bottom: 2}), Gap: ui.Some(float32(0)), MinSize: ui.Some(ui.Size{Height: 34}), MaxSize: ui.Some(ui.Size{Height: 34})}, Accessibility: ui.Accessibility{Name: "Inspector section"}, Children: actions}
		children = append([]ui.Node{header}, children...)
	}
	return ui.Node{ID: "inspector-rail", Kind: ui.Column, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 296}), MaxSize: ui.Some(ui.Size{Width: 296}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0)), Background: ui.Some(paper)}, Children: children}
}

func compactReadingStatus(text string) ui.Node {
	return ui.Node{ID: "simulation-info", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.Insets{Top: 5, Bottom: 5, Left: 12, Right: 12}), Gap: ui.Some(float32(2)), Background: ui.Some(paper), MaxSize: ui.Some(ui.Size{Width: 232})}, Children: []ui.Node{section("simulation-title", "Live simulation"), quiet("reading-status", text)}}
}
