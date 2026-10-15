package main

import (
	"context"
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeLiveControlsSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, setup := range []struct {
			id string
			p  any
		}{{"example.rc", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.pause", struct{}{}}} {
			if r := s.invoke(setup.id, setup.p); r.Error != nil {
				done <- fmt.Errorf("native live controls setup %s: %s", setup.id, r.Error.Message)
				return
			}
		}
		for _, action := range []struct {
			id string
			e  *ui.ChangeEvent
		}{
			{"setting-speed", &ui.ChangeEvent{Number: .001}}, {"setting-step", &ui.ChangeEvent{Text: "1 us"}},
			{"simulation.restart", nil}, {"input-charge", nil},
		} {
			if err := smokeUI(s, action.id, action.e); err != nil {
				done <- err
				return
			}
		}
		state := s.service.Snapshot()
		if len(state.LiveInputModes) != 3 || state.TransientSettings.PlaybackSpeed != .001 || state.TransientSettings.StepSeconds != 1e-6 || state.Transient.PendingMode != "charge" {
			done <- fmt.Errorf("native live controls disagree with published input/rate boundary")
			return
		}
		r := s.invoke("simulation.step.start", workbench.StepParams{Duration: .001})
		if r.Error != nil {
			done <- fmt.Errorf("native 1000-sample admission: %s", r.Error.Message)
			return
		}
		done <- nil
	}); err != nil {
		return err
	}
	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	state, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if state.StepWork == nil || state.StepWork.Status != "completed" || state.StepWork.CompletedSamples != 1000 || math.Abs(state.Transient.Sample.Time-.001) > 1e-15 || state.Transient.PendingMode != "" || state.Transient.Mode != "charge" {
		return fmt.Errorf("native boundary step failed or did not apply queued input")
	}
	return nil
}
