package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/vellum/app"
)

func waitNativeCursorAction(ctx context.Context, a *app.Application, s *screen, cursor string) error {
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		type outcome struct {
			pending bool
			err     error
		}
		done := make(chan outcome, 1)
		if err := a.PostWait(ctx, func() {
			result := outcome{pending: s.pendingCursorAction != nil}
			if !result.pending {
				if s.errorMessage != "" {
					result.err = fmt.Errorf("native cursor action: %s", s.errorMessage)
				} else if s.service.Snapshot().Scope.ActiveCursor != cursor {
					result.err = fmt.Errorf("native cursor action did not select %s", cursor)
				}
			}
			done <- result
		}); err != nil {
			return err
		}
		select {
		case result := <-done:
			if !result.pending {
				return result.err
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
