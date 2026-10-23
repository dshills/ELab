package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestInspectorPagesMatchAgentAndKeepControlsReachable(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	for _, section := range []string{"simulation", "document", "component"} {
		before := s.service.Snapshot()
		uiAction(t, s, "inspector-section-"+section, nil)
		after := s.service.Snapshot()
		if after.InspectorSection != section || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) {
			t.Fatal("page change did not preserve circuit")
		}
		root := s.view()
		tabs, ok := findNode(root, "inspector-section-"+section)
		if !ok || tabs.Style.Background.Value != paletteForTheme(after.Theme).Accent {
			t.Fatal("tab selection differs from shared state")
		}
		target := map[string]string{"component": "selection", "simulation": "setting-step", "document": "pdf-paper"}[section]
		if _, ok := ui.FocusTarget(root, target); !ok {
			t.Fatal("selected inspector control is unreachable", section, target)
		}
	}
	agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "document"})
	tabs, _ := findNode(s.view(), "inspector-section-document")
	if tabs.Style.Background.Value != paletteForTheme(s.service.Snapshot().Theme).Accent {
		t.Fatal("external selection did not appear in UI")
	}
	if _, ok := ui.FocusTarget(s.view(), "value-entry"); ok {
		t.Fatal("hidden component field remained focusable")
	}
}
