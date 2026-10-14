package main

import (
	"fmt"
	"path/filepath"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) projectWorkControls(state workbench.Snapshot) []ui.Node {
	w := state.ProjectWork
	if w == nil {
		return nil
	}
	title := "Project open"
	if w.ResultGeneration != 0 && w.ResultGeneration != state.Document.Generation || w.ResultGeneration == 0 && w.DocumentGeneration != state.Document.Generation {
		title = "Previous project open"
	}
	text := fmt.Sprintf("%s · %s · %s\n%d bytes read", filepath.Base(w.Path), w.Status, w.Phase, w.Bytes)
	nodes := []ui.Node{section("project-work-title", title), quiet("project-work-status", text)}
	if w.Status == "queued" || w.Status == "running" || w.Status == "canceling" {
		nodes = append(nodes, quiet("project-work-explanation", "The current circuit stays editable until the file is validated. Edits supersede this open request."), s.action("project-work-cancel", "Cancel opening project", "project.work.cancel", workbench.IDParams{ID: w.ID}))
	}
	if w.Error != nil {
		nodes = append(nodes, quiet("project-work-error", w.Error.Message))
	}
	return nodes
}
