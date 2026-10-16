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
	if err := owner(func() error {
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
	}); err != nil {
		return err
	}
	if err := owner(func() error {
		for _, action := range []struct {
			id string
			e  *ui.ChangeEvent
		}{
			{"value-entry", &ui.ChangeEvent{Text: "12 V"}}, {"apply-value", nil},
			{"selection", &ui.ChangeEvent{SelectedID: "V1"}}, {"value-entry", &ui.ChangeEvent{Text: "12 V"}}, {"apply-value", nil},
			{"selection", &ui.ChangeEvent{SelectedID: "W5"}}, {"delete-object", nil},
			{"tool-wire", nil}, {"wire-from", &ui.ChangeEvent{SelectedID: "V2.negative"}},
			{"wire-to", &ui.ChangeEvent{SelectedID: "GND1.reference"}}, {"connect-terminals", nil},
		} {
			if err := smokeUI(s, action.id, action.e); err != nil {
				return err
			}
		}
		if s.service.Snapshot().Paused {
			return smokeUI(s, "simulation.resume", nil)
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	_, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		return state.Mixed != nil && !state.Mixed.Stale && math.Abs(state.Mixed.Resistors[0].Voltage-24) < 1e-10 && math.Abs(state.Mixed.Resistors[0].Current-.024) < 1e-12
	})
	if err != nil {
		return err
	}
	if err := owner(func() error {
		for _, action := range []struct {
			id string
			e  *ui.ChangeEvent
		}{
			{"part-list", &ui.ChangeEvent{SelectedID: "led"}},
			{"position-x", &ui.ChangeEvent{Number: 350}}, {"position-y", &ui.ChangeEvent{Number: 100}}, {"place-at-position", nil},
		} {
			if err := smokeUI(s, action.id, action.e); err != nil {
				return err
			}
		}
		led := s.service.Snapshot().Selected
		for _, action := range []struct {
			id string
			e  *ui.ChangeEvent
		}{
			{"selection", &ui.ChangeEvent{SelectedID: "W2"}}, {"delete-object", nil}, {"tool-wire", nil},
			{"wire-from", &ui.ChangeEvent{SelectedID: "R1.2"}}, {"wire-to", &ui.ChangeEvent{SelectedID: led + ".anode"}}, {"connect-terminals", nil},
			{"wire-from", &ui.ChangeEvent{SelectedID: led + ".cathode"}}, {"wire-to", &ui.ChangeEvent{SelectedID: "V2.negative"}}, {"connect-terminals", nil},
		} {
			if err := smokeUI(s, action.id, action.e); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	_, err = waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool {
		return state.Mixed != nil && !state.Mixed.Stale && len(state.Mixed.LEDs) == 1 && state.Mixed.LEDs[0].Current > .021 && state.Mixed.LEDs[0].Current < .023 && state.Mixed.KCLResidual < 1e-10 && math.Abs(state.Mixed.PowerResidual) < 1e-8
	})
	return err
}
