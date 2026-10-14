package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

func openCloseTestWindow(t *testing.T, s *screen) {
	t.Helper()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned close test", Width: 1040, Height: 760, View: s.view, CloseRequest: s.requestClose}); err != nil {
		t.Fatal(err)
	}
}
func requestTestClose(t *testing.T, s *screen) {
	t.Helper()
	if err := s.app.CheckCloseWindow("main"); !errors.Is(err, app.ErrCloseRefused) {
		t.Fatal("native policy bypassed asynchronous service admission", err)
	}
}
func TestNativeCloseCancelDiscardAndCleanAdmission(t *testing.T) {
	for _, decision := range []string{"cancel-file-action", "discard-continue"} {
		t.Run(decision, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			openCloseTestWindow(t, s)
			uiAction(t, s, "example.reset", nil)
			before := s.service.Snapshot()
			requestTestClose(t, s)
			if s.pendingCommand != "app.close" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("close did not preserve dirty circuit behind its decision")
			}
			uiAction(t, s, decision, nil)
			state := s.service.Snapshot()
			if decision == "cancel-file-action" && (state.Closed || !reflect.DeepEqual(before, state)) || decision == "discard-continue" && !state.Closed {
				t.Fatal("close decision differs from shared semantics")
			}
		})
	}
	s, cleanup := testScreen(t)
	defer cleanup()
	openCloseTestWindow(t, s)
	requestTestClose(t, s)
	if !s.service.Snapshot().Closed || s.pendingCommand != "" {
		t.Fatal("clean empty session did not admit shared close")
	}
}

func TestNativeClosePreservesExistingDecisionAndBusyPicker(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	openCloseTestWindow(t, s)
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "project.clear", nil)
	command, params := s.pendingCommand, append(json.RawMessage(nil), s.pendingParams...)
	before := s.service.Snapshot()
	requestTestClose(t, s)
	if s.pendingCommand != command || !reflect.DeepEqual(params, s.pendingParams) || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("close replaced another unsaved-work decision")
	}
	uiAction(t, s, "cancel-file-action", nil)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) {
		close(entered)
		<-release
		return "", app.ErrCanceled
	}
	uiAction(t, s, "file.save-as", nil)
	<-entered
	before = s.service.Snapshot()
	requestTestClose(t, s)
	if !s.fileBusy || s.pendingCommand != "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("close disrupted an active file picker")
	}
	unblock()
	waitFileUI(t, s)
}

func TestNativeSaveBeforeCloseAndStaleDiscard(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	openCloseTestWindow(t, s)
	uiAction(t, s, "example.reset", nil)
	path := filepath.Join(t.TempDir(), "saved-before-close.elab")
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
	requestTestClose(t, s)
	uiAction(t, s, "save-continue", nil)
	waitFileUI(t, s)
	data, err := os.ReadFile(path)
	if err != nil || !json.Valid(data) || !s.service.Snapshot().Closed {
		t.Fatal("save did not finish before shared close", err)
	}
	other, finish := testScreen(t)
	defer finish()
	openCloseTestWindow(t, other)
	uiAction(t, other, "example.reset", nil)
	requestTestClose(t, other)
	agentAction(t, other.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	before := other.service.Snapshot()
	other.app.Do(func() {
		node, ok := findNode(other.view(), "discard-continue")
		if !ok {
			t.Fatal("close decision missing")
		}
		node.Activate()
	})
	if other.errorMessage == "" || !reflect.DeepEqual(before, other.service.Snapshot()) || other.service.Snapshot().Closed {
		t.Fatal("old close decision closed an agent's replacement document")
	}
}

func TestNativeSaveBeforeCloseRefusesAnAgentReplacementDuringPicker(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	openCloseTestWindow(t, s)
	uiAction(t, s, "example.reset", nil)
	path := filepath.Join(t.TempDir(), "old-close-save.elab")
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) {
		close(entered)
		<-release
		return path, nil
	}
	requestTestClose(t, s)
	uiAction(t, s, "save-continue", nil)
	<-entered
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	before := s.service.Snapshot()
	unblock()
	waitFileUI(t, s)
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) || s.service.Snapshot().Closed {
		t.Fatal("old Save-and-close affected the replacement document")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("old Save-and-close published a replacement project", err)
	}
}
