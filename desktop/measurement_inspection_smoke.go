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

func nativeMeasurementInspectionSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if r := s.invoke("example.reset", workbench.ReplaceParams{DiscardChanges: true}); r.Error != nil {
			ready <- fmt.Errorf("DC inspection setup: %s", r.Error.Message)
		} else if r := s.invoke("simulation.resume", struct{}{}); r.Error != nil {
			ready <- fmt.Errorf("DC inspection resume: %s", r.Error.Message)
		} else {
			ready <- nil
		}
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
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"simulation.pause", nil}, {"selection", &ui.ChangeEvent{SelectedID: "R1"}}, {"measurement-pin-i:R1", nil}, {"pin-reading-probe", &ui.ChangeEvent{SelectedID: "v:D1"}}, {"inspect-terminal", &ui.ChangeEvent{SelectedID: "GND.reference"}}} {
			s.view()
			if err := smokeUI(s, action.id, action.event); err != nil {
				ready <- err
				return
			}
		}
		state := s.service.Snapshot()
		if len(state.PinnedReadings) != 2 || state.MeasurementInspection.ResultKind != "dc" || len(state.MeasurementInspection.Readings) != 1 || state.MeasurementInspection.Readings[0].Value == nil || *state.MeasurementInspection.Readings[0].Value != 0 {
			ready <- fmt.Errorf("native DC inspection/pin/reference ground failed")
			return
		}
		if r := s.invoke("example.rc", workbench.ReplaceParams{DiscardChanges: true}); r.Error != nil {
			ready <- fmt.Errorf("RC inspection setup: %s", r.Error.Message)
			return
		}
		s.invoke("simulation.pause", struct{}{})
		s.invoke("simulation.restart", struct{}{})
		before := s.service.Snapshot()
		part, _ := (workbench.Graph{Components: before.Components, Wires: before.Wires}).Component("C1")
		at, _ := workbench.TerminalPosition(part, "1")
		s.canvasClick(at)
		state = s.service.Snapshot()
		in := state.MeasurementInspection
		if !in.Available || in.Target.Kind != "terminal" || in.Target.ID != "C1.1" || len(in.Wires) != 1 || in.Wires[0] != "W2" || state.RunID != before.RunID || !reflect.DeepEqual(state.Transient, before.Transient) {
			ready <- fmt.Errorf("native terminal click changes physics or loses its exact net")
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
			green := 0
			for y := 0; y < pixels.Bounds().Dy(); y++ {
				for x := 0; x < pixels.Bounds().Dx(); x++ {
					p := pixels.NRGBAAt(x, y)
					if p.A > 250 && p.R == 80 && p.G == 124 && p.B == 64 {
						green++
					}
				}
			}
			if green < 50 {
				ready <- fmt.Errorf("fresh inspected-net wire/terminal highlights absent: %d pixels", green)
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
