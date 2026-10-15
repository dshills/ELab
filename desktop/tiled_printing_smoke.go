package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeTiledPrintingSmoke(ctx context.Context, a *app.Application, s *screen) error {
	owner := func(fn func() error) error {
		done := make(chan error, 1)
		if err := a.PostWait(ctx, func() { done <- fn() }); err != nil {
			return err
		}
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := owner(func() error {
		for _, r := range []struct {
			id string
			p  any
		}{{"example.rc", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.pause", struct{}{}}, {"component.move", workbench.MoveParams{ID: "C1", Position: workbench.Point{X: 800, Y: 600}}}, {"export.page.set", workbench.SchematicPage{}}} {
			if response := s.invoke(r.id, r.p); response.Error != nil {
				return fmt.Errorf("tiled setup: %s", response.Error.Message)
			}
		}
		if err := smokeUI(s, "pdf-tile", &ui.ChangeEvent{Checked: true}); err != nil {
			return err
		}
		if err := smokeUI(s, "pdf-overlap", &ui.ChangeEvent{Number: 5}); err != nil {
			return err
		}
		return smokeUI(s, "print-preview", nil)
	}); err != nil {
		return err
	}
	waitPage := func(index int) error {
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
			ready := false
			err := owner(func() error {
				p := s.service.Snapshot().PrintPreview
				if p == nil {
					return fmt.Errorf("missing tiled preview")
				}
				if p.Error != nil {
					return fmt.Errorf("tiled preview: %s", p.Error.Message)
				}
				node, ok := findAction(s.view(), "preview-image")
				ready = p.Status == "completed" && p.PageIndex == index && p.PageCount >= 4 && ok && node.Image != nil
				return nil
			})
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		}
	}
	if err := waitPage(0); err != nil {
		return err
	}
	if err := owner(func() error { return smokeUI(s, "preview-next", nil) }); err != nil {
		return err
	}
	if err := waitPage(1); err != nil {
		return err
	}
	if err := owner(func() error { return smokeUI(s, "preview-previous", nil) }); err != nil {
		return err
	}
	if err := waitPage(0); err != nil {
		return err
	}
	return owner(func() error {
		if err := smokeUI(s, "preview-close", nil); err != nil {
			return err
		}
		return smokeUI(s, "pdf-fit", nil)
	})
}
