package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

type previewPixels struct {
	id      string
	image   *canvas.TiledImage
	loading bool
	err     string
}

// Fetch through the public chunk query, just like an external agent. Decode and
// tile off the owner thread; only publish if this captured preview is still open.
func (s *screen) loadPrintPreview(p *workbench.PrintPreview) {
	if p.Status != "completed" || s.previewPixels.id == p.ID {
		return
	}
	s.previewPixels = previewPixels{id: p.ID, loading: true}
	captured := *p
	go func() {
		ctx, cancel := context.WithTimeout(s.app.Context(), 10*time.Second)
		defer cancel()
		pixels, err := readPrintPreview(ctx, s.service, captured)
		_ = s.app.PostWindow("main", func() {
			current := s.service.Snapshot().PrintPreview
			if current == nil || !current.Visible || current.ID != captured.ID || s.previewPixels.id != captured.ID {
				return
			}
			s.previewPixels.loading = false
			s.previewPixels.image = pixels
			if err != nil {
				s.previewPixels.err = err.Error()
			}
			s.app.Invalidate("main")
		})
	}()
}
func readPrintPreview(ctx context.Context, service *workbench.Service, p workbench.PrintPreview) (*canvas.TiledImage, error) {
	if p.Bytes <= 0 || p.Bytes > workbench.MaxProjectBytes {
		return nil, fmt.Errorf("invalid preview size")
	}
	data := make([]byte, 0, p.Bytes)
	for len(data) < p.Bytes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(workbench.PreviewImageParams{ID: p.ID, Offset: len(data)})
		response := service.Dispatch(ctx, workbench.Request{Version: 1, ID: fmt.Sprintf("preview-image-%s-%d", p.ID, len(data)), Command: "print.preview.image", Params: raw}, "user")
		if response.Error != nil {
			return nil, fmt.Errorf("%s", response.Error.Message)
		}
		chunk, ok := response.Result.(workbench.PreviewImageChunk)
		if !ok || chunk.ID != p.ID || chunk.Offset != len(data) || chunk.Total != p.Bytes || chunk.SHA256 != p.SHA256 || chunk.Next != len(data)+len(chunk.PNG) || chunk.Next > p.Bytes || len(chunk.PNG) == 0 {
			return nil, fmt.Errorf("invalid preview chunk")
		}
		data = append(data, chunk.PNG...)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != p.SHA256 {
		return nil, fmt.Errorf("preview checksum mismatch")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width != p.Width || cfg.Height != p.Height || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16*1024*1024 {
		return nil, fmt.Errorf("invalid preview dimensions")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return ui.DecodeImage(bytes.NewReader(data))
}
func previewVisible(state workbench.Snapshot) bool {
	return state.PrintPreview != nil && state.PrintPreview.Visible
}
func previewActive(p *workbench.PrintPreview) bool {
	return p.Status == "queued" || p.Status == "running" || p.Status == "canceling"
}
func (s *screen) printPreviewPanel(state workbench.Snapshot) []ui.Node {
	p := state.PrintPreview
	s.loadPrintPreview(p)
	refresh := s.action("preview-refresh", "Refresh preview", "print.preview.start", workbench.PreviewParams{})
	refresh.Disabled = previewActive(p)
	toolbar := ui.Node{ID: "preview-tools", Kind: ui.Toolbar, Style: chromeStyle(), Children: []ui.Node{heading("preview-title", "Print preview"), refresh, s.action("preview-close", "Back to circuit", "print.preview.close", workbench.IDParams{ID: p.ID})}}
	if previewActive(p) {
		cancel := s.action("preview-cancel", "Cancel", "print.preview.cancel", workbench.IDParams{ID: p.ID})
		cancel.Disabled = p.Status == "canceling"
		toolbar.Children = append(toolbar.Children, cancel)
	}
	text := fmt.Sprintf("%s · %s · captured circuit", p.Page.Paper, p.Page.Orientation)
	if !p.Current {
		text += " · source changed; refresh to update"
	}
	body := quiet("preview-progress", "Preparing page…")
	if p.Error != nil {
		body = quiet("preview-error", p.Error.Message)
	} else if p.Status == "canceled" {
		body = quiet("preview-canceled", "Preview canceled. Refresh to try again.")
	} else if p.Status == "completed" {
		if s.previewPixels.err != "" {
			body = quiet("preview-image-error", s.previewPixels.err)
		} else if s.previewPixels.image != nil {
			body = ui.Node{ID: "preview-image", Kind: ui.Image, Image: s.previewPixels.image, Fit: ui.ImageContain, Grow: true, Text: "Captured schematic page", Accessibility: ui.Accessibility{Name: "Captured schematic page"}, Style: ui.Style{MinSize: ui.Some(ui.Size{Width: 240, Height: 160}), Background: ui.Some(surface)}}
		} else {
			body = quiet("preview-image-loading", "Loading captured page…")
		}
	}
	body.Grow = true
	return []ui.Node{toolbar, quiet("preview-caption", text), body, quiet("preview-help", "Print captured page submits this page. Export PDF creates a printable file from the current circuit.")}
}
