package main

import (
	"context"
	"fmt"

	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeLEDFadeSmoke(ctx context.Context, a *app.Application, s *screen) error {
	owner := func(run func() error) error {
		result := make(chan error, 1)
		if err := a.PostWait(ctx, func() { result <- run() }); err != nil {
			return err
		}
		select {
		case err := <-result:
			if err != nil {
				return err
			}
			return waitNativeComponentStartup(ctx, a, s)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := owner(func() error {
		if err := smokeUI(s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.led-fade"}); err != nil {
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
	for i := 0; i < 205; i++ {
		if err := owner(func() error { return smokeUI(s, "transient-step", nil) }); err != nil {
			return err
		}
	}
	return owner(func() error {
		state := s.service.Snapshot()
		if state.Example == nil || state.Example.Info.ID != "led-fade" || state.Mixed == nil || state.Mixed.Stale || state.Transient.Sample.Input != 0 || state.Mixed.LEDs[0].Current <= 0 || state.Mixed.Capacitors[0].Voltage < 1 || len(state.Scope.Traces) != 3 {
			return fmt.Errorf("native LED fade tail/scope absent")
		}
		if err := ui.Validate(s.view()); err != nil {
			return err
		}
		if _, err := s.mesh(*s.drawing.View); err != nil {
			return err
		}
		_, err := s.scopeMesh(*s.scope.View)
		return err
	})
}
