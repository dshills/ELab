package workbench

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestScopeVisibilityPreservesPhysicalAndMeasurementState(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	call(t, s, "simulation.step", StepParams{Duration: .02}, "step")
	call(t, s, "scope.cursor.set", CursorParams{Cursor: "a", Time: .01}, "cursor")
	call(t, s, "scope.window.set", map[string]any{"mode": "fixed", "start_s": 0, "end_s": .02}, "window")
	before := s.Snapshot()
	for _, visible := range []bool{false, true, false} {
		call(t, s, "scope.visibility.set", ScopeVisibilityParams{Visible: visible}, "visibility-"+time.Now().String())
		after := s.Snapshot()
		if after.ScopeVisible != visible || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Scope, after.Scope) || !reflect.DeepEqual(before.Document, after.Document) || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.UndoName != after.UndoName || before.View != after.View {
			t.Fatal("scope visibility changed circuit, time, measurements, history or view")
		}
	}
	current := s.Snapshot()
	refuse(t, s, "scope.visibility.set", map[string]any{"visible": nil}, "invalid-params")
	if !reflect.DeepEqual(current, s.Snapshot()) {
		t.Fatal("invalid visibility changed state")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if s.Snapshot().ScopeVisible {
		t.Fatal("project replacement reset session layout choice")
	}
}
func TestCollapsedScopeStillAcquiresSamples(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	if _, err := s.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	call(t, s, "scope.visibility.set", ScopeVisibilityParams{Visible: false}, "hide")
	before := s.Snapshot()
	deadline := time.Now().Add(time.Second)
	for {
		state := s.Snapshot()
		if state.Transient.Sample.Time > before.Transient.Sample.Time {
			if state.ScopeVisible || state.Paused || len(state.Transient.Samples) < len(before.Transient.Samples) {
				t.Fatal("collapse stopped acquisition")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("hidden scope did not acquire another sample")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestScopeVisibilityPreservesRecordedInspectionAndPlayback(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration := .02
	_, operation := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	call(t, s, "operation.select", IDParams{ID: operation.Operation.ID}, "select")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "recorded")
	call(t, s, "recording.cursor.set", CursorParams{Cursor: "a", Time: .01}, "cursor")
	before := s.Snapshot()
	call(t, s, "scope.visibility.set", ScopeVisibilityParams{Visible: false}, "hide")
	if after := s.Snapshot(); !reflect.DeepEqual(before.Recording, after.Recording) || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("collapse changed recorded/live state")
	}
	call(t, s, "recording.play", RecordingPlayParams{Speed: .001}, "play-hidden")
	if !s.Snapshot().Recording.Playing || s.Snapshot().ScopeVisible {
		t.Fatal("hidden scope disabled playback")
	}
	call(t, s, "recording.pause", struct{}{}, "pause-recording")
	paused := s.Snapshot().Recording
	call(t, s, "scope.visibility.set", ScopeVisibilityParams{Visible: true}, "show")
	if !reflect.DeepEqual(paused, s.Snapshot().Recording) {
		t.Fatal("expand changed recorded position or cursor")
	}
}
