package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func expectedViewportFit(t *testing.T, s *Service, width, height float64) View {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	v, err := s.fitViewLocked(FitParams{Width: width, Height: height})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestExamplesFitRegisteredViewportAndResizeWithoutPhysics(t *testing.T) {
	for _, example := range ExampleCatalog() {
		t.Run(example.ID, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "view.viewport.set", ViewportParams{Width: 520, Height: 138}, "small")
			call(t, s, "example.load", ExampleLoadParams{ID: example.ID}, "load")
			call(t, s, "simulation.pause", struct{}{}, "pause")
			before := settled(t, s)
			assertViewportContainsSchematic(t, before)
			if !before.Viewport.Registered || !before.Viewport.Following || before.Viewport.FitFailure != nil || before.View != expectedViewportFit(t, s, 520, 138) {
				t.Fatal("example did not fit actual viewport", before.Viewport, before.View)
			}
			call(t, s, "view.viewport.set", ViewportParams{Width: 520, Height: 420}, "large")
			after := s.Snapshot()
			assertViewportContainsSchematic(t, after)
			if after.View != expectedViewportFit(t, s, 520, 420) || after.View.Zoom < before.View.Zoom || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Mixed, before.Mixed) || !reflect.DeepEqual(after.NPN, before.NPN) || !reflect.DeepEqual(after.Result, before.Result) || after.Document != before.Document || after.UndoName != before.UndoName || after.RedoName != before.RedoName {
				t.Fatal("viewport resize changed physics/document/history", after.View, before.View)
			}
		})
	}
}

func TestViewportManualNavigationWinsAndFitCanFollowAgain(t *testing.T) {
	for _, navigation := range []struct {
		command string
		params  any
	}{
		{"view.pan", map[string]float64{"dx": 10, "dy": 20}},
		{"view.zoom", map[string]float64{"factor": 1.25}},
		{"view.selection.center", CenterSelectionParams{ID: "R1"}},
		{"selection.navigate", NavigateParams{Direction: "next", Center: true}},
		{"view.fit", FitParams{Width: 600, Height: 400}},
	} {
		t.Run(navigation.command, func(t *testing.T) {
			s := NewExample()
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "view.viewport.set", ViewportParams{Width: 500, Height: 150}, "size")
			call(t, s, "view.fit", FitParams{FollowViewport: true}, "follow")
			if s.Snapshot().View != expectedViewportFit(t, s, 500, 150) {
				t.Fatal("follow without dimensions ignored measured size")
			}
			call(t, s, "component.select", IDParams{ID: "R1"}, "select")
			call(t, s, navigation.command, navigation.params, "navigate")
			before := s.Snapshot()
			if before.Viewport.Following {
				t.Fatal("explicit navigation left following on")
			}
			call(t, s, "view.viewport.set", ViewportParams{Width: 900, Height: 600}, "resize")
			if s.Snapshot().View != before.View {
				t.Fatal("resize overwrote manual view")
			}
			call(t, s, "view.fit", FitParams{FollowViewport: true}, "fit-again")
			if s.Snapshot().View != expectedViewportFit(t, s, 900, 600) || !s.Snapshot().Viewport.Following {
				t.Fatal("Fit did not resume following")
			}
		})
	}
}

func TestViewportLateMeasurementWaitsForSizeAndPreservesNavigation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "nmos"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	if !s.Snapshot().Viewport.Following || s.Snapshot().Viewport.Registered {
		t.Fatal("example must await the first measurement")
	}
	call(t, s, "view.viewport.set", ViewportParams{Width: 520, Height: 138}, "first-size")
	if s.Snapshot().View != expectedViewportFit(t, s, 520, 138) {
		t.Fatal("late measurement did not fit")
	}
	before := s.Snapshot()
	call(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 0}, "pan")
	raw, _ := json.Marshal(ViewportParams{Width: 520, Height: 420})
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "stale-size", Key: "stale-size", Command: "view.viewport.set", Params: raw, ExpectedCommandRevision: &before.CommandRevision}, "test")
	if r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("stale measurement was admitted", r.Error)
	}
	manual := s.Snapshot()
	call(t, s, "view.viewport.set", ViewportParams{Width: 520, Height: 420}, "fresh-size")
	if s.Snapshot().View != manual.View {
		t.Fatal("late fresh report overrode manual navigation")
	}
}

func TestViewportValidationFailureOwnershipAndLargeDrawingRecovery(t *testing.T) {
	s := New()
	defer s.Close()
	for i, raw := range []string{`{}`, `{"width_px":520}`, `{"width_px":null,"height_px":138}`, `{"width_px":63,"height_px":138}`, `{"width_px":520,"height_px":100001}`, `{"width_px":"520","height_px":138}`, `{"width_px":520,"height_px":138,"extra":true}`} {
		before := s.Snapshot()
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: "view.viewport.set", Params: json.RawMessage(raw), ExpectedCommandRevision: &before.CommandRevision}, "test")
		if r.Error == nil || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid viewport mutated", raw, r.Error)
		}
	}
	for i, x := range []float64{-10000, 10000} {
		call(t, s, "component.place", PlaceParams{Kind: "resistor", ID: fmt.Sprintf("R%d", i), Position: Point{X: x}}, fmt.Sprint("place", i))
	}
	call(t, s, "view.fit", FitParams{Width: 1000, Height: 600, FollowViewport: true}, "fit-wide")
	before := s.Snapshot()
	call(t, s, "view.viewport.set", ViewportParams{Width: 64, Height: 64}, "tiny")
	failed := s.Snapshot()
	if failed.View != before.View || !failed.Viewport.Following || failed.Viewport.Width != 64 || failed.Viewport.FitFailure == nil || failed.Viewport.FitFailure.Code != "view-limit" {
		t.Fatal("unfittable viewport lied or changed view", failed.Viewport)
	}
	failed.Viewport.FitFailure.Message = "changed"
	if s.Snapshot().Viewport.FitFailure.Message == "changed" {
		t.Fatal("fit failure aliases service")
	}
	call(t, s, "view.viewport.set", ViewportParams{Width: 1000, Height: 600}, "recover")
	if s.Snapshot().Viewport.FitFailure != nil || s.Snapshot().View != before.View {
		t.Fatal("enlarging viewport did not recover")
	}
	before = s.Snapshot()
	raw := json.RawMessage(`{"follow_viewport":null}`)
	r := s.Dispatch(t.Context(), Request{Version: 1, ID: "null-follow", Key: "null-follow", Command: "view.fit", Params: raw, ExpectedCommandRevision: &before.CommandRevision}, "test")
	if r.Error == nil || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("null follow changed state")
	}
}

func TestViewportIsSessionOnlyAndSavedViewSurvivesOpen(t *testing.T) {
	s := NewExample()
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "view.fit", FitParams{Width: 520, Height: 138, FollowViewport: true}, "fit")
	before := s.Snapshot()
	path := filepath.Join(t.TempDir(), "view.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	call(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 0}, "pan")
	call(t, s, "project.load", LoadParams{Path: path}, "open")
	loaded := settled(t, s)
	if loaded.View != before.View || loaded.Viewport.Following || !loaded.Viewport.Registered {
		t.Fatal("open did not preserve saved view/manual mode")
	}
	call(t, s, "view.viewport.set", ViewportParams{Width: 900, Height: 600}, "resize")
	if s.Snapshot().View != before.View {
		t.Fatal("file view replaced on resize")
	}
	call(t, s, "example.load", ExampleLoadParams{ID: "pmos", DiscardChanges: true}, "example")
	if !s.Snapshot().Viewport.Following || math.Abs(s.Snapshot().View.Zoom-expectedViewportFit(t, s, 900, 600).Zoom) > 1e-15 {
		t.Fatal("new example did not re-enable following")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if s.Snapshot().Viewport.Following || !s.Snapshot().Viewport.Registered {
		t.Fatal("blank did not preserve dimensions and manual mode")
	}
}

// Check actual source corners, terminal positions and routed path vertices in
// screen coordinates, independently of the fit implementation's bound math.
func assertViewportContainsSchematic(t *testing.T, state Snapshot) {
	t.Helper()
	v := state.Viewport
	g := Graph{Components: state.Components, Wires: state.Wires}
	check := func(p Point) {
		x := (p.X-state.View.Center.X)*state.View.Zoom + v.Width/2
		y := (p.Y-state.View.Center.Y)*state.View.Zoom + v.Height/2
		if x < -1e-8 || x > v.Width+1e-8 || y < -1e-8 || y > v.Height+1e-8 {
			t.Fatalf("schematic point %+v outside measured %gx%g: %g,%g", p, v.Width, v.Height, x, y)
		}
	}
	for _, p := range g.Components {
		for _, pin := range p.Terminals {
			at, _ := TerminalPosition(p, pin)
			check(at)
		}
		annotated := false
		for _, a := range state.SourceAnnotations {
			annotated = annotated || a.ID == p.ID
		}
		if !annotated {
			_, text := CanvasPartLabel(p)
			at := CanvasPartLabelPositionAtZoom(p, text, state.View.Zoom)
			lo, hi := CanvasTextBounds(at, text, CanvasFontScale(state.View.Zoom))
			check(lo)
			check(hi)
		}
	}
	for _, w := range g.Wires {
		for _, at := range WirePath(g, w) {
			check(at)
		}
	}
	for _, a := range state.SourceAnnotations {
		lo, hi := SourceAnnotationBounds(a)
		check(lo)
		check(hi)
		part, _ := g.Component(a.ID)
		layout := LayoutCanvasAnnotation(a, part.Rotation, state.View.Zoom)
		for i, line := range a.Lines {
			lo, hi := CanvasTextBounds(layout.Position(line, i), line, layout.Scale)
			check(lo)
			check(hi)
		}
	}
}

func TestViewportDiscoveryIsPureOwnedAndBounded(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	for _, id := range []string{"view.fit", "view.viewport.set"} {
		data, _ := json.Marshal(CapabilityQuery{Command: id})
		r := s.Dispatch(t.Context(), Request{Version: 1, ID: id, Command: "capabilities.list", Params: data}, "test")
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		c := r.Result.(map[string]any)["capabilities"].([]Capability)[0]
		var schema map[string]any
		_ = json.Unmarshal(c.Input, &schema)
		contract := schema["x-command-contract"].(map[string]any)
		if contract["scope"] != "schematic-viewport" {
			t.Fatal("missing viewport contract")
		}
		for _, field := range []string{"width_px", "height_px"} {
			prop := schema["properties"].(map[string]any)[field].(map[string]any)
			if prop["minimum"] != float64(64) || prop["maximum"] != float64(100000) {
				t.Fatal("size bounds missing")
			}
		}
		c.Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("discovery aliases cached data")
		}
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("viewport discovery changed state")
	}
}
