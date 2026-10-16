package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
)

func nativeDisplaySmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			command string
			params  any
		}{{"example.reset", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.resume", struct{}{}}} {
			if r := s.invoke(action.command, action.params); r.Error != nil {
				ready <- fmt.Errorf("display setup: %s", r.Error.Message)
				return
			}
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		for _, id := range []string{"display-voltage", "display-current", "display-motion"} {
			s.view()
			if err := smokeUI(s, id, nil); err != nil {
				ready <- err
				return
			}
		}
		s.view()
		ready <- s.drawing.RebuildResources()
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	baseline, err := waitNativeViewport(ctx, a, s, func(workbench.Snapshot) bool { return true })
	if err != nil {
		return err
	}
	if !baseline.Display.Animate {
		return fmt.Errorf("native current motion did not enable")
	}
	type pixelsResult struct {
		hash [32]byte
		err  error
	}
	read := func() (hash [32]byte, err error) {
		result := make(chan pixelsResult, 1)
		if err = a.PostWait(ctx, func() {
			pixels, e := s.drawing.Snapshot()
			if e != nil {
				result <- pixelsResult{err: e}
				return
			}
			orange, purple := 0, 0
			for y := 0; y < pixels.Bounds().Dy(); y++ {
				for x := 0; x < pixels.Bounds().Dx(); x++ {
					p := pixels.NRGBAAt(x, y)
					if p.A > 250 && p.R == 213 && p.G == 106 && p.B == 57 {
						orange++
					}
					if p.A > 250 && p.R == 118 && p.G == 67 && p.B == 158 {
						purple++
					}
				}
			}
			if orange < 50 || purple < 20 {
				result <- pixelsResult{err: fmt.Errorf("fresh voltage/current colors absent: %d/%d pixels", orange, purple)}
				return
			}
			result <- pixelsResult{hash: sha256.Sum256(pixels.Pix)}
		}); err != nil {
			return hash, err
		}
		r := <-result
		return r.hash, r.err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	var first [32]byte
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		hash, err := read()
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		if err != nil {
			return err
		}
		first = hash
		break
	}
	// No forced rebuild here: the mounted drawing-only Sync cadence must cause
	// a new GPU frame while the DC result/editor revisions remain unchanged.
	changed := false
	for n := 0; n < 12; n++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		hash, err := read()
		if err != nil {
			return err
		}
		if hash != first {
			changed = true
			break
		}
	}
	if !changed {
		return fmt.Errorf("mounted current animation never repainted")
	}
	after := s.service.Snapshot()
	if after.Revision != baseline.Revision || after.CommandRevision != baseline.CommandRevision || after.RunID != baseline.RunID || !reflect.DeepEqual(after.Result, baseline.Result) {
		return fmt.Errorf("illustrative animation mutated physics/editor state")
	}
	if err := a.PostWait(ctx, func() {
		s.view()
		if err := smokeUI(s, "display-reduced-motion", nil); err != nil {
			ready <- err
			return
		}
		s.view()
		ready <- s.drawing.RebuildResources()
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	if s.service.Snapshot().Display.Animate {
		return fmt.Errorf("reduced motion still animates")
	}
	var frozen [32]byte
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		hash, err := read()
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		if err != nil {
			return err
		}
		frozen = hash
		break
	}
	for n := 0; n < 4; n++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		hash, err := read()
		if err != nil {
			return err
		}
		if hash != frozen {
			return fmt.Errorf("reduced-motion GPU frame changed without an edit")
		}
	}
	return nil
}
