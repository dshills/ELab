package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func nativePinnedReadingsSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		if r := s.invoke("example.rc", workbench.ReplaceParams{DiscardChanges: true}); r.Error != nil {
			ready <- fmt.Errorf("pin example: %s", r.Error.Message)
			return
		}
		for _, action := range []struct {
			id    string
			event *ui.ChangeEvent
		}{{"simulation.pause", nil}, {"simulation.restart", nil}, {"pin-reading-probe", &ui.ChangeEvent{SelectedID: "v:C1"}}, {"pin-X-v:C1", &ui.ChangeEvent{Number: 350}}, {"pin-Y-v:C1", &ui.ChangeEvent{Number: 250}}, {"pin-reading-probe", &ui.ChangeEvent{SelectedID: "i:C1"}}, {"pin-X-i:C1", &ui.ChangeEvent{Number: 350}}, {"pin-Y-i:C1", &ui.ChangeEvent{Number: 310}}} {
			s.view()
			if err := smokeUI(s, action.id, action.event); err != nil {
				ready <- err
				return
			}
		}
		if r := s.invoke("simulation.step.start", workbench.StepParams{Duration: .01}); r.Error != nil {
			ready <- fmt.Errorf("pin step: %s", r.Error.Message)
			return
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	state, err := s.service.Wait(ctx)
	if err != nil || len(state.PinnedReadings) != 2 || state.PinnedReadings[0].Value == nil || math.Abs(*state.PinnedReadings[0].Value-5*(1-math.Exp(-1))) > 1e-12 || state.PinnedReadings[1].Value == nil || *state.PinnedReadings[1].Value <= 0 {
		return fmt.Errorf("native pinned RC readings fail: %v", err)
	}
	if err := a.PostWait(ctx, func() {
		s.invoke("view.fit", workbench.FitParams{Width: 800, Height: 500})
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
			view := *s.drawing.View
			sx, sy := float64(pixels.Bounds().Dx())/view.Width, float64(pixels.Bounds().Dy())/view.Height
			for _, reading := range s.service.Snapshot().PinnedReadings {
				min, max := workbench.PinnedReadingBounds(reading)
				a, b := view.Screen(canvas.Point{X: min.X, Y: min.Y}), view.Screen(canvas.Point{X: max.X, Y: max.Y})
				cards, text := 0, 0
				for y := int(math.Max(0, a.Y*sy)); y < int(math.Min(float64(pixels.Bounds().Dy()), b.Y*sy)); y++ {
					for x := int(math.Max(0, a.X*sx)); x < int(math.Min(float64(pixels.Bounds().Dx()), b.X*sx)); x++ {
						p := pixels.NRGBAAt(x, y)
						if p.A > 250 && p.R == 243 && p.G == 250 && p.B == 251 {
							cards++
						}
						if p.A > 250 && p.R == 39 && p.G == 110 && p.B == 126 {
							text++
						}
					}
				}
				if cards < 100 || text < 20 {
					ready <- fmt.Errorf("fresh %s reading card/text absent: %d/%d pixels", reading.Definition.ID, cards, text)
					return
				}
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
