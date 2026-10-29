//go:build offscreen

package main

import (
	"image/color"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func inspectorScrollNode(t *testing.T, root ui.Node) ui.Node {
	t.Helper()
	if rail, ok := findInspectorScroll(root); ok {
		return rail
	}
	t.Fatal("inspector scroll host missing")
	return ui.Node{}
}

func findInspectorScroll(root ui.Node) (ui.Node, bool) {
	if root.Kind == ui.Scroll && len(root.Children) == 1 && root.Children[0].ID == "sidebar-content" {
		return root, true
	}
	for _, child := range root.Children {
		if found, ok := findInspectorScroll(child); ok {
			return found, true
		}
	}
	return ui.Node{}, false
}

func TestMountedInspectorScrollReturnsToStartForNewDocument(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.npn-pulse", nil)
	uiAction(t, s, "simulation.pause", nil)
	duration := .5
	agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration, ProbeIDs: []string{"v:VIN.positive", "v:Q1.collector"}})
	completedVerification(t, s.service, s.service.Snapshot().LastOperation)
	uiAction(t, s, "recording-show", nil)
	uiAction(t, s, "recorded-timing-probe", &ui.ChangeEvent{SelectedID: "v:VIN.positive"})
	uiAction(t, s, "inspector-section-component", nil)
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "RB"})
	view := func() ui.Node {
		return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{inspectorScrollNode(t, s.view())}}
	}
	theme := ui.Theme{Default: ui.Style{Foreground: ui.Some(color.NRGBA{R: 24, G: 24, B: 24, A: 255}), Background: ui.Some(color.NRGBA{R: 248, G: 248, B: 248, A: 255}), Typography: ui.Some(ui.Typography{Family: "Noto Sans", Size: 16})}}
	m := newMountedControlWindowSize(t, s, view, 340, 540, theme)
	topHeader := func() string {
		object, ok := findNode(s.view(), "inspector-group-object")
		if !ok || len(object.Children) == 0 {
			t.Fatal("missing object inspector header")
		}
		return object.Children[0].ID
	}
	headerID := topHeader()
	rail := mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID)
	header := mountedControlElement(t, m, headerID)
	if header.Bounds.Intersect(rail.Bounds).Empty() {
		t.Fatal("initial inspector header is not in view")
	}
	m.input(cogent.ManualInput{Type: events.Scroll, Target: rail.ID, TargetIncarnation: rail.Incarnation, Delta: math32.Vec2(0, 10000)})
	m.frame()
	header = mountedControlElement(t, m, headerID)
	if !header.Bounds.Intersect(rail.Bounds).Empty() {
		t.Fatal("fixture did not scroll past the inspector header")
	}
	before := header.Bounds
	m.frame()
	if mountedControlElement(t, m, headerID).Bounds != before {
		t.Fatal("ordinary repaint reset the current document's scroll")
	}
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "RB", Value: "12k"})
	if _, err := s.service.Wait(m.ctx); err != nil {
		t.Fatal(err)
	}
	m.frame()
	if current := mountedControlElement(t, m, rail.ID); current.Incarnation != rail.Incarnation || !mountedControlElement(t, m, headerID).Bounds.Intersect(current.Bounds).Empty() {
		t.Fatal("ordinary component edit reset the current inspector viewport")
	}
	agentAction(t, s.service, "example.reset", map[string]bool{"discard_changes": true})
	m.frame()
	rail = mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID)
	headerID = topHeader()
	header = mountedControlElement(t, m, headerID)
	if header.Bounds.Intersect(rail.Bounds).Empty() {
		t.Fatalf("new LED document inherited the long recording's scroll: header %v, rail %v", header.Bounds, rail.Bounds)
	}
}
