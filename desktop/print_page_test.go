package main

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativePDFPageControlsMatchAgentAndPreserveCircuit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "pdf-paper", &ui.ChangeEvent{SelectedID: "a4"})
	uiAction(t, s, "pdf-orientation", &ui.ChangeEvent{SelectedID: "portrait"})
	uiAction(t, s, "pdf-margin", &ui.ChangeEvent{Number: 12})
	uiAction(t, s, "pdf-color", &ui.ChangeEvent{Checked: true})
	state := s.service.Snapshot()
	if state.SchematicPage.Paper != "a4" || state.SchematicPage.Orientation != "portrait" || *state.SchematicPage.MarginMM != 12 || !state.SchematicPage.Color {
		t.Fatal("native page settings failed")
	}
	if state.Document != before.Document || state.RunID != before.RunID || !reflect.DeepEqual(state.Transient, before.Transient) || !reflect.DeepEqual(state.UndoEntry, before.UndoEntry) {
		t.Fatal("page settings changed source/physics/history")
	}
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "export.page.set", state.SchematicPage)
	if !reflect.DeepEqual(agent.Snapshot().SchematicPage, state.SchematicPage) {
		t.Fatal("page user/agent mismatch")
	}
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned PDF test", Width: 1040, Height: 760, View: s.view}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "page.pdf")
	s.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
		if len(options.Extensions) != 1 || options.Extensions[0] != "pdf" {
			t.Error("PDF picker format")
		}
		return path, nil
	}
	uiAction(t, s, "export-pdf", nil)
	waitFileUI(t, s)
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 5 || string(data[:5]) != "%PDF-" || s.errorMessage != "" {
		t.Fatal("native shared PDF export", err, s.errorMessage)
	}
	if s.service.Snapshot().LastExport.Page.Paper != "a4" {
		t.Fatal("native export lost shared page defaults")
	}
	path = filepath.Join(t.TempDir(), "page.png")
	s.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
		if len(options.Extensions) != 1 || options.Extensions[0] != "png" {
			t.Error("PNG picker format")
		}
		return path, nil
	}
	uiAction(t, s, "export-png", nil)
	waitFileUI(t, s)
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil || im.Bounds().Dx() != 1241 || im.Bounds().Dy() != 1754 || s.errorMessage != "" {
		t.Fatal("native shared PNG export", err, s.errorMessage)
	}
	if s.service.Snapshot().LastExport.Page.Paper != "a4" || s.service.Snapshot().LastExport.DPI != 150 {
		t.Fatal("native PNG lost shared page/default DPI")
	}
}
