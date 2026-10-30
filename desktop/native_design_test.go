package main

import (
	"image"
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"

	"cogentcore.org/core/core"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func TestNativeDesignKeepsSharedControlsAndValidPages(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "led"})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	for _, theme := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
		for _, section := range []string{"component", "simulation", "document"} {
			agentAction(t, s.service, "workspace.inspector.set", workbench.InspectorParams{Section: section})
			root := s.workspaceView(1440)
			if err := ui.Validate(root); err != nil {
				t.Fatal(theme, section, err)
			}
			if root.Kind != ui.Row {
				t.Fatal("native shell must keep full-height rails")
			}
			for _, id := range []string{"native-parts", "native-center", "header", "inspector-rail", "parts-toggle", "inspector-toggle", "examples-menu", "native-share", "tools", "native-zoom", "scope-view-scope", "scope-view-spectrum"} {
				if _, ok := findNode(root, id); !ok {
					t.Fatalf("%s/%s missing %s", theme, section, id)
				}
			}
			if _, ok := findNode(root, "footer-row"); ok {
				t.Fatal("legacy bottom status strip remains")
			}
			if _, ok := findNode(root, "readings"); ok {
				t.Fatal("legacy reading strip remains")
			}
		}
	}
}
func TestNativePanelWidthsUseSharedPreferences(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	before := s.service.Snapshot()
	for _, p := range []workbench.WorkspacePanels{workbench.DefaultWorkspacePanels(), {PartsVisible: true, InspectorVisible: true, PartsWidth: 300, InspectorWidth: 420}} {
		agentAction(t, s.service, "workspace.panels.set", p)
		root := s.workspaceView(1440)
		left, _ := findNode(root, "native-parts")
		right, _ := findNode(root, "inspector-rail")
		if left.Style.MinSize.Value.Width != float32(p.PartsWidth) || right.Style.MinSize.Value.Width != float32(p.InspectorWidth) {
			t.Fatal("rendered rails ignored shared panel widths")
		}
	}
	after := s.service.Snapshot()
	if before.Document != after.Document || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID {
		t.Fatal("panel layout changed circuit")
	}
}
func TestNativeZoomStepsKeepTheDocument(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "view.zoom", map[string]float64{"factor": 1.05})
	before := s.service.Snapshot()
	uiAction(t, s, "native-zoom-in", nil)
	if math.Abs(s.service.Snapshot().View.Zoom-1.1) > 1e-9 {
		t.Fatal("zoom did not choose next stop")
	}
	uiAction(t, s, "native-zoom-out", nil)
	after := s.service.Snapshot()
	if math.Abs(after.View.Zoom-1) > 1e-9 || before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) {
		t.Fatal("zoom changed document or failed to step down")
	}
}
func TestNativeQuickExampleRetainsDirtyReplacementProtection(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "led"})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	menu := s.nativeExamplesMenu(before)
	menu.Change(ui.ChangeEvent{SelectedID: "example.rc"})
	after := s.service.Snapshot()
	if s.pendingCommand == "" || before.Document != after.Document || !reflect.DeepEqual(before.Components, after.Components) {
		t.Fatal("quick example replaced unsaved work without the original prompt")
	}
}
func TestNativeProbePillsAreAcceptedCollisionFreePresentation(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	prepareSpectrumDock(t, s.service)
	state := s.service.Snapshot()
	view := canvas.Viewport{Width: 900, Height: 600, Zoom: 1}
	before := s.service.Snapshot()
	pills := nativeProbePills(state, view, resolvePalette("light", false))
	if len(pills) != len(state.Scope.Traces) {
		t.Fatal("accepted traces were not mirrored")
	}
	for i, p := range pills {
		if p.Width <= 0 || p.Height != 22 || p.Text == "" {
			t.Fatal("invalid pill geometry", p)
		}
		for _, q := range pills[:i] {
			if p.Left < q.Left+q.Width && p.Left+p.Width > q.Left && p.Top < q.Top+q.Height && p.Top+p.Height > q.Top {
				t.Fatal("pills overlap")
			}
		}
	}
	state.Scope.Stale = true
	if len(nativeProbePills(state, view, resolvePalette("light", false))) != 0 {
		t.Fatal("stale data presented as live")
	}
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("drawing pills mutated service")
	}
}
func TestNativeExamplesRetainCatalogAndFileActions(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	menu := s.nativeExamplesMenu(s.service.Snapshot())
	for _, id := range []string{"example.rc", "example.led", "example.speaker-amplifier", "example.rectifier", "example.555-flasher", "example.rlc", "browse", "file.open", "file.save", "file.save-as", "project.clear"} {
		found := false
		for _, o := range menu.Options {
			found = found || o.ID == id
		}
		if !found {
			t.Fatal("missing original file/browser action", id)
		}
	}
	if strings.Contains(menu.Text, "cloud") {
		t.Fatal("native menu advertises unsupported cloud storage")
	}
}

func TestNativeConnectPopoverCancelPreservesTheCircuit(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "wire-by-name", nil)
	if !s.nativeConnectOpen {
		t.Fatal("named wire action did not open its popover")
	}
	before := s.service.Snapshot()
	uiAction(t, s, "native-connect-cancel", nil)
	if s.nativeConnectOpen || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("popover cancellation changed accepted schematic state")
	}
}
func TestNativeQuantitiesKeepPrecisionAndSIUnits(t *testing.T) {
	for _, tc := range []struct {
		value      float64
		unit, want string
	}{
		{-1.1537e-6, "V", "-1.15 µV"}, {.0079, "A", "7.90 mA"}, {1100, "Hz", "1.10 kHz"}, {.27, "s", "270 ms"}, {0, "V", "0 V"},
	} {
		if got := nativeQuantity(tc.value, tc.unit); got != tc.want {
			t.Errorf("%g %s: got %q want %q", tc.value, tc.unit, got, tc.want)
		}
	}
}

func TestNativeSamplingNoticeUsesTheSharedFitCommand(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.load", workbench.ExampleLoadParams{ID: "speaker-amplifier"})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "example.load", workbench.ExampleLoadParams{ID: "speaker-amplifier"})
	agentAction(t, agent, "simulation.pause", struct{}{})
	agentAction(t, agent, "simulation.sampling.fit", struct{}{})
	before := s.service.Snapshot()
	if _, ok := findAction(s.view(), "native-fit-sampling"); !ok {
		t.Fatal("sampling notice has no direct recovery action")
	}
	uiAction(t, s, "native-fit-sampling", nil)
	after := s.service.Snapshot()
	if after.TransientSettings.StepSeconds >= before.TransientSettings.StepSeconds || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || after.TransientSettings != agent.Snapshot().TransientSettings {
		t.Fatal("sampling notice differs from the shared fit command")
	}
	if _, ok := findNode(s.view(), "native-sampling-notice"); ok {
		t.Fatal("resolved sampling notice remains")
	}

}

func TestNativeFloatingControlsFollowTheCurrentLayoutAllocation(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		for _, name := range []string{"tools", "native-zoom", "native-sampling-notice"} {
			st := styles.Style{}
			nativeOverlayStyle(name, false, &st)
			for _, size := range [][2]float32{{1358, 700}, {884, 580}, {484, 360}, {1358, 700}} {
				st.UnitContext.Defaults()
				st.UnitContext.DPI = units.DpPerInch * scale
				st.UnitContext.Paw = size[0] * scale
				st.UnitContext.Pah = size[1] * scale
				st.ToDots()
				x, y, w, h := st.Pos.X.Dots/scale, st.Pos.Y.Dots/scale, st.Min.X.Dots/scale, st.Min.Y.Dots/scale
				if x < 0 || y < 0 || x+w > size[0] || y+h > size[1] {
					t.Fatalf("%s escaped resized sheet %v at density %g: %g,%g %gx%g", name, size, scale, x, y, w, h)
				}
				if name == "tools" && math.Abs(float64(x+(w+10)/2-size[0]/2)) > .1 {
					t.Fatal("tool pill did not recenter", size, x, w)
				}
			}
		}
	}
}

func TestNativeEveryQuickExampleLoadsThroughTheSharedService(t *testing.T) {
	for _, id := range []string{"rc", "led", "speaker-amplifier", "rectifier", "555-flasher", "rlc"} {
		t.Run(id, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			menu := s.nativeExamplesMenu(s.service.Snapshot())
			menu.Change(ui.ChangeEvent{SelectedID: "example." + id})
			state := s.service.Snapshot()
			if state.Example == nil || state.Example.Info.ID != id || state.Document.Dirty {
				t.Fatal("quick example did not load cleanly through example.load", s.errorMessage, state.Example)
			}
		})
	}
}

func TestNativeProbePillTextContrast(t *testing.T) {
	linear := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= .04045 {
			return x / 12.92
		}
		return math.Pow((x+.055)/1.055, 2.4)
	}
	lum := func(c color.NRGBA) float64 { return .2126*linear(c.R) + .7152*linear(c.G) + .0722*linear(c.B) }
	for _, theme := range []string{"light", "dark", "apollo"} {
		p := paletteForTheme(theme)
		for _, name := range []string{"amber", "teal", "coral", "blue", "purple"} {
			fill := traceColor(probeColor(name), p)
			ink := nativePillTextColor(fill)
			a, b := lum(fill), lum(ink)
			if ratio := (max(a, b) + .05) / (min(a, b) + .05); ratio < 4.5 {
				t.Fatal("probe text contrast", theme, name, ratio)
			}
		}
	}
}

func TestNativeOverlayExclusionsCoverBridgeAndCoreCoordinates(t *testing.T) {
	scene := core.NewScene()
	parent := core.NewFrame(scene)
	sheet := core.NewFrame(parent)
	zoom := core.NewFrame(parent)
	zoom.SetName("native-zoom")
	tools := core.NewFrame(parent)
	tools.SetName("tools")
	zoom.Geom.TotalBBox = image.Rect(700, 500, 820, 540)
	tools.Geom.TotalBBox = image.Rect(200, 490, 500, 540)
	scene.SceneGeom.Pos = image.Pt(10, 20)
	rects := nativeSheetExclusions(sheet.AsWidget())
	if rects[0] != zoom.Geom.TotalBBox.Add(scene.SceneGeom.Pos) || rects[1] != tools.Geom.TotalBBox.Add(scene.SceneGeom.Pos) {
		t.Fatal("bridge exclusion coordinates", rects)
	}
	for _, position := range []image.Point{image.Pt(710, 510), image.Pt(300, 510)} {
		if !nativeSheetOverlayHit(sheet.AsWidget(), position) {
			t.Fatal("fallback admitted an overlay point", position)
		}
	}
	if nativeSheetOverlayHit(sheet.AsWidget(), image.Pt(600, 400)) {
		t.Fatal("fallback blocked exposed schematic")
	}
}

func TestNativePopoverChoicesRetainTheirOwnSceneRouting(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "wire-by-name", nil)
	root := s.view()
	for _, id := range []string{"wire-from", "wire-to"} {
		n, ok := findNode(root, id)
		if !ok || n.Kind != ui.Selector || n.Change == nil {
			t.Fatal("popover selector converted to a main-scene-only menu", id, n.Kind)
		}
	}
	uiAction(t, s, "native-connect-cancel", nil)
	uiAction(t, s, "sweep-open", nil)
	root = s.view()
	pop, ok := findNode(root, "native-sweep-popover")
	if !ok {
		t.Fatal("sweep popover missing")
	}
	var check func(ui.Node)
	check = func(n ui.Node) {
		if _, converted := n.Custom.(menuChoices); converted {
			t.Fatal("sweep popover retained main-scene menu proxy", n.ID)
		}
		for _, child := range n.Children {
			check(child)
		}
	}
	check(pop)
}

func TestNativeSweepSelectsOnlyItsOwnInstrumentTab(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	for _, theme := range []string{"light", "dark", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": theme})
		agentAction(t, s.service, "sweep.visibility.set", workbench.SweepVisibilityParams{Visible: true})
		root := s.view()
		p := paletteForTheme(theme)
		for _, id := range []string{"scope-view-scope", "scope-view-spectrum", "scope-view-sweep"} {
			n, ok := findNode(root, id)
			if !ok || (n.Style.Background.Value == p.Paper) != (id == "scope-view-sweep") {
				t.Fatal("incorrect sweep tab selection", theme, id)
			}
		}
		body, ok := findNode(root, "native-sweep-panel")
		if !ok || len(body.Children) == 0 || body.Children[0].ID != "sweep-no-record" {
			t.Fatal("empty sweep next-step explanation is hidden")
		}
	}
}
