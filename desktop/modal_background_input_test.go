package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// A hidden plot can still prepare a newer frame while the sheet is open.
// Fresh frame guards must not grant background UI input modal access. Agents
// remain able to issue the same public measurement commands deliberately.
func TestExamplesModalBlocksBackgroundScopeButKeepsAgentAccess(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	uiAction(t, s, "examples-menu", nil)
	s.app.Do(func() {
		if _, err := s.scopeMesh(*s.scope.View); err != nil {
			t.Fatal(err)
		}
	})
	before := s.service.Snapshot()
	s.app.Do(func() { f := s.scope.frame.Load(); s.scope.placeCursor((f.left + f.right) / 2) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("background scope changed a measurement behind the examples sheet")
	}
	agentAction(t, s.service, "scope.cursor.set", workbench.CursorParams{Cursor: "a", Time: .01})
	if a := s.service.Snapshot().Scope.A; a == nil || a.Requested != .01 {
		t.Fatal("modal UI blocked deliberate public-agent measurement", a)
	}
	uiAction(t, s, ui.SheetCancelID(exampleBrowserID), nil)
	s.app.Do(func() { _, _ = s.scopeMesh(*s.scope.View); f := s.scope.frame.Load(); s.scope.placeCursor(f.right) })
	if a := s.service.Snapshot().Scope.A; a == nil || a.Requested != .02 {
		t.Fatal("scope did not resume after dismissing the sheet", a)
	}
}

func TestExamplesModalBlocksBackgroundSweepButKeepsAgentAccess(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "sweep-points", &ui.ChangeEvent{Text: "11"})
	uiAction(t, s, "sweep-start", nil)
	awaitNativeSweep(t, s)
	uiAction(t, s, "examples-menu", nil)
	s.app.Do(func() {
		s.view()
		if _, err := s.sweepMesh(*s.sweepDrawing.View); err != nil {
			t.Fatal(err)
		}
	})
	before := s.service.Snapshot()
	s.app.Do(func() { f := s.sweepDrawing.frame.Load(); s.sweepDrawing.placeCursor((f.left + f.right) / 2) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("background sweep changed a measurement behind the examples sheet")
	}
	index := 5
	agentAction(t, s.service, "sweep.cursor.set", workbench.SweepCursorParams{OperationID: before.DCSweep.Selected, Index: &index})
	if c := s.service.Snapshot().DCSweep.Cursor; c == nil || *c != index {
		t.Fatal("modal UI blocked deliberate agent sweep measurement", c)
	}
	uiAction(t, s, ui.SheetCancelID(exampleBrowserID), nil)
	s.app.Do(func() {
		_, _ = s.sweepMesh(*s.sweepDrawing.View)
		f := s.sweepDrawing.frame.Load()
		s.sweepDrawing.placeCursor(f.right)
	})
	if c := s.service.Snapshot().DCSweep.Cursor; c == nil || *c != 10 {
		t.Fatal("sweep did not resume after dismissing the sheet", c)
	}
}

func TestExamplesModalBlocksBackgroundObjectNavigation(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "examples-menu", nil)
	before := s.service.Snapshot()
	s.app.Do(func() { s.drawing.navigateObject("last", before.CommandRevision, before.Document.Generation) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("background object shortcut changed selection behind the examples sheet")
	}
	agentAction(t, s.service, "selection.navigate", workbench.NavigateParams{Direction: "last", Center: true})
	if s.service.Snapshot().Selected == before.Selected {
		t.Fatal("modal UI blocked deliberate agent navigation")
	}
	uiAction(t, s, ui.SheetCancelID(exampleBrowserID), nil)
	current := s.service.Snapshot()
	s.app.Do(func() { s.drawing.navigateObject("first", current.CommandRevision, current.Document.Generation) })
	if s.service.Snapshot().Selected != current.Components[0].ID {
		t.Fatal("navigation did not resume after dismissing the sheet")
	}
}
