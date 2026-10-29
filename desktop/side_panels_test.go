package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestSidePanelsUseSharedStateAndRetainTheirWidths(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	p := before.WorkspacePanels
	p.PartsWidth = 320
	p.InspectorWidth = 480
	agentAction(t, s.service, "workspace.panels.set", p)
	for _, id := range []string{"parts-toggle", "inspector-toggle"} {
		uiAction(t, s, id, nil)
	}
	hidden := s.service.Snapshot()
	if hidden.WorkspacePanels.PartsVisible || hidden.WorkspacePanels.InspectorVisible {
		t.Fatal("toggles did not use service state")
	}
	for _, id := range []string{"parts-scroll", "inspector-rail", "part-search", "selection"} {
		if _, ok := findNode(s.view(), id); ok {
			t.Fatal("collapsed content remains mounted", id)
		}
	}
	for _, id := range []string{"parts-toggle", "inspector-toggle"} {
		if _, ok := ui.FocusTarget(s.view(), id); !ok {
			t.Fatal("panel cannot be reopened by keyboard", id)
		}
		uiAction(t, s, id, nil)
	}
	after := s.service.Snapshot()
	if after.WorkspacePanels != p || after.Document != before.Document || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Transient, before.Transient) {
		t.Fatal("panel toggle lost width or changed circuit")
	}
	if s.keyboardFocus.ID != "inspector-section-component" {
		t.Fatal("opening inspector did not continue focus")
	}
}

func TestSimulationChooserShowsOneCompleteToolAtATime(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "inspector-section-simulation", nil)
	targets := map[string]string{"live": "setting-step", "recordings": "verify-start", "spectrum": "spectrum-window", "sweeps": "sweep-target", "tolerances": "tolerance-run", "circuit": "circuit-capacity"}
	before := s.service.Snapshot()
	for _, option := range simulationTools() {
		uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: option.ID})
		state := s.service.Snapshot()
		root := s.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		choice, _ := findNode(root, "simulation-tool")
		if choice.SelectedID != option.ID || state.SimulationTool != option.ID {
			t.Fatal("chooser not shared")
		}
		for tool, id := range targets {
			_, shown := findNode(root, id)
			if shown != (tool == option.ID) {
				t.Fatalf("%s shown=%v while %s selected", id, shown, option.ID)
			}
		}
		if state.Document != before.Document || !reflect.DeepEqual(state.Transient, before.Transient) || !reflect.DeepEqual(state.Scope, before.Scope) {
			t.Fatal("task selection changed live circuit")
		}
	}
	// Each task gets its own document-scoped viewport, preventing an old long
	// form's offset from making the newly selected short form appear empty.
	state := s.service.Snapshot()
	a := inspectorScrollID(state)
	state.SimulationTool = "live"
	if a == inspectorScrollID(state) {
		t.Fatal("tool scroll viewport reused")
	}
}

func TestSidePanelSizingClampsDisplayWithoutChangingPreferences(t *testing.T) {
	for _, total := range []float32{1024, 1264, 1744} {
		p := workbench.WorkspacePanels{PartsVisible: true, InspectorVisible: true, PartsWidth: 360, InspectorWidth: 560}
		left, right := sidePanelWidths(total, p)
		if left < 224 || right < 296 || left+right > total-minimumDrawingWidth-2*sideDividerWidth {
			t.Fatal("side panels consumed drawing space", total, left, right)
		}
		s, cleanup := testScreen(t)
		agentAction(t, s.service, "workspace.panels.set", p)
		root := s.sidePanels(s.service.Snapshot(), ui.Node{ID: "parts", Kind: ui.Column}, ui.Node{ID: "sheet", Kind: ui.Column}, ui.Node{ID: "details", Kind: ui.Column}, total, left, right)
		if err := ui.Validate(root); err != nil {
			var dump func(ui.Node)
			dump = func(n ui.Node) {
				if n.Kind == ui.SplitPane {
					t.Logf("total=%g id=%s number=%.10g bounds=%.10g..%.10g", total, n.ID, n.Number, n.Numeric.Min, n.Numeric.Max)
				}
				for _, c := range n.Children {
					dump(c)
				}
			}
			dump(root)
			t.Fatal(err)
		}
		if s.service.Snapshot().WorkspacePanels != p {
			t.Fatal("layout pass overwrote preferred widths")
		}
		cleanup()
	}
}

func TestPanelAndToolHeldCallbacksRefuseNewerAgentContext(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "inspector-section-simulation", nil)
	toggle, _ := findAction(s.view(), "parts-toggle")
	chooser, _ := findAction(s.view(), "simulation-tool")
	agentAction(t, s.service, "workspace.simulation.set", workbench.SimulationToolParams{Tool: "spectrum"})
	before := s.service.Snapshot()
	s.app.Do(func() { toggle.Activate(); chooser.Change(ui.ChangeEvent{SelectedID: "tolerances"}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("held callback borrowed agent context")
	}
	uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: "recordings"})
	if s.service.Snapshot().SimulationTool != "recordings" {
		t.Fatal("fresh selection failed")
	}
}

func TestSimulationToolsRetainEveryExistingFormControl(t *testing.T) {
	s, cleanup := recordedCursorRC(t)
	defer cleanup()
	state := s.service.Snapshot()
	forms := [][]ui.Node{s.verificationInspector(state), s.spectrumInspector(state), s.toleranceInspector(state), s.sweepInspector(state), s.transientInspector(state), s.probeInspector(state), s.lcSamplingControls(state), s.npnControls(state)}
	expected := map[string]bool{}
	var visit func(ui.Node, map[string]bool, bool)
	visit = func(n ui.Node, into map[string]bool, controls bool) {
		if n.ID != "" && (!controls || n.Change != nil || n.Activate != nil) {
			into[n.ID] = true
		}
		for _, child := range n.Children {
			visit(child, into, controls)
		}
	}
	for _, form := range forms {
		for _, n := range form {
			visit(n, expected, true)
		}
	}
	uiAction(t, s, "inspector-section-simulation", nil)
	found := map[string]bool{}
	for _, option := range simulationTools() {
		uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: option.ID})
		visit(s.view(), found, false)
	}
	for id := range expected {
		if !found[id] {
			t.Error("existing control has no tool destination", id)
		}
	}
}
