package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dshills/vellum/app"
)

// Run last: the explicit Discard decision closes this owned smoke session.
func nativeLifecycleSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		before := s.service.Snapshot()
		if !before.Document.Dirty {
			done <- fmt.Errorf("close smoke requires the unsaved manual circuit")
			return
		}
		if err := a.CheckCloseWindow(s.window); !errors.Is(err, app.ErrCloseRefused) || s.pendingCommand != "app.close" || s.service.Snapshot().Closed {
			done <- fmt.Errorf("native close bypassed protected service admission: %v", err)
			return
		}
		if err := smokeUI(s, "cancel-file-action", nil); err != nil {
			done <- err
			return
		}
		if !reflect.DeepEqual(before, s.service.Snapshot()) {
			done <- fmt.Errorf("canceled close changed circuit/history/physics")
			return
		}
		// Dispatch through the actual menu descriptor and its selected option.
		// The harness's generic fixture helper translates example choices.
		menu, ok := findNode(s.view(), "examples-menu")
		closeItem, offered := findAction(menu, "app.close")
		if !ok || menu.Disabled || !offered || closeItem.Disabled || closeItem.Activate == nil {
			done <- fmt.Errorf("Close window is unavailable in the file actions menu")
			return
		}
		closeItem.Activate()
		if s.errorMessage != "" {
			done <- fmt.Errorf("Close window: %s", s.errorMessage)
			return
		}
		if s.pendingCommand != "app.close" {
			done <- fmt.Errorf("file Close did not share the window-close decision")
			return
		}
		if err := smokeUI(s, "discard-continue", nil); err != nil {
			done <- err
			return
		}
		if !s.service.Snapshot().Closed {
			done <- fmt.Errorf("explicit discard did not close owned service")
			return
		}
		done <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		if s.service.Snapshot().ReadyToExit() {
			return nil
		}
		return ctx.Err()
	}
	for {
		state := s.service.Snapshot()
		if state.ReadyToExit() {
			return nil
		}
		if state.Shutdown != nil && state.Shutdown.Status == "failed" {
			return fmt.Errorf("native shutdown failed: %s / %s", state.Shutdown.RecoveryError, state.Shutdown.AppearanceError)
		}
		select {
		case <-ctx.Done():
			if s.service.Snapshot().ReadyToExit() {
				return nil
			}
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
