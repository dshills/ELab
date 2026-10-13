package workbench

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileActionsRefuseReplacedDocument(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	generation := s.Snapshot().Document.Generation
	path := filepath.Join(t.TempDir(), "original.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save-original")
	call(t, s, "example.rc", ReplaceParams{}, "replace")
	before := s.Snapshot()
	exportPath := filepath.Join(t.TempDir(), "wrong.svg")
	refuse(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "wrong.elab"), ExpectedGeneration: &generation}, "conflict")
	refuse(t, s, "project.load", LoadParams{Path: path, DiscardChanges: true, ExpectedGeneration: &generation}, "conflict")
	refuse(t, s, "export.schematic.svg", ExportParams{Path: exportPath, ExpectedGeneration: &generation}, "conflict")
	if _, err := os.Stat(exportPath); !os.IsNotExist(err) {
		t.Fatal("stale file action published artifact")
	}
	// Only autonomous clock ticks may have changed; document and graph must remain.
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || before.Document != after.Document || before.CommandRevision != after.CommandRevision {
		t.Fatal("stale file action changed replacement project")
	}
	current := after.Document.Generation
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "current.elab"), ExpectedGeneration: &current}, "save-current")
}
func TestCursorOnDecimalWindowBoundary(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.03}, "time")
	call(t, s, "scope.window.set", ScopeWindow{Mode: "fixed", Start: .01, End: .02}, "window")
	call(t, s, "scope.cursor.set", CursorParams{Cursor: "a", Time: .01}, "a")
	call(t, s, "scope.cursor.set", CursorParams{Cursor: "b", Time: .02}, "b")
	scope := s.Snapshot().Scope
	if !scope.A.Valid || !scope.B.Valid || scope.DeltaSeconds == nil || math.Abs(*scope.DeltaSeconds-.01) > 1e-12 {
		t.Fatal("floating-point endpoint invalidated retained cursor")
	}
}
func TestDuplicateAtDrawingBoundaryFindsSpace(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{Kind: "capacitor", Position: Point{10000, 10000}, Value: "10uF"}, "corner")
	id := s.Snapshot().Selected
	call(t, s, "component.duplicate", DuplicateParams{ID: id}, "duplicate")
	state := s.Snapshot()
	p, ok := s.graph.Component(state.Selected)
	if !ok || p.ID == id || p.Position == (Point{10000, 10000}) || absOutOfBounds(p.Position) || p.Position.X > 10000 || p.Position.Y > 10000 {
		t.Fatal("automatic duplicate failed to find valid nearby position")
	}
}
