package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/internal/schematicpdf"
)

func awaitPreview(t *testing.T, s *Service, status string) Snapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot()
		if state.PrintPreview != nil && state.PrintPreview.Status == status {
			return state
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("preview did not reach %s: %+v", status, s.Snapshot().PrintPreview)
	return Snapshot{}
}
func previewBytes(t *testing.T, s *Service, p *PrintPreview) []byte {
	t.Helper()
	var data []byte
	for len(data) < p.Bytes {
		response := call(t, s, "print.preview.image", PreviewImageParams{ID: p.ID, Offset: len(data), Limit: 4096}, "chunk")
		chunk := response.Result.(PreviewImageChunk)
		if chunk.Next != len(data)+len(chunk.PNG) || chunk.Total != p.Bytes || chunk.SHA256 != p.SHA256 {
			t.Fatal("chunk identity mismatch")
		}
		data = append(data, chunk.PNG...)
	}
	return data
}
func TestPrintPreviewMatchesExportAndPreservesCircuit(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "export.page.set", SchematicPage{Paper: "a4", Orientation: "portrait", MarginMM: ptrFloat(12), Color: true}, "page")
	before := s.Snapshot()
	call(t, s, "print.preview.start", PreviewParams{}, "start")
	after := awaitPreview(t, s, "completed")
	p := after.PrintPreview
	if !p.Current || !p.Visible || p.Width != 1241 || p.Height != 1754 || p.Actor != "test" || p.DocumentGeneration != before.Document.Generation || p.RunID != before.RunID {
		t.Fatalf("capture identity: %+v", p)
	}
	if before.Document != after.Document || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.Components, after.Components) {
		t.Fatal("preview changed physics/source/history")
	}
	data := previewBytes(t, s, p)
	path := filepath.Join(t.TempDir(), "page.png")
	call(t, s, "export.schematic.png", PNGExportParams{Path: path}, "export")
	exported, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, exported) {
		t.Fatal("preview/export differ", err)
	}
	// Queries return owned chunks and snapshots, never inline the whole image.
	chunk := call(t, s, "print.preview.image", PreviewImageParams{ID: p.ID, Limit: 10}, "owned").Result.(PreviewImageChunk)
	chunk.PNG[0] = 0
	if !bytes.Equal(data, previewBytes(t, s, p)) {
		t.Fatal("query aliases retained pixels")
	}
	*after.PrintPreview.Page.MarginMM = 30
	if *s.Snapshot().PrintPreview.Page.MarginMM != 12 {
		t.Fatal("snapshot aliases page")
	}
	raw, _ := json.Marshal(s.Snapshot())
	if len(raw) > 512*1024 || bytes.Contains(raw, []byte("png_base64")) {
		t.Fatal("image leaked into snapshot")
	}
	refuse(t, s, "print.preview.image", PreviewImageParams{ID: p.ID, Limit: 65537}, "invalid-params")
	call(t, s, "component.move", MoveParams{ID: "R1", Position: Point{X: 100, Y: 100}}, "move")
	if s.Snapshot().PrintPreview.Current || !bytes.Equal(data, previewBytes(t, s, p)) {
		t.Fatal("edited source changed captured pixels or failed stale flag")
	}
	call(t, s, "print.preview.start", PreviewParams{}, "refresh")
	newP := awaitPreview(t, s, "completed").PrintPreview
	if !newP.Current || newP.ID == p.ID {
		t.Fatal("refresh identity")
	}
	refuse(t, s, "print.preview.close", IDParams{ID: p.ID}, "not-found")
	call(t, s, "print.preview.close", IDParams{ID: newP.ID}, "close")
	if s.Snapshot().PrintPreview.Visible || s.Snapshot().PrintPreview.Bytes != 0 {
		t.Fatal("close retained pixels")
	}
	refuse(t, s, "print.preview.image", PreviewImageParams{ID: newP.ID}, "not-ready")
}
func TestPrintPreviewCancellationCaptureAndCloseJoin(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	entered := make(chan exportCapture, 1)
	release := make(chan struct{})
	s.previewRender = func(ctx context.Context, capture exportCapture) ([]byte, error) {
		entered <- capture
		select {
		case <-release:
			return renderSchematicPage(ctx, capture)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call(t, s, "print.preview.start", PreviewParams{}, "start")
	capture := <-entered
	if capture.graph.Components[0].ID != before.Components[0].ID {
		t.Fatal("capture missing")
	}
	p := s.Snapshot().PrintPreview
	refuse(t, s, "print.preview.start", PreviewParams{}, "busy")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	if s.Snapshot().PrintPreview.Current {
		t.Fatal("stale capture shown current")
	}
	call(t, s, "print.preview.cancel", IDParams{ID: p.ID}, "cancel")
	awaitPreview(t, s, "canceled")
	refuse(t, s, "print.preview.image", PreviewImageParams{ID: p.ID}, "not-ready")
	// A new capture owns the edited graph and close cancels and joins its worker.
	call(t, s, "print.preview.start", PreviewParams{}, "again")
	<-entered
	done := make(chan struct{})
	go func() { s.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close did not join canceled render")
	}
	if s.previewWorkers != 0 || len(s.preview.data) != 0 || s.preview.view.Visible {
		t.Fatal("worker/image survived close")
	}
}
func TestPrintPreviewErrorAndExplicitPageOwnership(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	s.previewRender = func(context.Context, exportCapture) ([]byte, error) { return nil, context.DeadlineExceeded }
	page := SchematicPage{Paper: "a4", MarginMM: ptrFloat(12)}
	call(t, s, "print.preview.start", PreviewParams{Page: &page}, "start")
	*page.MarginMM = 20
	p := awaitPreview(t, s, "failed").PrintPreview
	if p.Error.Code != "preview-timeout" || *p.Page.MarginMM != 12 {
		t.Fatalf("deadline/page ownership: %+v", p)
	}
	call(t, s, "export.page.set", SchematicPage{Paper: "letter"}, "defaults")
	if !s.Snapshot().PrintPreview.Current {
		t.Fatal("unrelated defaults stale explicit-page preview")
	}
	refuse(t, s, "print.preview.cancel", IDParams{ID: p.ID}, "unavailable")
}

func TestBlockedPreviewRetainsOriginalPixelsAndPublishesEvents(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "print.preview.start", PreviewParams{}, "baseline")
	baseline := awaitPreview(t, s, "completed")
	data := previewBytes(t, s, baseline.PrintPreview)
	entered, release := make(chan struct{}), make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	s.previewRender = func(ctx context.Context, c exportCapture) ([]byte, error) {
		close(entered)
		select {
		case <-release:
			return renderSchematicPage(ctx, c)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	start := call(t, s, "print.preview.start", PreviewParams{}, "captured").Result.(Snapshot)
	<-entered
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	close(release)
	released = true
	after := awaitPreview(t, s, "completed")
	if after.PrintPreview.Current || !bytes.Equal(data, previewBytes(t, s, after.PrintPreview)) {
		t.Fatal("worker read later source or lost stale provenance")
	}
	zero := 0
	p := EventPollParams{AfterRevision: start.Revision, TimeoutMS: &zero}
	events := call(t, s, "events.poll", p, "events").Result.(EventPollResult)
	found := false
	for _, event := range events.Publications {
		if event.Topic == "preview" {
			found = true
		}
	}
	if !found {
		t.Fatal("preview completion missing publication")
	}
}

func TestPrintPreviewUnsupportedAndBase64Discovery(t *testing.T) {
	cap := capabilityIndex["print.preview.image"]
	var schema map[string]any
	if err := json.Unmarshal(cap.OutputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	field := schema["properties"].(map[string]any)["png_base64"].(map[string]any)
	if field["type"] != "string" || field["contentEncoding"] != "base64" {
		t.Fatal("PNG discovery disagrees with JSON byte encoding")
	}
	if !schematicpdf.Supported() {
		s := New()
		defer s.Close()
		refuse(t, s, "print.preview.start", PreviewParams{}, "unsupported")
		if s.Snapshot().PrintPreview != nil {
			t.Fatal("unsupported renderer admitted work")
		}
	}
}
