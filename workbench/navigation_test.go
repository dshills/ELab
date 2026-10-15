package workbench

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestFitContainsLongNetLabel(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{Kind: "net-label", Position: Point{}, Value: strings.Repeat("N", 32)}, "label")
	call(t, s, "view.fit", FitParams{Width: 650, Height: 400}, "fit")
	state := s.Snapshot()
	part := state.Components[0]
	// Native label is rendered to the right of the terminal marker. Even the
	// less conservative SVG fallback advance must fit within the actual view.
	right := part.Position.X + 50 + float64(len(part.ID)+len(part.Label)+3)*7.5
	if math.Abs((right-state.View.Center.X)*state.View.Zoom) > 650./2 {
		t.Fatal("Fit clips long net label", state.View)
	}
}

func TestNavigationOrderFiltersAndPresentationOnly(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.reset", ReplaceParams{DiscardChanges: true}, "fixture")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := s.Snapshot()
	call(t, s, "drawing.tool.set", ToolParams{Tool: "wire"}, "wire-tool")
	call(t, s, "wire.begin", map[string]string{"terminal": "R1.1"}, "wire-start")
	call(t, s, "selection.navigate", NavigateParams{Direction: "first", Center: true}, "first")
	state := s.Snapshot()
	if state.Selected != state.Components[0].ID || state.View.Center != state.Components[0].Position || state.View.Zoom != before.View.Zoom || state.Drawing.Tool != "select" || state.Drawing.WireStart != "" {
		t.Fatal("first navigation", state.Selected, state.View, state.Drawing)
	}
	call(t, s, "selection.navigate", NavigateParams{Direction: "previous"}, "previous")
	if got := s.Snapshot().Selected; got != before.Wires[len(before.Wires)-1].ID {
		t.Fatal("wrap", got)
	}
	call(t, s, "selection.navigate", NavigateParams{Direction: "next", Kind: "components"}, "filtered")
	if got := s.Snapshot().Selected; got != before.Components[0].ID {
		t.Fatal("filtered", got)
	}
	call(t, s, "selection.navigate", NavigateParams{Direction: "last", Kind: "components", Center: true}, "last")
	call(t, s, "view.pan", map[string]float64{"dx": 200, "dy": 200}, "pan")
	id := s.Snapshot().Selected
	call(t, s, "view.selection.center", CenterSelectionParams{ID: id}, "center")
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || before.Document.Dirty != after.Document.Dirty || before.UndoName != after.UndoName || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("navigation changed circuit/history/physics")
	}
}

func TestNavigationRefusalsAreAtomic(t *testing.T) {
	s := New()
	defer s.Close()
	awaitTestInitialization(t, s)
	for _, tc := range []struct {
		command string
		p       any
		code    string
	}{
		{"selection.navigate", NavigateParams{Direction: "next"}, "unavailable"},
		{"selection.navigate", NavigateParams{Direction: "sideways"}, "invalid-params"},
		{"view.selection.center", CenterSelectionParams{}, "unavailable"},
		{"view.selection.center", CenterSelectionParams{ID: "gone"}, "conflict"},
	} {
		before := s.Snapshot()
		raw, _ := json.Marshal(tc.p)
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: tc.command + tc.code, Key: tc.command + tc.code, Command: tc.command, Params: raw, ExpectedCommandRevision: &before.CommandRevision}, "test")
		if r.Error == nil || r.Error.Code != tc.code {
			t.Fatal("unexpected refusal", tc, r.Error)
		}
		after := s.Snapshot()
		if before.CommandRevision != after.CommandRevision || before.View != after.View || before.Selected != after.Selected || before.Drawing != after.Drawing {
			t.Fatal("refusal mutated", tc, r.Error)
		}
	}
}

func TestRoutingEnvelopeContainsRoutes(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.flasher", ReplaceParams{DiscardChanges: true}, "flasher")
	state := s.Snapshot()
	g := Graph{Components: state.Components, Wires: state.Wires}
	for _, w := range g.Wires {
		lo, hi, ok := WireRoutingBounds(g, w)
		if !ok {
			t.Fatal(w.ID)
		}
		for _, p := range WirePath(g, w) {
			if p.X < lo.X || p.X > hi.X || p.Y < lo.Y || p.Y > hi.Y {
				t.Fatal("outside envelope", w.ID, p, lo, hi)
			}
		}
	}
	call(t, s, "view.fit", FitParams{Width: 800, Height: 600}, "fit")
	v := s.Snapshot().View
	for _, w := range g.Wires {
		for _, p := range WirePath(g, w) {
			if math.Abs((p.X-v.Center.X)*v.Zoom) > 400 || math.Abs((p.Y-v.Center.Y)*v.Zoom) > 300 {
				t.Fatal("route clipped", w.ID, p, v)
			}
		}
	}
	call(t, s, "selection.navigate", NavigateParams{Direction: "first", Kind: "wires", Center: true}, "wire-center")
	if got := s.Snapshot(); got.Selected != g.Wires[0].ID || got.View.Zoom != v.Zoom {
		t.Fatal("wire center", got.Selected, got.View)
	}
}

func TestFitExtremeSpreadRefusesWithoutChangingView(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{Kind: "resistor", Position: Point{}, Value: "1k"}, "part1")
	call(t, s, "component.place", PlaceParams{Kind: "resistor", Position: Point{100, 0}, Value: "1k"}, "part2")
	s.mu.Lock()
	s.graph.Components[0].Position.X = -1e6
	s.graph.Components[1].Position.X = 1e6
	s.mu.Unlock()
	before := s.Snapshot()
	raw, _ := json.Marshal(FitParams{Width: 650, Height: 400})
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "fit-limit", Key: "fit-limit", Command: "view.fit", Params: raw, ExpectedCommandRevision: &before.CommandRevision}, "test")
	if r.Error == nil || r.Error.Code != "view-limit" || s.Snapshot().View != before.View || s.Snapshot().CommandRevision != before.CommandRevision {
		t.Fatal("fit-limit", r.Error)
	}
}
