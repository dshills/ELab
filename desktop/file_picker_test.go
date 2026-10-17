package main

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

func TestSavePickerDefaultNamesAndCancellation(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Save defaults", Width: 1040, Height: 760, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "transient-step", nil)
	for _, tc := range []struct{ control, filename, extension string }{
		{"file.save", "Untitled.elab", "elab"},
		{"export-svg", "Schematic.svg", "svg"},
		{"export-pdf", "Schematic.pdf", "pdf"},
		{"export-png", "Schematic.png", "png"},
		{"export-csv", "Waveforms.csv", "csv"},
	} {
		t.Run(tc.control, func(t *testing.T) {
			before := s.service.Snapshot()
			called := false
			s.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
				called = true
				want := tc.filename
				if runtime.GOOS == "darwin" {
					want = want[:len(want)-len(tc.extension)-1]
				}
				if options.Filename != want || options.Mode != app.SaveFile || !reflect.DeepEqual(options.Extensions, []string{tc.extension}) {
					t.Errorf("picker options = %+v; want filename %q and extension %q", options, want, tc.extension)
				}
				return "", app.ErrCanceled
			}
			uiAction(t, s, tc.control, nil)
			waitFileUI(t, s)
			after := s.service.Snapshot()
			if !called || s.errorMessage != "" || after.Document != before.Document || after.RunID != before.RunID || !reflect.DeepEqual(after.Components, before.Components) {
				t.Fatal("canceling native picker changed the project or produced an error")
			}
		})
	}
	path := filepath.Join(t.TempDir(), "My circuit.v2.elab")
	agentAction(t, s.service, "project.save", workbench.SaveParams{Path: path})
	s.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
		want := filepath.Base(path)
		if runtime.GOOS == "darwin" {
			want = "My circuit.v2"
		}
		if options.Filename != want || options.Directory != filepath.Dir(path) {
			t.Errorf("Save As options = %+v", options)
		}
		return "", app.ErrCanceled
	}
	uiAction(t, s, "file.save-as", nil)
	waitFileUI(t, s)
	if state := s.service.Snapshot(); state.Document.Path != path || state.Document.Dirty || s.errorMessage != "" {
		t.Fatal("canceled Save As changed the existing file association")
	}
}

func TestNativeFileOptionsPreserveNamesAndTypes(t *testing.T) {
	for _, tc := range []struct {
		name, platform, filename, want string
		mode                           app.FileMode
		extensions                     []string
	}{
		{"project", "darwin", "Untitled.elab", "Untitled", app.SaveFile, []string{"elab"}},
		{"compound", "darwin", "Timer.v2.elab", "Timer.v2", app.SaveFile, []string{"elab"}},
		{"unicode", "darwin", "Éclairage.elab", "Éclairage", app.SaveFile, []string{"elab"}},
		{"case", "darwin", "Timer.ELAB", "Timer", app.SaveFile, []string{"elab"}},
		{"json", "darwin", "Verification.json", "Verification", app.SaveFile, []string{"json"}},
		{"csv", "darwin", "DCSweep.csv", "DCSweep", app.SaveFile, []string{"csv"}},
		{"hidden", "darwin", ".elab", ".elab", app.SaveFile, []string{"elab"}},
		{"empty", "darwin", "", "", app.SaveFile, []string{"elab"}},
		{"bare", "darwin", "Timer", "Timer", app.SaveFile, []string{"elab"}},
		{"other suffix", "darwin", "Timer.txt", "Timer.txt", app.SaveFile, []string{"elab"}},
		{"multiple types", "darwin", "Timer.elab", "Timer.elab", app.SaveFile, []string{"elab", "json"}},
		{"no types", "darwin", "Timer.elab", "Timer.elab", app.SaveFile, nil},
		{"open", "darwin", "Timer.elab", "Timer.elab", app.OpenFile, []string{"elab"}},
		{"linux", "linux", "Timer.elab", "Timer.elab", app.SaveFile, []string{"elab"}},
		{"windows", "windows", "Timer.elab", "Timer.elab", app.SaveFile, []string{"elab"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := app.FileOptions{Mode: tc.mode, Title: "Choose circuit", Directory: "/circuits", Filename: tc.filename, Extensions: tc.extensions}
			want := original
			want.Filename = tc.want
			if got := nativeFileOptions(original, tc.platform); !reflect.DeepEqual(got, want) || original.Filename != tc.filename {
				t.Fatalf("native options = %+v, want %+v; original %+v", got, want, original)
			}
		})
	}
}
