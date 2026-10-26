//go:build offscreen

package main

import (
	"image"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// Retain the actual assembled collection, bindings and theme in a reduced
// host. Native modal placement and hardware event rates are separate scopes.
func mountedExampleCollection(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindowSize(t, s, func() ui.Node {
		root := s.view()
		grid, ok := findNode(root, "example-grid")
		if !ok && (!s.service.Snapshot().ExampleBrowser.Visible || s.pendingCommand != "" || s.fileBusy) {
			root.Children = []ui.Node{{ID: "browser-closed", Kind: ui.Label, Text: "Examples closed"}}
			root.Focus = ui.FocusRequest{}
			return root
		}
		if !ok || grid.Collection == nil {
			t.Fatal("assembled browser collection missing")
		}
		root.Children = []ui.Node{grid}
		root.Focus = ui.FocusRequest{ID: "example-grid", Sequence: 1}
		return root
	}, 760, 360, inspectorLayoutTestTheme())
}

func TestMountedQueuedExampleCollectionKeepsAgentContext(t *testing.T) {
	fixtures := []struct{ theme, input, agent string }{}
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		for _, input := range []string{"select", "select-all", "activate"} {
			fixtures = append(fixtures, struct{ theme, input, agent string }{theme, input, "draft"})
		}
	}
	fixtures = append(fixtures, struct{ theme, input, agent string }{"light", "activate", "component"})
	for _, interleave := range []string{"search", "replace"} {
		for _, input := range []string{"select", "select-all", "activate"} {
			fixtures = append(fixtures, struct{ theme, input, agent string }{"light", input, interleave})
		}
	}
	for _, fixture := range fixtures {
		t.Run(fixture.theme+"/"+fixture.input+"/"+fixture.agent, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": fixture.theme})
			agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
			seedExampleBrowserImage(t, s)
			m := mountedExampleCollection(t, s)
			code, mods := key.CodeRightArrow, key.Modifiers(0)
			if fixture.input == "select-all" {
				code, mods = key.CodeA, 1<<key.Meta
			}
			if fixture.input == "activate" {
				code = key.CodeReturnEnter
			}
			before := s.service.Snapshot()
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.KeyChord, Code: code, Modifiers: mods}); err != nil {
				t.Fatal(err)
			}
			step, err := m.manual.Step(m.ctx, 1)
			if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || s.app.Pending().Ordinary == 0 || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("originating collection event not isolated before owner delivery", step, err)
			}
			switch fixture.agent {
			case "draft":
				agentAction(t, s.service, "editor.value.set", workbench.DraftParams{Value: "2k"})
			case "component":
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			case "search":
				agentAction(t, s.service, "examples.search.set", workbench.SearchParams{Query: "motor"})
			case "replace":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			}
			agent := s.service.Snapshot()
			if err := s.app.Invalidate("main"); err != nil {
				t.Fatal(err)
			}
			if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(740, 340)}); err != nil {
				t.Fatal(err)
			}
			m.settle()
			after := s.service.Snapshot()
			if !reflect.DeepEqual(agent, after) || s.pendingCommand != "" {
				t.Fatalf("queued collection %s borrowed agent %s context: revision %d -> %d; generation %d -> %d; selected %q -> %q; draft %q -> %q; pending %q", fixture.input, fixture.agent, agent.CommandRevision, after.CommandRevision, agent.Document.Generation, after.Document.Generation, agent.ExampleBrowser.SelectedID, after.ExampleBrowser.SelectedID, agent.Drawing.Value, after.Drawing.Value, s.pendingCommand)
			}
			if !after.ExampleBrowser.Visible {
				agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
				m.frame()
				agent = s.service.Snapshot()
			}
			// Agent catalogue replacement or explicit reopening can remove the
			// focused row. A new user Tab re-enters the reduced host's collection.
			if m.snapshot().FocusedID != "example-grid" {
				m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeTab})
				if m.snapshot().FocusedID != "example-grid" || !reflect.DeepEqual(agent, s.service.Snapshot()) {
					t.Fatal("fresh keyboard retry did not reacquire collection focus")
				}
			}
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: code, Modifiers: mods})
			fresh := s.service.Snapshot()
			if fixture.input == "activate" {
				if fixture.agent == "component" {
					if s.pendingCommand != "example.load" || !reflect.DeepEqual(agent, fresh) {
						t.Fatal("fresh activation lost dirty-document protection")
					}
				} else if fresh.Document.Generation != agent.Document.Generation+1 || fresh.ExampleBrowser.Visible || fresh.Example == nil || fresh.LastActor != "user" {
					t.Fatal("fresh activation did not resume protected example loading")
				}
			} else if fresh.CommandRevision != agent.CommandRevision+1 || fresh.Document != agent.Document || fresh.Drawing != agent.Drawing || fresh.LastActor != "user" {
				t.Fatal("fresh selection did not resume the shared command")
			}
		})
	}
}

func TestMountedQueuedCollectionOwnKeysContinueWithoutLosingFocus(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agentAction(t, s.service, "examples.visibility.set", workbench.ScopeVisibilityParams{Visible: true})
	seedExampleBrowserImage(t, s)
	m := mountedExampleCollection(t, s)
	before := s.service.Snapshot()
	for range 3 {
		if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.KeyChord, Code: key.CodeRightArrow}); err != nil {
			t.Fatal(err)
		}
		step, err := m.manual.Step(m.ctx, 1)
		if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || !reflect.DeepEqual(before, s.service.Snapshot()) {
			t.Fatal("uninterrupted originating keys were not queued before owner delivery", step, err)
		}
	}
	m.settle()
	after := s.service.Snapshot()
	if after.CommandRevision != before.CommandRevision+3 || after.ExampleBrowser.SelectedID != before.ExampleBrowser.MatchingIDs[3] || after.Document != before.Document || after.Drawing != before.Drawing || m.snapshot().FocusedID != "example-grid" {
		t.Fatal("own queued navigation lost continuation, focus or document", after.ExampleBrowser.SelectedID, after.CommandRevision-before.CommandRevision, m.snapshot().FocusedID)
	}
}
