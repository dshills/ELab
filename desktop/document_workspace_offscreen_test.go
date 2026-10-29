//go:build offscreen

package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func TestMountedTaskGuidanceStaysVisibleWithoutOpeningDetails(t *testing.T) {
	for _, section := range []string{"component", "simulation", "document"} {
		for _, themeName := range []string{"light", "dark", "apollo", "system"} {
			t.Run(section+"/"+themeName, func(t *testing.T) {
				s, cleanup := testScreen(t)
				t.Cleanup(cleanup)
				uiAction(t, s, "example.rc", nil)
				uiAction(t, s, "simulation.pause", nil)
				uiAction(t, s, "simulation.restart", nil)
				agentAction(t, s.service, "simulation.step", workbench.StepParams{Duration: .02})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": themeName})
				uiAction(t, s, "inspector-section-"+section, nil)
				s.app.Do(func() {
					field, _ := findNode(s.view(), "cursor-time-a")
					field.Change(ui.ChangeEvent{Text: "oops"})
				})
				before := s.service.Snapshot()
				view := func() ui.Node {
					root := s.view()
					details, ok := findNode(root, "diagnostic-details")
					if !ok || details.Checked {
						t.Fatal("guidance requires opening full details")
					}
					notice, ok := findNode(root, "operation-notice-area")
					if !ok {
						t.Fatal("missing visible correction")
					}
					action, ok := findNode(notice, "message-action")
					if !ok || s.requestHelp == nil || action.Text != s.requestHelp.Action {
						t.Fatal("visible next step differs from shared request help")
					}
					root.Children = []ui.Node{notice}
					root.Focus = ui.FocusRequest{} // This focused fixture excludes the owner toolbar.
					return root
				}
				theme := inspectorLayoutTestTheme()
				// Preserve the production palette on unstyled child groups; the
				// width-test helper's white default would falsify dark rendering.
				theme.Default.Background = ui.Some(color.NRGBA{})
				m := newMountedControlWindowSize(t, s, view, 320, 360, theme)
				rail := mountedControlElement(t, m, "root")
				action := mountedControlElement(t, m, "message-action")
				if !action.Visible || action.Bounds.Dy() < 28 || !action.Bounds.In(rail.Bounds) {
					t.Fatalf("correction instructions are clipped: action=%v rail=%v", action.Bounds, rail.Bounds)
				}
				assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
				if artifacts := os.Getenv("ELAB_GUIDANCE_LAYOUT_ARTIFACTS"); artifacts != "" {
					im, err := m.manual.Capture(m.ctx, "main")
					if err != nil {
						t.Fatal(err)
					}
					file, err := os.OpenFile(filepath.Join(artifacts, fmt.Sprintf("%s-%s.png", section, themeName)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(file, im)
					closeErr := file.Close()
					if err != nil || closeErr != nil {
						t.Fatal(err, closeErr)
					}
				}
			})
		}
	}
}

func TestMountedInspectorNavigationStaysVisibleAndPreservesCircuit(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	view := func() ui.Node {
		rail, ok := findNode(s.view(), "inspector-rail")
		if !ok {
			t.Fatal("missing inspector")
		}
		return ui.Node{ID: "root", Kind: ui.Column, Grow: true, Children: []ui.Node{rail}}
	}
	m := newMountedControlWindowSize(t, s, view, 320, 540, inspectorLayoutTestTheme())
	clickMountedControl(t, m, "inspector-section-simulation")
	if s.service.Snapshot().InspectorSection != "simulation" {
		t.Fatal("physical navigation did not use the shared command")
	}
	header := mountedControlElement(t, m, "inspector-section-document").Bounds
	step := mountedControlElement(t, m, "setting-step")
	m.input(cogent.ManualInput{Type: events.Scroll, Target: step.ID, TargetIncarnation: step.Incarnation, Delta: math32.Vec2(0, 10000)})
	m.frame()
	if current := mountedControlElement(t, m, "inspector-section-document").Bounds; current.Empty() || current != header {
		t.Fatal("scroll moved inspector navigation", header, current)
	}
	clickMountedControl(t, m, "inspector-section-document")
	if _, ok := ui.FocusTarget(s.view(), "setting-step"); ok {
		t.Fatal("hidden simulation field remained reachable")
	}
	preview := mountedControlElement(t, m, "print-preview")
	if !preview.Bounds.In(mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID).Bounds) {
		t.Fatal("document output inherited another section's scroll")
	}
	after := s.service.Snapshot()
	if after.InspectorSection != "document" || before.Document != after.Document || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("inspector navigation changed the circuit")
	}
}
