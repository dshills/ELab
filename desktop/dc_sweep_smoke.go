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

func nativeDCSweepSmoke(ctx context.Context, a *app.Application, s *screen) error {
	result := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if r := s.invoke("example.reset", workbench.ReplaceParams{DiscardChanges: true}); r.Error != nil {
			result <- fmt.Errorf("sweep example: %s", r.Error.Message)
			return
		}
		result <- smokeUI(s, "simulation.pause", nil)
	}); err != nil {
		return err
	}
	if err := <-result; err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	before := s.service.Snapshot()
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"sweep-probe", &ui.ChangeEvent{SelectedID: "i:D1"}}, {"sweep-points", &ui.ChangeEvent{Text: "11"}}, {"sweep-start", nil}} {
			if err := smokeUI(s, action.id, action.event); err != nil {
				result <- err
				return
			}
		}
		result <- nil
	}); err != nil {
		return err
	}
	if err := <-result; err != nil {
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
		state := s.service.Snapshot()
		if state.DCSweep == nil {
			continue
		}
		op := selectedSweep(state.DCSweep)
		if op != nil && op.Error != nil {
			return fmt.Errorf("native sweep failed: %s", op.Error.Message)
		}
		if state.DCSweep.Result == nil {
			continue
		}
		if len(state.DCSweep.Result.Points) != 11 || before.RunID != state.RunID || before.ElectricalRevision != state.ElectricalRevision || op.Actor != "user" {
			return fmt.Errorf("native sweep source/result identity")
		}
		if err := a.PostWait(ctx, func() {
			s.view()
			pixels, err := s.sweepDrawing.Snapshot()
			if err != nil {
				result <- err
				return
			}
			accent := paletteForTheme(state.Theme).Accent
			ink := 0
			for y := pixels.Bounds().Min.Y; y < pixels.Bounds().Max.Y; y++ {
				for x := pixels.Bounds().Min.X; x < pixels.Bounds().Max.X; x++ {
					if pixels.NRGBAAt(x, y) == accent {
						ink++
					}
				}
			}
			if ink < 100 {
				result <- cogent.ErrFrameNotReady
				return
			}
			frame := s.sweepDrawing.frame.Load()
			if frame == nil {
				result <- cogent.ErrFrameNotReady
				return
			}
			s.sweepDrawing.placeCursor(frame.left + (frame.right-frame.left)/2)
			if current := s.service.Snapshot(); current.DCSweep.Cursor == nil || *current.DCSweep.Cursor != 5 {
				result <- fmt.Errorf("native sweep measured cursor failed")
				return
			}
			result <- smokeUI(s, "sweep-back", nil)
		}); err != nil {
			return err
		}
		err := <-result
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		return err
	}
}
