package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestWorkspaceDividerMatchesAgentAndRejectsStaleChanges(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.nmos"})
	uiAction(t, s, "simulation.pause", nil)
	for _, fraction := range []float32{.35, .85, .65} {
		uiAction(t, s, "workspace-split", &ui.ChangeEvent{Number: fraction})
		expected := float64(fraction)
		if fraction == .35 {
			expected = .35
		} else if fraction == .85 {
			expected = .85
		}
		if s.service.Snapshot().WorkspaceLayout.SchematicFraction != expected || s.errorMessage != "" {
			t.Fatal("native divider did not use shared layout command", s.errorMessage)
		}
	}
	old, ok := findNode(s.view(), "workspace-split")
	if !ok || old.Kind != ui.SplitPane || old.Orientation != ui.Vertical || len(old.Children) != 2 || old.Children[1].Kind != ui.Scroll {
		t.Fatal("missing vertical divider and scrollable scope")
	}
	agentAction(t, s.service, "workspace.layout.set", workbench.WorkspaceLayout{SchematicFraction: .75})
	old.Change(ui.ChangeEvent{Number: .45})
	if s.service.Snapshot().WorkspaceLayout.SchematicFraction != .75 || s.errorMessage == "" {
		t.Fatal("stale divider overwrote agent layout")
	}
	uiAction(t, s, "scope-visibility", nil)
	if _, ok := findNode(s.view(), "workspace-split"); ok {
		t.Fatal("collapsed scope left the divider mounted")
	}
	uiAction(t, s, "scope-expand", nil)
	if split, ok := findNode(s.view(), "workspace-split"); !ok || split.Number != .75 {
		t.Fatal("expanding scope lost accepted split")
	}
}
