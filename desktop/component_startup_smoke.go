package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/vellum/app"
)

// Call off the application owner: a startup action first settles its mounted
// field asynchronously. Service.Wait alone cannot observe that UI admission.
func waitNativeComponentStartup(ctx context.Context, a *app.Application, s *screen) error {
	for {
		type result struct {
			pending bool
			err     error
		}
		ready := make(chan result, 1)
		if err := a.PostWait(ctx, func() {
			r := result{pending: s.pendingComponentStartup != nil}
			if !r.pending && s.errorMessage != "" {
				r.err = fmt.Errorf("native startup admission: %s", s.errorMessage)
			}
			ready <- r
		}); err != nil {
			return err
		}
		select {
		case r := <-ready:
			if !r.pending {
				return r.err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
