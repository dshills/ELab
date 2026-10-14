package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
	uiAction(t, user, "verify-duration", &ui.ChangeEvent{Number: 10})
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
