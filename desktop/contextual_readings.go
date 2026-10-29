package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// One experiment status; paused never hides an invalid circuit or stale data.
func simulationStatus(state workbench.Snapshot) string {
	status := "Complete the circuit"
	switch state.Status {
	case "running":
		status = "Live · DC simulation"
		if workbench.IsTransientAnalysis(state.Analysis) {
			status = "Live · " + state.Analysis
		}
	case "paused":
		status = "Paused"
	case "error":
		status = "Check circuit"
	case "unsupported":
		status = "Circuit not supported"
	case "updating":
		status = "Solving…"
	}
	if state.Paused && (state.Status == "running" || state.Status == "paused") {
		status = "Paused"
	}
	if state.Paused && state.Status == "updating" {
		status = "Initializing · paused"
	}
	if state.StepWork != nil && (state.StepWork.Status == "queued" || state.StepWork.Status == "running" || state.StepWork.Status == "canceling") {
		status = fmt.Sprintf("Stepping · %.0f%%", 100*state.StepWork.Progress)
	}
	if state.Result != nil && state.Result.Stale || state.NPN != nil && state.NPN.Stale || state.Mixed != nil && state.Mixed.Stale || state.Transient != nil && state.Transient.Stale {
		status += " · stale readings"
	}
	return status
}

func contextualReadings(state workbench.Snapshot, metric func(string, string, string) ui.Node, overview []ui.Node) []ui.Node {
	inspection := state.MeasurementInspection
	if inspection.Target != nil {
		nodes := []ui.Node{}
		for _, r := range inspection.Readings {
			if len(nodes) == 3 {
				break
			}
			// Reuse the accepted display, including signed units and stale/unavailable
			// qualifiers. No solver output is calculated by the presentation layer.
			_, value, ok := strings.Cut(r.Display, " · ")
			if !ok {
				value = r.Display
			}
			n := metric("context-reading-"+r.Probe.ID, r.Probe.Label, value)
			n.Accessibility.Name = r.Display
			n.Accessibility.Help = fmt.Sprintf("%s · %s · sample %.6g s · run %d · electrical revision %d", r.ReferenceDisplay, r.Reason, r.Time, r.RunID, r.ElectricalRevision)
			nodes = append(nodes, n)
		}
		if len(nodes) > 0 {
			return nodes
		}
		return []ui.Node{quiet("measurement-context-empty", inspection.Target.ID+" · "+inspection.Reason)}
	}
	if state.Result == nil && state.NPN == nil && state.Mixed == nil && state.Transient == nil {
		return []ui.Node{quiet("measurement-context-empty", "Select a component or wire to see its measurements.")}
	}
	return overview
}
