package main

import (
	"context"
	"fmt"
	"math"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeNMOSSmoke(ctx context.Context, a *app.Application, s *screen) error {
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
	if err := owner(func() error {
		if err := smokeUI(s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.nmos"}); err != nil {
			return err
		}
		if s.pendingCommand == "example.load" {
			return smokeUI(s, "discard-continue", nil)
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	for _, id := range []string{"simulation.pause", "simulation.restart"} {
		if err := owner(func() error { return smokeUI(s, id, nil) }); err != nil {
			return err
		}
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	check := func(region string) error {
		return owner(func() error {
			state := s.service.Snapshot()
			if state.Example == nil || state.Example.Info.ID != "nmos" || state.Mixed == nil || state.Mixed.Stale || len(state.Mixed.MOSFETs) != 1 || state.Mixed.MOSFETs[0].State != region || state.Mixed.MOSFETs[0].GateCurrent != 0 || state.Mixed.KCLResidual > 1e-9 || math.Abs(state.Mixed.PowerResidual) > 1e-8 {
				return fmt.Errorf("native MOSFET %s result missing", region)
			}
			if err := ui.Validate(s.view()); err != nil {
				return err
			}
			for _, id := range []string{"plot-v:M1", "plot-v:M1.gate-source", "plot-i:M1.drain"} {
				if _, ok := findNode(s.view(), id); !ok {
					return fmt.Errorf("native MOSFET action %s missing", id)
				}
			}
			if _, err := s.mesh(*s.drawing.View); err != nil {
				return err
			}
			_, err := s.scopeMesh(*s.scope.View)
			return err
		})
	}
	if err := check("triode"); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		if err := owner(func() error { return smokeUI(s, "transient-step", nil) }); err != nil {
			return err
		}
	}
	return check("cutoff")
}
