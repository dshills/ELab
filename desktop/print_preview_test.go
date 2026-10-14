package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func waitPreviewUI(t *testing.T, s *screen) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.app.Drain()
		ready := false
		s.app.Do(func() {
			root := s.view()
			if err := ui.Validate(root); err != nil {
				t.Fatal(err)
			}
			node, ok := findAction(root, "preview-image")
			ready = ok && node.Image != nil
		})
		if ready {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("preview failed to load: %+v", s.service.Snapshot().PrintPreview)
}
func TestNativePrintPreviewUsesSharedCommandsAndProtectsHiddenCanvas(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned preview test", Width: 1040, Height: 760, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	original := s.service.Snapshot()
	s.app.Do(func() { s.drawing.dragID = "R1"; s.drawing.dragging = true })
	uiAction(t, s, "print-preview", nil)

	waitPreviewUI(t, s)
	s.app.Do(func() {
		if s.drawing.dragID != "" || s.drawing.dragging {
			t.Fatal("preview retained an unfinished drag")
		}
	})
	s.app.Do(func() {
		root := s.view()
		for _, id := range []string{"scope-panel", "readings", "part-search"} {
			if _, ok := findAction(root, id); ok {
				t.Fatalf("preview wastes workspace on %s", id)
			}
		}
		if _, ok := findAction(root, "pdf-paper"); !ok {
			t.Fatal("preview omitted page settings")
		}
	})
	if now := s.service.Snapshot(); now.ScopeVisible != original.ScopeVisible || !reflect.DeepEqual(now.Scope, original.Scope) {
		t.Fatal("preview changed scope acquisition/preferences")
	}
	p := s.service.Snapshot().PrintPreview
	if p.Actor != "user" || !p.Current || p.Width != 1650 {
		t.Fatalf("native capture: %+v", p)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	pixels, err := readPrintPreview(ctx, s.service, *p)
	if err != nil || pixels == nil {
		t.Fatal("agent-compatible chunk decoding", err)
	}
	before := s.service.Snapshot()
	s.app.Do(func() {
		s.drawing.routePointer(0, workbench.Point{X: 100, Y: 100}, workbench.Point{})
		s.drawing.routePointer(3, workbench.Point{}, workbench.Point{Y: 3})
	})
	if s.service.Snapshot().CommandRevision != before.CommandRevision {
		t.Fatal("hidden canvas accepted gestures")
	}
	uiAction(t, s, "pdf-paper", &ui.ChangeEvent{SelectedID: "a4"})
	if s.service.Snapshot().PrintPreview.Current {
		t.Fatal("page edit did not stale preview")
	}
	uiAction(t, s, "preview-refresh", nil)
	waitPreviewUI(t, s)
	newP := s.service.Snapshot().PrintPreview
	if newP.ID == p.ID || newP.Page.Paper != "a4" {
		t.Fatal("refresh did not capture latest defaults")
	}
	uiAction(t, s, "preview-close", nil)
	s.app.Do(func() {
		root := s.view()
		if _, ok := findAction(root, "schematic"); !ok {
			t.Fatal("back did not restore editor")
		}
		if _, ok := findAction(root, "scope-panel"); !ok {
			t.Fatal("back did not restore scope")
		}
		if _, ok := findAction(root, "readings"); !ok {
			t.Fatal("back did not restore readings")
		}
		if s.previewPixels.image != nil {
			t.Fatal("closed pixels retained")
		}
	})
}
