package main

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
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
	return <-ready
}
