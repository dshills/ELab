package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Startup uses the same versioned command an external agent sends.
func setupRecovery(service *workbench.Service, directory string) error {
	if directory == "" {
		var err error
		directory, err = workbench.DefaultRecoveryDirectory()
		if err != nil {
			return err
		}
	}
	raw, _ := json.Marshal(workbench.RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 2})
	revision := service.Snapshot().Revision
	response := service.Dispatch(context.Background(), workbench.Request{Version: 1, ID: "startup-recovery", Command: "recovery.configure", Params: raw, ExpectedRevision: &revision, Key: "startup-recovery"}, "user")
	if response.Error != nil {
		return fmt.Errorf("%s", response.Error.Message)
	}
	return nil
}
func (s *screen) recoveryInspector(state workbench.Snapshot) []ui.Node {
	recovery := state.Recovery
	busy := activeProjectWork(state)
	nodes := []ui.Node{section("recovery-title", "Autosave & recovery"), quiet("recovery-help", "Recovery snapshots preserve your edits separately from saved projects. Restoring opens a copy for Save As.")}
	if recovery.Directory == "" {
		directory, err := workbench.DefaultRecoveryDirectory()
		if err == nil {
			enable := s.action("recovery-enable", "Enable autosave", "recovery.configure.start", workbench.RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 2})
			enable.Disabled = enable.Disabled || busy
			nodes = append(nodes, enable)
		}
		return nodes
	}
	text := "Enable autosave"
	if recovery.Enabled {
		text = "Disable autosave"
	}
	toggle := s.action("recovery-toggle", text, "recovery.configure.start", workbench.RecoveryConfigureParams{Directory: recovery.Directory, Enabled: !recovery.Enabled, Interval: recovery.Interval})
	toggle.Disabled = toggle.Disabled || busy
	nodes = append(nodes, toggle, quiet("recovery-interval-label", "Autosave interval · 1–60 s"), timeField("recovery-interval", "Autosave interval", recovery.Interval, busy, func(text string) {
		s.invokeAtCommandRevision("recovery.configure.start", textFieldParams(workbench.RecoveryConfigureParams{Directory: recovery.Directory, Enabled: recovery.Enabled, Interval: recovery.Interval}, "interval_s", text), state.CommandRevision)
	}), s.action("recovery-checkpoint", "Save recovery snapshot now", "recovery.work.start", workbench.RecoveryWorkParams{Kind: "checkpoint"}), s.action("recovery-scan", "Refresh recovery list", "recovery.work.start", workbench.RecoveryWorkParams{Kind: "scan"}))
	if recovery.LastSavedAt != "" {
		nodes = append(nodes, quiet("recovery-last-saved", "Last snapshot · "+recoveryTime(recovery.LastSavedAt)))
	}
	if recovery.Error != "" {
		nodes = append(nodes, quiet("recovery-error", recovery.Error))
	}
	if state.Document.RecoveredFrom != "" {
		nodes = append(nodes, quiet("recovered-from", "Recovered copy · "+filepath.Base(state.Document.RecoveredFrom)))
	}
	options := []ui.Option{{ID: "__none", Label: "Choose a recovery snapshot"}}
	var chosen *workbench.RecoveryCandidate
	for _, candidate := range recovery.Candidates {
		candidate := candidate
		source := "Unsaved schematic"
		if candidate.SourcePath != "" {
			source = filepath.Base(candidate.SourcePath)
		}
		status := ""
		if candidate.ActiveSession {
			status = " · active session"
		}
		options = append(options, ui.Option{ID: candidate.ID, Label: source + " · " + recoveryTime(candidate.SavedAt) + status})
		if candidate.ID == recovery.Selected {
			chosen = &candidate
		}
	}
	selected := recovery.Selected
	if selected == "" {
		selected = "__none"
	}
	nodes = append(nodes, ui.Node{ID: "recovery-choice", Kind: ui.Selector, Options: options, SelectedID: selected, Accessibility: ui.Accessibility{Name: "Recovery snapshot"}, Change: func(e ui.ChangeEvent) {
		id := e.SelectedID
		if id == "__none" {
			id = ""
		}
		s.invoke("recovery.select", workbench.IDParams{ID: id})
	}})
	if chosen != nil {
		choice := workbench.RecoveryChoiceParams{ID: chosen.ID, SHA256: chosen.SHA256, ExpectedGeneration: &state.Document.Generation}
		nodes = append(nodes, quiet("recovery-summary", fmt.Sprintf("%d components · %d recordings · %s", chosen.Components, chosen.Recordings, recoveryTime(chosen.SavedAt))), buttonStyle(ui.Node{ID: "recovery-restore", Kind: ui.Button, Text: "Restore as a new copy", Disabled: s.fileBusy || s.pendingCommand != "", Accessibility: ui.Accessibility{Name: "Restore recovery as a new copy"}, Activate: func() { s.requestReplace("recovery.restore", choice) }}, true))
		dismiss := s.action("recovery-dismiss", "Dismiss this snapshot", "recovery.dismiss.start", choice)
		// Other live sessions keep their snapshots. The shared service enforces ownership.
		dismiss.Disabled = dismiss.Disabled || busy || chosen.ActiveSession && !chosen.CurrentSession
		nodes = append(nodes, dismiss)
		nodes = append(nodes, s.recoveryReadControls(state, *chosen)...)
	}
	for _, problem := range recovery.Problems {
		nodes = append(nodes, quiet("recovery-problem-"+problem.ID, "Snapshot could not be validated · "+problem.Message))
		if problem.SHA256 != "" {
			dismiss := s.action("recovery-dismiss-"+problem.ID, "Dismiss damaged snapshot", "recovery.dismiss.start", workbench.RecoveryChoiceParams{ID: problem.ID, SHA256: problem.SHA256})
			dismiss.Disabled = dismiss.Disabled || busy
			nodes = append(nodes, dismiss)
		}
	}
	return nodes
}
func recoveryTime(value string) string {
	instant, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return instant.Local().Format("Jan 2, 15:04:05")
}
