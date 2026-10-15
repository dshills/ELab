package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeShutdownFailureUsesSharedExplicitAcknowledgement(t *testing.T) {
	for _, action := range []string{"button", "window"} {
		t.Run(action, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			openCloseTestWindow(t, s)
			directory := t.TempDir()
			if err := os.Chmod(directory, 0700); err != nil {
				t.Fatal(err)
			}
			agentAction(t, s.service, "recovery.configure", workbench.RecoveryConfigureParams{Directory: directory, Interval: 1})
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "session-") {
				t.Fatal("owned fault fixture has no unique lease", err)
			}
			lease := filepath.Join(directory, entries[0].Name())
			if err := os.Remove(lease); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(lease, 0700); err != nil {
				t.Fatal(err)
			}
			protected := filepath.Join(lease, "retain")
			if err := os.WriteFile(protected, []byte("fault fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			before := s.service.Snapshot()
			requestTestClose(t, s)
			deadline := time.Now().Add(3 * time.Second)
			for {
				state := s.service.Snapshot()
				if state.Shutdown != nil && state.Shutdown.Status == "failed" {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("shutdown did not report cleanup failure")
				}
				time.Sleep(time.Millisecond)
			}
			failed := s.service.Snapshot()
			if failed.ReadyToExit() || failed.Shutdown.Actor != "user" || failed.Shutdown.RecoveryError == "" {
				t.Fatal("native failure lost shared state/actor")
			}
			s.app.Do(func() {
				root := s.view()
				if err := ui.Validate(root); err != nil {
					t.Fatal(err)
				}
				if n, ok := findNode(root, "shutdown-recovery-error"); !ok || n.Text != failed.Shutdown.RecoveryError {
					t.Fatal("native shutdown hides actual cleanup error")
				}
				if _, ok := findNode(root, "value-entry"); ok {
					t.Fatal("closed session leaves editing controls visible")
				}
			})
			s.chooseFile = func(context.Context, app.FileOptions) (string, error) {
				t.Error("closing session opened a new picker")
				return "", app.ErrCanceled
			}
			s.app.Do(func() { s.beginFile("project.load", true, workbench.LoadParams{}, nil) })
			if s.fileBusy {
				t.Fatal("closing session started file work")
			}
			if action == "button" {
				uiAction(t, s, "shutdown-acknowledge", nil)
			} else {
				requestTestClose(t, s)
			}
			after := s.service.Snapshot()
			if !after.ReadyToExit() || after.Shutdown.Status != "failed" || before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) {
				t.Fatal("native acknowledgement lost failure or changed project")
			}
			if _, err := os.Stat(protected); err != nil {
				t.Fatal("acknowledgement removed protected fixture", err)
			}
		})
	}
}
