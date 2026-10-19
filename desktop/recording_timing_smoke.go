package main

import (
	"context"
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

// Wait for mounted fields, invoke the production direction callback once, then
// observe completion. Owner-only unit fixtures have no text backend; their
// callback finishes synchronously and cannot establish native FIFO coverage.
func nativeRecordedTimingDirectionSmoke(ctx context.Context, a *app.Application, s *screen, direction string) error {
	owner := func(run func() error) error {
		done := make(chan error, 1)
		if err := a.PostWait(ctx, func() { done <- run() }); err != nil {
			return err
		}
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	var mounted bool
	if err := owner(func() error { mounted = s.settleField != nil; return nil }); err != nil {
		return err
	}
	if mounted {
		if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
			return state.Recording != nil && state.Recording.Timing != nil
		}); err != nil {
			return err
		}
	}
	if err := owner(func() error {
		return smokeUI(s, "recorded-timing-direction", &ui.ChangeEvent{SelectedID: direction})
	}); err != nil {
		return err
	}
	if mounted {
		if _, err := waitNativeViewport(ctx, a, s, func(workbench.Snapshot) bool {
			return s.pendingRecordedTimingPage == nil
		}); err != nil {
			return err
		}
	}
	return owner(func() error {
		state := s.service.Snapshot()
		if s.errorMessage != "" || s.pendingRecordedTimingPage != nil || state.Recording == nil || state.Recording.Timing == nil || state.Recording.Timing.Settings.Direction != direction {
			return fmt.Errorf("native recorded direction did not complete: %s", s.errorMessage)
		}
		return nil
	})
}
