package main

import (
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Reorder existing controls without replacing their field/action callbacks.
// Advanced settings remain controlled by the same service as ordinary commands.
func (s *screen) focusSimulationForm(state workbench.Snapshot, tool string, nodes []ui.Node) []ui.Node {
	var basic, advanced []ui.Node
	for _, n := range nodes {
		keep := false
		switch tool {
		case "recordings":
			switch n.ID {
			case "verification-title", "verification-help", "recording-import", "verify-label-duration", "verify-duration", "verify-start", "verify-operation", "verify-progress", "verify-cancel", "verify-source-changed", "verify-failure-summary", "verify-failure-action", "verify-failure-details", "verify-error", "verify-dc-help", "recording-show", "recording-keep", "verify-recorded", "verify-export-json", "verify-export-csv":
				keep = true
			}
		case "spectrum":
			switch n.ID {
			case "spectrum-title", "spectrum-help", "spectrum-operation", "spectrum-probe", "spectrum-analyze", "spectrum-summary", "spectrum-thd", "spectrum-thd-unavailable", "spectrum-plot", "spectrum-export-json", "spectrum-export-csv":
				keep = true
			}
		case "sweeps":
			switch n.ID {
			case "sweep-settings-title", "sweep-settings-help", "sweep-import", "sweep-target", "sweep-start-label", "sweep-range-start", "sweep-stop-label", "sweep-range-stop", "sweep-start", "sweep-record", "sweep-select", "sweep-show", "sweep-error", "sweep-keep", "sweep-export-json", "sweep-export-csv":
				keep = true
			}
			keep = keep || strings.HasPrefix(n.ID, "sweep-cancel-") || strings.HasPrefix(n.ID, "sweep-progress-") || strings.HasPrefix(n.ID, "noise-") || strings.HasPrefix(n.ID, "loop-")
		case "tolerances":
			switch n.ID {
			case "tolerance-title", "tolerance-help", "tolerance-mode", "tolerance-source-label", "tolerance-source", "tolerance-probe-label", "tolerance-probe", "tolerance-frequency-label", "tolerance-frequency", "tolerance-run", "tolerance-cancel", "tolerance-progress", "tolerance-failure", "tolerance-summary", "tolerance-statistics", "tolerance-export-json", "tolerance-export-csv":
				keep = true
			}
		case "live":
			switch n.ID {
			case "transient-title", "transient-help", "setting-label-step", "setting-step", "setting-label-speed", "setting-speed", "step-work-error":
				keep = true
			}
		}
		if keep {
			basic = append(basic, n)
		} else {
			advanced = append(advanced, n)
		}
	}
	if tool == "spectrum" {
		hasRecording := false
		for _, op := range state.Operations {
			hasRecording = hasRecording || op.Status == "completed"
		}
		if !hasRecording {
			// Preserve the original settings under Advanced, including import-independent
			// preferences. The initial view gives a useful next action instead of blanks.
			advanced = nodes
			basic = []ui.Node{section("spectrum-empty-title", "Recorded spectrum"), quiet("spectrum-prerequisite", "Record a run or open a saved recording to analyze its frequencies. Live Spectrum is available in the instrument dock."), s.action("spectrum-record-first", "Record a run…", "workspace.simulation.set", workbench.SimulationToolParams{Tool: "recordings"}), {ID: "spectrum-import-recording", Kind: ui.Button, Text: "Open recorded run…", Accessibility: ui.Accessibility{Name: "Open recording for spectrum"}, Disabled: s.fileBusy || s.pendingCommand != "", Activate: func() { s.beginFile("verification.import", true, workbench.RecordingImportParams{}, nil) }}}
		}
	}
	if tool == "recordings" || tool == "sweeps" || tool == "tolerances" {
		_, _, problem := (workbench.Graph{Components: state.Components, Wires: state.Wires}).Compile()
		if problem != nil {
			basic = append([]ui.Node{quiet(tool+"-prerequisite", "Complete and connect a supported circuit before running this analysis. Existing captured results and file import remain available.")}, basic...)
		}
	}
	if len(advanced) > 0 {
		basic = append(basic, s.detailsSection(state, tool, "Advanced", advanced))
	}
	return basic
}

// Remove a visible primary action from its form before giving it a fixed slot.
// Closed Advanced bodies keep their controls and editing ownership intact.
func takeVisibleControl(nodes *[]ui.Node, id string) (ui.Node, bool) {
	for i := range *nodes {
		n := &(*nodes)[i]
		if n.ID == id {
			out := *n
			*nodes = append((*nodes)[:i], (*nodes)[i+1:]...)
			return out, true
		}
		if n.Kind == ui.Disclosure && !n.Checked {
			continue
		}
		if out, ok := takeVisibleControl(&n.Children, id); ok {
			return out, true
		}
	}
	return ui.Node{}, false
}
