package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/vellum/app"
)

// Native field settlement is asynchronous. Fixture consumers must observe a
// fresh admission before inspecting completion, exporting or comparing captures;
// an older completed LastOperation is not evidence that this click finished.
func waitNativeVerificationAdmission(ctx context.Context, a *app.Application, s *screen, previous string) (string, error) {
	for {
		type admission struct {
			id  string
			err error
		}
		ready := make(chan admission, 1)
		if err := a.PostWait(ctx, func() {
			var result admission
			state := s.service.Snapshot()
			if s.pendingVerificationAction == nil {
				if s.errorMessage != "" {
					result.err = fmt.Errorf("native verification admission: %s", s.errorMessage)
				} else if state.LastOperation != previous {
					result.id = state.LastOperation
				}
			}
			ready <- result
		}); err != nil {
			return "", err
		}
		var result admission
		select {
		case result = <-ready:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		if result.err != nil {
			return "", result.err
		}
		if result.id != "" {
			return result.id, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
