package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestRequestGuidanceRemainsInTheActiveInspector(t *testing.T) {
	for _, section := range []string{"component", "simulation", "document"} {
		t.Run(section, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.rc", nil)
			uiAction(t, s, "simulation.pause", nil)
			uiAction(t, s, "simulation.restart", nil)
			agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
			uiAction(t, s, "inspector-section-"+section, nil)
			before := s.service.Snapshot()
			s.app.Do(func() {
				field, ok := findNode(s.view(), "cursor-time-a")
				if !ok {
					t.Fatal("missing live cursor field")
				}
				field.Change(ui.ChangeEvent{Text: "oops"})
				root := s.view()
				if err := ui.Validate(root); err != nil {
					t.Fatal(err)
				}
				action, ok := findNode(root, "diagnostic-0-action")
				if !ok || s.requestHelp == nil || action.Text != s.requestHelp.Action || !strings.Contains(action.Text, "0.02 s") || !strings.Contains(action.Text, "ms") {
					t.Fatal("active inspector hides the shared correction instructions", section, s.requestHelp)
				}
				message, ok := findNode(root, "messages")
				if !ok || message.Accessibility.Help != s.requestHelp.Action {
					t.Fatal("summary omits the shared instructions from accessible/hover help")
				}
			})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("showing request guidance changed accepted state")
			}
		})
	}
}

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
