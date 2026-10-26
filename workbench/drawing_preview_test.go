package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func drawingQuery(t *testing.T, s *Service, at Point) DrawingPreview {
	t.Helper()
	raw, _ := json.Marshal(PositionParams{Position: at})
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "preview", Command: "drawing.preview", Params: raw}, "agent-reader")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	return r.Result.(DrawingPreview)
}

func commitDrawingPreview(t *testing.T, s *Service, preview DrawingPreview) Response {
	t.Helper()
	return s.Dispatch(context.Background(), Request{Version: 1, ID: "commit", Key: "commit", Command: preview.Command, Params: preview.Params, ExpectedCommandRevision: &preview.CommandRevision, ExpectedDocumentGeneration: &preview.DocumentGeneration}, "agent")
}

func TestDrawingPreviewPlacementPurityAdmissionAndUndo(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "drawing.tool.set", ToolParams{Tool: "place", Kind: "resistor"}, "tool")
	before := s.Snapshot()
	p := drawingQuery(t, s, Point{15, -15})
	if !p.Valid || p.Component == nil || p.Position != (Point{20, -20}) || p.Component.Position != p.Position || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("preview did not expose snapped placement without side effects", p)
	}
	// Query results are owned copies, including nested component metadata.
	p.Component.Position.X = 500
	if r := commitDrawingPreview(t, s, p); r.Error != nil {
		t.Fatal(r.Error)
	}
	after := s.Snapshot()
	if len(after.Components) != 1 || after.Components[0].Position != (Point{20, -20}) || after.UndoName != "Place component" {
		t.Fatal("preview command did not produce the advertised placement")
	}
	call(t, s, "drawing.tool.set", ToolParams{Tool: "place", Kind: "led"}, "led-tool")
	occupied := drawingQuery(t, s, Point{21, -19})
	if occupied.Valid || occupied.Command != "" || occupied.Component == nil {
		t.Fatal("occupied placement not shown as an invalid ghost", occupied)
	}
	call(t, s, "edit.undo", HistoryParams{AllowOtherActor: true, ExpectedEntryRevision: &after.UndoEntry.CommandRevision}, "undo")
	if len(s.Snapshot().Components) != 0 {
		t.Fatal("placement did not undo as one operation")
	}
}

func TestDrawingPreviewNearestPinAmbiguityAndFreshGuard(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "component.place", PlaceParams{ID: "A", Kind: "junction", Position: Point{}}, "a")
	call(t, s, "component.place", PlaceParams{ID: "B", Kind: "junction", Position: Point{10, 0}}, "b")
	call(t, s, "drawing.tool.set", ToolParams{Tool: "wire"}, "wire")
	before := s.Snapshot()
	closer := drawingQuery(t, s, Point{9, 0})
	if !closer.Valid || closer.Terminal != "B.node" || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("closer pin lost to component ordering", closer)
	}
	tie := drawingQuery(t, s, Point{5, 0})
	if tie.Valid || tie.Terminal != "" || tie.Command != "" {
		t.Fatal("ambiguous midpoint chose a pin", tie)
	}
	call(t, s, "component.move", MoveParams{ID: "B", Position: Point{100, 0}}, "agent-move")
	stable := s.Snapshot()
	if r := commitDrawingPreview(t, s, closer); r.Error == nil || r.Error.Code != "conflict" || !reflect.DeepEqual(stable, s.Snapshot()) {
		t.Fatal("old preview adopted intervening input", r.Error)
	}
	fresh := drawingQuery(t, s, Point{100, 0})
	if r := commitDrawingPreview(t, s, fresh); r.Error != nil || s.Snapshot().Drawing.WireStart != "B.node" {
		t.Fatal("fresh guarded preview could not start a wire", r.Error)
	}
	link := drawingQuery(t, s, Point{})
	if !link.Valid || link.Command != "wire.connect" || len(link.Path) < 2 || link.Path[0] != (Point{100, 0}) || link.Path[len(link.Path)-1] != (Point{}) {
		t.Fatal("connection preview has the wrong endpoints", link)
	}
	// This is a distinct intent/key from the earlier successful wire.begin.
	r := call(t, s, link.Command, link.Params, "connect")
	if r.Error != nil || len(s.Snapshot().Wires) != 1 {
		t.Fatal("connection preview could not commit")
	}
	call(t, s, "wire.begin", map[string]string{"terminal": "B.node"}, "again")
	if duplicate := drawingQuery(t, s, Point{}); duplicate.Valid || duplicate.Command != "" {
		t.Fatal("duplicate wire was advertised as committable")
	}
}

func TestDrawingPreviewStrictInputsAndDiscovery(t *testing.T) {
	s := New()
	defer s.Close()
	for _, raw := range []string{`{}`, `{"position":{"x":1}}`, `{"position":null}`, `{"position":{"x":0,"y":0},"extra":1}`} {
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: "invalid", Command: "drawing.preview", Params: json.RawMessage(raw)}, "agent")
		if r.Error == nil || r.Error.Code != "invalid-params" {
			t.Fatal("malformed preview accepted", raw)
		}
	}
	for _, kind := range Library() {
		call(t, s, "drawing.tool.set", ToolParams{Tool: "place", Kind: kind.Kind}, "tool-"+kind.Kind)
		p := drawingQuery(t, s, Point{100, 100})
		if !p.Valid || p.Component == nil || p.Component.Kind != kind.Kind {
			t.Fatal("existing component has no valid ghost", kind.Kind, p.Message)
		}
	}
	for _, cap := range Catalog() {
		if cap.ID == "drawing.preview" {
			if cap.Mutation || len(cap.OutputSchema) == 0 {
				t.Fatal("preview not discoverable as a typed pure query")
			}
			return
		}
	}
	t.Fatal("preview missing from public catalog")
}
