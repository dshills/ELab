package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"

	"github.com/dshills/elab/internal/schematicpdf"
)

var previewContractCommands = []string{"print.preview.start", "print.preview.cancel", "print.preview.close", "print.preview.image"}

func TestPreviewContractsAreOwnedAndKeepDefaultDiscovery(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range previewContractCommands {
		details, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(details[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		c := schema["x-command-contract"].(map[string]any)
		if c["scope"] != "captured-print-preview" || len(c["effects"].([]any)) == 0 || len(c["errors"].([]any)) == 0 || schema["x-preview-inventory"] == nil {
			t.Fatal("missing preview state contract", id)
		}
		if id == "print.preview.start" && schema["x-page-selection"] == nil {
			t.Fatal("preview contract dropped existing tiled-page guidance")
		}
		details[0].Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("caller changed later detailed preview schema", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("preview contracts changed ordinary discovery or outputs")
	}
}

func TestPreviewImageEOFNullDefaultsOwnershipAndReadiness(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "print.preview.start", map[string]any{"page": nil, "page_index": nil}, "preview")
	p := awaitPreview(t, s, "completed").PrintPreview
	before := s.Snapshot()
	chunk := call(t, s, "print.preview.image", map[string]any{"id": p.ID, "offset": nil, "limit": nil}, "nulls").Result.(PreviewImageChunk)
	if chunk.Offset != 0 || chunk.Next != min(p.Bytes, 65536) || chunk.Total != p.Bytes || chunk.SHA256 != p.SHA256 || len(chunk.PNG) == 0 {
		t.Fatal("nullable/default byte chunk differs from discovery")
	}
	original := append([]byte{}, chunk.PNG...)
	chunk.PNG[0] ^= 255
	again := call(t, s, "print.preview.image", PreviewImageParams{ID: p.ID}, "again").Result.(PreviewImageChunk)
	if !bytes.Equal(original, again.PNG) {
		t.Fatal("returned byte chunk aliases preview pixels")
	}
	eof := call(t, s, "print.preview.image", PreviewImageParams{ID: p.ID, Offset: p.Bytes}, "eof").Result.(PreviewImageChunk)
	if len(eof.PNG) != 0 || eof.Next != p.Bytes || eof.Total != p.Bytes || eof.SHA256 != p.SHA256 {
		t.Fatal("inclusive EOF did not return an empty terminal chunk")
	}
	for _, params := range []PreviewImageParams{{ID: p.ID, Offset: -1}, {ID: p.ID, Offset: p.Bytes + 1}, {ID: p.ID, Limit: -1}, {ID: p.ID, Limit: 65537}} {
		refuse(t, s, "print.preview.image", params, "invalid-params")
	}
	refuse(t, s, "print.preview.image", PreviewImageParams{ID: "old", Offset: -1}, "not-found")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("chunk query/refusal changed source or preview state")
	}
	call(t, s, "print.preview.close", IDParams{ID: p.ID}, "close")
	closed := s.Snapshot()
	if closed.PrintPreview.Visible || closed.PrintPreview.Status != "completed" || closed.PrintPreview.Bytes != 0 || closed.PrintPreview.SHA256 != "" || closed.PrintPreview.Width != p.Width {
		t.Fatal("close did not retain metadata while releasing bytes")
	}
	refuse(t, s, "print.preview.image", PreviewImageParams{ID: p.ID, Offset: -1}, "not-ready")
	refuse(t, s, "print.preview.cancel", IDParams{ID: p.ID}, "unavailable")
	call(t, s, "print.preview.close", IDParams{ID: p.ID}, "close-again")
	if !reflect.DeepEqual(closed.PrintPreview, s.Snapshot().PrintPreview) {
		t.Fatal("repeat close changed retained metadata")
	}
}

func TestPreviewCancelAndCloseKeepWorkerSlotUntilActualReturn(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer func() { once.Do(func() { close(release) }) }()
	s.mu.Lock()
	s.previewRender = func(ctx context.Context, _ exportCapture) ([]byte, error) {
		close(entered)
		<-release // Deliberately model an OS renderer that has not returned yet.
		return nil, ctx.Err()
	}
	s.mu.Unlock()
	call(t, s, "print.preview.start", PreviewParams{}, "start")
	awaitLiveSampleBlock(t, entered)
	id := s.Snapshot().PrintPreview.ID
	call(t, s, "print.preview.cancel", IDParams{ID: id}, "cancel")
	call(t, s, "print.preview.cancel", IDParams{ID: id}, "cancel-again")
	refuse(t, s, "print.preview.start", PreviewParams{}, "busy")
	call(t, s, "print.preview.close", IDParams{ID: id}, "close")
	call(t, s, "print.preview.close", IDParams{ID: id}, "close-again")
	refuse(t, s, "print.preview.start", PreviewParams{}, "busy")
	if p := s.Snapshot().PrintPreview; p.Visible || p.Status != "canceling" || p.Bytes != 0 {
		t.Fatal("active close released worker or published partial pixels")
	}
	once.Do(func() { close(release) })
	awaitPreview(t, s, "canceled")
	s.mu.Lock()
	s.previewRender = renderSchematicPage
	s.mu.Unlock()
	call(t, s, "print.preview.start", PreviewParams{}, "new")
	newPreview := awaitPreview(t, s, "completed").PrintPreview
	if newPreview.ID == id || newPreview.Bytes == 0 {
		t.Fatal("renderer could not admit fresh work after actual return")
	}
	refuse(t, s, "print.preview.close", IDParams{ID: id}, "not-found")
}
