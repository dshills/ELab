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

func TestNativeTiledPageControlsNavigationAndCapturedExport(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned tiled output test", Width: 1040, Height: 760, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "component.move", workbench.MoveParams{ID: "C1", Position: workbench.Point{X: 800, Y: 600}})
	before := s.service.Snapshot()
	uiAction(t, s, "pdf-tile", &ui.ChangeEvent{Checked: true})
	uiAction(t, s, "pdf-overlap", &ui.ChangeEvent{Number: 7})
	settings := s.service.Snapshot().SchematicPage
	if !settings.Tile || settings.Scale == nil || *settings.Scale != 1 || *settings.OverlapMM != 7 {
		t.Fatal("native tiling settings", settings)
	}
	uiAction(t, s, "print-preview", nil)
	waitPreviewUI(t, s)
	first := s.service.Snapshot().PrintPreview
	if first.PageCount < 4 {
		t.Fatal("native tiled preview has no grid", first)
	}
	uiAction(t, s, "preview-next", nil)
	waitPreviewUI(t, s)
	second := s.service.Snapshot().PrintPreview
	if first.ID == second.ID || second.PageIndex != 1 {
		t.Fatal("native page navigation did not use captured page command")
	}
	path := filepath.Join(t.TempDir(), "shown.png")
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
	uiAction(t, s, "preview-export-png", nil)
	waitFileUI(t, s)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || s.service.Snapshot().LastExport.OperationID != second.ID {
		t.Fatal("native page export lost capture")
	}
	uiAction(t, s, "preview-previous", nil)
	waitPreviewUI(t, s)
	current := s.service.Snapshot().PrintPreview
	if current.PageIndex != 0 {
		t.Fatal("previous page did not navigate")
	}
	png2 := filepath.Join(t.TempDir(), "first.png")
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return png2, nil }
	uiAction(t, s, "preview-export-png", nil)
	waitFileUI(t, s)
	other, _ := os.ReadFile(png2)
	if bytes.Equal(data, other) {
		t.Fatal("native pages identical")
	}
	after := s.service.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("output controls changed circuit")
	}
	uiAction(t, s, "pdf-fit", nil)
	if s.service.Snapshot().SchematicPage.Tile {
		t.Fatal("fit did not restore single-page output")
	}
}
