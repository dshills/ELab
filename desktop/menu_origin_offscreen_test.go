//go:build offscreen

package main

import (
	"encoding/base64"
	"image"
	"reflect"
	"strings"
	"testing"

	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Use the actual assembled managed selector. Production keeps this control in
// the example sheet's scene instead of converting it to an AppKit proxy. This
// reduced host tests the retained driver queue, not native modal positioning.
func mountedExampleCategory(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindowSize(t, s, func() ui.Node {
		root := s.view()
		n, ok := findNode(root, "example-category")
		if !ok && !s.service.Snapshot().ExampleBrowser.Visible {
			root.Kind = ui.Column // Reduced fixtures stack controls independently of the full-height native shell.
			root.Children = []ui.Node{{ID: "closed-browser", Kind: ui.Label, Text: "Examples closed"}}
			root.Focus = ui.FocusRequest{}
			return root
		}
		if !ok || n.Kind != ui.Selector {
			t.Fatal("assembled browser lost its managed selector")
		}
		root.Kind = ui.Column // Reduced fixtures stack controls independently of the full-height native shell.
		root.Children = []ui.Node{n}
		if root.Focus.ID != n.ID {
			root.Focus = ui.FocusRequest{}
		}
		return root
	}, 700, 600, inspectorLayoutTestTheme())
}

func mountedMenuItem(t *testing.T, m *mountedControlWindow, owner, option string) cogent.ManualElement {
	t.Helper()
	for _, e := range m.snapshot().Elements {
		if strings.HasPrefix(e.ID, "menu:") && e.Parent == owner && strings.HasSuffix(e.ID, ":"+base64.RawURLEncoding.EncodeToString([]byte(option))) {
			return e
		}
	}
	t.Fatal("menu item missing", owner, option, m.snapshot().Elements)
	return cogent.ManualElement{}
}

func TestMountedQueuedExampleCategoryRetainsAgentSearch(t *testing.T) {
	for _, fixture := range []struct{ theme, interleave string }{
		{"light", "search"}, {"dark", "search"}, {"system", "search"}, {"apollo", "search"},
		{"light", "editor"}, {"light", "replace"},
	} {
		t.Run(fixture.theme+"/"+fixture.interleave, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": fixture.theme})
			agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
			seedExampleBrowserImage(t, s)
			m := mountedExampleCategory(t, s)
			clickMountedControl(t, m, "example-category")
			item := mountedMenuItem(t, m, "example-category", "timers")
			at := item.Bounds.Min.Add(item.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			before := s.service.Snapshot()
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseUp, Button: events.Left, Position: at}); err != nil {
				t.Fatal(err)
			}
			step, err := m.manual.Step(m.ctx, 1)
			if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || s.app.Pending().Ordinary == 0 {
				t.Fatal("menu action was not isolated before owner delivery", step, err)
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("menu selection changed state before owner delivery")
			}
			switch fixture.interleave {
			case "search":
				agentAction(t, s.service, "examples.search.set", workbench.SearchParams{Query: "motor"})
			case "editor":
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "2k"})
			case "replace":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			agent := s.service.Snapshot()
			if err := s.app.Invalidate("main"); err != nil {
				t.Fatal(err)
			}
			// Reconcile before draining the queued menu action, through the real
			// driver's public event admission, just as an intervening agent redraw.
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(680, 580)}); err != nil {
				t.Fatal(err)
			}
			m.settle()
			after := s.service.Snapshot()
			if !reflect.DeepEqual(agent, after) {
				t.Fatalf("queued category replaced agent search: query %q -> %q, category %q -> %q, revision %d -> %d", agent.ExampleBrowser.Query, after.ExampleBrowser.Query, agent.ExampleBrowser.Category, after.ExampleBrowser.Category, agent.CommandRevision, after.CommandRevision)
			}
			if !after.ExampleBrowser.Visible {
				agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
				m.frame()
				agent = s.service.Snapshot()
			}
			clickMountedControl(t, m, "example-category")
			item = mountedMenuItem(t, m, "example-category", "timers")
			clickMountedControl(t, m, item.ID)
			fresh := s.service.Snapshot()
			if m.snapshot().FocusedID != "example-category" {
				t.Fatal("fresh category choice lost keyboard focus")
			}
			if fresh.ExampleBrowser.Query != "" || fresh.ExampleBrowser.Category != "timers" || fresh.LastActor != "user" || fresh.CommandRevision != agent.CommandRevision+1 {
				t.Fatal("fresh category choice did not resume shared commands", fresh.ExampleBrowser, s.errorMessage)
			}
		})
	}
}

func TestMountedHeldExampleCategoryRetainsAgentSearch(t *testing.T) {
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		t.Run(theme, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
			agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
			seedExampleBrowserImage(t, s)
			m := mountedExampleCategory(t, s)
			clickMountedControl(t, m, "example-category")
			item := mountedMenuItem(t, m, "example-category", "timers")
			at := item.Bounds.Min.Add(item.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			agentAction(t, s.service, "examples.search.set", workbench.SearchParams{Query: "motor"})
			agent := s.service.Snapshot()
			m.frame()
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			if !reflect.DeepEqual(agent, s.service.Snapshot()) {
				t.Fatal("held category release replaced agent search")
			}
			clickMountedControl(t, m, "example-category")
			item = mountedMenuItem(t, m, "example-category", "timers")
			clickMountedControl(t, m, item.ID)
			fresh := s.service.Snapshot()
			if fresh.ExampleBrowser.Category != "timers" || fresh.ExampleBrowser.Query != "" || fresh.LastActor != "user" || fresh.CommandRevision != agent.CommandRevision+1 {
				t.Fatal("fresh category after held refusal did not work")
			}
		})
	}
}

func TestMountedQueuedNativeFitRetainsAgentViewport(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "view.pan", map[string]float64{"dx": 20, "dy": 0})
	m := newMountedControlWindowSize(t, s, func() ui.Node {
		root := s.view()
		bar, ok := findNode(root, "header")
		if !ok {
			t.Fatal("missing assembled native toolbar")
		}
		root.Kind = ui.Column
		root.Children, root.Focus = []ui.Node{bar}, ui.FocusRequest{}
		return root
	}, 1040, 180, inspectorLayoutTestTheme())
	item := mountedControlElement(t, m, "view.fit")
	at := item.Bounds.Min.Add(item.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	before := s.service.Snapshot()
	if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseUp, Button: events.Left, Position: at}); err != nil {
		t.Fatal(err)
	}
	step, err := m.manual.Step(m.ctx, 1)
	if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || s.app.Pending().Ordinary == 0 || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("Fit was not isolated in the real owner queue", step, err)
	}
	agentAction(t, s.service, "view.pan", map[string]float64{"dx": 40, "dy": 0})
	agent := s.service.Snapshot()
	if err := s.app.Invalidate("main"); err != nil {
		t.Fatal(err)
	}
	if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(300, 160)}); err != nil {
		t.Fatal(err)
	}
	m.settle()
	if !reflect.DeepEqual(agent, s.service.Snapshot()) {
		t.Fatal("old native Fit overwrote agent viewport")
	}
	clickMountedControl(t, m, "view.fit")
	fresh := s.service.Snapshot()
	if !fresh.Viewport.Following || fresh.CommandRevision != agent.CommandRevision+1 || fresh.LastActor != "user" || !reflect.DeepEqual(agent.Components, fresh.Components) || !reflect.DeepEqual(agent.Result, fresh.Result) {
		t.Fatal("fresh Fit did not resume the shared viewport command")
	}
}

func TestMountedFreshExampleCategoryFocus(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
	seedExampleBrowserImage(t, s)
	m := mountedExampleCategory(t, s)
	clickMountedControl(t, m, "example-category")
	item := mountedMenuItem(t, m, "example-category", "timers")
	clickMountedControl(t, m, item.ID)
	if m.snapshot().FocusedID != "example-category" {
		t.Fatal("normal category selection lost keyboard focus")
	}
}
