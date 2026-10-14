package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func nativeDiagnosticSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			command string
			params  any
		}{{"example.reset", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.resume", struct{}{}}} {
			if r := s.invoke(action.command, action.params); r.Error != nil {
				ready <- fmt.Errorf("diagnostic setup: %s", r.Error.Message)
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
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			command string
			params  any
		}{{"simulation.pause", struct{}{}}, {"object.delete", workbench.IDParams{ID: "W1"}}, {"drawing.tool.set", workbench.ToolParams{Tool: "wire"}}} {
			if r := s.invoke(action.command, action.params); r.Error != nil {
				ready <- fmt.Errorf("diagnostic action: %s", r.Error.Message)
				return
			}
		}
		s.canvasClick(workbench.Point{X: -200, Y: -34})
		before := s.service.Snapshot()
		s.view()
		if err := smokeUI(s, "diagnostic-details", &ui.ChangeEvent{Checked: true}); err != nil {
			ready <- err
			return
		}
		after := s.service.Snapshot()
		if !after.DiagnosticPresentation.DetailsVisible || after.Drawing != before.Drawing || after.RunID != before.RunID || len(after.DiagnosticPresentation.Items) == 0 || len(after.DiagnosticPresentation.Items[0].Terminals) != 1 || after.DiagnosticPresentation.Items[0].Terminals[0] != "V1.positive" {
			ready <- fmt.Errorf("native diagnostic disclosure loses source/gesture")
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
	read := func() (int, error) {
		type result struct {
			count int
			err   error
		}
		done := make(chan result, 1)
		if err := a.PostWait(ctx, func() {
			pixels, err := s.drawing.Snapshot()
			if err != nil {
				done <- result{err: err}
				return
			}
			red := 0
			for y := 0; y < pixels.Bounds().Dy(); y++ {
				for x := 0; x < pixels.Bounds().Dx(); x++ {
					p := pixels.NRGBAAt(x, y)
					if p.A > 250 && p.R == 192 && p.G == 73 && p.B == 50 {
						red++
					}
				}
			}
			done <- result{count: red}
		}); err != nil {
			return 0, err
		}
		r := <-done
		return r.count, r.err
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		red, err := read()
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		if err != nil {
			return err
		}
		if red < 50 {
			return fmt.Errorf("fresh known diagnostic highlight absent: %d pixels", red)
		}
		break
	}
	if err := a.PostWait(ctx, func() {
		if r := s.invoke("wire.connect", workbench.ConnectParams{From: "V1.positive", To: "R1.1"}); r.Error != nil {
			ready <- fmt.Errorf("diagnostic correction: %s", r.Error.Message)
			return
		}
		if len(s.service.Snapshot().DiagnosticPresentation.Items) != 0 {
			ready <- fmt.Errorf("corrected circuit retains diagnostic marker")
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
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		red, err := read()
		if errors.Is(err, cogent.ErrFrameNotReady) {
			continue
		}
		if err != nil {
			return err
		}
		if red != 0 {
			return fmt.Errorf("corrected GPU frame retains diagnostic pixels: %d", red)
		}
		return nil
	}
}
