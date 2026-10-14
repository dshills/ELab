package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestPinnedNativeCommandsAndDragCommitOnce(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "pin-reading-probe", &ui.ChangeEvent{SelectedID: "v:C1"})
	state := s.service.Snapshot()
	if len(state.PinnedReadings) != 1 || state.PinnedReadings[0].State != "live" || state.RunID != before.RunID || !reflect.DeepEqual(state.Transient, before.Transient) {
		t.Fatal("native pin changes physics or loses its live reading")
	}
	uiAction(t, s, "pin-X-v:C1", &ui.ChangeEvent{Number: 400})
	uiAction(t, s, "pin-Y-v:C1", &ui.ChangeEvent{Number: 300})
	pinned := s.service.Snapshot()
	origin := pinned.PinnedReadings[0].Definition.Position
	s.app.Do(func() {
		s.drawing.routePointer(0, workbench.Point{X: origin.X + 10, Y: origin.Y + 10}, workbench.Point{})
		s.drawing.routePointer(1, workbench.Point{}, workbench.Point{X: 20, Y: 30})
	})
	preview := s.service.Snapshot()
	if preview.CommandRevision != pinned.CommandRevision || preview.PinnedReadings[0].Definition.Position != origin {
		t.Fatal("drag preview commits presentation state")
	}
	s.app.Do(func() { s.drawing.routePointer(2, workbench.Point{}, workbench.Point{X: 10, Y: 0}) })
	moved := s.service.Snapshot()
	if moved.CommandRevision != pinned.CommandRevision+1 || moved.PinnedReadings[0].Definition.Position != (workbench.Point{X: 430, Y: 330}) || moved.RunID != pinned.RunID || !reflect.DeepEqual(moved.Transient, pinned.Transient) || moved.UndoName != "Move pinned reading" {
		t.Fatal("native drag did not make one shared presentation edit")
	}
	uiAction(t, s, "edit.undo", nil)
	if s.service.Snapshot().PinnedReadings[0].Definition.Position != origin {
		t.Fatal("one undo does not reverse complete pin gesture")
	}
	agentAction(t, s.service, "measurement.pin.remove", workbench.IDParams{ID: "v:C1"})
	if len(s.service.Snapshot().PinnedReadings) != 0 {
		t.Fatal("agent remove differs from native pin state")
	}
}

func TestPinnedNativeGestureRefusesInterveningMoveOrProjectReplacement(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "pin-reading-probe", &ui.ChangeEvent{SelectedID: "v:C1"})
	origin := s.service.Snapshot().PinnedReadings[0].Definition.Position
	s.app.Do(func() {
		s.drawing.routePointer(0, workbench.Point{X: origin.X + 10, Y: origin.Y + 10}, workbench.Point{})
	})
	agentAction(t, s.service, "measurement.pin.move", workbench.MoveParams{ID: "v:C1", Position: workbench.Point{X: -400, Y: 200}})
	s.app.Do(func() {
		s.drawing.routePointer(1, workbench.Point{}, workbench.Point{X: 20})
		s.drawing.routePointer(2, workbench.Point{}, workbench.Point{})
	})
	if s.errorMessage == "" || s.service.Snapshot().PinnedReadings[0].Definition.Position.X != -400 {
		t.Fatal("old gesture overwrites intervening move")
	}
	s.errorMessage = ""
	s.app.Do(func() { s.drawing.routePointer(0, workbench.Point{X: -390, Y: 210}, workbench.Point{}) })
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	before := s.service.Snapshot()
	s.app.Do(func() {
		s.drawing.routePointer(1, workbench.Point{}, workbench.Point{X: 20})
		s.drawing.routePointer(2, workbench.Point{}, workbench.Point{})
	})
	if after := s.service.Snapshot(); after.CommandRevision != before.CommandRevision || len(after.PinnedReadings) != 0 {
		t.Fatal("old pin gesture edited replacement project")
	}
}
