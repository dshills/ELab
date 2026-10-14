package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"time"

	"github.com/dshills/elab/internal/schematicpdf"
)

type PreviewParams struct {
	Page *SchematicPage `json:"page,omitempty"`
}
type PrintPreview struct {
	ID                 string        `json:"id"`
	Status             string        `json:"status"`
	Visible            bool          `json:"visible"`
	Current            bool          `json:"source_current"`
	Page               SchematicPage `json:"page"`
	RunID              uint64        `json:"run_id"`
	ElectricalRevision uint64        `json:"electrical_revision"`
	DocumentGeneration uint64        `json:"document_generation"`
	Actor              string        `json:"actor"`
	Width              int           `json:"width_px"`
	Height             int           `json:"height_px"`
	Bytes              int           `json:"bytes"`
	SHA256             string        `json:"sha256,omitempty"`
	Error              *Failure      `json:"error,omitempty"`
}
type PreviewImageParams struct {
	ID     string `json:"id"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type PreviewImageChunk struct {
	ID     string `json:"id"`
	Offset int    `json:"offset"`
	Next   int    `json:"next_offset"`
	Total  int    `json:"total"`
	SHA256 string `json:"sha256"`
	PNG    []byte `json:"png_base64"`
}
type previewJob struct {
	view        PrintPreview
	ctx         context.Context
	cancel      context.CancelFunc
	capture     exportCapture
	data        []byte
	sourceHash  [32]byte
	defaultPage bool
}

func clonePrintPreview(in *PrintPreview) *PrintPreview {
	if in == nil {
		return nil
	}
	out := *in
	out.Page = cloneSchematicPage(in.Page)
	if in.Error != nil {
		e := *in.Error
		out.Error = &e
	}
	return &out
}

func (s *Service) previewSourceHashLocked(page SchematicPage) [32]byte {
	value := struct {
		Content     [32]byte
		Annotations []SourceAnnotation
		Display     *CircuitDisplay
		Readings    []PinnedReading
	}{Content: contentHash(s.graph, s.transientSettings), Annotations: s.sourceAnnotationsLocked()}
	if page.Measurements {
		d := s.circuitDisplayLocked()
		d.Animate = false
		value.Display, value.Readings = &d, s.pinnedReadingsLocked()
	}
	data, _ := json.Marshal(value)
	return sha256.Sum256(data)
}
func (s *Service) printPreviewLocked() *PrintPreview {
	if s.preview == nil {
		return nil
	}
	out := clonePrintPreview(&s.preview.view)
	out.Current = out.DocumentGeneration == s.documentGeneration && s.preview.sourceHash == s.previewSourceHashLocked(out.Page)
	if s.preview.defaultPage {
		page, _, _ := normalizeSchematicPage(&s.exportPage)
		a, _ := json.Marshal(page)
		b, _ := json.Marshal(out.Page)
		out.Current = out.Current && bytes.Equal(a, b)
	}
	return out
}
func (s *Service) startPrintPreviewLocked(p PreviewParams) error {
	if s.previewWorkers != 0 {
		return &actionError{"busy", "Finish or cancel the active preview before refreshing"}
	}
	if !schematicpdf.Supported() {
		return &actionError{"unsupported", "Print preview requires the macOS CoreGraphics build"}
	}
	settings := p.Page
	if settings == nil {
		settings = &s.exportPage
	}
	page, layout, err := normalizeSchematicPage(settings)
	if err != nil {
		return err
	}
	layout.RasterDPI = 150
	capture := exportCapture{kind: "schematic-png", graph: s.graph.Clone(), annotations: s.sourceAnnotationsLocked(), settings: s.transientSettings, page: layout, artifact: Artifact{Format: "png", Page: &page, DPI: 150, RunID: s.run, ElectricalRevision: s.electrical, DocumentGeneration: s.documentGeneration}, display: CircuitDisplay{Preferences: DefaultDisplayPreferences()}}
	if page.Measurements {
		capture.display = cloneCircuitDisplay(s.circuitDisplayLocked())
		capture.display.Animate = false
		capture.readings = clonePinnedReadings(s.pinnedReadingsLocked())
	}
	s.previewSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	job := &previewJob{ctx: ctx, cancel: cancel, capture: capture, defaultPage: p.Page == nil, sourceHash: s.previewSourceHashLocked(page), view: PrintPreview{ID: fmt.Sprintf("preview-%d", s.previewSerial), Status: "queued", Visible: true, Page: page, RunID: s.run, ElectricalRevision: s.electrical, DocumentGeneration: s.documentGeneration}}
	s.preview = job
	if s.sweeps != nil {
		s.sweeps.visible = false
	}
	s.previewWorkers++
	s.liveWorkers.Add(1)
	render := s.previewRender
	go s.runPrintPreview(job, render)
	return nil
}
func (s *Service) runPrintPreview(job *previewJob, render func(context.Context, exportCapture) ([]byte, error)) {
	defer s.liveWorkers.Done()
	s.mu.Lock()
	if job.ctx.Err() == nil {
		job.view.Status = "running"
		s.revision++
		s.signalLocked("preview")
	}
	s.mu.Unlock()
	data, err := render(job.ctx, job.capture)
	var width, height int
	if err == nil {
		if len(data) > MaxProjectBytes {
			err = fmt.Errorf("preview exceeds 8 MiB")
		} else {
			var cfgErr error
			cfg, cfgErr := png.DecodeConfig(bytes.NewReader(data))
			err = cfgErr
			width, height = cfg.Width, cfg.Height
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.previewWorkers--
	if s.preview != job || s.closed || errors.Is(job.ctx.Err(), context.Canceled) || job.view.Status == "canceling" {
		job.view.Status = "canceled"
	} else if errors.Is(job.ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		job.view.Status = "failed"
		job.view.Error = &Failure{Code: "preview-timeout", Message: "Print preview exceeded its 30-second rendering limit"}
	} else if err != nil {
		job.view.Status = "failed"
		job.view.Error = &Failure{Code: "preview-failed", Message: err.Error()}
	} else {
		job.data = data
		sum := sha256.Sum256(data)
		job.view.Status = "completed"
		job.view.Width = width
		job.view.Height = height
		job.view.Bytes = len(data)
		job.view.SHA256 = hex.EncodeToString(sum[:])
	}
	job.capture = exportCapture{}
	job.cancel()
	s.revision++
	s.signalLocked("preview")
}
func (s *Service) cancelPrintPreviewLocked(p IDParams, close bool) error {
	if s.preview == nil || p.ID != s.preview.view.ID {
		return &actionError{"not-found", "Preview is no longer current; inspect before closing or canceling"}
	}
	if !close && !operationActive(s.preview.view.Status) {
		return &actionError{"unavailable", "Only an active preview can be canceled"}
	}
	if operationActive(s.preview.view.Status) {
		s.preview.view.Status = "canceling"
		s.preview.cancel()
	}
	if close {
		s.preview.view.Visible = false
		s.preview.data = nil
		s.preview.view.Bytes = 0
		s.preview.view.SHA256 = ""
	}
	return nil
}
func (s *Service) previewImageLocked(p PreviewImageParams) (PreviewImageChunk, error) {
	if s.preview == nil || p.ID != s.preview.view.ID {
		return PreviewImageChunk{}, &actionError{"not-found", "Preview was replaced or closed"}
	}
	j := s.preview
	if !j.view.Visible || j.view.Status != "completed" || len(j.data) == 0 {
		return PreviewImageChunk{}, &actionError{"not-ready", "A visible completed preview is required"}
	}
	if p.Limit == 0 {
		p.Limit = 65536
	}
	if p.Offset < 0 || p.Offset > len(j.data) || p.Limit < 1 || p.Limit > 65536 {
		return PreviewImageChunk{}, fmt.Errorf("preview chunks require valid offset and 1–65536 bytes")
	}
	end := min(len(j.data), p.Offset+p.Limit)
	return PreviewImageChunk{ID: p.ID, Offset: p.Offset, Next: end, Total: len(j.data), SHA256: j.view.SHA256, PNG: append([]byte{}, j.data[p.Offset:end]...)}, nil
}
