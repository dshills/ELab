package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeUnknownRecoveryOwnerKeepsCopyAndDisablesDismiss(t *testing.T) {
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	source := workbench.NewExample()
	defer source.Close()
	agentAction(t, source, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 2})
	agentAction(t, source, "recovery.checkpoint", struct{}{})
	leases, err := filepath.Glob(filepath.Join(directory, "session-*.json"))
	if err != nil || len(leases) != 1 {
		t.Fatal("source lease missing", err)
	}
	if err := os.WriteFile(leases[0], []byte("damaged lease"), 0600); err != nil {
		t.Fatal(err)
	}
	user, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, user.service, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Enabled: false, Interval: 2})
	uiAction(t, user, "inspector-section-document", nil)
	state := user.service.Snapshot()
	if len(state.Recovery.Candidates) != 1 || state.Recovery.Candidates[0].OwnerStatus != workbench.RecoveryOwnerUnknown {
		t.Fatal("native state lost unknown owner")
	}
	user.app.Do(func() {
		root := user.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		dismiss, ok := findAction(root, "recovery-dismiss")
		if !ok || !dismiss.Disabled {
			t.Fatal("unverified foreign owner has a destructive action")
		}
		restore, ok := findAction(root, "recovery-restore")
		if !ok || restore.Disabled {
			t.Fatal("copy recovery unavailable")
		}
		note, ok := findAction(root, "recovery-owner-unknown")
		if !ok || !strings.Contains(note.Text, "could not be verified") {
			t.Fatal("unknown owner was described as active/inactive")
		}
		selector, ok := findAction(root, "recovery-choice")
		if !ok {
			t.Fatal("missing choices")
		}
		found := strings.Contains(selector.Text, "owner not verified")
		for _, option := range selector.Options {
			if strings.Contains(option.Label, "owner not verified") {
				found = true
			}
		}
		if !found {
			t.Fatal("selector omits ownership uncertainty")
		}
	})
}
