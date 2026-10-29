package main

import (
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) detailsSection(state workbench.Snapshot, section, title string, children []ui.Node) ui.Node {
	return ui.Node{ID: "workspace-details-" + section, Kind: ui.Disclosure, Text: title, Checked: state.WorkspaceDetails.Expanded(section), Accessibility: ui.Accessibility{Name: title, Help: "Show or hide additional controls. Accepted settings and live simulation are preserved."}, Children: children, Change: func(e ui.ChangeEvent) {
		params := workbench.WorkspaceDetailsParams{Section: section, Visible: e.Checked}
		if (section == "component" || section == "selection") && state.Selected != "" {
			graph := workbench.Graph{Components: state.Components, Wires: state.Wires}
			if part, ok := graph.Component(state.Selected); ok && (part.Unit != "" || part.Kind == "net-label") {
				// The focus transfer may have just accepted this value field's
				// own draft. Reuse the same bounded continuation as Rotate/Plot;
				// never apply the component value or refresh an agent conflict.
				action := s.valueRelatedAction(state, "details-value-handoff", "Details", workbench.ValueParams{ID: part.ID}, true, "workspace.details.set", func(workbench.ValueParams) any { return params })
				action.Activate()
				return
			}
		}
		if section == "recordings" {
			s.verificationRelatedAction(state, false, &params).Activate()
			return
		}
		if section == "sweeps" && state.DCSweep != nil {
			s.sweepRelatedAction(state, &params).Activate()
			return
		}
		s.invokeAtCommandRevision("workspace.details.set", params, state.CommandRevision, state.Document.Generation)
	}}
}

// Reorganize existing descriptors, preserving identities, editing buffers and
// captured commands. All advanced fields remain behind one shared disclosure.
func (s *screen) compactComponentInspector(state workbench.Snapshot, part workbench.Component, nodes []ui.Node) []ui.Node {
	var basic, details, plots []ui.Node
	var rotate *ui.Node
	for _, n := range nodes {
		switch n.ID {
		case "inspector", "value-title", "label-title", "value-entry", "apply-value":
			basic = append(basic, n)
		case "rotate-part":
			v := n
			v.Accessibility.Name = "Rotate component 90 degrees"
			v.Text = "Rotate"
			rotate = &v
		default:
			if strings.HasPrefix(n.ID, "source-sampling-") || n.ID == "fit-waveform-sampling" {
				basic = append(basic, n)
				continue
			}
			if (strings.HasPrefix(n.ID, "plot-") && n.Kind == ui.Button) || strings.HasPrefix(n.ID, "opamp-plot-row-") {
				if len(plots) < 2 {
					plots = append(plots, n)
				} else {
					details = append(details, n)
				}
			} else if basicActuation(n.ID) {
				basic = append(basic, n)
			} else {
				details = append(details, n)
			}
		}
	}
	// Explanation readings are scoped to the selected part even when the general
	// measurement inspector is following an independently chosen terminal/probe.
	if x := state.Explanation; x != nil && x.ID == part.ID {
		reading := "No current reading"
		for _, r := range x.Readings {
			if r.Value != nil {
				reading = r.Display
				if state.Paused && r.State == "live" {
					reading = strings.Replace(reading, "Live ", "Paused ", 1)
				}
				break
			}
		}
		basic = append(basic, quiet("component-live-summary", reading))
		model := part.Model
		if part.Device != nil && part.Device.Passive != nil {
			model = workbench.PassiveLossModelID
		}
		summary := quiet("component-model-summary", "Model details")
		summary.Accessibility.Help = "Model: " + model + ". " + x.Limitations
		// A configured model must be visible without implying manufacturer identity.
		if part.Thermal != nil || part.Device != nil {
			summary.Text = "Configured educational model · Details below"
		} else {
			summary.Text = "Educational model · Details below"
		}
		basic = append(basic, summary)
	}
	if len(plots) == 2 && plots[0].Kind == ui.Button && plots[1].Kind == ui.Button {
		basic = append(basic, ui.Node{ID: "component-plot-actions", Kind: ui.Row, Children: plots})
	} else {
		basic = append(basic, plots...)
	}
	if rotate != nil {
		basic = append(basic, *rotate)
	}
	basic = append(basic, s.detailsSection(state, "component", "Details", details))
	return basic
}

func basicActuation(id string) bool {
	switch id {
	case "switch-title", "switch-state", "switch-actuate", "switch-actuation-help", "button-title", "button-state", "button-click", "button-hold", "sensor-title", "sensor-environment", "sensor-slider", "sensor-exact", "sensor-help", "potentiometer-title", "wiper-label", "pot-wiper", "pot-wiper-exact", "pot-actuation-help", "motor-load-title", "motor-load-value", "motor-load-slider", "motor-load-exact", "pwm-title", "pwm-frequency", "pwm-frequency-label", "pwm-duty", "pwm-duty-label", "battery-startup-title", "battery-startup-help", "battery-charge-label", "battery-charge-slider", "battery-charge-exact":
		return true
	}
	return false
}

// Smoke and electrical fixtures navigate the real disclosure before invoking
// an action. This helper never dispatches a hidden electrical action directly.
func (s *screen) reachableAction(id string) (ui.Node, bool) {
	for i := 0; i < 12; i++ {
		root := s.view()
		var closed func(ui.Node) (ui.Node, bool)
		closed = func(n ui.Node) (ui.Node, bool) {
			if n.Kind == ui.Disclosure && !n.Checked && strings.HasPrefix(n.ID, "workspace-details-") {
				if _, ok := findNode(n, id); ok && n.ID != id {
					return n, true
				}
			}
			for _, child := range n.Children {
				if found, ok := closed(child); ok {
					return found, true
				}
			}
			return ui.Node{}, false
		}
		if n, ok := closed(root); ok {
			n.Change(ui.ChangeEvent{Checked: true})
			if s.errorMessage != "" {
				return ui.Node{}, false
			}
			continue
		}
		return findAction(root, id)
	}
	return ui.Node{}, false
}

func compactObjectActions(nodes []ui.Node) []ui.Node {
	var rotate, remove ui.Node
	for _, n := range nodes {
		if n.ID == "rotate-part" {
			rotate = n
		}
		if n.ID == "delete-object" {
			remove = n
		}
	}
	if rotate.ID == "" || remove.ID == "" {
		return nodes
	}
	remove.Text = "Delete"
	remove.Accessibility.Name = "Delete selected object"
	var out []ui.Node
	for _, n := range nodes {
		if n.ID == "rotate-part" {
			out = append(out, ui.Node{ID: "object-main-actions", Kind: ui.Row, Children: []ui.Node{rotate, remove}})
		} else if n.ID != "delete-object" {
			out = append(out, n)
		}
	}
	return out
}
