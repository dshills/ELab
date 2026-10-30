//go:build offscreen

package main

import (
	"io/fs"
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/text/fonts"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

func mountedUsabilityInspector(t *testing.T, s *screen, height float32) *mountedControlWindow {
	t.Helper()
	view := func() ui.Node {
		root := s.view()
		rail, ok := findNode(root, "inspector-rail")
		if !ok {
			t.Fatal("missing production inspector")
		}
		root.Kind = ui.Column // Reduced fixtures stack controls independently of the full-height native shell.
		root.Children = []ui.Node{rail}
		return root // Preserve the production theme, including Apollo's real mono font.
	}
	return newMountedControlWindowSize(t, s, view, 320, height, inspectorLayoutTestTheme())
}
func TestMountedInspectorPrimaryActionsFitShortRail(t *testing.T) {
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		for _, part := range []string{"R1", "C1"} {
			t.Run(theme+"/"+part, func(t *testing.T) {
				s, cleanup := testScreen(t)
				defer cleanup()
				prepareSpectrumDock(t, s.service)
				if part == "R1" {
					agentAction(t, s.service, "thermal.configure", workbench.ThermalParams{ID: part, Enabled: true, Parameters: workbench.DefaultThermalConfig("resistor")})
				} else {
					agentAction(t, s.service, "passive.configure", workbench.PassiveParams{ID: part, Enabled: true, Parameters: workbench.PassiveConfig{Version: workbench.PassiveLossModelID, ESR: 1, Leakage: 1e6}})
				}
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: part})
				agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
				before := s.service.Snapshot()
				m := mountedUsabilityInspector(t, s, 480)
				scroll := mountedControlElement(t, m, inspectorScrollNode(t, s.view()).ID).Bounds
				for _, id := range []string{"value-entry", "apply-value", "plot-v:" + part, "plot-i:" + part} {
					e := mountedControlElement(t, m, id)
					if !e.Visible || e.Bounds.Empty() || !e.Bounds.In(scroll) {
						t.Errorf("primary control %s needs scrolling: %v / %v", id, e.Bounds, scroll)
					}
				}
				assertCursorMeasurementPreservesPhysics(t, before, s.service.Snapshot())
			})
		}
	}
}
func TestMountedInspectorCheckboxCaptionsFitMonoFont(t *testing.T) {
	face := inspectorCaptionFont(t)
	for _, item := range []struct{ example, part, kind string }{{"discrete-class-ab", "Q6", ""}, {"opamp-follower", "U1", ""}, {"rc", "C1", ""}, {"rc", "R1", ""}, {"hbridge-motor", "MTR1", ""}, {"", "PART", "transformer"}, {"", "PART", "electronic-load"}, {"", "PART", "current-source"}} {
		t.Run(item.example+"/"+item.part+item.kind, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			if item.example != "" {
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: item.example})
			} else {
				agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: item.part, Kind: item.kind})
			}
			switch item.part {
			case "Q6":
				agentAction(t, s.service, "transistor.profile.set", workbench.TransistorProfileParams{ID: item.part, Enabled: true, Parameters: workbench.DefaultTransistorProfile()})
			case "C1":
				agentAction(t, s.service, "passive.configure", workbench.PassiveParams{ID: item.part, Enabled: true, Parameters: workbench.PassiveConfig{Version: workbench.PassiveLossModelID, ESR: 1, Leakage: 1e6}})
			case "R1":
				agentAction(t, s.service, "thermal.configure", workbench.ThermalParams{ID: item.part, Enabled: true, Parameters: workbench.DefaultThermalConfig("resistor")})
			}
			agentAction(t, s.service, "simulation.pause", struct{}{})
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: item.part})
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": "apollo"})
			assertMountedInspectorCaptions(t, s, face, func() ui.Node {
				object, ok := findNode(s.view(), "inspector-group-object")
				if !ok {
					t.Fatal("missing object inspector")
				}
				return object
			})
		})
	}
}

// Drive the production field/button through the retained driver, rather than
// invoking a callback with an already committed value.
func TestMountedInspectorPlotSettlesOnlyOwnValueDraft(t *testing.T) {
	for _, probe := range []string{"v:R1", "i:R1"} {
		t.Run(probe, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareSpectrumDock(t, s.service)
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
			reference := workbench.New()
			t.Cleanup(reference.Close)
			prepareSpectrumDock(t, reference)
			agentAction(t, reference, "component.select", workbench.IDParams{ID: "R1"})
			agentAction(t, reference, "scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1"}})
			// Make the plot change observable (the prepared fixture has i:R1 already).
			agentAction(t, s.service, "scope.probes.set", workbench.ProbeParams{IDs: []string{"v:C1"}})
			m := mountedValuePreset(t, s, "plot-"+probe)
			before := s.service.Snapshot()
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, r := range "unfinished value" {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("private typing changed service")
			}
			clickMountedControl(t, m, "plot-"+probe)
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || after.Drawing.Value != "unfinished value" {
				t.Fatalf("one plot click must settle only its draft: %d→%d %s", before.CommandRevision, after.CommandRevision, s.errorMessage)
			}
			if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision {
				t.Fatal("plot applied draft or changed physics")
			}
			agentAction(t, reference, "editor.value.set", workbench.DraftParams{Value: "unfinished value"})
			agentAction(t, reference, "scope.probe.plot", workbench.IDParams{ID: probe})
			want := reference.Snapshot()
			if !reflect.DeepEqual(after.Scope.Selected, want.Scope.Selected) || !reflect.DeepEqual(after.TraceFocus, want.TraceFocus) || !reflect.DeepEqual(after.Components, want.Components) || after.Drawing.Value != want.Drawing.Value || after.ScopeVisible != want.ScopeVisible {
				t.Fatal("field/plot workflow differs from the shared agent commands")
			}
			found := false
			for _, p := range after.Scope.Selected {
				found = found || p == probe
			}
			if !found {
				t.Fatal("trace not plotted")
			}
		})
	}
}

func TestMountedInspectorPlotRefusesHeldAgentAndDocumentEdits(t *testing.T) {
	for _, scenario := range []string{"agent", "document"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareSpectrumDock(t, s.service)
			agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
			m := mountedValuePreset(t, s, "plot-v:R1")
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeA, Modifiers: 1 << key.Meta})
			for _, r := range "9k" {
				m.input(cogent.ManualInput{Type: events.KeyChord, Rune: r})
			}
			b := mountedControlElement(t, m, "plot-v:R1")
			at := b.Bounds.Min.Add(b.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			if scenario == "agent" {
				agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "4k"})
			} else {
				agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "rc", DiscardChanges: true})
				agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
			}
			before := s.service.Snapshot()
			m.frame()
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("held plot adopted intervening work")
			}
			// A fresh click on the current control remains useful.
			clickMountedControl(t, m, "plot-v:R1")
			if after := s.service.Snapshot(); s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("fresh plot failed", s.errorMessage)
			}
		})
	}
}

func inspectorCaptionFont(t *testing.T) font.Face {
	t.Helper()
	data, err := fs.ReadFile(fonts.Default, "robotomono/RobotoMono-Regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := opentype.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 12, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = face.Close() })
	return face
}

func assertMountedInspectorCaptions(t *testing.T, s *screen, face font.Face, source func() ui.Node) {
	t.Helper()
	var collect func(ui.Node) []ui.Node
	collect = func(n ui.Node) []ui.Node {
		var boxes []ui.Node
		if n.Kind == ui.Checkbox {
			boxes = append(boxes, n)
		}
		for _, child := range n.Children {
			boxes = append(boxes, collect(child)...)
		}
		return boxes
	}
	view := func() ui.Node {
		root := s.view()
		boxes := collect(source())
		if len(boxes) == 0 {
			t.Fatal("fixture has no checkboxes")
		}
		root.Kind = ui.Column // Reduced fixtures stack controls independently of the full-height native shell.
		root.Children = []ui.Node{panel("caption-test-rail", 280, boxes...)}
		return root
	}
	m := newMountedControlWindowSize(t, s, view, 320, 760, inspectorLayoutTestTheme())
	for _, n := range collect(source()) {
		box := mountedControlElement(t, m, n.ID).Bounds
		if box.Empty() {
			t.Fatal("unlaid checkbox", n.ID)
		}
		// The shipped native Switch caption is single-line. Its glyph and gap
		// consume 20 points; measure both names with the actual Apollo font.
		available := box.Dx() - 20
		for _, caption := range []string{n.Text, n.Accessibility.Name} {
			if width := font.MeasureString(face, caption).Ceil(); width > available {
				t.Errorf("%s caption clips: %q needs %dpx; %dpx available in %v", n.ID, caption, width, available, box)
			}
		}
	}
}

func TestMountedRemainingInspectorCaptionsFitMonoFont(t *testing.T) {
	face := inspectorCaptionFont(t)
	for _, page := range []string{"output", "comparison", "sweep"} {
		t.Run(page, func(t *testing.T) {
			var s *screen
			if page == "comparison" {
				s, _ = capturedComparisonScreen(t)
			} else {
				var cleanup func()
				s, cleanup = testScreen(t)
				t.Cleanup(cleanup)
				agentAction(t, s.service, "example.reset", workbench.ReplaceParams{})
				agentAction(t, s.service, "simulation.pause", struct{}{})
			}
			section := "simulation"
			expected := []string{"sweep-keep"}
			if page == "output" {
				section = "document"
				expected = []string{"pdf-tile", "pdf-color", "pdf-measurements"}
			}
			if page == "comparison" {
				expected = []string{"recorded-compare-overlay", "recorded-compare-statistics"}
			}
			if page == "sweep" {
				uiAction(t, s, "sweep-points", &ui.ChangeEvent{Text: "11"})
				uiAction(t, s, "sweep-start", nil)
				awaitNativeSweep(t, s)
			}
			agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: section})
			if page == "comparison" {
				agentAction(t, s.service, "workspace.simulation.set", workbench.SimulationToolParams{Tool: "recordings"})
			}
			agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": "apollo"})
			for _, id := range expected {
				if _, ok := findNode(s.view(), id); !ok {
					t.Fatal("missing fixture control", id)
				}
			}
			assertMountedInspectorCaptions(t, s, face, s.view)
		})
	}
}
