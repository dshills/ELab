package main

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

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
	var parts, document, help, experiment []ui.Node
	// Output/sweep workspaces retain their own left-hand controls. The right
	// rail still selects one task instead of reverting to every inspector form.
	if previewVisible(state) {
		parts, library = library, nil
	}
	reachedOutput := false
	for _, n := range library {
		if strings.HasPrefix(n.ID, "example-") || n.ID == "example.current.reset" {
			experiment = append(experiment, n)
			continue
		}
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
		case "guidance", "proposal":
			// Errors belong to the current task. Moving to another page clears
			// request feedback, so an Inspect-only group is unreachable elsewhere.
			for page := range pages {
				pages[page] = append(pages[page], group)
			}
			continue
		case "circuit", "live-sampling", "probes", "transient", "verification", "sweep", "spectrum", "tolerance":
			section = "simulation"
		case "appearance", "display", "output", "recovery", "project-work":
			section = "document"
		}
		pages[section] = append(pages[section], group)
	}
	pages["document"] = append(pages["document"], inspectorGroup("blocks", "Circuit blocks", s.blockLibraryControls(state))...)
	pages["document"] = append(pages["document"], inspectorGroup("document-actions", "Share and print", document)...)
	pages["document"] = append(pages["document"], inspectorGroup("document-help", "Help and examples", append(help, experiment...))...)
	pages["simulation"] = append(pages["simulation"], inspectorGroup("experiment", "Experiment", experiment)...)
	priority := map[string]int{"source-sampling": 0, "transient": 1, "probes": 2, "verification": 3, "sweep": 4, "circuit": 5, "document-actions": 0, "output": 1, "appearance": 2, "display": 3, "recovery": 4, "project-work": 5, "blocks": 6, "document-help": 7}
	priority["proposal"] = -2
	priority["experiment"] = 3
	if state.InspectorSection != "component" {
		priority["guidance"] = -1
	}
	selected := pages[state.InspectorSection]
	if state.InspectorSection == "simulation" {
		filtered := make([]ui.Node, 0, len(selected))
		for _, group := range selected {
			if tool := simulationGroupTool(group.ID); tool == "" || tool == state.SimulationTool {
				filtered = append(filtered, group)
			}
		}
		selected = filtered
	}
	slices.SortStableFunc(selected, func(a, b ui.Node) int {
		return priority[strings.TrimPrefix(a.ID, "inspector-group-")] - priority[strings.TrimPrefix(b.ID, "inspector-group-")]
	})
	return parts, selected
}

func (s *screen) inspectorRail(state workbench.Snapshot, groups []ui.Node, width float32) ui.Node {
	var primary []ui.Node
	if state.InspectorSection == "simulation" {
		id := map[string]string{"recordings": "verify-start", "spectrum": "spectrum-analyze", "sweeps": "sweep-start", "tolerances": "tolerance-run"}[state.SimulationTool]
		if id != "" {
			if n, ok := takeVisibleControl(&groups, id); ok {
				primary = append(primary, buttonStyle(n, true))
			}
		}
	}
	// Each document and section owns its scroll viewport. Navigation remains
	// visible outside the scroll area, with no nested native tab-page scroller.
	rail := ui.Node{ID: inspectorScrollID(state), Kind: ui.Scroll, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper)}, Children: []ui.Node{panel("sidebar-content", width-40, groups...)}}
	children := []ui.Node{rail}
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
	// Keep vertical spacing outside the height-constrained toolbar. Native
	// sibling allocation otherwise lets the scrolling form cover its underline.
	inset := ui.Node{ID: "inspector-navigation-top-space", Kind: ui.Row, Style: ui.Style{MinSize: ui.Some(ui.Size{Height: 14}), MaxSize: ui.Some(ui.Size{Height: 14}), Padding: ui.Some(ui.UniformInsets(0)), Background: ui.Some(paper)}}
	header := ui.Node{ID: "inspector-navigation", Kind: ui.Toolbar, Style: ui.Style{Background: ui.Some(paper), Padding: ui.Some(ui.Insets{Left: 16, Right: 16}), Gap: ui.Some(float32(0)), MinSize: ui.Some(ui.Size{Height: 34}), MaxSize: ui.Some(ui.Size{Height: 34})}, Accessibility: ui.Accessibility{Name: "Inspector section"}, Children: actions}
	navigation := []ui.Node{inset, header}
	if state.InspectorSection == "simulation" {
		navigation = append(navigation, s.simulationToolChooser(state, width))
		if len(primary) > 0 {
			constrainRail(primary, width-32)
			navigation = append(navigation, ui.Node{ID: "simulation-primary-actions", Kind: ui.Column, Style: ui.Style{Padding: ui.Some(ui.Insets{Left: 16, Right: 16, Bottom: 8}), Gap: ui.Some(float32(0))}, Children: primary})
		}
	}
	children = append(navigation, children...)
	if state.InspectorSection == "simulation" && state.SimulationTool == "live" {
		step := s.action("native-simulate-step", "Step +1 sample", "simulation.step.start", workbench.StepParams{Duration: state.TransientSettings.StepSeconds})
		step.Disabled = !state.Paused || state.Transient == nil || state.Transient.Stale
		children = append(children, quiet("native-run-summary", nativeStepSummary(state)), ui.Node{ID: "native-simulate-actions", Kind: ui.Row, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(10)), Gap: ui.Some(float32(8))}, Children: []ui.Node{step, s.action("native-simulate-restart", "Restart", "simulation.restart", struct{}{})}})
	}

	return ui.Node{ID: "inspector-rail", Kind: ui.Column, Grow: true, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: width}), MaxSize: ui.Some(ui.Size{Width: width}), Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0)), Background: ui.Some(paper)}, Children: children}
}

func compactReadingStatus(text string) ui.Node {
	return ui.Node{ID: "simulation-info", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.Insets{Top: 5, Bottom: 5, Left: 12, Right: 12}), Gap: ui.Some(float32(2)), Background: ui.Some(paper), MinSize: ui.Some(ui.Size{Width: 232}), MaxSize: ui.Some(ui.Size{Width: 232})}, Children: []ui.Node{section("simulation-title", "Circuit overview"), quiet("reading-status", text)}}
}
