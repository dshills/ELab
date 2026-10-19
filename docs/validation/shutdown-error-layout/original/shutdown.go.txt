package main

import (
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) shutdownView(state workbench.Snapshot) ui.Node {
	work := state.Shutdown
	nodes := []ui.Node{heading("shutdown-title", "Closing ELab"), quiet("shutdown-help", "This window closes after recovery, settings and background work have finished.")}
	if work != nil {
		phase := map[string]string{"queued": "Preparing to close", "finalizing-recovery": "Finishing recovery snapshots", "joining-workers": "Waiting for background work to finish", "failed": "Review the cleanup errors before closing", "completed": "Ready to close"}[work.Phase]
		nodes = append(nodes, quiet("shutdown-phase", phase))
		if work.Status == "failed" {
			nodes = append(nodes, heading("shutdown-error-title", "Some shutdown work failed"), quiet("shutdown-recovery-error", work.RecoveryError), quiet("shutdown-appearance-error", work.AppearanceError), s.action("shutdown-acknowledge", "Close window anyway", "app.close.acknowledge", workbench.IDParams{ID: work.ID}))
		}
	}
	return ui.Node{ID: "shutdown", Kind: ui.Column, Grow: true, Style: ui.Style{Padding: ui.Some(ui.UniformInsets(24)), Gap: ui.Some(float32(16)), Background: ui.Some(paper), Foreground: ui.Some(ink)}, Children: nodes}
}
