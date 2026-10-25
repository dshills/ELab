package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"strings"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/text/shaped"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
)

type exampleThumbnails struct {
	requested bool
	images    map[string]*canvas.TiledImage
	errors    map[string]string
}

func queryExamplePreview(ctx context.Context, service *workbench.Service, id string) (workbench.ExamplePreview, error) {
	raw, _ := json.Marshal(workbench.IDParams{ID: id})
	r := service.Dispatch(ctx, workbench.Request{Version: 1, ID: "example-thumbnail-" + id, Command: "example.preview", Params: raw}, "user")
	if r.Error != nil {
		return workbench.ExamplePreview{}, fmt.Errorf("%s", r.Error.Message)
	}
	preview, ok := r.Result.(workbench.ExamplePreview)
	if !ok || preview.Info.ID != id {
		return workbench.ExamplePreview{}, fmt.Errorf("invalid template preview response")
	}
	return preview, nil
}

// Small, bounded CPU renders use the same pure SVG query as external agents.
// No live document, solver, filesystem or network image source is involved.
func renderExampleThumbnail(p workbench.ExamplePreview) (*canvas.TiledImage, error) {
	if len(p.SVG) == 0 || len(p.SVG) > 256*1024 {
		return nil, fmt.Errorf("template drawing exceeds thumbnail bounds")
	}
	size := math32.Vec2(480, 300)
	drawing := svg.NewSVG(size)
	// Shaper outputs may be reused by its next call. Keep this worker's
	// temporary runs separate from concurrent native icon/text rendering.
	drawing.TextShaper = shaped.NewShaper()
	if err := drawing.ReadXML(strings.NewReader(p.SVG)); err != nil {
		return nil, err
	}
	drawing.SetSize(size)
	drawing.Root.ViewBox.PreserveAspectRatio.Align.Set(svg.AlignMid)
	drawing.Root.ViewBox.PreserveAspectRatio.MeetOrSlice = svg.Meet
	renderer := paint.NewImageRenderer(size)
	pixels := renderer.Render(drawing.Render(nil).RenderDone()).Image()
	out := image.NewNRGBA(image.Rect(0, 0, 480, 300))
	draw.Draw(out, out.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), pixels, pixels.Bounds().Min, draw.Over)
	return canvas.NewTiledImage(out)
}

func (s *screen) requestExampleThumbnails() {
	if s.exampleThumbnails.requested {
		return
	}
	window, err := s.windowReference()
	if err != nil {
		return // Unmounted semantic fixtures do not start presentation workers.
	}
	s.exampleThumbnails.requested = true
	go func() {
		images := make(map[string]*canvas.TiledImage)
		failures := make(map[string]string)
		for _, info := range workbench.ExampleCatalog() {
			if window.Context().Err() != nil {
				return
			}
			preview, err := queryExamplePreview(window.Context(), s.service, info.ID)
			var pixels *canvas.TiledImage
			if err == nil {
				pixels, err = renderExampleThumbnail(preview)
			}
			if err != nil {
				failures[info.ID] = err.Error()
			} else {
				images[info.ID] = pixels
			}
		}
		_ = s.app.PostWindow(window, func() {
			s.exampleThumbnails.images, s.exampleThumbnails.errors = images, failures
			s.app.Invalidate("main")
		})
	}()
}
