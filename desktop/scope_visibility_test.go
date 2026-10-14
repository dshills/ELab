package main

import (
	"reflect"
	"testing"

	"github.com/dshills/vellum/ui"
)

func TestNativeScopeCollapsePreservesDataAndBlocksHiddenCanvasGestures(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.rc"})
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "simulation.restart", nil)
	uiAction(t, s, "transient-step", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "scope-visibility", nil)
	if _, ok := findNode(s.view(), "rc-scope"); ok {
		t.Fatal("collapsed scope kept plot mounted")
	}
	if _, ok := findNode(s.view(), "scope-expand"); !ok {
		t.Fatal("collapsed scope cannot be restored")
	}
	collapsed := s.service.Snapshot()
	s.scope.placeCursor(100)
	if !reflect.DeepEqual(collapsed, s.service.Snapshot()) {
		t.Fatal("hidden canvas gesture changed cursor")
	}
	uiAction(t, s, "scope-expand", nil)
	after := s.service.Snapshot()
	if !after.ScopeVisible || !reflect.DeepEqual(before.Scope, after.Scope) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Document, after.Document) {
		t.Fatal("expand lost live state")
	}
	if _, ok := findNode(s.view(), "rc-scope"); !ok {
		t.Fatal("expanded scope did not remount descriptor")
	}
}
