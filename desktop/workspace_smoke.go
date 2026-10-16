package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"reflect"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Runs after nativeViewportSmoke leaves a paused, fitted MOSFET example.
func nativeWorkspaceSmoke(ctx context.Context, s *screen) error {
	owner := func(run func() error) error {
		done := make(chan error, 1)
		if err := s.app.PostWait(ctx, func() { done <- run() }); err != nil {
			return err
		}
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	before, err := waitNativeViewport(ctx, s.app, s, func(state workbench.Snapshot) bool {
		return state.ScopeVisible && state.Paused && state.Viewport.Following
	})
	if err != nil {
		return err
	}
	if err := owner(func() error {
		return smokeUI(s, "workspace-split", &ui.ChangeEvent{Number: .8})
	}); err != nil {
		return err
	}
	large, err := waitNativeViewport(ctx, s.app, s, func(state workbench.Snapshot) bool {
		if state.Viewport.Height <= before.Viewport.Height || state.WorkspaceLayout.SchematicFraction <= .79 || s.scope.widget == nil {
			return false
		}
		// Drawing preparation can finish before the scope's new scrollbar is
		// positioned. HasScroll alone also includes an unconfigured slider with
		// its default 0..1 range; wait for actual overflow geometry and params.
		host, ok := s.scope.widget.ParentByName("scope-scroll").(core.Layouter)
		if !ok {
			return false
		}
		frame := host.AsFrame()
		bar := frame.Scrolls[math32.Y]
		maximum, visible, _ := frame.ScrollValues(math32.Y)
		return frame.HasScroll[math32.Y] && bar != nil && maximum > visible && visible > 0 && bar.Max == maximum && bar.Step == 1 && bar.Geom.Size.Actual.Content.Y > 0
	})
	if err != nil {
		return err
	}
	if large.RunID != before.RunID || large.ElectricalRevision != before.ElectricalRevision || large.Document != before.Document || !reflect.DeepEqual(large.Mixed, before.Mixed) || !reflect.DeepEqual(large.Transient, before.Transient) || !reflect.DeepEqual(large.Scope, before.Scope) {
		return fmt.Errorf("divider changed physics/document/measurements")
	}
	if err := owner(func() error {
		if s.scope.View.Width < s.drawing.View.Width {
			return fmt.Errorf("scope failed to fill split width: %g < %g", s.scope.View.Width, s.drawing.View.Width)
		}
		plot := s.scope.widget
		if plot == nil {
			return fmt.Errorf("scope plot has no mounted widget")
		}
		host, ok := plot.ParentByName("scope-scroll").(core.Widget)
		if !ok {
			return fmt.Errorf("scope has no scroll host")
		}
		frame, ok := host.(core.Layouter)
		if !ok || !frame.AsFrame().HasScroll[math32.Y] {
			return fmt.Errorf("short scope has no active scroll range")
		}
		bar := frame.AsFrame().Scrolls[math32.Y]
		// Prior captured/live views can retain any valid offset, including the
		// bottom. Exercise the wheel from an explicit start instead of assuming
		// another fixture left room to scroll farther down.
		frame.AsFrame().ScrollDimToContentEnd(math32.Y)
		if !frame.AsFrame().ScrollDimToContentStart(math32.Y) || !frame.AsFrame().IsDimAtContentStart(math32.Y) {
			return fmt.Errorf("scope wheel fixture could not return to content start")
		}
		position, zoom := bar.Value, s.scope.View.Zoom
		plot.HandleEvent(events.NewScroll(image.Point{}, math32.Vec2(0, 100), 0))
		if bar.Value <= position || s.scope.View.Zoom != zoom {
			return fmt.Errorf("plot wheel failed: scroll %g to %g, range %g to %g, zoom %g to %g", position, bar.Value, bar.Min, bar.Max, zoom, s.scope.View.Zoom)
		}
		r := s.invoke("view.pan", map[string]float64{"dx": 20, "dy": 0})
		if r.Error != nil {
			return fmt.Errorf("manual split navigation: %s", r.Error.Message)
		}
		return smokeUI(s, "workspace-split", &ui.ChangeEvent{Number: .4})
	}); err != nil {
		return err
	}
	manual := s.service.Snapshot().View
	small, err := waitNativeViewport(ctx, s.app, s, func(state workbench.Snapshot) bool {
		return state.Viewport.Height < large.Viewport.Height && !state.Viewport.Following
	})
	if err != nil {
		return err
	}
	if small.View != manual {
		return fmt.Errorf("split resize overwrote manual positioning")
	}
	// Restore this harness's layout so later pixel fixtures retain their own
	// expected drawing space rather than inheriting this test's 40% setting.
	if err := owner(func() error {
		return smokeUI(s, "workspace-split", &ui.ChangeEvent{Number: float32(before.WorkspaceLayout.SchematicFraction)})
	}); err != nil {
		return err
	}
	if _, err := waitNativeViewport(ctx, s.app, s, func(state workbench.Snapshot) bool {
		return state.Viewport.Height == before.Viewport.Height && !state.Viewport.Following && state.View == manual
	}); err != nil {
		return err
	}
	evidence, _ := json.Marshal(map[string]any{"event": "elab.workspace.smoke", "passed": true, "default_viewport": before.Viewport, "large_viewport": large.Viewport, "small_viewport": small.Viewport, "manual_resize_preserved": true, "scope_fills_width": true, "plot_wheel_scrolls_controls_without_zoom": true})
	log.Print(string(evidence))
	return nil
}
