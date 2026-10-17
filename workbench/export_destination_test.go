package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/dshills/elab/internal/schematicpdf"
)

func TestExportCannotReplaceAssociatedSchematicThroughParentAlias(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	path := filepath.Join(directory, "source.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	raw, _ := json.Marshal(ExportParams{Path: filepath.Join(alias, "source.elab"), Overwrite: true})
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "alias", Key: "alias", Command: "export.schematic.svg", Params: raw, ExpectedCommandRevision: &before.CommandRevision}, "test")
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if response.Error == nil || response.Error.Code != "invalid-params" || !bytes.Equal(original, after) {
		t.Fatalf("export must refuse the associated file alias: error=%+v saved_bytes_replaced=%t", response.Error, !bytes.Equal(original, after))
	}
}

func TestExportAliasProtectionCoversSharedKindsAndPreservesOtherDestinations(t *testing.T) {
	for _, kind := range []string{"schematic-svg", "schematic-pdf", "schematic-png", "scope-csv", "verification-json", "verification-csv"} {
		t.Run(kind, func(t *testing.T) {
			if (kind == "schematic-pdf" || kind == "schematic-png") && !schematicpdf.Supported() {
				t.Skip("macOS renderer")
			}
			s := pausedRC(t)
			defer s.Close()
			call(t, s, "simulation.step", StepParams{Duration: .002}, "step")
			duration := .002
			_, record := verify(t, s, VerifyParams{Duration: &duration}, "capture")
			directory := t.TempDir()
			alias := filepath.Join(t.TempDir(), "alias")
			if err := os.Symlink(directory, alias); err != nil {
				t.Skipf("directory symlink unavailable: %v", err)
			}
			path := filepath.Join(directory, "source.elab")
			call(t, s, "project.save", SaveParams{Path: path}, "save")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			p := ExportStartParams{Kind: kind, Path: filepath.Join(alias, "source.elab"), Overwrite: true}
			if kind == "verification-json" || kind == "verification-csv" {
				p.ID = record.Operation.ID
			}
			before := s.Snapshot()
			call(t, s, "export.start", p, "alias")
			after := awaitProjectStatus(t, s, "failed")
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(original, actual) || after.ProjectWork.Error.Code != "invalid-params" || before.Document != after.Document || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(after.LastExport, before.LastExport) {
				t.Fatal("alias refusal modified saved bytes, live data or baseline", err)
			}
			p.Path = filepath.Join(alias, "different-output")
			call(t, s, "export.start", p, "different")
			awaitProjectStatus(t, s, "completed")
			if _, err := os.ReadFile(p.Path); err != nil {
				t.Fatal("distinct destination in aliased directory refused", err)
			}
		})
	}
}

func TestExportAliasGuardProtectsMissingAssociatedNameAndHardLinks(t *testing.T) {
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	path := filepath.Join(directory, "source.elab")
	if checkExportDestination(filepath.Join(alias, "source.elab"), path) == nil {
		t.Fatal("removed associated filename can be recreated by export through alias")
	}
	if err := os.WriteFile(path, []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	hardLink := filepath.Join(directory, "other-name")
	if err := os.Link(path, hardLink); err != nil {
		t.Skipf("hard link unavailable: %v", err)
	}
	if checkExportDestination(hardLink, path) == nil {
		t.Fatal("same-file identity bypassed export protection")
	}
}

func TestExportRechecksProtectedAssociationAfterPreparation(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	directory := t.TempDir()
	first, second := filepath.Join(directory, "first"), filepath.Join(directory, "second")
	for _, folder := range []string{first, second} {
		if err := os.Mkdir(folder, 0700); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(directory, "current")
	if err := os.Symlink(first, alias); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	call(t, s, "project.save", SaveParams{Path: filepath.Join(alias, "source.elab")}, "save")
	original, err := os.ReadFile(filepath.Join(first, "source.elab"))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(second, "source.elab")
	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer func() { once.Do(func() { close(release) }) }()
	s.mu.Lock()
	s.projectExport = func(ctx context.Context, capture exportCapture, progress func(string, int)) (*preparedFile, error) {
		prepared, err := prepareProjectExport(ctx, capture, progress)
		close(entered)
		<-release
		return prepared, err
	}
	s.mu.Unlock()
	call(t, s, "export.start", ExportStartParams{Kind: "schematic-svg", Path: target, Overwrite: true}, "export")
	awaitLiveSampleBlock(t, entered)
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, alias); err != nil {
		t.Fatal(err)
	}
	once.Do(func() { close(release) })
	after := awaitProjectStatus(t, s, "failed")
	actual, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(original, actual) || after.ProjectWork.Error.Code != "invalid-params" {
		t.Fatal("late association alias was not protected", err)
	}
	oldSource, err := os.ReadFile(filepath.Join(first, "source.elab"))
	if err != nil || !bytes.Equal(original, oldSource) {
		t.Fatal("original saved source changed after alias retarget", err)
	}
	temps, err := filepath.Glob(filepath.Join(second, ".elab-export-*"))
	if err != nil || len(temps) != 0 {
		t.Fatal("refused publication left temporary files", err, temps)
	}
}

func TestExportPreviewPNGRetainsCurrentSchematicAliasProtection(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	path := filepath.Join(directory, "source.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	call(t, s, "print.preview.start", PreviewParams{}, "preview")
	preview := awaitPreview(t, s, "completed").PrintPreview
	before := s.Snapshot()
	raw, _ := json.Marshal(PNGExportParams{Path: filepath.Join(alias, "source.elab"), PreviewID: preview.ID, Overwrite: true})
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "alias", Key: "alias", Command: "export.schematic.png", Params: raw, ExpectedCommandRevision: &before.CommandRevision}, "test")
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, actual) || response.Error == nil || response.Error.Code != "invalid-params" || !reflect.DeepEqual(preview, s.Snapshot().PrintPreview) {
		t.Fatal("exact preview export bypassed associated-file protection or changed preview", err, response.Error)
	}
}
