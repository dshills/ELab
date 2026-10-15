package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Read the actual scope surface after rebuilding its current shared state.
func smokeScopeHash(ctx context.Context, a *app.Application, s *screen, previous *[32]byte) ([32]byte, error) {
	bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ready := make(chan error, 1)
	if err := a.PostWait(bounded, func() { s.view(); ready <- s.scope.RebuildResources() }); err != nil {
		return [32]byte{}, err
	}
	if err := <-ready; err != nil {
		return [32]byte{}, err
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		var hash [32]byte
		if err := a.PostWait(bounded, func() {
			pixels, err := s.scope.Snapshot()
			if err == nil {
				hash = sha256.Sum256(pixels.Pix)
			}
			ready <- err
		}); err != nil {
			return [32]byte{}, err
		}
		err := <-ready
		if err != nil && !errors.Is(err, cogent.ErrFrameNotReady) {
			return [32]byte{}, err
		}
		if err == nil && (previous == nil || hash != *previous) {
			return hash, nil
		}
		select {
		case <-bounded.Done():
			return [32]byte{}, fmt.Errorf("reference overlay did not produce a fresh changed scope frame: %w", bounded.Err())
		case <-ticker.C:
		}
	}
}

func nativeRecordingOverlaySmoke(ctx context.Context, a *app.Application, s *screen, reference string) error {
	before, err := smokeScopeHash(ctx, a, s, nil)
	if err != nil {
		return err
	}
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "recorded-compare-overlay", &ui.ChangeEvent{Checked: true}); err != nil {
			ready <- err
			return
		}
		state := s.service.Snapshot()
		if state.Recording.ReferenceOverlay == nil || state.Recording.ReferenceOverlay.ID != reference || !state.Recording.ReferenceOverlay.Scope.WindowValid {
			ready <- fmt.Errorf("native reference overlay missing shared capture")
			return
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	_, err = smokeScopeHash(ctx, a, s, &before)
	return err
}
