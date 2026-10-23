package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeAutomaticRecoveryFailureAndRetryPresentation(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	base := t.TempDir()
	directory := filepath.Join(base, "recovery")
	agentAction(t, user.service, "example.rc", struct{}{})
	agentAction(t, user.service, "simulation.pause", struct{}{})
	agentAction(t, user.service, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 1})
	moved := directory + "-preserved"
	if err := os.Rename(directory, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(directory, []byte("owned unavailable storage"), 0600); err != nil {
		t.Fatal(err)
	}
	repaired := false
	repair := func() {
		if !repaired {
			if err := os.Remove(directory); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(moved, directory); err != nil {
				t.Fatal(err)
			}
			repaired = true
		}
	}
	defer repair()
	await := func(failed bool) workbench.Snapshot {
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			state := user.service.Snapshot()
			if failed && state.Recovery.Error != "" || !failed && state.Recovery.Error == "" && state.Recovery.LastSnapshot != "" {
				return state
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("automatic storage state did not appear", failed)
		return workbench.Snapshot{}
	}
	failed := await(true)
	uiAction(t, user, "inspector-section-document", nil)
	user.app.Do(func() {
		root := user.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		detail, ok := findAction(root, "recovery-error")
		if !ok || detail.Text != failed.Recovery.Error {
			t.Fatal("automatic failure missing from native recovery controls")
		}
		message, ok := findAction(root, "messages")
		if !ok || !strings.Contains(message.Text, "Autosave needs attention") {
			t.Fatal("automatic failure missing from native status", message.Text)
		}
	})
	repair()
	agentAction(t, user.service, "component.set", workbench.ValueParams{ID: "R1", Value: "30k"})
	ready := await(false)
	if len(ready.Recovery.Candidates) != 1 || !ready.Document.Dirty {
		t.Fatal("automatic retry did not preserve a dirty recoverable copy")
	}
	user.app.Do(func() {
		root := user.view()
		if _, found := findAction(root, "recovery-error"); found {
			t.Fatal("native failure remained after automatic retry")
		}
	})
}
