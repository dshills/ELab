package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeDisplayPreferencesAndGeometryMatchSharedFrames(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "display-voltage", nil)
	uiAction(t, s, "display-current", nil)
	uiAction(t, s, "display-motion", nil)
	after := s.service.Snapshot()
	if !after.Display.Preferences.VoltageColor || !after.Display.Preferences.CurrentCues || !after.Display.Animate || after.RunID != before.RunID || !reflect.DeepEqual(after.Result, before.Result) {
		t.Fatal("native preference controls disagree with semantic display state")
	}
	a, err := schematic(after, *s.drawing.View, .1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := schematic(after, *s.drawing.View, .35)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(a.Vertices(), b.Vertices()) {
		t.Fatal("native mesh ignores shared animation phase")
	}
	uiAction(t, s, "display-direction", &ui.ChangeEvent{SelectedID: "electron"})
	if s.service.Snapshot().Display.Preferences.Direction != "electron" {
		t.Fatal("native direction selector did not invoke shared preferences")
	}
	uiAction(t, s, "display-reduced-motion", nil)
	reduced := s.service.Snapshot()
	a, err = schematic(reduced, *s.drawing.View, .1)
	if err != nil {
		t.Fatal(err)
	}
	b, err = schematic(reduced, *s.drawing.View, .35)
	if err != nil {
		t.Fatal(err)
	}
	if reduced.Display.Animate || !reflect.DeepEqual(a.Vertices(), b.Vertices()) {
		t.Fatal("reduced motion does not freeze native geometry")
	}
	uiAction(t, s, "display-voltage-range", &ui.ChangeEvent{Number: .01})
	if s.service.Snapshot().Display.Preferences.VoltageRange != .01 {
		t.Fatal("native minimum range loses precision at float32 boundary")
	}
	agentAction(t, s.service, "display.preferences.set", workbench.DefaultDisplayPreferences())
	if s.service.Snapshot().Display.Animate {
		t.Fatal("agent disabled preferences leave native motion enabled")
	}
}
