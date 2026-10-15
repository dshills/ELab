package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
		if err != nil {
			return err
		}
		return nativeSweepImportSmoke(ctx, a, s)
	}
}

// File work stays outside the owner; the owned picker selects a temporary
// curve exported by the same shared commands available to an agent.
func nativeSweepImportSmoke(ctx context.Context, a *app.Application, s *screen) error {
	directory, err := os.MkdirTemp("", "elab-sweep-import-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "curve.json")
	before := s.service.Snapshot()
	action := make(chan error, 1)
	defer func() { _ = a.PostWait(ctx, func() { s.chooseFile = nil }) }()
	for _, control := range []string{"sweep-export-json", "sweep-import"} {
		if err := a.PostWait(ctx, func() {
			s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
			action <- smokeUI(s, control, nil)
		}); err != nil {
			return err
		}
		if err := <-action; err != nil {
			return err
		}
		if err := waitNativeFile(ctx, a, s); err != nil {
			return err
		}
	}
	if err := a.PostWait(ctx, func() {
		state := s.service.Snapshot()
		op := selectedSweep(state.DCSweep)
		if s.errorMessage != "" || state.ProjectWork == nil || state.ProjectWork.Status != "completed" || state.ProjectWork.Command != "sweep.import" || state.ProjectWork.Actor != "user" || op == nil || op.Kind != "sweep-import" || op.SourceUnchanged || !reflect.DeepEqual(state.DCSweep.Result, before.DCSweep.Result) || state.RunID != before.RunID || !reflect.DeepEqual(state.Result, before.Result) {
			action <- fmt.Errorf("native sweep import capture/source identity: %s", s.errorMessage)
			return
		}
		s.view()
		_, err := s.sweepMesh(*s.sweepDrawing.View)
		if err != nil {
			action <- err
			return
		}
		frame := s.sweepDrawing.frame.Load()
		s.sweepDrawing.placeCursor(frame.left + (frame.right-frame.left)/2)
		if current := s.service.Snapshot(); current.DCSweep.Cursor == nil || *current.DCSweep.Cursor != 5 {
			action <- fmt.Errorf("imported sweep measured cursor failed")
			return
		}
		action <- smokeUI(s, "sweep-back", nil)
	}); err != nil {
		return err
	}
	return <-action
}
