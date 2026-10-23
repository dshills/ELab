//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedInspectorNavigationStaysVisibleAndPreservesCircuit(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	view := func() ui.Node {
		rail, ok := findNode(s.view(), "inspector-rail")
		if !ok {
			t.Fatal("missing inspector")
		}
		return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
	}
	m := newMountedControlWindowSize(t, s, view, 320, 540, inspectorLayoutTestTheme())
	clickMountedControl(t, m, "inspector-section-simulation")
	if s.service.Snapshot().InspectorSection != "simulation" {
		t.Fatal("physical navigation did not use the shared command")
	}
	header := mountedControlElement(t, m, "inspector-section-document").Bounds
	step := mountedControlElement(t, m, "setting-step")
	m.input(cogent.ManualInput{Type: events.Scroll, Target: step.ID, TargetIncarnation: step.Incarnation, Delta: math32.Vec2(0, 10000)})
	m.frame()
	if current := mountedControlElement(t, m, "inspector-section-document").Bounds; current.Empty() || current != header {
		t.Fatal("scroll moved inspector navigation", header, current)
	}
	clickMountedControl(t, m, "inspector-section-document")
	if _, ok := ui.FocusTarget(s.view(), "setting-step"); ok {
		t.Fatal("hidden simulation field remained reachable")
	}
	preview := mountedControlElement(t, m, "print-preview")
	if !preview.Bounds.In(mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID).Bounds) {
		t.Fatal("document output inherited another section's scroll")
	}
	after := s.service.Snapshot()
	if after.InspectorSection != "document" || before.Document != after.Document || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("inspector navigation changed the circuit")
	}
}
