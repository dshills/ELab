package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecordedExportRetainsSelectedOperationParameters(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	if err := user.app.Open(app.Window{ID: "main", Title: "Verify export", Width: 1260, Height: 900, View: user.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, user, "example.rc", nil)
	uiAction(t, user, "simulation.pause", nil)
	uiAction(t, user, "simulation.restart", nil)
	uiAction(t, user, "verify-duration", &ui.ChangeEvent{Text: "10 ms"})
	uiAction(t, user, "verify-start", nil)
	view := completedVerification(t, user.service, user.service.Snapshot().LastOperation)
	for _, format := range []string{"json", "csv"} {
		path := filepath.Join(t.TempDir(), "record."+format)
		user.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
			if options.Mode != app.SaveFile || len(options.Extensions) != 1 || options.Extensions[0] != format {
				t.Errorf("wrong recorded picker %+v", options)
			}
			return path, nil
		}
		uiAction(t, user, "verify-export-"+format, nil)
		waitFileUI(t, user)
		if user.errorMessage != "" {
			t.Fatal(user.errorMessage)
		}
		artifact := user.service.Snapshot().LastExport
		if artifact == nil || artifact.OperationID != view.Operation.ID || artifact.Format != "verification."+format {
			t.Fatal("native picker dropped record ID/format")
		}
		work := user.service.Snapshot().ProjectWork
		if work == nil || work.Status != "completed" || work.Command != "verification.export" || work.Actor != "user" || work.ArtifactSHA256 != artifact.SHA256 || artifact.SHA256 == "" {
			t.Fatal("native recorded export lacks its shared completed job/artifact identity")
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			t.Fatal("record export missing")
		}
	}
}

func TestNativeSchematicExportRefusesAliasOfSavedProject(t *testing.T) {
	user, cleanup := testScreen(t)
	defer cleanup()
	if err := user.app.Open(app.Window{ID: "main", Title: "Export protection", Width: 1260, Height: 900, View: user.view}); err != nil {
		t.Fatal(err)
	}
	agentAction(t, user.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, user.service, "simulation.pause", struct{}{})
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	path := filepath.Join(directory, "source.svg")
	agentAction(t, user.service, "project.save", workbench.SaveParams{Path: path})
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := user.service.Snapshot()
	user.chooseFile = func(context.Context, app.FileOptions) (string, error) {
		return filepath.Join(alias, "source.svg"), nil
	}
	uiAction(t, user, "export-svg", nil)
	waitFileUI(t, user)
	after := user.service.Snapshot()
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, actual) || user.errorMessage == "" || after.ProjectWork == nil || after.ProjectWork.Error == nil || after.ProjectWork.Error.Code != "invalid-params" || after.ProjectWork.Actor != "user" || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.UndoEntry, before.UndoEntry) {
		t.Fatal("native export did not preserve saved source or expose shared refusal", err)
	}
}
