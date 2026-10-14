package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dshills/vellum/drivers/cogent"
)

func nativeScopeVisibilitySmoke(ctx context.Context, s *screen) error {
	activate := func(id string) error {
		completed := make(chan error, 1)
		if err := s.app.PostWait(ctx, func() { completed <- smokeUI(s, id, nil) }); err != nil {
			return err
		}
		return <-completed
	}
	before := s.service.Snapshot()
	var epoch uint64
	if err := s.app.PostWait(ctx, func() { epoch = s.scope.mountEpoch }); err != nil {
		return err
	}
	if err := activate("scope-visibility"); err != nil {
		return err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		ready := make(chan bool, 1)
		if err := s.app.PostWait(ctx, func() { _, err := s.scope.Snapshot(); ready <- errors.Is(err, cogent.ErrFrameNotReady) }); err != nil {
			return err
		}
		if <-ready {
			break
		}
	}
	if state := s.service.Snapshot(); state.ScopeVisible || !reflect.DeepEqual(before.Scope, state.Scope) || !reflect.DeepEqual(before.Transient, state.Transient) {
		return fmt.Errorf("native collapse changed measurement/physics state")
	}
	if err := activate("scope-expand"); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		ready := make(chan error, 1)
		if err := s.app.PostWait(ctx, func() {
			pixels, err := s.scope.Snapshot()
			if err != nil {
				ready <- err
				return
			}
			if s.scope.mountEpoch <= epoch {
				ready <- cogent.ErrFrameNotReady
				return
			}
			colored := 0
			for y := 0; y < pixels.Bounds().Dy(); y++ {
				for x := 0; x < pixels.Bounds().Dx(); x++ {
					p := pixels.NRGBAAt(x, y)
					if p.A > 100 && p.R < 80 && p.G > 70 && p.B > 70 {
						colored++
					}
				}
			}
			if colored < 100 {
				ready <- cogent.ErrFrameNotReady
				return
			}
			ready <- nil
		}); err != nil {
			return err
		}
		err := <-ready
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		if err != nil {
			return err
		}
		break
	}
	after := s.service.Snapshot()
	if !after.ScopeVisible || !reflect.DeepEqual(before.Scope, after.Scope) || !reflect.DeepEqual(before.Transient, after.Transient) {
		return fmt.Errorf("native remount lost scope/physics")
	}
	return nil
}
