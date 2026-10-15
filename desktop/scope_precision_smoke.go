package main

import (
	"context"
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeScopePrecisionSmoke(ctx context.Context, a *app.Application, s *screen) error {
	done := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		for _, setup := range []struct {
			id string
			p  any
		}{
			{"example.rc", workbench.ReplaceParams{DiscardChanges: true}}, {"simulation.pause", struct{}{}},
			{"scope.visibility.set", workbench.ScopeVisibilityParams{Visible: true}},
			{"scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1"}}},
			{"simulation.step.start", workbench.StepParams{Duration: 1e-12}},
		} {
			if r := s.invoke(setup.id, setup.p); r.Error != nil {
				done <- fmt.Errorf("scope precision setup %s: %s", setup.id, r.Error.Message)
				return
			}
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
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct{ id, text string }{{"window-start", "0.4 ps"}, {"window-end", "0.6 ps"}} {
			if err := smokeUI(s, action.id, &ui.ChangeEvent{Text: action.text}); err != nil {
				done <- err
				return
			}
		}
		state := s.service.Snapshot()
		if state.Scope.WindowValid || len(state.Scope.Traces[0].Points) != 0 || len(state.Scope.Statistics) != 0 {
			done <- fmt.Errorf("native narrow interval borrowed outside rows")
			return
		}
		csv, ok := findNode(s.view(), "export-csv")
		if !ok || !csv.Disabled {
			done <- fmt.Errorf("native empty interval permits CSV export")
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
	empty, err := smokeScopeHash(ctx, a, s, nil)
	if err != nil {
		return err
	}
	if err := a.PostWait(ctx, func() {
		for _, action := range []struct{ id, text string }{{"window-end", "1 ps"}, {"window-start", "0.75 ps"}} {
			if err := smokeUI(s, action.id, &ui.ChangeEvent{Text: action.text}); err != nil {
				done <- err
				return
			}
		}
		sc := s.service.Snapshot().Scope
		if !sc.WindowValid || len(sc.Traces[0].Points) != 1 {
			done <- fmt.Errorf("native one-row interval failed")
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
	_, err = smokeScopeHash(ctx, a, s, &empty)
	return err
}
