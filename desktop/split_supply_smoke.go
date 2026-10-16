package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeSplitSupplySmoke(ctx context.Context, a *app.Application, s *screen) error {
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
		if err := smokeUI(s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.split-supply"}); err != nil {
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
	if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		return state.Example != nil && state.Example.Info.ID == "split-supply" && state.Mixed != nil && !state.Mixed.Stale
	}); err != nil {
		return err
	}
	return owner(func() error {
		state := s.service.Snapshot()
		if math.Abs(state.Mixed.Resistors[0].Voltage-6) > 1e-12 || math.Abs(state.Mixed.Resistors[0].Current-.006) > 1e-12 {
			return fmt.Errorf("native split supply differs from Kirchhoff reference")
		}
		if err := smokeUI(s, "selection", &ui.ChangeEvent{SelectedID: "V2"}); err != nil {
			return err
		}
		reading, ok := findNode(s.view(), "measurement-value-v:V2.negative")
		if !ok || !strings.Contains(reading.Text, "-3") || !strings.Contains(reading.Text, "V") {
			return fmt.Errorf("native split supply has no signed negative-rail reading")
		}
		return nil
	})
}
