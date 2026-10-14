package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func nativePrintPreviewSmoke(ctx context.Context, a *app.Application, s *screen) error {
	action := make(chan error, 1)
	activate := func(id string) error {
		if err := a.PostWait(ctx, func() { action <- smokeUI(s, id, nil) }); err != nil {
			return err
		}
		return <-action
	}
	if err := a.PostWait(ctx, func() {
		r := s.invoke("example.rectifier", workbench.ReplaceParams{DiscardChanges: true})
		if r.Error != nil {
			action <- fmt.Errorf("preview example: %s", r.Error.Message)
			return
		}
		action <- smokeUI(s, "simulation.pause", nil)
	}); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := activate("print-preview"); err != nil {
		return err
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		ready := make(chan bool, 1)
		if err := a.PostWait(ctx, func() {
			state := s.service.Snapshot()
			root := s.view()
			if state.PrintPreview != nil && state.PrintPreview.Error != nil {
				action <- fmt.Errorf("preview: %s", state.PrintPreview.Error.Message)
				ready <- true
				return
			}
			node, ok := findAction(root, "preview-image")
			ready <- ok && node.Kind == ui.Image && node.Image != nil
			action <- nil
		}); err != nil {
			return err
		}
		ok := <-ready
		err := <-action
		if err != nil {
			return err
		}
		if ok {
			break
		}
	}
	p := s.service.Snapshot().PrintPreview
	if !p.Current || p.Actor != "user" || p.Width != 1650 || p.Height != 1275 {
		return fmt.Errorf("preview capture identity: %+v", p)
	}
	if err := a.PostWait(ctx, func() {
		r := s.invoke("component.set", workbench.ValueParams{ID: "RL", Value: "20k"})
		if r.Error != nil {
			action <- fmt.Errorf("preview edit: %s", r.Error.Message)
			return
		}
		if s.service.Snapshot().PrintPreview.Current {
			action <- fmt.Errorf("preview lost stale status")
			return
		}
		action <- nil
	}); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	if err := activate("preview-close"); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		root := s.view()
		_, ok := findAction(root, "schematic")
		if !ok || s.service.Snapshot().PrintPreview.Visible || s.previewPixels.image != nil {
			action <- fmt.Errorf("preview close failed to restore editor")
			return
		}
		action <- nil
	}); err != nil {
		return err
	}
	if err := <-action; err != nil {
		return err
	}
	rebuilt := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if err := a.PostWait(ctx, func() {
			if !rebuilt {
				if err := s.drawing.RebuildResources(); err != nil {
					action <- err
					return
				}
				rebuilt = true
			}
			pixels, err := s.drawing.Snapshot()
			if err != nil {
				action <- err
				return
			}
			ink := 0
			for y := pixels.Bounds().Min.Y; y < pixels.Bounds().Max.Y; y++ {
				for x := pixels.Bounds().Min.X; x < pixels.Bounds().Max.X; x++ {
					p := pixels.NRGBAAt(x, y)
					if p.A > 200 && p.R < 120 && p.G > 60 && p.B > 60 {
						ink++
					}
				}
			}
			if ink < 200 {
				action <- cogent.ErrFrameNotReady
				return
			}
			action <- nil
		}); err != nil {
			return err
		}
		err := <-action
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		return err
	}
}
