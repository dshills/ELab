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

func TestNativeDisableUnavailableRecoveryUsesSharedCommandAndRetainsLastError(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	directory := filepath.Join(t.TempDir(), "recovery")
	agentAction(t, user.service, "example.rc", struct{}{})
	agentAction(t, user.service, "simulation.pause", struct{}{})
	agentAction(t, user.service, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: true, Interval: 1})
	moved := directory + "-preserved"
	if err := os.Rename(directory, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(directory, []byte("owned unavailable folder"), 0600); err != nil {
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
	deadline := time.Now().Add(5 * time.Second)
	for user.service.Snapshot().Recovery.Error == "" {
		if time.Now().After(deadline) {
			t.Fatal("automatic storage error did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	before := user.service.Snapshot()
	uiAction(t, user, "recovery-toggle", nil)
	after := awaitNativeRecoveryConfiguration(t, user)
	if after.Recovery.Enabled || after.Recovery.Error != before.Recovery.Error || *after.ProjectWork.CatalogApplied || !*after.ProjectWork.RecoveryConfigured || after.ProjectWork.Actor != "user" {
		t.Fatal("native disable lost shared scheduling/status outcome", after.ProjectWork)
	}
	user.app.Do(func() {
		root := user.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		toggle, ok := findAction(root, "recovery-toggle")
		if !ok || toggle.Disabled || toggle.Text != "Enable autosave" {
			t.Fatal("native toggle did not reflect disabled scheduling")
		}
		message, ok := findAction(root, "messages")
		if !ok || !strings.Contains(message.Text, "Autosave is off") {
			t.Fatal("last storage error was presented as active autosave", message.Text)
		}
		detail, ok := findAction(root, "recovery-error")
		if !ok || detail.Text != before.Recovery.Error {
			t.Fatal("native disable hid unverified storage error")
		}
		preserved, ok := findAction(root, "project-work-catalog-preserved")
		if !ok || strings.Contains(preserved.Text, "newer") {
			t.Fatal("settings-only completion implied a newer catalog observation")
		}
	})
	repair()
	agentAction(t, user.service, "component.set", workbench.ValueParams{ID: "R1", Value: "20k"})
	uiAction(t, user, "recovery-toggle", nil)
	awaitNativeRecoveryConfiguration(t, user)
	deadline = time.Now().Add(5 * time.Second)
	for user.service.Snapshot().Recovery.LastSnapshot == "" {
		if time.Now().After(deadline) {
			t.Fatal("re-enabled native autosave did not capture the dirty circuit")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !user.service.Snapshot().Recovery.Enabled || user.service.Snapshot().Recovery.Error != "" || !user.service.Snapshot().Document.Dirty {
		t.Fatal("re-enable did not validate/recover without marking work saved")
	}
}
