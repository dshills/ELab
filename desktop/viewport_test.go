package main

import (
	"reflect"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func testViewportScreen(t *testing.T, capacity int) *screen {
	t.Helper()
	a, err := app.New(app.Config{QueueCapacity: capacity})
	if err != nil {
		t.Fatal(err)
	}
	s := &screen{app: a, service: workbench.New(), drawing: &commandCanvas{widget: &core.WidgetBase{}}}
	t.Cleanup(func() { a.Close(); s.service.Close() })
	return s
}
func drainViewport(t *testing.T, s *screen, ready func() bool) {
	t.Helper()
	end := time.Now().Add(time.Second)
	for time.Now().Before(end) {
		if err := s.app.Drain(); err != nil {
			t.Fatal(err)
		}
		if ready() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("viewport callback did not publish")
}

func TestNativeViewportDefersCoalescesAndPreservesMessages(t *testing.T) {
	s := testViewportScreen(t, 8)
	s.errorMessage = "existing error"
	s.fileMessage = "Saved existing file"
	v := canvas.Viewport{Width: 520, Height: 138, Zoom: 1}
	before := s.service.Snapshot()
	_ = s.app.Do(func() {
		s.observeSchematicViewport(canvas.Viewport{Width: 400, Height: 200, Zoom: 1}, before)
		s.observeSchematicViewport(v, before)
	})
	if s.service.Snapshot().Viewport.Registered {
		t.Fatal("viewport mutated the model inside paint")
	}
	drainViewport(t, s, func() bool { return s.service.Snapshot().Viewport.Registered })
	after := s.service.Snapshot()
	if after.CommandRevision != before.CommandRevision+1 || after.Viewport.Width != 520 || after.Viewport.Height != 138 || after.View != before.View || s.errorMessage != "existing error" || s.fileMessage != "Saved existing file" {
		t.Fatal("measurement did not coalesce or damaged presentation", after.Viewport)
	}
	_ = s.app.Do(func() { s.observeSchematicViewport(v, after) })
	_ = s.app.Drain()
	if s.service.Snapshot().CommandRevision != after.CommandRevision {
		t.Fatal("same size keeps publishing")
	}
}

func TestNativeStaleViewportAndRetiredWidgetRefuse(t *testing.T) {
	for _, change := range []string{"pan", "replace", "retire"} {
		t.Run(change, func(t *testing.T) {
			s := testViewportScreen(t, 8)
			before := s.service.Snapshot()
			v := canvas.Viewport{Width: 520, Height: 138, Zoom: 1}
			_ = s.app.Do(func() { s.observeSchematicViewport(v, before) })
			if change == "pan" {
				agentAction(t, s.service, "view.pan", map[string]float64{"dx": 20, "dy": 0})
			}
			if change == "replace" {
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "pmos"})
				agentAction(t, s.service, "simulation.pause", struct{}{})
			}
			if change == "retire" {
				_ = s.app.Do(func() { s.drawing.widget = &core.WidgetBase{} })
			}
			current := s.service.Snapshot()
			_ = s.app.Drain()
			if !reflect.DeepEqual(current, s.service.Snapshot()) || s.viewportPending != nil || s.errorMessage != "" {
				t.Fatal("stale/retired viewport changed current state or user error")
			}
		})
	}
}

func TestNativeViewportResizeReturningToMeasuredSizeCancelsPendingReport(t *testing.T) {
	s := testViewportScreen(t, 8)
	agentAction(t, s.service, "view.viewport.set", workbench.ViewportParams{Width: 520, Height: 138})
	before := s.service.Snapshot()
	_ = s.app.Do(func() {
		s.observeSchematicViewport(canvas.Viewport{Width: 520, Height: 420, Zoom: 1}, before)
		s.observeSchematicViewport(canvas.Viewport{Width: 520, Height: 138, Zoom: 1}, before)
	})
	_ = s.app.Drain()
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.viewportPending != nil {
		t.Fatal("resize back to measured size published the superseded report")
	}
}

func TestNativeViewportBackpressureRetainsOneDelivery(t *testing.T) {
	s := testViewportScreen(t, 1)
	if err := s.app.Post(func() {}); err != nil {
		t.Fatal(err)
	}
	before := s.service.Snapshot()
	_ = s.app.Do(func() { s.observeSchematicViewport(canvas.Viewport{Width: 520, Height: 138, Zoom: 1}, before) })
	drainViewport(t, s, func() bool { return s.service.Snapshot().Viewport.Registered })
	if s.service.Snapshot().CommandRevision != before.CommandRevision+1 {
		t.Fatal("backpressure duplicated/lost viewport publication")
	}
}

func TestNativeFitAndMeasuredResizeMatchAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.pmos"})
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "pmos"})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, s, command, nil)
		agentAction(t, agent, command, struct{}{})
	}
	for i, size := range []workbench.ViewportParams{{Width: 520, Height: 138}, {Width: 520, Height: 420}} {
		_ = s.app.Do(func() {
			s.drawing.widget = &core.WidgetBase{}
			_ = s.drawing.View.SetSize(size.Width, size.Height)
			s.observeSchematicViewport(*s.drawing.View, s.service.Snapshot())
		})
		drainViewport(t, s, func() bool {
			v := s.service.Snapshot().Viewport
			return v.Width == size.Width && v.Height == size.Height
		})
		agentAction(t, agent, "view.viewport.set", size)
		left, right := s.service.Snapshot(), agent.Snapshot()
		if left.View != right.View || left.Viewport.Following != right.Viewport.Following || !reflect.DeepEqual(left.Mixed, right.Mixed) {
			t.Fatalf("native measurement %d diverged", i)
		}
	}
	// A native Fit explicitly re-enables following, using the same measured size.
	agentAction(t, s.service, "view.pan", map[string]float64{"dx": 10, "dy": 0})
	agentAction(t, agent, "view.pan", map[string]float64{"dx": 10, "dy": 0})
	uiAction(t, s, "view.fit", nil)
	agentAction(t, agent, "view.fit", workbench.FitParams{Width: 520, Height: 420, FollowViewport: true})
	if s.service.Snapshot().View != agent.Snapshot().View || !s.service.Snapshot().Viewport.Following {
		t.Fatal("Fit mode diverged")
	}
}
