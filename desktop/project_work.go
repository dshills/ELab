package main

import (
	"fmt"
	"path/filepath"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func activeProjectWork(state workbench.Snapshot) bool {
	return state.ProjectWork != nil && (state.ProjectWork.Status == "queued" || state.ProjectWork.Status == "running" || state.ProjectWork.Status == "canceling")
}

func (s *screen) projectWorkControls(state workbench.Snapshot) []ui.Node {
	w := state.ProjectWork
	if w == nil {
		return nil
	}
	action := "open"
	if w.Command == "project.save" {
		action = "save"
	} else if w.Command == "verification.import" || w.Command == "sweep.import" {
		action = "record import"
	} else if w.Command == "recovery.dismiss" || w.Command == "recovery.configure" || w.Command == "recovery.scan" || w.Command == "recovery.checkpoint" || w.Command == "recovery.restore" {
		action = "recovery"
	} else if w.Command == "export.schematic.svg" || w.Command == "export.scope.csv" || w.Command == "verification.export" {
		action = "export"
	}
	title := "Project " + action
	if w.ResultGeneration != 0 && w.ResultGeneration != state.Document.Generation || w.ResultGeneration == 0 && w.DocumentGeneration != state.Document.Generation {
		title = "Previous project " + action
	}
	if w.Command == "recovery.configure" {
		title = "Recovery setup"
	}
	text := fmt.Sprintf("%s · %s · %s\n%d bytes prepared", filepath.Base(w.Path), w.Status, w.Phase, w.Bytes)
	nodes := []ui.Node{section("project-work-title", title), quiet("project-work-status", text)}
	if w.Status == "queued" || w.Status == "running" || w.Status == "canceling" {
		explanation := "The current circuit stays editable until the file is validated. Edits supersede this open request."
		if action == "save" {
			explanation = "Saving the captured circuit. Later edits remain unsaved. Cancellation is available before publication."
		}
		if action == "export" {
			explanation = "Exporting captured drawing or readings. Editing stays available; cancellation ends when publication begins."
		}
		if action == "record import" {
			explanation = "Validating recorded data. Live simulation stays separate; later viewer choices are preserved."
		}
		if w.Command == "recovery.configure" {
			explanation = "Preparing recovery settings. Existing recovery stays active until ready; drawing and simulation stay available."
		} else if w.Command == "recovery.dismiss" {
			explanation = "Validating the exact recovery snapshot. Cancellation is available until removal starts; saved schematics stay unchanged."
		} else if action == "recovery" {
			explanation = "Recovery work runs separately from editing. Checkpoint cancellation ends when publication begins. Saved project files are preserved."
		}
		if w.Command == "recovery.restore" {
			explanation = "Validating a recovery copy. Edits supersede this restore request; saved project files are preserved."
		}
		cancel := s.action("project-work-cancel", "Cancel project "+action, "project.work.cancel", workbench.IDParams{ID: w.ID})
		cancel.Disabled = !w.Cancelable
		nodes = append(nodes, quiet("project-work-explanation", explanation), cancel)
	}
	if w.Error != nil {
		nodes = append(nodes, quiet("project-work-error", w.Error.Message))
	}
	if w.CleanupWarning != "" {
		nodes = append(nodes, quiet("project-work-cleanup-warning", w.CleanupWarning))
	}
	if w.Associated != nil && !*w.Associated {
		nodes = append(nodes, quiet("project-work-unassociated", "Published the captured file; the current project was preserved."))
	}
	if w.ViewerSelected != nil && !*w.ViewerSelected {
		nodes = append(nodes, quiet("project-work-viewer-preserved", "Imported and kept the recording without switching to its view."))
	}
	if w.RecoveryDismissed != nil && *w.RecoveryDismissed {
		nodes = append(nodes, quiet("project-work-recovery-dismissed", "Removed the confirmed recovery snapshot."))
	}
	if w.CatalogApplied != nil && !*w.CatalogApplied {
		nodes = append(nodes, quiet("project-work-catalog-preserved", "Recovery work completed; newer recovery state was preserved."))
	}
	return nodes
}
