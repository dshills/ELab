package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
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
