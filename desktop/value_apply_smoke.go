package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

// Exercise controlled Change -> mounted refresh -> asynchronous Apply. This
// helper runs on the smoke worker, never blocks the owner waiting for its FIFO,
// and does not claim physical typing coverage (the CUA audit covers that).
// An empty value leaves the displayed draft untouched before Apply.
func nativeValueApplySmoke(ctx context.Context, a *app.Application, s *screen, id, value string, wantError bool) error {
	prepared := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if err := smokeUI(s, "selection", &ui.ChangeEvent{SelectedID: id}); err != nil {
			prepared <- err
			return
		}
		if value == "" {
			value = s.service.Snapshot().Drawing.Value
			prepared <- nil
			return
		}
		prepared <- smokeUI(s, "value-entry", &ui.ChangeEvent{Text: value})
	}); err != nil {
		return err
	}
	select {
	case err := <-prepared:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		return state.Selected == id && state.Drawing.Value == value
	}); err != nil {
		return err
	}
	before := s.service.Snapshot()
	started := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		node, ok := findAction(s.view(), "apply-value")
		if !ok || node.Disabled {
			started <- fmt.Errorf("mounted Apply unavailable")
			return
		}
		node.Activate()
		started <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-started:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		type sample struct {
			pending bool
			message string
		}
		observed := make(chan sample, 1)
		if err := a.PostWait(ctx, func() { observed <- sample{s.pendingValueApply != nil, s.errorMessage} }); err != nil {
			return err
		}
		select {
		case got := <-observed:
			if !got.pending {
				if (got.message != "") != wantError {
					return fmt.Errorf("mounted Apply error=%q; wantError=%t", got.message, wantError)
				}
				if wantError && !reflect.DeepEqual(before, s.service.Snapshot()) {
					return fmt.Errorf("refused mounted Apply changed accepted state")
				}
				_, err := s.service.Wait(ctx)
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-tick.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
