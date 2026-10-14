package workbench

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/internal/schematicpdf"
)

func TestSchematicPageBoundsAndOwnership(t *testing.T) {
	page, layout, err := normalizeSchematicPage(nil)
	if err != nil || page.Paper != "letter" || page.Orientation != "landscape" || layout.Width != 792 || layout.Height != 612 || !layout.Grayscale {
		t.Fatal("page defaults", page, layout, err)
	}
	for _, bad := range []SchematicPage{{Paper: "bad"}, {Orientation: "bad"}, {MarginMM: ptrFloat(-1)}, {MarginMM: ptrFloat(math.NaN())}, {Scale: ptrFloat(0)}, {Scale: ptrFloat(5)}} {
		if _, _, err := normalizeSchematicPage(&bad); err == nil {
			t.Fatal("invalid page accepted", bad)
		}
	}
	input := SchematicPage{MarginMM: ptrFloat(12), Scale: ptrFloat(.5)}
	copy, _, err := normalizeSchematicPage(&input)
	if err != nil {
		t.Fatal(err)
	}
	*input.MarginMM = 25
	*input.Scale = 2
	if *copy.MarginMM != 12 || *copy.Scale != .5 {
		t.Fatal("page capture aliases caller")
	}
}
func TestPrintableSchematicExportPreservesProjectAndPhysics(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS PDF backend")
	}
	for _, info := range ExampleCatalog() {
		t.Run(info.ID, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.load", ExampleLoadParams{ID: info.ID}, "example")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			before := settled(t, s)
			path := filepath.Join(t.TempDir(), "schematic.pdf")
			call(t, s, "export.schematic.pdf", PDFExportParams{Path: path}, "pdf")
			after := s.Snapshot()
			data, err := os.ReadFile(path)
			if err != nil || !strings.HasPrefix(string(data), "%PDF-") || after.LastExport == nil || after.LastExport.Format != "pdf" || after.LastExport.Page == nil {
				t.Fatal("missing PDF/identity", err)
			}
			if before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
				t.Fatal("PDF export changes source/physics/history")
			}
			*after.LastExport.Page.MarginMM = 39
			if *s.Snapshot().LastExport.Page.MarginMM != 10 {
				t.Fatal("artifact page aliases snapshot caller")
			}
			refuse(t, s, "export.schematic.pdf", PDFExportParams{Path: path}, "file-exists")
		})
	}
	// Incomplete drawings are documentation too; explicit oversized scale must
	// refuse without clipping or leaving an output file.
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{ID: "R1", Kind: "resistor", Position: Point{X: 2000}}, "part")
	path := filepath.Join(t.TempDir(), "incomplete.pdf")
	call(t, s, "export.schematic.pdf", PDFExportParams{Path: path}, "incomplete")
	large := filepath.Join(t.TempDir(), "clipped.pdf")
	before := s.Snapshot()
	raw := PDFExportParams{Path: large, Page: &SchematicPage{Scale: ptrFloat(4)}}
	refuse(t, s, "export.schematic.pdf", raw, "file-error")
	after := s.Snapshot()
	if before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) || before.RunID != after.RunID {
		t.Fatal("failed scale changed source")
	}
	if _, err := os.Stat(large); !os.IsNotExist(err) {
		t.Fatal("clipped PDF published", err)
	}
}
func TestCanceledPrintableEncodingStopsBeforePublication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := schematicpdf.Render(ctx, schematicSVG(ExampleGraph(), 1, 1), schematicpdf.Page{Width: 792, Height: 612, Margin: 30})
	if err != context.Canceled {
		t.Fatal("canceled renderer continued", err)
	}
}

func ptrFloat(v float64) *float64 { return &v }
