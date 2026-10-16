package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"reflect"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

// waitNativeViewport waits for a measured viewport and the model's current
// command/generation to reach real mesh preparation, beyond PostWait admission.
func waitNativeViewport(ctx context.Context, a *app.Application, s *screen, ready func(workbench.Snapshot) bool) (workbench.Snapshot, error) {
	type sample struct {
		state                             workbench.Snapshot
		width, height, zoom               float64
		preparedCommand, preparedDocument uint64
		wanted, matched                   bool
	}
	last := sample{}
	observed := false
	failure := func(err error) (workbench.Snapshot, error) {
		if !observed {
			return last.state, fmt.Errorf("%w; no completed owner viewport sample", err)
		}
		return last.state, fmt.Errorf("%w; viewport registered=%t measured=%gx%g drawing=%gx%g zoom=%g/%g command=%d/%d document=%d/%d predicate=%t", err, last.state.Viewport.Registered, last.state.Viewport.Width, last.state.Viewport.Height, last.width, last.height, last.state.View.Zoom, last.zoom, last.state.CommandRevision, last.preparedCommand, last.state.Document.Generation, last.preparedDocument, last.wanted)
	}
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		// PostWait admits owner work; it does not await callback completion.
		// Send an owned result rather than sharing mutable locals with a
		// callback that may finish after the caller's context is canceled.
		done := make(chan sample, 1)
		if err := a.PostWait(ctx, func() {
			state := s.service.Snapshot()
			v := state.Viewport
			p := sample{state: state, width: s.drawing.View.Width, height: s.drawing.View.Height, zoom: s.drawing.View.Zoom, preparedCommand: s.drawing.preparedCommandRevision, preparedDocument: s.drawing.preparedGeneration, wanted: ready(state)}
			p.matched = v.Registered && s.drawing.widget != nil && v.Width == p.width && v.Height == p.height && p.preparedCommand == state.CommandRevision && p.preparedDocument == state.Document.Generation && p.zoom == state.View.Zoom && p.wanted
			done <- p
		}); err != nil {
			return failure(err)
		}
		select {
		case p := <-done:
			last = p
			observed = true
			if p.matched {
				return p.state, nil
			}
		case <-ctx.Done():
			return failure(ctx.Err())
		}
		select {
		case <-ctx.Done():
			return failure(ctx.Err())
		case <-tick.C:
		}
	}
}

func nativeViewportSmoke(ctx context.Context, a *app.Application, s *screen) error {
	owner := func(run func() error) error {
		done := make(chan error, 1)
		if err := a.PostWait(ctx, func() { done <- run() }); err != nil {
			return err
		}
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	wait := func(ready func(workbench.Snapshot) bool) (workbench.Snapshot, error) {
		return waitNativeViewport(ctx, a, s, ready)
	}
	if err := owner(func() error {
		if err := smokeUI(s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.pmos"}); err != nil {
			return err
		}
		if s.pendingCommand == "example.load" {
			if err := smokeUI(s, "discard-continue", nil); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if err := owner(func() error {
		if err := smokeUI(s, "simulation.pause", nil); err != nil {
			return err
		}
		if !s.service.Snapshot().ScopeVisible {
			return smokeUI(s, "scope-expand", nil)
		}
		return nil
	}); err != nil {
		return err
	}
	small, err := wait(func(state workbench.Snapshot) bool {
		return state.Viewport.Following && state.Viewport.FitFailure == nil
	})
	if err != nil {
		return err
	}
	contains := func(state workbench.Snapshot) error {
		check := func(p workbench.Point) error {
			x := (p.X-state.View.Center.X)*state.View.Zoom + state.Viewport.Width/2
			y := (p.Y-state.View.Center.Y)*state.View.Zoom + state.Viewport.Height/2
			if x < -1e-6 || x > state.Viewport.Width+1e-6 || y < -1e-6 || y > state.Viewport.Height+1e-6 {
				return fmt.Errorf("mounted fit clips source point %v", p)
			}
			return nil
		}
		for _, part := range state.Components {
			for _, pin := range part.Terminals {
				p, _ := workbench.TerminalPosition(part, pin)
				if err := check(p); err != nil {
					return err
				}
			}
		}
		for _, annotation := range state.SourceAnnotations {
			lo, hi := workbench.SourceAnnotationBounds(annotation)
			if err := check(lo); err != nil {
				return err
			}
			if err := check(hi); err != nil {
				return err
			}
		}
		return nil
	}
	if err := contains(small); err != nil {
		return err
	}
	if err := owner(func() error { return smokeUI(s, "scope-visibility", nil) }); err != nil {
		return err
	}
	large, err := wait(func(state workbench.Snapshot) bool {
		return state.Viewport.Following && state.Viewport.Height > small.Viewport.Height
	})
	if err != nil {
		return err
	}
	if err := contains(large); err != nil {
		return err
	}
	if large.View.Zoom < small.View.Zoom || large.RunID != small.RunID || large.ElectricalRevision != small.ElectricalRevision || !reflect.DeepEqual(large.Transient, small.Transient) || !reflect.DeepEqual(large.Mixed, small.Mixed) || large.Document != small.Document {
		return fmt.Errorf("scope collapse changed physics/document or failed to enlarge fitted view")
	}
	if err := owner(func() error {
		r := s.invoke("view.pan", map[string]float64{"dx": 20, "dy": 0})
		if r.Error != nil {
			return fmt.Errorf("manual pan: %s", r.Error.Message)
		}
		return smokeUI(s, "scope-expand", nil)
	}); err != nil {
		return err
	}
	manual := s.service.Snapshot().View
	restored, err := wait(func(state workbench.Snapshot) bool {
		return !state.Viewport.Following && state.Viewport.Height < large.Viewport.Height
	})
	if err != nil {
		return err
	}
	if restored.View != manual {
		return fmt.Errorf("mounted resize replaced manual navigation")
	}
	if err := owner(func() error { return smokeUI(s, "view.fit", nil) }); err != nil {
		return err
	}
	fitted, err := wait(func(state workbench.Snapshot) bool {
		return state.Viewport.Following && state.Viewport.FitFailure == nil
	})
	if err != nil {
		return err
	}
	if err := contains(fitted); err != nil {
		return err
	}
	if math.Abs(fitted.View.Zoom-small.View.Zoom) > 1e-6 {
		return fmt.Errorf("fit did not return to current measured workspace")
	}
	evidence, _ := json.Marshal(map[string]any{"event": "elab.viewport.smoke", "passed": true, "expanded_scope": small.Viewport, "collapsed_scope": large.Viewport, "expanded_zoom": small.View.Zoom, "collapsed_zoom": large.View.Zoom, "manual_resize_preserved": true})
	log.Print(string(evidence))
	return nil
}
