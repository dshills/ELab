package main

import (
	"math"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
)

func scopeContextRC(t *testing.T) (*screen, func()) {
	t.Helper()
	s, cleanup := testScreen(t)
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "simulation.restart", struct{}{})
	agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
	s.app.Do(func() {
		if _, err := s.scopeMesh(*s.scope.View); err != nil {
			t.Fatal(err)
		}
	})
	return s, cleanup
}

func TestScopeShortDragUsesStartAndReleaseWithoutMoveEvents(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	before := s.service.Snapshot()
	s.app.Do(func() {
		frame := s.scope.frame.Load()
		point := func(f float64) float64 { return frame.left + f*(frame.right-frame.left) }
		s.scope.cursorPointer(frame, events.MouseDown, events.Left, point(.1))
		s.scope.cursorPointer(frame, events.SlideStart, events.Left, point(.6))
		if got := s.service.Snapshot().Scope.A; got == nil || math.Abs(got.Requested-.012) > 1e-12 {
			t.Fatal("single movement did not update cursor", got)
		}
		s.scope.cursorPointer(frame, events.SlideStop, events.Left, point(.9))
	})
	after := s.service.Snapshot()
	if after.Scope.A == nil || math.Abs(after.Scope.A.Requested-.018) > 1e-12 || s.scope.gesture != nil {
		t.Fatal("final release position lost or gesture retained", after.Scope.A)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestScopeFastReleaseWithoutAnyDragClassification(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	before := s.service.Snapshot()
	s.app.Do(func() {
		frame := s.scope.frame.Load()
		point := func(f float64) float64 { return frame.left + f*(frame.right-frame.left) }
		s.scope.cursorPointer(frame, events.MouseDown, events.Left, point(.1))
		s.scope.cursorPointer(frame, events.MouseUp, events.Left, point(.9))
	})
	after := s.service.Snapshot()
	if after.Scope.A == nil || math.Abs(after.Scope.A.Requested-.018) > 1e-12 || s.scope.gesture != nil {
		t.Fatal("final release position lost or gesture retained", after.Scope.A)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestScopeSecondaryClickAndReleaseCannotRetargetMeasurements(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	s.app.Do(func() {
		frame := s.scope.frame.Load()
		s.scope.cursorPointer(frame, events.MouseDown, events.Left, frame.left)
	})
	before := s.service.Snapshot()
	s.app.Do(func() {
		frame := s.scope.frame.Load()
		for _, kind := range []events.Types{events.MouseDown, events.SlideStart, events.SlideMove, events.SlideStop} {
			s.scope.cursorPointer(frame, kind, events.Right, frame.right)
		}
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.scope.gesture != nil {
		t.Fatal("secondary gesture changed a measurement")
	}
	s.app.Do(func() {
		frame := s.scope.frame.Load()
		s.scope.cursorPointer(frame, events.MouseDown, events.Left, frame.left)
	})
	agentAction(t, s.service, "scope.cursor.select", workbench.CursorSelectParams{Cursor: "b"})
	before = s.service.Snapshot()
	s.app.Do(func() { s.scope.cursorPointer(s.scope.frame.Load(), events.SlideStop, events.Left, 500) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("release adopted an intervening agent change")
	}
}

func TestScopeClickRejectsStaleWindowAndFreshFrameUsesCapturedAxis(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	old := s.scope.frame.Load()
	if old == nil {
		t.Fatal("fresh scope has no context")
	}
	agentAction(t, s.service, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: .004, End: .016})
	before := s.service.Snapshot()
	s.app.Do(func() { s.scope.placeCursor((old.left + old.right) / 2) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old frame used new interval")
	}
	s.app.Do(func() {
		if _, err := s.scopeMesh(*s.scope.View); err != nil {
			t.Fatal(err)
		}
		frame := s.scope.frame.Load()
		s.scope.placeCursor((frame.left + frame.right) / 2)
	})
	if got := s.service.Snapshot().Scope.A; got == nil || got.Requested != .01 {
		t.Fatal("fresh frame did not use its captured interval", got)
	}
}

func TestScopeDragOnlyContinuesItsOwnCursorWrites(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	before := s.service.Snapshot()
	s.app.Do(func() {
		frame := s.scope.frame.Load()
		s.scope.cursorGesture(frame, frame.left+(frame.right-frame.left)*.25, true)
		s.scope.cursorGesture(frame, frame.left+(frame.right-frame.left)*.75, false)
	})
	state := s.service.Snapshot()
	if state.Scope.A == nil || state.Scope.A.Requested != .015 || state.CommandRevision != before.CommandRevision+2 || s.scope.gesture == nil {
		t.Fatal("drag cannot continue before its own repaint")
	}
	agentAction(t, s.service, "scope.cursor.select", workbench.CursorSelectParams{Cursor: "b"})
	intervened := s.service.Snapshot()
	s.app.Do(func() { s.scope.cursorGesture(s.scope.frame.Load(), 100, false) })
	if !reflect.DeepEqual(intervened, s.service.Snapshot()) || s.scope.gesture != nil {
		t.Fatal("drag overwrote an intervening actor/cursor selection")
	}
	if state.RunID != before.RunID || state.Transient.Sample != before.Transient.Sample || state.Document != before.Document || !reflect.DeepEqual(state.UndoEntry, before.UndoEntry) {
		t.Fatal("cursor drag changed live circuit/history")
	}
}

func TestScopeGestureStopsWhenPreparedContextIsInvalidated(t *testing.T) {
	s, cleanup := scopeContextRC(t)
	defer cleanup()
	s.app.Do(func() {
		frame := s.scope.frame.Load()
		s.scope.cursorGesture(frame, (frame.left+frame.right)/2, true)
		before := s.service.Snapshot()
		s.scope.frame.Store(nil)
		s.scope.cursorGesture(frame, frame.right, false)
		if !reflect.DeepEqual(before, s.service.Snapshot()) || s.scope.gesture != nil {
			t.Fatal("invalidated canvas still accepts a queued gesture")
		}
	})
}

func TestScopeClickCannotRetargetAnotherCaptureBeforeRedraw(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	duration := .02
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
	first := s.service.Snapshot().LastOperation
	completedVerification(t, s.service, first)
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
	second := s.service.Snapshot().LastOperation
	completedVerification(t, s.service, second)
	agentAction(t, s.service, "operation.select", workbench.IDParams{ID: first})
	agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
	s.app.Do(func() {
		if _, err := s.scopeMesh(*s.scope.View); err != nil {
			t.Fatal(err)
		}
	})
	agentAction(t, s.service, "operation.select", workbench.IDParams{ID: second})
	before := s.service.Snapshot()
	s.app.Do(func() { s.scope.placeCursor(58 + (s.scope.View.Width-78-58)*.5) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old scope click placed a cursor on the newly selected capture")
	}
}

func assertCursorMeasurementPreservesPhysics(t *testing.T, before, after workbench.Snapshot) {
	t.Helper()
	if before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Document != after.Document ||
		!reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) ||
		!reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Mixed, after.Mixed) ||
		!reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.RedoEntry, after.RedoEntry) {
		t.Fatal("cursor measurement changed physics, document or circuit history")
	}
}
