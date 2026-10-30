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
	assertSelection := func(root ui.Node, state workbench.Snapshot) {
		t.Helper()
		palette := paletteForTheme(state.Theme)
		for _, section := range []string{"component", "simulation", "document"} {
			foreground, background := palette.Muted, nativeSegmentColor(palette)
			if section == state.InspectorSection {
				foreground, background = palette.Ink, palette.Paper
			}
			tab, ok := findNode(root, "inspector-section-"+section)
			if !ok || tab.Style.Foreground.Value != foreground || tab.Style.Background.Value != background {
				t.Fatal("tab selection differs from shared state", section, state.InspectorSection)
			}
		}
	}
	for _, section := range []string{"simulation", "document", "component"} {
		before := s.service.Snapshot()
		uiAction(t, s, "inspector-section-"+section, nil)
		after := s.service.Snapshot()
		if after.InspectorSection != section || after.Document != before.Document || !reflect.DeepEqual(after.Transient, before.Transient) {
			t.Fatal("page change did not preserve circuit")
		}
		root := s.view()
		assertSelection(root, after)
		target := map[string]string{"component": "workspace-details-selection", "simulation": "setting-step", "document": "workspace-details-native-document"}[section]
		if _, ok := ui.FocusTarget(root, target); !ok {
			t.Fatal("selected inspector control is unreachable", section, target)
		}
	}
	agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: "document"})
	assertSelection(s.view(), s.service.Snapshot())
	if _, ok := ui.FocusTarget(s.view(), "value-entry"); ok {
		t.Fatal("hidden component field remained focusable")
	}
}
