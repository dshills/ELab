package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func activeInputWork(state workbench.Snapshot) bool {
	return state.InputWork != nil && (state.InputWork.Status == "queued" || state.InputWork.Status == "running" || state.InputWork.Status == "canceling")
}
func (s *screen) inputWorkControls(state workbench.Snapshot) []ui.Node {
	if state.InputWork == nil {
		return nil
	}
	w := state.InputWork
	text := fmt.Sprintf("%s · %s · accepted time %.5g ms", w.Command, w.Status, w.Time*1000)
	if w.Wiper != nil {
		text += fmt.Sprintf("\nRequested %s wiper %.4g%%", w.Wiper.ID, w.Wiper.Position*100)
	}
	if w.Mode != "" {
		text += "\nRequested input " + w.Mode
	}
	title := "Input calculation"
	if w.RunID != state.RunID || w.ElectricalRevision != state.ElectricalRevision {
		title = "Previous input calculation"
	}
	nodes := []ui.Node{section("input-work-title", title), quiet("input-work-status", text)}
	if activeInputWork(state) {
		nodes = append(nodes, quiet("input-work-committed", "Graph/readings remain committed until this calculation succeeds. Save captures the committed circuit."), s.action("input-work-cancel", "Cancel applying input", "simulation.work.cancel", workbench.IDParams{ID: w.ID}))
	}
	if w.Error != nil {
		nodes = append(nodes, quiet("input-work-error", w.Error.Message))
	}
	return nodes
}
