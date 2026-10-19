package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

func nativeVerificationContractsSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	var before workbench.Snapshot
	if err := a.PostWait(ctx, func() {
		if response := s.invoke("example.reset", workbench.ReplaceParams{DiscardChanges: true}); response.Error != nil {
			ready <- fmt.Errorf("verification reset smoke: %s", response.Error.Message)
			return
		}
		ready <- smokeUI(s, "simulation.pause", nil)
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		cfg := s.service.Snapshot().VerificationSettings
		initial := 1.
		cfg.InitialVoltage, cfg.InputMode, cfg.Step = &initial, "charge", .002
		if response := s.invoke("verification.settings.set", cfg); response.Error != nil {
			ready <- fmt.Errorf("verification smoke defaults: %s", response.Error.Message)
			return
		}
		before = s.service.Snapshot()
		if err := smokeUI(s, "verify-project-startup", nil); err != nil {
			ready <- err
			return
		}
		ready <- smokeUI(s, "verify-start", nil)
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		state := s.service.Snapshot()
		for _, operation := range state.Operations {
			if operation.ID != state.LastOperation {
				continue
			}
			if operation.Status == "failed" || operation.Status == "canceled" {
				return fmt.Errorf("DC reset verification: %+v", operation)
			}
			if operation.Status != "completed" {
				continue
			}
			if state.Verification == nil || state.Verification.Duration != 0 || state.Verification.SampleCount != 1 || state.Verification.InputMode != "auto" || state.VerificationSettings.InitialVoltage != nil || state.VerificationSettings.Step != 0 {
				return fmt.Errorf("DC reset did not clear incompatible defaults/capture operating point")
			}
			if !reflect.DeepEqual(before.Components, state.Components) || !reflect.DeepEqual(before.Wires, state.Wires) || before.RunID != state.RunID || !reflect.DeepEqual(before.Result, state.Result) || !reflect.DeepEqual(before.UndoEntry, state.UndoEntry) {
				return fmt.Errorf("DC reset/capture changed live circuit or history")
			}
			return nil
		}
	}
}
