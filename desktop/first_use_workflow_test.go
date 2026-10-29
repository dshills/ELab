package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

// Engineering counterpart of task 4: actual desktop file bindings use a
// disposable picker, and an independent service reads the saved file.
func TestFirstUsePracticeSaveReopenAndExport(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned practice workflow", Width: 1200, Height: 800, View: s.view}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	project := filepath.Join(dir, "practice.elab")
	s.chooseFile = func(_ context.Context, o app.FileOptions) (string, error) {
		if len(o.Extensions) == 1 && (o.Extensions[0] == "svg" || o.Extensions[0] == "pdf") {
			return filepath.Join(dir, "practice."+o.Extensions[0]), nil
		}
		return project, nil
	}
	uiAction(t, s, "example.flasher", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "file.save-as", nil)
	waitFileUI(t, s)
	saved := s.service.Snapshot()
	if s.errorMessage != "" || saved.Document.Dirty || saved.Document.Path != project {
		t.Fatal("practice save", s.errorMessage)
	}
	uiAction(t, s, "project.clear", nil)
	uiAction(t, s, "file.open", nil)
	waitFileUI(t, s)
	reopened := s.service.Snapshot()
	if s.errorMessage != "" || reopened.Document.Dirty || !reflect.DeepEqual(saved.Components, reopened.Components) || !reflect.DeepEqual(saved.Wires, reopened.Wires) {
		t.Fatal("reopened graph differs", s.errorMessage)
	}
	reference := workbench.New()
	defer reference.Close()
	agentAction(t, reference, "project.load", workbench.LoadParams{Path: project})
	if !reflect.DeepEqual(reference.Snapshot().Components, reopened.Components) || !reflect.DeepEqual(reference.Snapshot().Wires, reopened.Wires) {
		t.Fatal("agent cannot read desktop file")
	}
	for _, format := range []string{"svg", "pdf"} {
		uiAction(t, s, "export-"+format, nil)
		waitFileUI(t, s)
		data, err := os.ReadFile(filepath.Join(dir, "practice."+format))
		if err != nil || s.errorMessage != "" {
			t.Fatal("practice export", format, err, s.errorMessage)
		}
		if format == "pdf" && !strings.HasPrefix(string(data), "%PDF-") {
			t.Fatal("invalid PDF")
		}
		if format == "svg" {
			for _, part := range saved.Components {
				if !strings.Contains(string(data), part.ID) {
					t.Fatal("SVG missing part", part.ID)
				}
			}
		}
	}
	if after := s.service.Snapshot(); !reflect.DeepEqual(reopened.Components, after.Components) || !reflect.DeepEqual(reopened.Wires, after.Wires) || after.Document.Dirty {
		t.Fatal("export changed schematic")
	}
}
