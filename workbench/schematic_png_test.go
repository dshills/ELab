package workbench

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/schematicpdf"
)

func TestSchematicPNGExportPreservesSource(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS rendering backend")
	}
	for _, info := range ExampleCatalog() {
		t.Run(info.ID, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: info.ID}, "example")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			before := settled(t, s)
			path := filepath.Join(t.TempDir(), "schematic.png")
			call(t, s, "export.schematic.png", PNGExportParams{Path: path}, "png")
			after := s.Snapshot()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			im, err := png.Decode(bytes.NewReader(data))
			if err != nil || im.Bounds().Dx() != 1650 || im.Bounds().Dy() != 1275 || after.LastExport.Format != "png" || after.LastExport.DPI != 150 || after.LastExport.Page.Paper != "letter" {
				t.Fatal("missing PNG/page identity", err)
			}
			if before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Transient, after.Transient) || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
				t.Fatal("PNG export changes source/physics/history")
			}
			refuse(t, s, "export.schematic.png", PNGExportParams{Path: path}, "file-exists")
			for _, dpi := range []int{-1, 71, 301} {
				refuse(t, s, "export.schematic.png", PNGExportParams{Path: path, DPI: dpi}, "invalid-params")
			}
		})
	}
}

func TestBlockedPageExportKeepsOriginalPageAndPixels(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS rendering backend")
	}
	s := pausedRC(t)
	defer s.Close()
	dir := t.TempDir()
	baseline, captured := filepath.Join(dir, "baseline.png"), filepath.Join(dir, "captured.png")
	settings := SchematicPage{Paper: "a4", Orientation: "portrait", MarginMM: ptrFloat(12), Color: true}
	call(t, s, "export.page.set", settings, "page")
	call(t, s, "export.schematic.png", PNGExportParams{Path: baseline, DPI: 72}, "baseline")
	before := s.Snapshot()
	entered, release := blockExportPreparation(t, s)
	defer release()
	promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "schematic-png", Path: captured, DPI: 72})
	awaitLiveSampleBlock(t, entered)
	call(t, s, "export.page.set", SchematicPage{MarginMM: ptrFloat(30)}, "later-page")
	promptLiveCommand(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"})
	edited := s.Snapshot()
	release()
	after := awaitProjectStatus(t, s, "completed")
	one, err := os.ReadFile(baseline)
	if err != nil {
		t.Fatal(err)
	}
	two, err := os.ReadFile(captured)
	if err != nil {
		t.Fatal(err)
	}
	// PNG encoding includes captured identity (no native PDF timestamps).
	if !bytes.Equal(one, two) || after.LastExport.Page.Paper != "a4" || after.LastExport.DPI != 72 || after.LastExport.ElectricalRevision != before.ElectricalRevision {
		t.Fatal("page/image capture changed after admission")
	}
	if after.Document != edited.Document || !reflect.DeepEqual(after.Components, edited.Components) || !reflect.DeepEqual(after.SchematicPage, edited.SchematicPage) {
		t.Fatal("export overwrote later source/session choices")
	}
}

func TestCanceledPNGExportPreservesDestination(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS rendering backend")
	}
	s := pausedRC(t)
	defer s.Close()
	path := filepath.Join(t.TempDir(), "existing.png")
	original := []byte("existing user file")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	entered, release := blockExportPreparation(t, s)
	defer release()
	before := s.Snapshot()
	admitted := promptLiveCommand(t, s, "export.start", ExportStartParams{Kind: "schematic-png", Path: path, Overwrite: true}).Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	promptLiveCommand(t, s, "project.work.cancel", IDParams{ID: admitted.ProjectWork.ID})
	release()
	after := awaitProjectStatus(t, s, "canceled")
	data, err := os.ReadFile(path)
	temps, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".elab-export-*"))
	if err != nil || !bytes.Equal(data, original) || len(temps) != 0 || after.LastExport != before.LastExport || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("canceled PNG changed source/destination or leaked a temporary", err)
	}
}
