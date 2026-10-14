package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/drivers/cogent"
)

func constructCapacity(s *screen) error {
	for _, action := range []struct {
		command string
		params  any
	}{{"project.clear", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.pause", struct{}{}}} {
		if r := s.invoke(action.command, action.params); r.Error != nil {
			return fmt.Errorf("%s", r.Error.Message)
		}
	}
	edit := func(command string, p any) workbench.BatchEdit {
		raw, _ := json.Marshal(p)
		return workbench.BatchEdit{Command: command, Params: raw}
	}
	parts := []workbench.BatchEdit{edit("component.place", workbench.PlaceParams{Kind: "pulse-source", Position: workbench.Point{X: -800}}), edit("component.place", workbench.PlaceParams{Kind: "ground", Position: workbench.Point{X: 800}})}
	for i := 0; i < 50; i++ {
		parts = append(parts, edit("component.place", workbench.PlaceParams{Kind: "potentiometer", Position: workbench.Point{X: float64(i%8)*160 - 560, Y: float64(i/8)*160 - 560}}))
	}
	if r := s.invoke("edit.batch", workbench.BatchParams{Name: "Capacity components", Edits: parts}); r.Error != nil {
		return fmt.Errorf("%s", r.Error.Message)
	}
	state := s.service.Snapshot()
	source, ground := "", ""
	pots := []string{}
	for _, p := range state.Components {
		switch p.Kind {
		case "pulse-source":
			source = p.ID
		case "ground":
			ground = p.ID
		case "potentiometer":
			pots = append(pots, p.ID)
		}
	}
	wires := []workbench.BatchEdit{}
	previous := source + ".positive"
	for _, id := range pots {
		wires = append(wires, edit("wire.connect", workbench.ConnectParams{From: previous, To: id + ".1"}))
		previous = id + ".2"
	}
	for _, pair := range [][2]string{{previous, ground + ".reference"}, {source + ".negative", ground + ".reference"}, {pots[49] + ".wiper", pots[49] + ".1"}} {
		wires = append(wires, edit("wire.connect", workbench.ConnectParams{From: pair[0], To: pair[1]}))
	}
	for _, action := range []struct {
		command string
		params  any
	}{{"edit.batch", workbench.BatchParams{Name: "Capacity wires", Edits: wires}}, {"scope.probes.set", workbench.ProbeParams{IDs: []string{"v:" + source, "v:" + pots[0] + ".wiper", "v:" + pots[24] + ".wiper", "i:" + pots[48] + ".1"}}}, {"simulation.restart", struct{}{}}, {"view.fit", workbench.FitParams{Width: 800, Height: 500}}} {
		if r := s.invoke(action.command, action.params); r.Error != nil {
			return fmt.Errorf("%s", r.Error.Message)
		}
	}
	return nil
}
func nativeCapacitySmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() { ready <- constructCapacity(s) }); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	state, err := s.service.Wait(ctx)
	if err != nil {
		return err
	}
	if len(state.Nets) != 100 || state.SimulatedComponents != 51 || state.Mixed == nil || state.Mixed.Stale {
		return fmt.Errorf("native capacity initialization failed")
	}
	if err := a.PostWait(ctx, func() {
		if r := s.invoke("simulation.step.start", workbench.StepParams{Duration: .06}); r.Error != nil {
			ready <- fmt.Errorf("%s", r.Error.Message)
		} else {
			ready <- nil
		}
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	if state, err = s.service.Wait(ctx); err != nil || state.StepWork == nil || state.StepWork.Status != "completed" || len(state.Scope.Traces) != 4 {
		return fmt.Errorf("native capacity step: %v", err)
	}
	// Snapshot reads the last completed GPU frame. Rebuild after updating the
	// view so earlier smoke pixels cannot masquerade as a capacity render.
	if err := a.PostWait(ctx, func() {
		s.view()
		for _, canvas := range []*cogent.MeshCanvas{s.drawing.MeshCanvas, s.scope.MeshCanvas} {
			if err := canvas.RebuildResources(); err != nil {
				ready <- err
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
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if err := a.PostWait(ctx, func() {
			for _, canvas := range []*cogent.MeshCanvas{s.drawing.MeshCanvas, s.scope.MeshCanvas} {
				pixels, err := canvas.Snapshot()
				if err != nil {
					ready <- err
					return
				}
				colored := 0
				for y := 0; y < pixels.Bounds().Dy(); y++ {
					for x := 0; x < pixels.Bounds().Dx(); x++ {
						p := pixels.NRGBAAt(x, y)
						if p.A > 100 && p.R < 130 && p.G > 40 && p.B > 40 {
							colored++
						}
					}
				}
				if colored < 20 {
					ready <- cogent.ErrFrameNotReady
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
