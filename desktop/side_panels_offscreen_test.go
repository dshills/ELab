//go:build offscreen

package main

import (
	"encoding/base64"
	"fmt"
	"image"
	"math"
	"reflect"
	"testing"
	"time"

	"cogentcore.org/core/core"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

// The manual driver refuses arbitrary canvas extensions. Use the production
// descriptor at the fixture's known width with only canvas paint replaced by a
// plain growing host; every tested panel, divider and control stays real.
func sidePanelPaintView(s *screen, width float32) ui.Node {
	root := s.workspaceView(width - 16)
	var omit func(*ui.Node)
	omit = func(n *ui.Node) {
		if n.Kind == ui.Extension {
			n.Kind, n.Custom = ui.Row, nil
		}
		for i := range n.Children {
			omit(&n.Children[i])
		}
	}
	omit(&root)
	return root
}

func sidePanelGrip(t *testing.T, m *mountedControlWindow, id string) image.Rectangle {
	t.Helper()
	target := "split:" + base64.RawURLEncoding.EncodeToString([]byte(id))
	for _, e := range m.snapshot().Elements {
		if e.ID == target && !e.Bounds.Empty() {
			return e.Bounds
		}
	}
	t.Fatal("missing divider", id)
	return image.Rectangle{}
}

func TestMountedSidePanelsResizeCollapseAndKeepCanvas(t *testing.T) {
	for _, theme := range []string{"light", "dark", "apollo", "system"} {
		for _, width := range []float32{1040, 1440} {
			t.Run(fmt.Sprintf("%s/%g", theme, width), func(t *testing.T) {
				s, cleanup := testScreen(t)
				defer cleanup()
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
				m := newMountedControlWindowSize(t, s, func() ui.Node { return sidePanelPaintView(s, width) }, width, 800, inspectorLayoutTestTheme())
				before := s.service.Snapshot()
				for _, id := range []string{"parts-toggle", "inspector-toggle", "simulation.pause"} {
					e := mountedControlElement(t, m, id)
					if !e.Visible || !e.Bounds.In(image.Rect(0, 0, int(width), 800)) {
						t.Fatal("header control clipped", id, e.Bounds)
					}
				}
				for _, id := range []string{"parts-split", "inspector-split"} {
					grip := sidePanelGrip(t, m, id)
					start := grip.Min.Add(grip.Size().Div(2))
					delta := 30
					if id == "inspector-split" {
						delta = -30
					}
					m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
					m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(delta, 0))})
					m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(delta, 0))})
					after := s.service.Snapshot()
					actual := after.WorkspacePanels.PartsWidth
					if id == "inspector-split" {
						actual = after.WorkspacePanels.InspectorWidth
					}
					original := before.WorkspacePanels.PartsWidth
					if id == "inspector-split" {
						original = before.WorkspacePanels.InspectorWidth
					}
					panelID := "parts-scroll"
					if id == "inspector-split" {
						panelID = "inspector-rail"
					}
					bounds := mountedControlElement(t, m, panelID).Bounds
					if math.Abs(float64(bounds.Dx())-actual) > 2 {
						t.Fatalf("visible panel width differs from shared value: %s %v / %g", panelID, bounds, actual)
					}
					if math.Abs(actual-original-30) > 3 {
						t.Fatalf("drag did not resize panel by requested amount: %s %g -> %g (%s)", id, original, actual, s.errorMessage)
					}
				}
				stored := s.service.Snapshot().WorkspacePanels
				for _, id := range []string{"parts-toggle", "inspector-toggle"} {
					clickMountedControl(t, m, id)
				}
				if _, ok := findNode(s.view(), "parts-scroll"); ok {
					t.Fatal("collapsed library still mounted")
				}
				for _, id := range []string{"parts-toggle", "inspector-toggle"} {
					clickMountedControl(t, m, id)
				}
				if s.service.Snapshot().WorkspacePanels != stored {
					t.Fatal("collapse lost width")
				}
				sheet := mountedControlElement(t, m, "schematic").Bounds
				if sheet.Dx() < 320 || sheet.Empty() {
					t.Fatal("side panels hid drawing", sheet)
				}
				assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
			})
		}
	}
}

func TestMountedSidePanelQueuedResizeRefusesAgentRevision(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	m := newMountedControlWindowSize(t, s, func() ui.Node { return sidePanelPaintView(s, 1440) }, 1440, 800, inspectorLayoutTestTheme())
	grip := sidePanelGrip(t, m, "parts-split")
	at := grip.Min.Add(grip.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if m.snapshot().FocusedID != "parts-split" {
		t.Fatal("divider not keyboard focusable")
	}
	if err := m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.KeyChord, Code: key.CodeRightArrow}); err != nil {
		t.Fatal(err)
	}
	step, err := m.manual.Step(m.ctx, 1)
	if err != nil || step.Events != 1 || step.OwnerCallbacks != 0 || s.app.Pending().Ordinary == 0 {
		t.Fatal("origin queue not isolated", step, err)
	}
	p := s.service.Snapshot().WorkspacePanels
	p.InspectorWidth = 400
	agentAction(t, s.service, "workspace.panels.set", p)
	accepted := s.service.Snapshot()
	if err = s.app.Invalidate("main"); err != nil {
		t.Fatal(err)
	}
	if err = m.manual.Input(m.ctx, cogent.ManualInput{Window: "main", WindowIncarnation: m.window.Incarnation, Type: events.MouseMove, Position: image.Pt(10, 10)}); err != nil {
		t.Fatal(err)
	}
	m.settle()
	after := s.service.Snapshot()
	// Native viewport publications may follow layout; the user's old resize must
	// not change the newer panel state, circuit or its analysis preferences.
	if after.WorkspacePanels != accepted.WorkspacePanels || !reflect.DeepEqual(after.Components, accepted.Components) {
		t.Fatal("old divider borrowed newer agent context")
	}
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeRightArrow})
	if s.service.Snapshot().WorkspacePanels.PartsWidth <= p.PartsWidth {
		t.Fatal("fresh divider input failed", s.errorMessage)
	}
}

func TestMountedSimulationToolChooserKeepsNavigationVisible(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "inspector-section-simulation", nil)
	s.keyboardFocus = ui.FocusRequest{}
	m := mountedUsabilityInspector(t, s, 480)
	for _, option := range simulationTools() {
		agentAction(t, s.service, "workspace.simulation.set", workbench.SimulationToolParams{Tool: option.ID})
		m.frame()
		choice := mountedControlElement(t, m, "simulation-tool")
		if !choice.Visible || !choice.Bounds.In(image.Rect(0, 0, 320, 480)) {
			t.Fatal("tool navigation scrolls offscreen", choice.Bounds)
		}
		if _, ok := findNode(s.view(), inspectorScrollID(s.service.Snapshot())); !ok {
			t.Fatal("tool lost its scroll host")
		}
	}
}

func TestMountedMaximumPanelPreferencesKeepToolsAndBothGripsUsable(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	p := workbench.WorkspacePanels{PartsVisible: true, InspectorVisible: true, PartsWidth: 360, InspectorWidth: 560}
	agentAction(t, s.service, "workspace.panels.set", p)
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "wire"})
	m := newMountedControlWindowSize(t, s, func() ui.Node { return sidePanelPaintView(s, 1040) }, 1040, 800, inspectorLayoutTestTheme())
	if s.service.Snapshot().WorkspacePanels != p {
		t.Fatal("native allocation overwrote preferred widths")
	}
	sheet := mountedControlElement(t, m, "schematic-panel").Bounds
	for _, id := range []string{"tool-select", "tool-wire", "tool-pan", "drawing.cancel"} {
		control := mountedControlElement(t, m, id)
		if !control.Visible || !control.Bounds.In(sheet) {
			t.Fatal("maximum side panels clip drawing action", id, control.Bounds, sheet)
		}
	}
	header := mountedControlElement(t, m, "header").Bounds
	for _, id := range []string{"sweep-open", "view.fit"} {
		e := mountedControlElement(t, m, id)
		if !e.Visible || !e.Bounds.In(header) {
			t.Fatal("native toolbar action clipped", id, e.Bounds, header)
		}
	}
	// The inspector is already at its minimum displayed width. Its grip must
	// still grow it by taking available space from the opposite panel.
	grip := sidePanelGrip(t, m, "inspector-split")
	start := grip.Min.Add(grip.Size().Div(2))
	original := mountedControlElement(t, m, "inspector-rail").Bounds.Dx()
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
	m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Sub(image.Pt(24, 0))})
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Sub(image.Pt(24, 0))})
	width := mountedControlElement(t, m, "inspector-rail").Bounds.Dx()
	if math.Abs(float64(width-original-24)) > 2 {
		t.Fatal("constrained inspector could not grow", original, width, s.errorMessage)
	}
	sheet = mountedControlElement(t, m, "schematic-panel").Bounds
	for _, id := range []string{"tool-select", "tool-wire", "tool-pan", "drawing.cancel"} {
		control := mountedControlElement(t, m, id)
		if !control.Visible || !control.Bounds.In(sheet) {
			t.Fatal("resize clipped drawing action", id, control.Bounds, sheet)
		}
	}
}

func TestMountedDrawerResizeContinuesWithinOnePointerGesture(t *testing.T) {
	for _, id := range []string{"parts-split", "inspector-split"} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			m := newMountedControlWindowSize(t, s, func() ui.Node { return sidePanelPaintView(s, 1440) }, 1440, 800, inspectorLayoutTestTheme())
			grip := sidePanelGrip(t, m, id)
			start := grip.Min.Add(grip.Size().Div(2))
			direction := 1
			if id == "inspector-split" {
				direction = -1
			}
			before := s.service.Snapshot().WorkspacePanels
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
			if _, err := m.clock.Advance(core.TimingSettings.SlideStartTime + time.Millisecond); err != nil {
				t.Fatal(err)
			}
			m.settle()
			for _, delta := range []int{16, 48, 72} {
				m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(direction*delta, 0))})
				current := s.service.Snapshot().WorkspacePanels
				change := current.PartsWidth - before.PartsWidth
				if id == "inspector-split" {
					change = current.InspectorWidth - before.InspectorWidth
				}
				if math.Abs(change-float64(delta)) > 3 {
					t.Fatalf("ongoing move %d stopped at %g: %s", delta, change, s.errorMessage)
				}
			}
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(direction*72, 0))})
			after := s.service.Snapshot().WorkspacePanels
			change := after.PartsWidth - before.PartsWidth
			if id == "inspector-split" {
				change = after.InspectorWidth - before.InspectorWidth
			}
			if math.Abs(change-72) > 3 {
				t.Fatalf("drawer stopped before the pointer finished: change=%g want=72; %s", change, s.errorMessage)
			}
		})
	}
}

func TestMountedDrawersCloseAtEdgesAndReopenByLabel(t *testing.T) {
	for _, theme := range []string{"light", "dark", "apollo", "system"} {
		t.Run(theme, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
			m := newMountedControlWindowSize(t, s, func() ui.Node { return sidePanelPaintView(s, 1040) }, 1040, 800, inspectorLayoutTestTheme())
			before := s.service.Snapshot()
			left := mountedControlElement(t, m, "parts-toggle")
			right := mountedControlElement(t, m, "inspector-toggle")
			leftGap := sidePanelGrip(t, m, "parts-split").Min.X - left.Bounds.Max.X
			rightGap := sidePanelGrip(t, m, "inspector-split").Min.X - right.Bounds.Max.X
			if !left.Visible || !right.Visible || leftGap < 0 || leftGap > 24 || rightGap < 0 || rightGap > 24 {
				t.Fatalf("close controls are not beside their dividers: parts gap %d, inspector gap %d", leftGap, rightGap)
			}
			for _, id := range []string{"parts-toggle", "inspector-toggle"} {
				clickMountedControl(t, m, id)
			}
			for _, id := range []string{"parts-scroll", "inspector-rail", "parts-split", "inspector-split"} {
				if _, ok := findNode(s.view(), id); ok {
					t.Fatal("closed drawer remains mounted", id)
				}
			}
			for _, id := range []string{"parts-toggle", "inspector-toggle"} {
				n := mountedControlElement(t, m, id)
				if !n.Visible || n.Bounds.Empty() {
					t.Fatal("no visible edge tab", id)
				}
				clickMountedControl(t, m, id)
			}
			after := s.service.Snapshot()
			if after.WorkspacePanels != before.WorkspacePanels {
				t.Fatal("close/reopen lost preferred width")
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestMountedDrawersDragAndKeyboardCloseWithoutLosingWidth(t *testing.T) {
	for _, inspector := range []bool{false, true} {
		for _, keyboard := range []bool{false, true} {
			t.Run(fmt.Sprintf("inspector=%v/keyboard=%v", inspector, keyboard), func(t *testing.T) {
				s, cleanup := testScreen(t)
				defer cleanup()
				m := newMountedControlWindowSize(t, s, func() ui.Node { return sidePanelPaintView(s, 1440) }, 1440, 800, inspectorLayoutTestTheme())
				before := s.service.Snapshot()
				id, toggle, panel := "parts-split", "parts-toggle", "parts-scroll"
				delta := -200
				code := key.CodeHome
				if inspector {
					id, toggle, panel = "inspector-split", "inspector-toggle", "inspector-rail"
					delta = 240
					code = key.CodeEnd
				}
				b := sidePanelGrip(t, m, id)
				start := b.Min.Add(b.Size().Div(2))
				m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
				if keyboard {
					m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start})
					m.input(cogent.ManualInput{Type: events.KeyChord, Code: code})
				} else {
					if _, err := m.clock.Advance(core.TimingSettings.SlideStartTime + time.Millisecond); err != nil {
						t.Fatal(err)
					}
					m.settle()
					m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(delta/4, 0))})
					m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(delta, 0))})
					m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(delta, 0))})
				}
				after := s.service.Snapshot()
				visible := after.WorkspacePanels.PartsVisible
				if inspector {
					visible = after.WorkspacePanels.InspectorVisible
				}
				if visible {
					t.Fatal("edge gesture did not close the drawer", s.errorMessage)
				}
				if _, ok := findNode(s.view(), panel); ok {
					t.Fatal("closed form still mounted")
				}
				if m.snapshot().FocusedID != toggle {
					t.Fatal("closing lost focus instead of returning to edge tab", m.snapshot().FocusedID)
				}
				if after.WorkspacePanels.PartsWidth != before.WorkspacePanels.PartsWidth || after.WorkspacePanels.InspectorWidth != before.WorkspacePanels.InspectorWidth {
					t.Fatal("closing discarded preferred width")
				}
				clickMountedControl(t, m, toggle)
				if s.service.Snapshot().WorkspacePanels != before.WorkspacePanels {
					t.Fatal("edge tab did not reopen drawer")
				}
				assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
			})
		}
	}
}

func TestMountedDrawerClosingDragRefusesInterveningAgent(t *testing.T) {
	for _, inspector := range []bool{false, true} {
		t.Run(fmt.Sprint(inspector), func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			m := newMountedControlWindowSize(t, s, func() ui.Node { return sidePanelPaintView(s, 1440) }, 1440, 800, inspectorLayoutTestTheme())
			id := "parts-split"
			delta := -200
			if inspector {
				id = "inspector-split"
				delta = 240
			}
			b := sidePanelGrip(t, m, id)
			start := b.Min.Add(b.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
			if _, err := m.clock.Advance(core.TimingSettings.SlideStartTime + time.Millisecond); err != nil {
				t.Fatal(err)
			}
			m.settle()
			p := s.service.Snapshot().WorkspacePanels
			if inspector {
				p.PartsWidth = 300
			} else {
				p.InspectorWidth = 400
			}
			agentAction(t, s.service, "workspace.panels.set", p)
			m.frame()
			m.input(cogent.ManualInput{Type: events.MouseMove, Position: start.Add(image.Pt(delta, 0))})
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(delta, 0))})
			if s.service.Snapshot().WorkspacePanels != p {
				t.Fatal("pre-agent gesture closed a newer drawer state")
			}
			b = sidePanelGrip(t, m, id)
			start = b.Min.Add(b.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: start})
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: start.Add(image.Pt(delta, 0))})
			current := s.service.Snapshot().WorkspacePanels
			if inspector && current.InspectorVisible || !inspector && current.PartsVisible {
				t.Fatal("fresh closing gesture failed", s.errorMessage)
			}
		})
	}
}
