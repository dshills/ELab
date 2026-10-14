package main

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeProjectPickerKeepsRecordedAssetsAndFreshStartup(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	if err := user.app.Open(app.Window{ID: "main", Title: "Project recordings", Width: 1260, Height: 900, View: user.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, user, "example.rc", nil)
	uiAction(t, user, "simulation.pause", nil)
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, user.service, "simulation.step", workbench.StepParams{Duration: .015})
	uiAction(t, user, "verify-duration", &ui.ChangeEvent{Text: "10 ms"})
	uiAction(t, user, "verify-start", nil)
	original := completedVerification(t, user.service, user.service.Snapshot().LastOperation)
	uiAction(t, user, "recording-show", nil)
	path := filepath.Join(t.TempDir(), "experiment.elab")
	user.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
		if len(options.Extensions) != 1 || options.Extensions[0] != "elab" {
			t.Errorf("unexpected picker %+v", options)
		}
		return path, nil
	}
	uiAction(t, user, "file.save", nil)
	waitFileUI(t, user)
	if user.errorMessage != "" || user.service.Snapshot().Document.Dirty {
		t.Fatal("native project save failed", user.errorMessage)
	}
	uiAction(t, user, "file.open", nil)
	waitFileUI(t, user)
	loaded := user.service.Snapshot()
	if user.errorMessage != "" || loaded.Document.Dirty || !loaded.Paused || loaded.Transient.Sample.Time != 0 || loaded.Recording == nil || !loaded.Recording.Viewing || loaded.Document.RecordingCount != 1 {
		t.Fatal("native project reopen lost recorded/fresh state", user.errorMessage)
	}
	reopened := completedVerification(t, user.service, loaded.Recording.ID)
	if !reflect.DeepEqual(original.Metadata, reopened.Metadata) {
		t.Fatal("native reopen changed recorded data")
	}
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "project.load", workbench.LoadParams{Path: path})
	uiAction(t, user, "recording-keep", nil)
	agentAction(t, agent, "recording.keep.set", workbench.RecordingKeepParams{ID: agent.Snapshot().Recording.ID, Keep: false})
	if user.service.Snapshot().Document.RecordingCount != agent.Snapshot().Document.RecordingCount || user.service.Snapshot().Document.RecordingsDirty != agent.Snapshot().Document.RecordingsDirty || user.service.Snapshot().Document.Path != agent.Snapshot().Document.Path || !user.service.Snapshot().Document.RecordingsDirty || user.service.Snapshot().Document.SchematicDirty {
		t.Fatal("native/agent recorded inclusion diverged")
	}
	uiAction(t, user, "project.clear", nil)
	if user.pendingCommand != "project.clear" {
		t.Fatal("unsaved recorded changes bypassed the prompt")
	}
}
