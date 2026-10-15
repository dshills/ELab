package main

import (
	"context"
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

func nativeNavigationSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct {
			id string
			p  any
		}{
			{"example.reset", workbench.ReplaceParams{DiscardChanges: true}},
			{"simulation.pause", struct{}{}},
		} {
			if r := s.invoke(action.id, action.p); r.Error != nil {
				done <- fmt.Errorf("navigation setup: %s", r.Error.Message)
				return
			}
		}
		before := s.service.Snapshot()
		for _, id := range []string{"selection-next", "selection-previous", "selection-center"} {
			if err := smokeUI(s, id, nil); err != nil {
				done <- err
				return
			}
		}
		after := s.service.Snapshot()
		if after.Selected != after.Wires[len(after.Wires)-1].ID || after.View.Zoom != before.View.Zoom || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || after.UndoName != before.UndoName || after.Document.Dirty != before.Document.Dirty {
			done <- fmt.Errorf("native navigation changed source/physics/history")
			return
		}
		s.drawing.navigateObject("first", after.CommandRevision, after.Document.Generation)
		after = s.service.Snapshot()
		if after.Selected != after.Components[0].ID {
			done <- fmt.Errorf("native guarded first object")
			return
		}
		if r := s.invoke("view.fit", workbench.FitParams{Width: 650, Height: 400}); r.Error != nil {
			done <- fmt.Errorf("native Fit: %s", r.Error.Message)
			return
		}
		// Keep the lifecycle smoke's expected unsaved circuit.
		if r := s.invoke("component.set", workbench.ValueParams{ID: "R1", Value: "2.2k"}); r.Error != nil {
			done <- fmt.Errorf("native lifecycle setup: %s", r.Error.Message)
			return
		}
		done <- nil
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
