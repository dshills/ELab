package workbench

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestTraceFocusPreservesPhysicsAndRestoresSourceAfterUndo(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"i:D1", "v:C1"}}, "probes")
	before := s.Snapshot()
	focused := call(t, s, "scope.trace.select", IDParams{ID: "i:D1"}, "focus").Result.(Snapshot)
	if focused.TraceFocus == nil || focused.TraceFocus.Unavailable || !reflect.DeepEqual(focused.TraceFocus.Objects, []string{"D1"}) || len(focused.TraceFocus.Terminals) != 2 || focused.RunID != before.RunID || focused.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(focused.Transient, before.Transient) || focused.Selected != before.Selected || focused.Document.Dirty != before.Document.Dirty {
		t.Fatal("focus changed physics/selection or lost source")
	}
	focused.TraceFocus.Objects[0] = "mutated"
	if s.Snapshot().TraceFocus.Objects[0] != "D1" {
		t.Fatal("focus snapshot aliases state")
	}
	call(t, s, "component.move", MoveParams{ID: "D1", Position: Point{500, 500}}, "move")
	if s.Snapshot().TraceFocus.Objects[0] != "D1" {
		t.Fatal("geometry lost focus")
	}
	call(t, s, "object.delete", IDParams{ID: "D1"}, "delete")
	if !s.Snapshot().TraceFocus.Unavailable || len(s.Snapshot().TraceFocus.Objects) != 0 {
		t.Fatal("deleted source masquerades as current")
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if s.Snapshot().TraceFocus.Unavailable {
		t.Fatal("undo did not restore source")
	}
	call(t, s, "scope.probe.remove", IDParams{ID: "i:D1"}, "remove")
	if s.Snapshot().TraceFocus != nil {
		t.Fatal("removed trace retained focus")
	}
}

func TestTraceFocusDefaultProbeReferencesAndProjectReset(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	state := call(t, s, "scope.trace.select", IDParams{ID: "v:Q1.collector"}, "focus").Result.(Snapshot)
	if state.TraceFocus == nil || !reflect.DeepEqual(state.TraceFocus.Terminals, []string{"Q1.collector"}) || !reflect.DeepEqual(state.TraceFocus.Objects, []string{"Q1"}) || len(state.TraceFocus.Nets) != 1 {
		t.Fatal("ground-referenced probe source association is wrong")
	}
	path := filepath.Join(t.TempDir(), "focus.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	call(t, s, "project.load", LoadParams{Path: path}, "load")
	if s.Snapshot().TraceFocus != nil {
		t.Fatal("project replacement retained ephemeral focus")
	}
	call(t, s, "scope.trace.select", IDParams{ID: "v:Q1.collector"}, "again")
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if s.Snapshot().TraceFocus != nil {
		t.Fatal("new project retained focus")
	}
}
func TestTraceFocusValidatesSelectedTraceAndRecordedSeparation(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:C1"}}, "probes")
	refuse(t, s, "scope.trace.select", IDParams{ID: "i:D1"}, "not-found")
	call(t, s, "scope.trace.select", IDParams{ID: "v:C1"}, "focus")
	duration := .01
	_, record := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	call(t, s, "operation.select", IDParams{ID: record.Operation.ID}, "select")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "recorded")
	if !s.Snapshot().TraceFocus.Unavailable {
		t.Fatal("recorded plot highlighted current live circuit")
	}
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "live"}, "live")
	if s.Snapshot().TraceFocus.Unavailable {
		t.Fatal("live focus did not return")
	}
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{}}, "hide")
	if s.Snapshot().TraceFocus != nil {
		t.Fatal("hidden trace kept focus")
	}
}
