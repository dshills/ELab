package main

import (
	"fmt"
	"math"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/units"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// The pinned native Handle reserves a six-unit grip and six-unit margins on
// each side. Include all eighteen units when converting widths to split shares.
const sideDividerWidth float32 = 18
const minimumDrawingWidth float32 = 400

// Native geometry is sampled only while constructing the owner view. Prefer
// the existing workspace's actual allocation; initial/unmounted views use the
// normal window width. Widths are logical units, independent of display density.
func (s *screen) workspaceWidth() float32 {
	if s.drawing != nil && s.drawing.widget != nil {
		for p := s.drawing.widget.Parent; p != nil; p = p.AsTree().Parent {
			if p.AsTree().Name != "workspace" {
				continue
			}
			if widget, ok := p.(core.Widget); ok {
				w := widget.AsWidget()
				scale := w.Styles.UnitContext.DPI / units.DpPerInch
				if scale > 0 && w.Geom.Size.Alloc.Content.X > 0 {
					return w.Geom.Size.Alloc.Content.X / scale
				}
			}
		}
	}
	return minimumWindowWidthDP - 16
}

// Clamp the displayed widths, not the stored preference. Re-expanding a panel
// or enlarging a window restores its preferred size without a service mutation.
func sidePanelWidths(total float32, p workbench.WorkspacePanels) (float32, float32) {
	left, right := float32(0), float32(0)
	if p.PartsVisible {
		left = float32(p.PartsWidth)
	}
	if p.InspectorVisible {
		right = float32(p.InspectorWidth)
	}
	available := max(float32(0), sidePanelBudget(total, p)) // drawing plus both divider grips
	if excess := left + right - available; excess > 0 {
		delta := min(excess, max(float32(0), right-workbench.MinInspectorPanelWidth))
		right -= delta
		excess -= delta
		delta = min(excess, max(float32(0), left-workbench.MinPartsPanelWidth))
		left -= delta
	}
	return left, right
}

func (s *screen) panelToggle(state workbench.Snapshot, inspector bool) ui.Node {
	p := state.WorkspacePanels
	id, text, visible := "parts-toggle", "Parts", p.PartsVisible
	if previewVisible(state) {
		text = "Print"
	} else if sweepVisible(state) {
		text = "Sweep"
	}
	if inspector {
		id, text, visible = "inspector-toggle", "Inspector", p.InspectorVisible
		p.InspectorVisible = !visible
	} else {
		p.PartsVisible = !visible
	}
	action := s.action(id, text, "workspace.panels.set", p)
	action = buttonStyle(action, visible)
	verb := "Show"
	if visible {
		verb = "Hide"
	}
	action.Accessibility.Name = verb + " " + text + " panel"
	action.Accessibility.Help = verb + " the side panel. Drag its divider to resize; use arrow keys when the divider is focused. The circuit stays unchanged."
	action.Activate = func() {
		response := s.invokeAtCommandRevision("workspace.panels.set", p, state.CommandRevision, state.Document.Generation)
		if response.Error == nil {
			target := id
			if !visible {
				if inspector {
					target = "inspector-section-" + state.InspectorSection
				} else {
					target = "part-search"
					if previewVisible(state) {
						target = "printer-destination"
					} else if sweepVisible(state) {
						target = "sweep-target"
					}
				}
			}
			s.requestKeyboardFocus(target)
		}
	}
	return action
}

func (s *screen) sidePanels(state workbench.Snapshot, parts, sheet, inspector ui.Node, total, left, right float32) ui.Node {
	// Reserve the native grip including its hit margins.
	// Horizontal widths are capped before deriving shares, preserving a useful sheet.
	content := sheet
	remaining := total
	if state.WorkspacePanels.PartsVisible {
		remaining -= left + sideDividerWidth
	}
	if state.WorkspacePanels.InspectorVisible {
		span := max(float32(1), remaining-sideDividerWidth)
		fraction := 1 - right/span
		content = ui.Node{ID: "inspector-split", Kind: ui.SplitPane, Grow: true, Number: fraction,
			Numeric:       panelSplitRange(1-panelMaximumWidth(total, state.WorkspacePanels, true)/span, 1-workbench.MinInspectorPanelWidth/span, span),
			Accessibility: ui.Accessibility{Name: "Inspector width divider", Help: "Drag to resize the inspector. Left/Right changes width; Home/End selects its limits."},
			Children:      []ui.Node{sheet, inspector}, Change: func(e ui.ChangeEvent) {
				p := state.WorkspacePanels
				p.InspectorWidth = math.Round(float64((1 - e.Number) * span))
				p.InspectorWidth = max(workbench.MinInspectorPanelWidth, min(workbench.MaxInspectorPanelWidth, p.InspectorWidth))
				if p.PartsVisible {
					p.PartsWidth = max(workbench.MinPartsPanelWidth, min(float64(left), float64(sidePanelBudget(total, p))-p.InspectorWidth))
				}
				s.invokeAtCommandRevision("workspace.panels.set", p, state.CommandRevision, state.Document.Generation)
			}}
	}
	if state.WorkspacePanels.PartsVisible {
		span := max(float32(1), total-sideDividerWidth)
		maxWidth := panelMaximumWidth(total, state.WorkspacePanels, false)
		content = ui.Node{ID: "parts-split", Kind: ui.SplitPane, Grow: true, Number: left / span,
			Numeric:       panelSplitRange(workbench.MinPartsPanelWidth/span, max(float32(workbench.MinPartsPanelWidth), maxWidth)/span, span),
			Accessibility: ui.Accessibility{Name: "Parts width divider", Help: "Drag to resize the parts panel. Left/Right changes width; Home/End selects its limits."},
			Children:      []ui.Node{parts, content}, Change: func(e ui.ChangeEvent) {
				p := state.WorkspacePanels
				p.PartsWidth = math.Round(float64(e.Number * span))
				p.PartsWidth = max(workbench.MinPartsPanelWidth, min(workbench.MaxPartsPanelWidth, p.PartsWidth))
				if p.InspectorVisible {
					p.InspectorWidth = max(workbench.MinInspectorPanelWidth, min(float64(right), float64(sidePanelBudget(total, p))-p.PartsWidth))
				}
				s.invokeAtCommandRevision("workspace.panels.set", p, state.CommandRevision, state.Document.Generation)
			}}
	}
	return ui.Node{ID: "workspace", Kind: ui.Row, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(0)), Gap: ui.Some(float32(0))}, Children: []ui.Node{content}}
}

func sidePanelBudget(total float32, p workbench.WorkspacePanels) float32 {
	budget := total - minimumDrawingWidth
	if p.PartsVisible {
		budget -= sideDividerWidth
	}
	if p.InspectorVisible {
		budget -= sideDividerWidth
	}
	return budget
}

// At a narrow window, dragging one edge may shrink the other visible panel.
// Its minimum, rather than its current width, determines the usable drag range.
func panelMaximumWidth(total float32, p workbench.WorkspacePanels, inspector bool) float32 {
	budget := sidePanelBudget(total, p)
	if inspector {
		if p.PartsVisible {
			budget -= workbench.MinPartsPanelWidth
		}
		return min(workbench.MaxInspectorPanelWidth, budget)
	}
	if p.InspectorVisible {
		budget -= workbench.MinInspectorPanelWidth
	}
	return min(workbench.MaxPartsPanelWidth, budget)
}

// Numeric requires a step no larger than its range. At the narrowest window,
// the inspector may have less than a default two-percent share left to move.
func panelSplitRange(low, high, span float32) ui.Numeric {
	return ui.Numeric{Min: low, Max: high, Step: min(16/span, high-low)}
}

func simulationTools() []ui.Option {
	return []ui.Option{{ID: "live", Label: "Live controls"}, {ID: "recordings", Label: "Recorded runs"}, {ID: "spectrum", Label: "Recorded spectrum"}, {ID: "sweeps", Label: "DC / AC / noise sweeps"}, {ID: "tolerances", Label: "Component tolerances"}, {ID: "circuit", Label: "Circuit details"}}
}

func (s *screen) simulationToolChooser(state workbench.Snapshot, width float32) ui.Node {
	n := ui.Node{ID: "simulation-tool", Kind: ui.Selector, SelectedID: state.SimulationTool, Options: simulationTools(),
		Accessibility: ui.Accessibility{Name: "Simulate tool", Help: "Show one tool's settings. Other tools keep their settings and results; live simulation continues."},
		Change: func(e ui.ChangeEvent) {
			s.invokeAtCommandRevision("workspace.simulation.set", workbench.SimulationToolParams{Tool: e.SelectedID}, state.CommandRevision, state.Document.Generation)
		}}
	nodes := []ui.Node{n}
	constrainRail(nodes, width-32)
	return ui.Node{ID: "simulation-tool-navigation", Kind: ui.Column, Style: ui.Style{Padding: ui.Some(ui.Insets{Left: 16, Right: 16, Top: 6, Bottom: 6}), Gap: ui.Some(float32(0))}, Children: nodes}
}

func simulationGroupTool(id string) string {
	switch id {
	case "inspector-group-live-sampling", "inspector-group-source-sampling", "inspector-group-transient", "inspector-group-probes", "inspector-group-experiment":
		return "live"
	case "inspector-group-verification":
		return "recordings"
	case "inspector-group-spectrum":
		return "spectrum"
	case "inspector-group-sweep":
		return "sweeps"
	case "inspector-group-tolerance":
		return "tolerances"
	case "inspector-group-circuit":
		return "circuit"
	}
	return ""
}

func inspectorScrollID(state workbench.Snapshot) string {
	id := fmt.Sprintf("sidebar-%d-%s", state.Document.Generation, state.InspectorSection)
	if state.InspectorSection == "simulation" {
		id += "-" + state.SimulationTool
	}
	return id
}
