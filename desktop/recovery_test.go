package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecoveryControlsRestoreSameCopyAsAgents(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	source := workbench.NewExample()
	agentAction(t, source, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 2})
	agentAction(t, source, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	agentAction(t, source, "recovery.checkpoint", struct{}{})
	source.Close()
	user, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	if err := setupRecovery(user.service, directory); err != nil {
		t.Fatal(err)
	}
	agentAction(t, agent, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 2})
	uiAction(t, user, "recovery-interval", &ui.ChangeEvent{Text: "1 s"})
	configured := awaitNativeRecoveryConfiguration(t, user)
	if configured.ProjectWork.RecoveryConfigured == nil || !*configured.ProjectWork.RecoveryConfigured || configured.ProjectWork.Actor != "user" {
		t.Fatal("native setup lacks shared outcome/actor")
	}
	agentAction(t, agent, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 1})
	uiAction(t, user, "recovery-toggle", nil)
	awaitNativeRecoveryConfiguration(t, user)
	agentAction(t, agent, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 1})
	if user.service.Snapshot().Recovery.Enabled || agent.Snapshot().Recovery.Enabled {
		t.Fatal("disable failed")
	}
	candidates := agent.Snapshot().Recovery.Candidates
	if len(candidates) != 1 {
		t.Fatal("recovery choice missing")
	}
	uiAction(t, user, "recovery-choice", &ui.ChangeEvent{SelectedID: candidates[0].ID})
	// Exercise the normal unsaved-change decision before restoring.
	p := workbench.PlaceParams{Kind: "resistor", Position: workbench.Point{X: 0, Y: 0}, Value: "1k"}
	agentAction(t, user.service, "component.place", p)
	agentAction(t, agent, "component.place", p)
	uiAction(t, user, "recovery-restore", nil)
	if user.pendingCommand != "recovery.restore" {
		t.Fatal("restore bypassed dirty prompt")
	}
	uiAction(t, user, "discard-continue", nil)
	choice := workbench.RecoveryChoiceParams{ID: candidates[0].ID, SHA256: candidates[0].SHA256, DiscardChanges: true}
	agentAction(t, agent, "recovery.restore", choice)
	u, a := user.service.Snapshot(), agent.Snapshot()
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || u.Document != a.Document || !u.Document.Dirty || u.Document.Path != "" {
		t.Fatal("native/agent restored copy differs")
	}
	uiAction(t, user, "recovery-scan", nil)
	beforeDismiss := user.service.Snapshot()
	uiAction(t, user, "recovery-dismiss", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	removed, err := user.service.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if removed.ProjectWork.Command != "recovery.dismiss" || removed.ProjectWork.Actor != "user" || removed.ProjectWork.RecoveryDismissed == nil || !*removed.ProjectWork.RecoveryDismissed || removed.ProjectWork.SourceSHA256 != choice.SHA256 {
		t.Fatal("native dismissal lost shared result")
	}
	if _, err := os.Stat(filepath.Join(directory, choice.ID+".elab-recovery")); !os.IsNotExist(err) {
		t.Fatal("native dismissal left confirmed file", err)
	}
	if !reflect.DeepEqual(beforeDismiss.Components, removed.Components) || beforeDismiss.Document != removed.Document || !reflect.DeepEqual(beforeDismiss.Result, removed.Result) || !reflect.DeepEqual(beforeDismiss.Transient, removed.Transient) {
		t.Fatal("native dismissal changes live circuit")
	}
	if err := ui.Validate(user.view()); err != nil {
		t.Fatal(err)
	}
}

func awaitNativeRecoveryConfiguration(t *testing.T, s *screen) workbench.Snapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.app.Drain()
		state := s.service.Snapshot()
		if state.ProjectWork != nil && state.ProjectWork.Command == "recovery.configure" {
			if state.ProjectWork.Status == "completed" {
				return state
			}
			if state.ProjectWork.Error != nil {
				t.Fatal(state.ProjectWork.Error)
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("native recovery setup did not complete")
	return workbench.Snapshot{}
}
