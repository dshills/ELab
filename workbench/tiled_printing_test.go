package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/printspool"
	"github.com/dshills/elab/internal/schematicpdf"
)

func tiledFixture(t *testing.T) *Service {
	t.Helper()
	s := pausedRC(t)
	t.Cleanup(s.Close)
	call(t, s, "component.move", MoveParams{ID: "C1", Position: Point{X: 800, Y: 600}}, "spread")
	call(t, s, "export.page.set", SchematicPage{Tile: true, Scale: ptrFloat(1), OverlapMM: ptrFloat(5)}, "page")
	return s
}
func TestTiledPreviewNavigationCaptureAndPNG(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS rendering backend")
	}
	s := tiledFixture(t)
	call(t, s, "print.preview.start", PreviewParams{}, "preview")
	first := awaitPreview(t, s, "completed").PrintPreview
	if first.PageCount < 4 || first.PageIndex != 0 || first.Rows < 2 || first.Columns < 2 {
		t.Fatal("missing tiled grid", first)
	}
	raw, _ := json.Marshal(PreviewPageParams{ID: first.ID, Index: 0})
	revision := s.Snapshot().CommandRevision
	result := s.Dispatch(context.Background(), Request{Version: 1, ID: "same-page-other-actor", Key: "same-page-other-actor", Command: "print.preview.page.set", Params: raw, ExpectedCommandRevision: &revision}, "other-actor")
	if result.Error != nil || s.Snapshot().PrintPreview.Actor != first.Actor || s.Snapshot().PrintPreview.ID != first.ID {
		t.Fatal("same-page reentry changed capture actor or identity", result.Error)
	}
	oldBytes := previewBytes(t, s, first)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "later-edit")
	before := s.Snapshot()
	call(t, s, "print.preview.page.set", PreviewPageParams{ID: first.ID, Index: first.PageCount - 1}, "last-page")
	last := awaitPreview(t, s, "completed").PrintPreview
	if last.ID == first.ID || last.Current || last.PageIndex != first.PageCount-1 || last.RunID != first.RunID || last.DocumentGeneration != first.DocumentGeneration {
		t.Fatal("page navigation recaptured source", last)
	}
	refuse(t, s, "print.preview.image", PreviewImageParams{ID: first.ID}, "not-found")
	stable := s.Snapshot()
	refuse(t, s, "print.preview.page.set", PreviewPageParams{ID: last.ID, Index: last.PageCount}, "invalid-params")
	if !reflect.DeepEqual(stable, s.Snapshot()) {
		t.Fatal("invalid page mutated preview")
	}
	pageBytes := previewBytes(t, s, last)
	if bytes.Equal(oldBytes, pageBytes) {
		t.Fatal("page selection returned old image")
	}
	path := filepath.Join(t.TempDir(), "captured-page.png")
	call(t, s, "export.schematic.png", PNGExportParams{Path: path, PreviewID: last.ID}, "captured-png")
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, pageBytes) {
		t.Fatal("shown-page export differs", err)
	}
	artifact := s.Snapshot().LastExport
	if artifact.PageIndex == nil || *artifact.PageIndex != last.PageIndex || artifact.OperationID != last.ID {
		t.Fatal("captured page identity missing", artifact)
	}
	*artifact.PageIndex = 99
	*artifact.Page.OverlapMM = 19
	if *s.Snapshot().LastExport.PageIndex != last.PageIndex || *s.Snapshot().LastExport.Page.OverlapMM != 5 {
		t.Fatal("page output aliases caller")
	}
	refuse(t, s, "export.schematic.png", PNGExportParams{Path: filepath.Join(t.TempDir(), "bad.png"), PreviewID: last.ID, DPI: 72}, "invalid-params")
	call(t, s, "print.preview.close", IDParams{ID: last.ID}, "close")
	refuse(t, s, "export.schematic.png", PNGExportParams{Path: path, PreviewID: last.ID}, "not-ready")
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || before.Document != after.Document || before.RunID != after.RunID {
		t.Fatal("output changed circuit physics")
	}
}
func TestTiledPrintSubmitsEntireCapturedDocument(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS rendering backend")
	}
	s := tiledFixture(t)
	call(t, s, "print.preview.start", PreviewParams{}, "preview")
	p := awaitPreview(t, s, "completed").PrintPreview
	call(t, s, "print.preview.page.set", PreviewPageParams{ID: p.ID, Index: 1}, "page2")
	p = awaitPreview(t, s, "completed").PrintPreview
	var captured []byte
	s.printBackend = &fakePrintSpool{submit: func(ctx context.Context, printer, path, title string, options printspool.Options) (printspool.Receipt, error) {
		var err error
		captured, err = os.ReadFile(path)
		return printspool.Receipt{ID: 42}, err
	}}
	s.printRender = func(ctx context.Context, c exportCapture) ([]byte, error) {
		if c.page.PageIndex != nil || c.artifact.PageIndex != nil || c.page.RasterDPI != 0 {
			t.Error("spool capture kept raster-page selection")
		}
		plan, err := schematicpdf.Plan(schematicPageSVG(c), c.page)
		if err != nil || len(plan.Pages) != p.PageCount {
			t.Error("spool capture lost tile grid", err)
		}
		return renderSchematicPage(ctx, c)
	}
	call(t, s, "print.printers.refresh", struct{}{}, "printers")
	awaitPrintWork(t, s, "completed")
	call(t, s, "print.submit", PrintSubmitParams{PreviewID: p.ID, PrinterID: "lab", Copies: 1}, "submit")
	state := awaitPrintWork(t, s, "completed")
	if !bytes.HasPrefix(captured, []byte("%PDF-")) || state.Jobs[0].PageCount != p.PageCount {
		t.Fatal("full tiled print not retained")
	}
}

func TestTiledBoundsAndCapturedPNGSurvivePreviewClose(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS rendering backend")
	}
	s := tiledFixture(t)
	for _, p := range []SchematicPage{{Tile: true}, {Tile: true, Scale: ptrFloat(1), OverlapMM: ptrFloat(21)}, {OverlapMM: ptrFloat(5)}} {
		before := s.Snapshot()
		refuse(t, s, "export.page.set", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("bad tile settings mutated state")
		}
	}
	call(t, s, "print.preview.start", PreviewParams{}, "preview")
	p := awaitPreview(t, s, "completed").PrintPreview
	expected := previewBytes(t, s, p)
	entered, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	s.projectExport = func(ctx context.Context, c exportCapture, progress func(string, int)) (*preparedFile, error) {
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return prepareProjectExport(ctx, c, progress)
	}
	path := filepath.Join(t.TempDir(), "closed-capture.png")
	call(t, s, "export.start", ExportStartParams{Kind: "schematic-png", Path: path, PreviewID: p.ID}, "admit")
	<-entered
	call(t, s, "print.preview.close", IDParams{ID: p.ID}, "close")
	close(release)
	awaitProjectStatus(t, s, "completed")
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, expected) {
		t.Fatal("closing preview corrupted admitted PNG", err)
	}
	// Full-document tiling is bounded and publishes no partial asset.
	s.projectExport = prepareProjectExport
	call(t, s, "component.move", MoveParams{ID: "C1", Position: Point{X: 10000, Y: 10000}}, "huge")
	path = filepath.Join(t.TempDir(), "too-many.pdf")
	refuse(t, s, "export.schematic.pdf", PDFExportParams{Path: path}, "file-error")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("partial oversized PDF published", err)
	}
}
