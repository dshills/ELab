package main

import (
	"context"
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeSharedPulseClockSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, request := range []struct {
			id string
			p  any
		}{{"example.npn-pulse", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.pause", struct{}{}}, {"component.select", workbench.IDParams{ID: "VIN"}}} {
			if r := s.invoke(request.id, request.p); r.Error != nil {
				done <- fmt.Errorf("shared clock setup: %s", r.Error.Message)
				return
			}
		}
		for _, field := range []struct{ id, text string }{{"pulse-frequency", "20 Hz"}, {"pulse-duty", "25 %"}, {"pulse-period-time", "100 ms"}} {
			if err := smokeUI(s, field.id, &ui.ChangeEvent{Text: field.text}); err != nil {
				done <- err
				return
			}
		}
		state := s.service.Snapshot()
		if state.TransientSettings.HighSeconds != .025 || math.Abs(state.TransientSettings.LowSeconds-.075) > 1e-16 || !state.PulseTimings[0].Shared {
			done <- fmt.Errorf("native shared clock fields did not update experiment phases")
			return
		}
		_, err := s.mesh(*s.drawing.View)
		done <- err
	}); err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
