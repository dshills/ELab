package main

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func nativeExamplesSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		r := s.invoke("example.load", workbench.ExampleLoadParams{ID: "rc", DiscardChanges: true})
		if r.Error != nil {
			ready <- fmt.Errorf("guided RC setup: %s", r.Error.Message)
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
		s.invoke("simulation.pause", struct{}{})
		original := s.service.Snapshot()
		root := s.view()
		for _, id := range []string{"example-question", "example-expected", "example-try", "example-limitation"} {
			node, ok := findNode(root, id)
			if !ok || node.Text == "" {
				ready <- fmt.Errorf("guided field missing: %s", id)
				return
			}
		}
		if r := s.invoke("component.set", workbench.ValueParams{ID: "R1", Value: "20k"}); r.Error != nil {
			ready <- fmt.Errorf("guided edit: %s", r.Error.Message)
			return
		}
		s.view()
		if err := smokeUI(s, "example.current.reset", nil); err != nil {
			ready <- err
			return
		}
		if s.pendingCommand != "example.current.reset" || !s.service.Snapshot().Example.Modified {
			ready <- fmt.Errorf("guided reset bypassed unsaved guard")
			return
		}
		if err := smokeUI(s, "cancel-file-action", nil); err != nil {
			ready <- err
			return
		}
		if !s.service.Snapshot().Example.Modified {
			ready <- fmt.Errorf("cancelled reset lost edited circuit")
			return
		}
		if err := smokeUI(s, "example.current.reset", nil); err != nil {
			ready <- err
			return
		}
		if err := smokeUI(s, "discard-continue", nil); err != nil {
			ready <- err
			return
		}
		state := s.service.Snapshot()
		if state.Example == nil || state.Example.Info.ID != "rc" || state.Example.Modified || !state.Paused || state.Transient == nil || state.Transient.Stale || state.Transient.Sample.Time != 0 || !reflect.DeepEqual(state.Components, original.Components) {
			ready <- fmt.Errorf("native guided reset changes the wrong example or startup")
			return
		}
		ready <- nil
	}); err != nil {
		return err
	}
	if err := <-ready; err != nil {
		return err
	}
	return nativeManualTransistorSmoke(ctx, a, s)
}

func nativeManualTransistorSmoke(ctx context.Context, a *app.Application, s *screen) error {
	ready := make(chan error, 1)
	if err := a.PostWait(ctx, func() {
		r := s.invoke("example.load", workbench.ExampleLoadParams{ID: "switch-npn", DiscardChanges: true})
		if r.Error != nil {
			ready <- fmt.Errorf("manual transistor setup: %s", r.Error.Message)
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
	for _, closed := range []bool{true, false} {
		if err := a.PostWait(ctx, func() {
			s.view()
			ready <- smokeUI(s, "switch-actuate", nil)
		}); err != nil {
			return err
		}
		if err := <-ready; err != nil {
			return err
		}
		state, err := s.service.Wait(ctx)
		if err != nil || state.Mixed == nil || state.Mixed.Stale || state.Selected != "S1" || state.SwitchStates[0].DesiredClosed != closed {
			return fmt.Errorf("manual transistor switch state: wanted closed=%t; selected=%q, status=%q, states=%+v, diagnostics=%+v, mixed=%+v; error=%v", closed, state.Selected, state.Status, state.SwitchStates, state.Diagnostics, state.Mixed, err)
		}
		current := state.Mixed.LEDs[0].Current
		if closed && current < .002 || !closed && current > 1e-8 {
			return fmt.Errorf("manual transistor switch current: %g A", current)
		}
	}
	if err := a.PostWait(ctx, func() {
		root := s.view()
		meshErr := ui.Validate(root)
		if meshErr == nil {
			_, meshErr = s.mesh(*s.drawing.View)
		}
		ready <- meshErr
	}); err != nil {
		return err
	}
	return <-ready
}
