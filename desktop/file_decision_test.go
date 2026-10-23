package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestFileDecisionOldCompletionCannotUseReopenedIntent(t *testing.T) {
	for _, control := range []string{"save-continue", "discard-continue", "cancel-file-action", "dismiss"} {
		t.Run(control, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			// Save decisions require an actual edit; shipped templates start clean.
			agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
			uiAction(t, s, "project.clear", nil)
			var old ui.Node
			s.app.Do(func() {
				if control == "dismiss" {
					old = s.filePrompt()
				} else {
					old, _ = findNode(s.filePrompt(), control)
				}
			})
			original := s.service.Snapshot()
			serial := s.pendingFileSerial
			uiAction(t, s, "cancel-file-action", nil)
			uiAction(t, s, "project.clear", nil)
			before := s.service.Snapshot()
			currentSerial := s.pendingFileSerial
			if before.CommandRevision != original.CommandRevision || currentSerial == serial {
				t.Fatal("fixture must reopen the same intent without a domain command")
			}
			s.app.Do(func() {
				if control == "dismiss" {
					old.Dismiss()
				} else {
					old.Activate()
				}
			})
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingCommand != "project.clear" || s.pendingFileSerial != currentSerial || s.fileBusy {
				t.Fatal("old completion acted on reopened intent", control, s.errorMessage)
			}
			uiAction(t, s, "cancel-file-action", nil)
			if s.pendingCommand != "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("fresh Cancel failed")
			}
		})
	}
}

func TestFileDecisionGuardReachesSharedCommandAdmission(t *testing.T) {
	for _, kind := range []string{"clear", "load"} {
		t.Run(kind, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			openCloseTestWindow(t, s)
			uiAction(t, s, "example.reset", nil)
			// Save decisions require an actual edit; shipped templates start clean.
			agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
			if kind == "load" {
				fixture := workbench.NewExample()
				t.Cleanup(fixture.Close)
				path := filepath.Join(t.TempDir(), "incoming.elab")
				agentAction(t, fixture, "project.save", workbench.SaveParams{Path: path})
				s.app.Do(func() { s.beginFile("project.load", false, workbench.LoadParams{Path: path}, nil) })
				waitFileUI(t, s)
			} else {
				uiAction(t, s, "project.clear", nil)
			}
			command, serial := s.pendingCommand, s.pendingFileSerial
			var intervening workbench.Snapshot
			s.app.Do(func() {
				decision := s.captureFileDecision()
				s.withFileDecision(decision, true, func() {
					// Mutate after the UI predicate but before shared command admission.
					agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
					intervening = s.service.Snapshot()
					s.continuePending(true)
				})
			})
			waitFileUI(t, s)
			if !reflect.DeepEqual(intervening, s.service.Snapshot()) || s.pendingCommand != command || s.pendingFileSerial != serial || !strings.Contains(s.errorMessage, "expected command revision") || s.fileDecisionContext != nil {
				t.Fatal("decision guard lost between UI predicate and dispatch", kind, s.errorMessage)
			}
		})
	}
}

func TestFileDecisionSaveRefusesReplacedDocumentBeforePicker(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	openCloseTestWindow(t, s)
	uiAction(t, s, "example.reset", nil)
	// Save decisions require an actual edit; shipped templates start clean.
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	uiAction(t, s, "project.clear", nil)
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "R2", Kind: "resistor", Value: "3k"})
	before := s.service.Snapshot()
	calls := atomic.Int32{}
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { calls.Add(1); return "", app.ErrCanceled }
	s.app.Do(func() { node, _ := findNode(s.view(), "save-continue"); node.Activate() })
	if !reflect.DeepEqual(before, s.service.Snapshot()) || calls.Load() != 0 || s.fileBusy || !strings.Contains(s.errorMessage, "Schematic changed") || s.pendingCommand != "project.clear" {
		t.Fatal("old prompt saved a replacement document", s.errorMessage)
	}
	uiAction(t, s, "cancel-file-action", nil)
	if s.pendingCommand != "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("fresh Cancel could not dismiss obsolete document intent")
	}
}

func TestFileDecisionSavePickerKeepsCapturedAdmissionGuard(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	openCloseTestWindow(t, s)
	uiAction(t, s, "example.reset", nil)
	// Save decisions require an actual edit; shipped templates start clean.
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	uiAction(t, s, "project.clear", nil)
	path := filepath.Join(t.TempDir(), "must-not-save.elab")
	entered, release := make(chan struct{}), make(chan struct{})
	var released atomic.Bool
	unblock := func() {
		if released.CompareAndSwap(false, true) {
			close(release)
		}
	}
	t.Cleanup(unblock)
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { close(entered); <-release; return path, nil }
	uiAction(t, s, "save-continue", nil)
	<-entered
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	serial := s.pendingFileSerial
	unblock()
	waitFileUI(t, s)
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingCommand != "project.clear" || s.pendingFileSerial != serial || !strings.Contains(s.errorMessage, "expected command revision") || s.fileDecisionContext != nil {
		t.Fatal("save picker adopted later agent context", s.errorMessage)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("stale save published a file", err)
	}
}

func TestFileDecisionBusyPickerPreservesIntentAndReopensAfterCancel(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	openCloseTestWindow(t, s)
	uiAction(t, s, "example.reset", nil)
	// Save decisions require an actual edit; shipped templates start clean.
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	uiAction(t, s, "project.clear", nil)
	var buttons []ui.Node
	s.app.Do(func() {
		for _, id := range []string{"save-continue", "discard-continue", "cancel-file-action"} {
			node, _ := findNode(s.filePrompt(), id)
			buttons = append(buttons, node)
		}
	})
	entered, release := make(chan struct{}), make(chan struct{})
	var released atomic.Bool
	unblock := func() {
		if released.CompareAndSwap(false, true) {
			close(release)
		}
	}
	t.Cleanup(unblock)
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) {
		close(entered)
		<-release
		return "", app.ErrCanceled
	}
	uiAction(t, s, "save-continue", nil)
	<-entered
	before := s.service.Snapshot()
	serial := s.pendingFileSerial
	s.app.Do(func() {
		if s.filePrompt().Open {
			t.Fatal("busy decision modal hides file progress/cancellation")
		}
		for _, button := range buttons {
			button.Activate()
		}
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingCommand != "project.clear" || s.pendingFileSerial != serial || !s.fileBusy {
		t.Fatal("busy callback changed pending intent")
	}
	unblock()
	waitFileUI(t, s)
	s.app.Do(func() {
		if !s.filePrompt().Open {
			t.Fatal("picker cancellation lost decision")
		}
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingFileSerial != serial || s.errorMessage != "" {
		t.Fatal("canceled picker changed graph/intent", s.errorMessage)
	}
	uiAction(t, s, "cancel-file-action", nil)
}
