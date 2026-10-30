package main

import (
	"image/color"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestWidgetSearchClearParityAndStaleAction(t *testing.T) {
	user, closeUser := testScreen(t)
	t.Cleanup(closeUser)
	agent, closeAgent := testScreen(t)
	t.Cleanup(closeAgent)
	uiAction(t, user, "part-search", &ui.ChangeEvent{Text: "MOSFET"})
	agentAction(t, agent.service, "parts.search.set", workbench.SearchParams{Query: "MOSFET"})
	uiAction(t, user, "part-search-clear", nil)
	agentAction(t, agent.service, "parts.search.set", workbench.SearchParams{Query: ""})
	if !sameBrowserState(user.service.Snapshot(), agent.service.Snapshot()) || user.keyboardFocus.ID != "part-search" || user.fieldContext != nil || user.fieldAccepted != nil {
		t.Fatal("Clear diverges from the shared search command or loses entry focus")
	}
	uiAction(t, user, "part-search", &ui.ChangeEvent{Text: "resistor"})
	var old ui.Node
	user.app.Do(func() { old, _ = findNode(user.view(), "part-search-clear") })
	agentAction(t, user.service, "parts.search.set", workbench.SearchParams{Query: "battery"})
	before := user.service.Snapshot()
	user.app.Do(func() {
		user.view() // Refresh the retained composite's current callback.
		old.Activate()
	})
	if !reflect.DeepEqual(before, user.service.Snapshot()) {
		t.Fatal("old Clear action erased an intervening agent query")
	}
}

func TestWidgetSystemReducedMotionRepaintsWithoutChangingPreferences(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "display-current", nil)
	uiAction(t, s, "display-motion", nil)
	before := s.service.Snapshot()
	if !before.Display.Animate {
		t.Fatal("fixture has no current animation")
	}
	s.app.Do(func() {
		appearance := cogent.SystemAppearance{Known: true, Accent: accent}
		s.observeSystemAppearance(appearance)
		s.view()
		previous := s.renderRevision
		appearance.ReduceMotion = true
		s.observeSystemAppearance(appearance)
		root := s.view()
		if s.renderRevision == previous || !root.Theme.ReducedMotion {
			t.Fatal("motion-only preference failed to invalidate the schematic")
		}
		s.displayStarted = time.Now().Add(-100 * time.Millisecond)
		a, err := s.mesh(*s.drawing.View)
		if err != nil {
			t.Fatal(err)
		}
		s.displayStarted = time.Now().Add(-400 * time.Millisecond)
		b, err := s.mesh(*s.drawing.View)
		if err != nil || !reflect.DeepEqual(a.Vertices(), b.Vertices()) {
			t.Fatal("system reduced motion left time-varying current dots", err)
		}
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("environmental reduced motion overwrote saved display settings or physics")
	}
}

func TestWidgetFormValidationKeepsSharedGuidanceAndEditor(t *testing.T) {
	s, closeScreen := testScreen(t)
	t.Cleanup(closeScreen)
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	before := s.service.Snapshot()
	s.app.Do(func() {
		s.invoke("component.set", workbench.ValueParams{ID: "R1", Value: "0"})
		root := s.view()
		field, ok := findNode(root, "value-entry")
		group, grouped := findNode(root, "inspector-field-value-entry")
		errorLabel, hasError := findNode(root, "inspector-field-value-entry/field/error")
		message, messageOK := findNode(root, "messages")
		if s.requestHelp == nil || !ok || !grouped || group.Kind != ui.FormGroup || !hasError || !messageOK {
			t.Fatal("missing field, shared notice or field validation")
		}
		if field.Kind != ui.TextField || field.Change == nil || !strings.Contains(field.Accessibility.Help, s.requestHelp.Summary) ||
			errorLabel.Text != "Error: "+s.requestHelp.Summary || message.Text != s.requestHelp.Summary {
			t.Fatal("field validation does not preserve the shared response")
		}
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("presenting refused value guidance changed accepted state")
	}
}

func TestWidgetSystemAppearancePreservesExplicitThemesAndModel(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, closeScreen := testScreen(t)
			t.Cleanup(closeScreen)
			uiAction(t, s, "example.reset", nil)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			before := s.service.Snapshot()
			value := cogent.SystemAppearance{Known: true, Dark: true, Accent: color.NRGBA{R: 140, G: 65, B: 220, A: 255}}
			s.app.Do(func() {
				s.observeSystemAppearance(value)
				root := s.view()
				p := s.colorsForTheme(mode)
				if root.Theme == nil || !root.Theme.Palette.Set || root.Theme.Palette.Value.Accent != p.Accent {
					t.Fatal("scoped controls do not share the session accent")
				}
				if mode == "system" {
					if p.Paper != resolvePalette("dark", false).Paper || p.Accent != value.Accent {
						t.Fatal("System did not follow observed dark/accent preferences")
					}
				} else if p != resolvePalette(mode, false) {
					t.Fatal("system observation replaced an explicit theme")
				}
				if err := ui.Validate(root); err != nil {
					t.Fatal(err)
				}
				value.HighContrast, value.ReduceMotion = true, true
				s.observeSystemAppearance(value)
				root = s.view()
				p = s.colorsForTheme(mode)
				if !root.Theme.ReducedMotion || p.Muted != p.Ink || p.Outline != p.Ink {
					t.Fatal("accessibility preferences were lost in an explicit theme")
				}
			})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("appearance observation changed document, settings or simulation")
			}
		})
	}
}

func TestWidgetCanvasAppearanceRevisionInvalidatesWithoutModelEdits(t *testing.T) {
	var revision canvasAppearanceRevision
	light, dark := resolvePalette("light", false), resolvePalette("dark", false)
	a := revision.next(7, light)
	if revision.next(7, light) != a || revision.next(7, dark) == a {
		t.Fatal("appearance-only canvas update lost or identical appearance repainted")
	}
	b := revision.next(7, dark)
	if revision.next(8, dark) == b {
		t.Fatal("model repaint was lost")
	}
}

func TestNativeExamplesMenuRetainsCommandFreshness(t *testing.T) {
	s, closeScreen := testScreen(t)
	t.Cleanup(closeScreen)
	uiAction(t, s, "example.reset", nil)
	var old ui.Node
	s.app.Do(func() { old, _ = findNode(s.view(), "examples-menu") })
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	before := s.service.Snapshot()
	s.app.Do(func() {
		s.view()
		old.Change(ui.ChangeEvent{SelectedID: "browse"})
	})
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old examples menu choice borrowed a refreshed command context")
	}
}
