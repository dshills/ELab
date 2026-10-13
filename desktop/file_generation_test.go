package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

func TestPickerRefusesAnInterveningProjectReplacement(t *testing.T) {
	for _, control := range []string{"file.save-as", "file.open", "export-svg"} {
		t.Run(control, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			if err := s.app.Open(app.Window{ID: "main", Title: "File race", Width: 1260, Height: 900, View: s.view}); err != nil {
				t.Fatal(err)
			}
			uiAction(t, s, "example.reset", nil)
			chosen := filepath.Join(t.TempDir(), "chosen.elab")
			if control == "file.open" {
				fixture := workbench.NewExample()
				defer fixture.Close()
				agentAction(t, fixture, "project.save", workbench.SaveParams{Path: chosen})
			}
			entered, release := make(chan struct{}), make(chan struct{})
			s.chooseFile = func(context.Context, app.FileOptions) (string, error) { close(entered); <-release; return chosen, nil }
			uiAction(t, s, control, nil)
			<-entered
			// The external agent replaces the project while the user's native picker waits.
			agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			generation := s.service.Snapshot().Document.Generation
			close(release)
			waitFileUI(t, s)
			if !strings.Contains(s.errorMessage, "Schematic changed") || s.service.Snapshot().Document.Generation != generation || len(s.service.Snapshot().Components) != 0 {
				t.Fatal("file picker operated on replacement project")
			}
			if control != "file.open" {
				if _, err := os.Stat(chosen); !os.IsNotExist(err) {
					t.Fatal("file chooser published stale target")
				}
			}
		})
	}
}
