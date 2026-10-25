//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func widgetTestTheme() ui.Theme {
	theme := desktopTheme()
	theme.Default.Background = ui.Some(paper)
	return theme
}

func TestMountedWidgetSearchClearFitsAndUsesSharedCommand(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			agentAction(t, s.service, "parts.search.set", workbench.SearchParams{Query: "MOSFET"})
			before := s.service.Snapshot()
			view := func() ui.Node {
				root := s.view()
				palette, _ := findNode(root, "palette")
				root.Children = []ui.Node{palette}
				return root
			}
			m := newMountedControlWindowSize(t, s, view, 224, 760, widgetTestTheme())
			box := mountedControlElement(t, m, "part-search-box")
			for _, id := range []string{"part-search", "part-search-clear"} {
				control := mountedControlElement(t, m, id)
				if control.Bounds.Empty() || !control.Bounds.In(box.Bounds) || !control.Visible {
					t.Fatal("search affordance is clipped", id, control.Bounds, box.Bounds)
				}
			}
			clickMountedControl(t, m, "part-search-clear")
			state := s.service.Snapshot()
			if state.PartSearch != "" || m.snapshot().FocusedID != "part-search" || state.Document != before.Document || state.ElectricalRevision != before.ElectricalRevision {
				t.Fatal("Clear lost entry focus or changed the circuit")
			}
		})
	}
}

func TestMountedWidgetOverflowMenuRoutesFitAndRetainsActions(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		t.Run(mode, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
			agentAction(t, s.service, "view.pan", map[string]float64{"dx": 20, "dy": 0})
			width := float32(80)
			view := func() ui.Node {
				root := s.view()
				bar, _ := findNode(root, "drawing-secondary-actions")
				bar.Style.MinSize = ui.Some(ui.Size{Width: width, Height: 30})
				bar.Style.MaxSize = bar.Style.MinSize
				bar.Style.GrowX = ui.Some(float32(0))
				root.Children = []ui.Node{bar}
				root.Focus = ui.FocusRequest{}
				return root
			}
			m := newMountedControlWindowSize(t, s, view, 320, 180, widgetTestTheme())
			moreID := ui.ToolbarOverflowID("drawing-secondary-actions")
			more := mountedControlElement(t, m, moreID)
			if !more.Visible || !more.Bounds.In(mountedControlElement(t, m, "drawing-secondary-actions").Bounds) {
				t.Fatal("More is clipped in the constrained toolbar", more.Bounds)
			}
			clickMountedControl(t, m, moreID)
			originalMore := mountedControlElement(t, m, moreID)
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
			m.frame()
			if mountedControlElement(t, m, moreID).Incarnation == originalMore.Incarnation {
				t.Fatal("an open More popup survived an intervening agent command")
			}
			for _, element := range m.snapshot().Elements {
				if element.Name == "Fit" && element.Visible && element.ID != "view.fit" {
					t.Fatal("retired overflow menu still exposes its old action")
				}
			}
			before := s.service.Snapshot()
			clickMountedControl(t, m, moreID)
			var fit cogent.ManualElement
			for _, element := range m.snapshot().Elements {
				if element.Name == "Fit" && element.Visible && element.ID != "view.fit" {
					fit = element
				}
			}
			if fit.ID == "" {
				t.Fatal("overflow menu lost Fit", m.snapshot().Elements)
			}
			clickMountedControl(t, m, fit.ID)
			after := s.service.Snapshot()
			if !after.Viewport.Following || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Components, before.Components) {
				t.Fatal("overflow Fit bypassed the shared viewport command")
			}
			width = 144
			m.frame()
			for _, id := range []string{"sweep-open", "view.fit"} {
				if !mountedControlElement(t, m, id).Visible {
					t.Fatal("expanded toolbar did not restore its action", id)
				}
			}
		})
	}
}

func TestMountedWidgetCompactStepperKeyboardAndWheel(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.buck", nil)
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "PWM"})
	view := func() ui.Node {
		root := s.view()
		rail, _ := findNode(root, "inspector-rail")
		root.Children = []ui.Node{rail}
		root.Focus = ui.FocusRequest{ID: "pwm-duty", Sequence: 1}
		return root
	}
	m := newMountedControlWindowSize(t, s, view, 320, 700, widgetTestTheme())
	field, _ := findNode(s.view(), "pwm-duty")
	if field.NumberButtons != ui.NumberButtonsStepper {
		t.Fatal("inspector did not adopt the compact stepper")
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeUpArrow})
	after := s.service.Snapshot()
	for _, part := range after.Components {
		if part.ID == "PWM" && part.Device.PWM.Duty != .46 {
			t.Fatal("stepper changed duty by the wrong amount", part.Device.PWM.Duty)
		}
	}
	control := mountedControlElement(t, m, "pwm-duty")
	m.input(cogent.ManualInput{Type: events.Scroll, Position: control.Bounds.Min.Add(control.Bounds.Size().Div(2)), Delta: math32.Vec2(0, 32)})
	if !reflect.DeepEqual(after, s.service.Snapshot()) {
		t.Fatal("wheel over the compact stepper changed its accepted value")
	}
}
