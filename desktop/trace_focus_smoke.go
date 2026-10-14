package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func nativeTraceFocusSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			command string
			params  any
		}{{"example.flasher", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.pause", struct{}{}}, {"scope.probes.set", workbench.ProbeParams{IDs: []string{"i:D1", "v:C1"}}}, {"view.fit", workbench.FitParams{Width: 800, Height: 500}}} {
			if r := s.invoke(action.command, action.params); r.Error != nil {
				ready <- fmt.Errorf("trace focus setup: %s", r.Error.Message)
				return
			}
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	before, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		s.view()
		if err := smokeUI(s, "scope-trace-focus", &ui.ChangeEvent{SelectedID: "i:D1"}); err != nil {
			ready <- err
			return
		}
		state := s.service.Snapshot()
		if state.TraceFocus == nil || state.TraceFocus.Unavailable || !reflect.DeepEqual(state.TraceFocus.Objects, []string{"D1"}) || state.RunID != before.RunID || !reflect.DeepEqual(state.Transient, before.Transient) {
			ready <- fmt.Errorf("trace focus changes physical state or loses source")
			return
		}
		s.view()
		ready <- s.drawing.RebuildResources()
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
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
		if err := a.PostWait(ctx, func() {
			pixels, err := s.drawing.Snapshot()
			if err != nil {
				ready <- err
				return
			}
			purple := 0
			for y := 0; y < pixels.Bounds().Dy(); y++ {
				for x := 0; x < pixels.Bounds().Dx(); x++ {
					p := pixels.NRGBAAt(x, y)
					if p.A > 100 && p.R > 140 && p.R < 170 && p.G > 65 && p.G < 90 && p.B > 180 && p.B < 205 {
						purple++
					}
				}
			}
			if purple < 50 {
				ready <- fmt.Errorf("fresh trace source highlight absent: %d pixels", purple)
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
		return err
	}
}
