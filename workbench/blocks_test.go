package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"math/cmplx"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBlocksTransparentRCAndBoundaryWiring(t *testing.T) {
	original := StepResponseExampleGraph()
	g := RCBlockExampleGraph()
	physical, err := g.PhysicalGraph()
	if err != nil {
		t.Fatal(err)
	}
	if !ElectricalEqual(original, g) || SimulatedComponentCount(original) != SimulatedComponentCount(g) {
		t.Fatal("block changed primitive network")
	}
	a, _, da := original.Compile()
	b, _, db := g.Compile()
	if da != nil || db != nil {
		t.Fatal(da, db)
	}
	if !reflect.DeepEqual(a.Circuit, b.Circuit) || b.TerminalNet["FILTER.out"] != b.TerminalNet["C1.1"] {
		t.Fatal("boundary aliases", b.TerminalNet)
	}
	if len(physical.Components) != len(original.Components) {
		t.Fatal("hidden parts lost")
	}
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-block"}, "block")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration, step := .001, .0001
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:C1"}}, "capture")
	s.mu.Lock()
	rows := s.operations[op.Operation.ID].record.rows
	got := rows[len(rows)-1].Values[0]
	s.mu.Unlock()
	want := 5 * (1 - math.Exp(-1))
	if math.Abs(got-want) > 1e-5 {
		t.Fatal("independent RC", got, want)
	}
	projected := g.VisibleGraph()
	if len(projected.Components) != len(g.Components)-2 {
		t.Fatal("collapsed projection")
	}
	hidden, _ := g.Component("R1")
	at, _ := TerminalPosition(hidden, "1")
	if ref, _ := TerminalAt(g, at, 1); ref == "R1.1" {
		t.Fatal("hidden pin acquired")
	}
	physical.Components[0].Name = "changed"
	if g.Components[0].Name == "changed" {
		t.Fatal("physical projection aliases")
	}
}
func TestBlocksSharedCommandsCopyHistoryPersistenceAndIsolation(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-block"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	call(t, s, "block.view.set", BlockViewParams{ID: "FILTER", Expanded: true}, "open")
	opened := settled(t, s)
	if opened.RunID != before.RunID || opened.ElectricalRevision != before.ElectricalRevision {
		t.Fatal("visual expansion restarted physics")
	}
	if len((Graph{Components: opened.Components, Wires: opened.Wires}).VisibleGraph().Components) != len(opened.Components)-1 {
		t.Fatal("internal projection")
	}
	call(t, s, "block.parameter.set", BlockValueParams{ID: "FILTER", Parameter: "resistance", Value: "2k"}, "parameter")
	after := settled(t, s)
	r, _ := (Graph{Components: after.Components, Wires: after.Wires}).Component("R1")
	if r.Value != 2000 {
		t.Fatal(r)
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	if !reflect.DeepEqual(s.Snapshot().Components, opened.Components) {
		t.Fatal("parameter undo")
	}
	call(t, s, "edit.redo", struct{}{}, "redo")
	call(t, s, "block.copy", IDParams{ID: "FILTER"}, "copy")
	if !s.Snapshot().BlockClipboardAvailable {
		t.Fatal("clipboard missing")
	}
	path := filepath.Join(t.TempDir(), "blocks.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	settled(t, s)
	call(t, s, "project.load", LoadParams{Path: path, DiscardChanges: true}, "reopen")
	reopened := settled(t, s)
	if !reflect.DeepEqual(reopened.Components, after.Components) {
		t.Fatal("block save round trip")
	}
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "new")
	call(t, s, "block.paste", BlockPasteParams{ID: "FILTER2", Position: Point{X: 600, Y: 300}}, "paste")
	pasted := s.Snapshot()
	if len(pasted.Components) != 3 || len(pasted.Wires) != 1 {
		t.Fatal("copy included outside circuit", pasted.Components, pasted.Wires)
	}
	if p, _ := (Graph{Components: pasted.Components, Wires: pasted.Wires}).Component("FILTER2_P1"); p.Value != 2000 {
		t.Fatal("copy lacked parameter", p)
	}
	call(t, s, "block.parameter.set", BlockValueParams{ID: "FILTER2", Parameter: "resistance", Value: "3k"}, "independent")
	s.mu.Lock()
	copiedValue := s.blockClipboard.Graph.Components[0].Value
	s.mu.Unlock()
	if copiedValue != 2000 {
		t.Fatal("paste modified clipboard")
	}
	call(t, s, "block.release", IDParams{ID: "FILTER2"}, "release")
	if len(s.Snapshot().Components) != 2 {
		t.Fatal("release removed primitive")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-release")
	call(t, s, "object.delete", IDParams{ID: "FILTER2"}, "delete-block")
	if len(s.Snapshot().Components) != 0 || len(s.Snapshot().Wires) != 0 {
		t.Fatal("orphaned block member")
	}
}
func TestBlocksRejectInvalidGraphsAtomicallyAndMoveTogether(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-block"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	for _, p := range []BlockValueParams{{"FILTER", "missing", "1k"}, {"FILTER", "resistance", "-1"}, {"missing", "resistance", "1k"}} {
		refuse(t, s, "block.parameter.set", p, "invalid-params")
	}
	refuse(t, s, "object.delete", IDParams{ID: "R1"}, "invalid-params")
	foundWire := false
	for _, w := range before.Wires {
		if w.From == "R1.2" && w.To == "C1.1" || w.To == "R1.2" && w.From == "C1.1" {
			foundWire = true
			refuse(t, s, "wire.junction.insert", JunctionInsertParams{WireIDs: []string{w.ID}}, "invalid-params")
		}
	}
	refuse(t, s, "wire.connect", ConnectParams{From: "V1.positive", To: "C1.1"}, "invalid-params")
	if !foundWire {
		t.Fatal("internal junction fixture missing")
	}
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("refusal mutation")
	}
	p, _ := (Graph{Components: before.Components, Wires: before.Wires}).Component("FILTER")
	call(t, s, "component.move", MoveParams{ID: p.ID, Position: Point{X: p.Position.X + 400, Y: p.Position.Y}}, "move")
	moved := s.Snapshot()
	for _, id := range []string{"FILTER", "R1", "C1"} {
		a, _ := (Graph{Components: before.Components}).Component(id)
		b, _ := (Graph{Components: moved.Components}).Component(id)
		if b.Position.X != a.Position.X+400 {
			t.Fatal("member did not move", id)
		}
	}
	if moved.ElectricalRevision != before.ElectricalRevision {
		t.Fatal("move electrical revision")
	}
	g := RCBlockExampleGraph()
	for _, variant := range []string{"nested", "missing", "pin", "member-label", "crossing"} {
		t.Run(variant, func(t *testing.T) {
			bad := g.Clone()
			k := len(bad.Components) - 1
			b := bad.Components[k].Block
			switch variant {
			case "nested":
				b.Members[0] = "FILTER"
			case "missing":
				b.Members[0] = "absent"
			case "pin":
				b.Ports[0].Terminal = "C1.nope"
			case "member-label":
				for k, c := range bad.Components {
					if c.ID == "R1" {
						bad.Components[k].Kind = "net-label"
					}
				}
			case "crossing":
				bad.Wires = append(bad.Wires, Wire{ID: "EVIL", From: "V1.positive", To: "C1.1", Connected: true})
			}
			if _, err := bad.PhysicalGraph(); err == nil {
				t.Fatal("invalid block admitted")
			}
		})
	}
	s.mu.Lock()
	raw, err := json.Marshal(s.documentLocked())
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeDocument(raw); err != nil {
		t.Fatal("own document invalid", err)
	}
	svg := string(schematicSVG(g, 1, 1))
	if !strings.Contains(svg, "block_graph") || !strings.Contains(svg, `id="FILTER"`) {
		t.Fatal("schematic lost block metadata/symbol")
	}
}

func TestBlocksIndependentInstancesAndNamedPortProbes(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-block"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "block.copy", IDParams{ID: "FILTER"}, "copy")
	call(t, s, "block.paste", BlockPasteParams{ID: "SECOND", Position: Point{X: 600, Y: 300}}, "paste")
	for _, w := range []ConnectParams{{From: "V1.positive", To: "SECOND.in"}, {From: "GND.reference", To: "SECOND.ground"}} {
		call(t, s, "wire.connect", w, "connect-"+w.To)
	}
	call(t, s, "block.parameter.set", BlockValueParams{ID: "SECOND", Parameter: "resistance", Value: "2k"}, "second-r")
	call(t, s, "block.parameter.set", BlockValueParams{ID: "SECOND", Parameter: "capacitance", Value: "2uF"}, "second-c")
	duration, step := .004, .00002
	_, op := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: []string{"v:FILTER.out", "v:SECOND.out"}}, "capture")
	s.mu.Lock()
	rows := s.operations[op.Operation.ID].record.rows
	values := append([]float64{}, rows[len(rows)-1].Values...)
	s.mu.Unlock()
	for k, tau := range []float64{.001, .004} {
		want := 5 * (1 - math.Exp(-duration/tau))
		if math.Abs(values[k]-want) > 1e-5 {
			t.Fatal("independent block RC response", k, values, want)
		}
	}
}

func TestCatalogSummariesRetainFullDetailedDescriptions(t *testing.T) {
	shortened := 0
	for _, c := range Catalog() {
		if c.Description == c.fullDescription {
			continue
		}
		shortened++
		d, err := detailedCapabilities(c.ID)
		if err != nil || len(d) != 1 || d[0].Description != c.fullDescription {
			t.Fatal("description lost", c.ID, err)
		}
	}
	if shortened == 0 {
		t.Fatal("catalog summaries not exercised")
	}
}

func TestBlocksPreserveConservativeWireCurrents(t *testing.T) {
	s := NewExample()
	defer s.Close()
	before := settled(t, s)
	p := BlockCreateParams{ID: "LAMP", Name: "LED module", Position: Point{Y: 300}, Members: []string{"R1", "D1"}, Ports: []BlockPort{{Name: "in", Terminal: "R1.1"}, {Name: "out", Terminal: "D1.cathode"}}, Parameters: []BlockParameter{{Name: "resistance", ComponentID: "R1"}}}
	call(t, s, "block.create", p, "group")
	after := s.Snapshot()
	if after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision {
		t.Fatal("group restarted physics")
	}
	for _, w := range after.Display.Wires {
		if w.Current == nil || w.Voltage == nil {
			t.Fatal("group lost wire reading", w)
		}
		if math.Abs(*w.Current-before.Result.Current) > 1e-12 {
			t.Fatal("independent LED wire current", w)
		}
	}
}

func TestBlockInspectOwnsNestedDataAndViewUndo(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-block"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	cfg := PassiveConfig{Version: PassiveLossModelID}
	cfg.ESR = 10
	call(t, s, "passive.configure", PassiveParams{ID: "C1", Enabled: true, Parameters: cfg}, "loss")
	before := s.Snapshot()
	raw, err := json.Marshal(IDParams{ID: "FILTER"})
	if err != nil {
		t.Fatal(err)
	}
	response := s.Dispatch(t.Context(), Request{Version: 1, ID: "inspect", Command: "block.inspect", Params: raw}, "reader")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	template, ok := response.Result.(*BlockTemplate)
	if !ok || template == nil {
		t.Fatal("template absent")
	}
	template.Block.Ports[0].Terminal = "changed"
	template.Block.Members[0] = "changed"
	for k, c := range template.Graph.Components {
		if c.ID == "C1" {
			template.Graph.Components[k].Device.Passive.ESR = 999
		}
	}
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("query aliases service")
	}
	call(t, s, "block.view.set", BlockViewParams{ID: "FILTER", Expanded: true}, "expand")
	call(t, s, "edit.undo", struct{}{}, "undo-view")
	if !reflect.DeepEqual(before.Components, s.Snapshot().Components) {
		t.Fatal("view history corrupted")
	}
}

func TestBlocksHideSourceAnnotationsFromFitAndExport(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "component.move", MoveParams{ID: "V1", Position: Point{X: -9000}}, "source-away")
	p := BlockCreateParams{ID: "CIRCUIT", Name: "LED circuit", Position: Point{X: 300, Y: 300}, Members: []string{"V1", "R1", "D1"}, Ports: []BlockPort{{Name: "sourceReturn", Terminal: "V1.negative"}, {Name: "ledReturn", Terminal: "D1.cathode"}}, Parameters: []BlockParameter{}}
	call(t, s, "block.create", p, "group")
	call(t, s, "view.viewport.set", ViewportParams{Width: 520, Height: 200}, "viewport")
	call(t, s, "view.fit", FitParams{FollowViewport: true}, "fit")
	state := s.Snapshot()
	if state.View.Zoom < .2 {
		t.Fatal("hidden source annotation enlarged fit", state.View)
	}
	assertViewportContainsSchematic(t, state)
	s.mu.Lock()
	g := s.graph.Clone()
	annotations := s.sourceAnnotationsLocked()
	s.mu.Unlock()
	svg := string(schematicSVGWithAnnotations(g, 1, 1, state.Display, annotations, state.TransientSettings))
	if strings.Contains(svg, `id="V1" transform=`) {
		t.Fatal("hidden source drawn")
	}
	// The viewBox must frame the visible sheet; metadata still retains -9000.
	var x, y, w, h float64
	start := strings.Index(svg, `viewBox="`)
	if start < 0 {
		t.Fatal("viewbox missing")
	}
	if _, err := fmt.Sscanf(svg[start+9:], "%f %f %f %f", &x, &y, &w, &h); err != nil {
		t.Fatal(err)
	}
	if w > 2000 {
		t.Fatal("hidden source annotation enlarged SVG", w)
	}
}

func TestBlocksCollapsedPlacementAndExpansionCollisionAreExplicit(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc-block"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	member, _ := (Graph{Components: before.Components}).Component("R1")
	call(t, s, "component.place", PlaceParams{ID: "EXTERNAL", Kind: "resistor", Position: member.Position, Value: "1k"}, "visible-empty")
	rejectedBefore := s.Snapshot()
	refuse(t, s, "block.view.set", BlockViewParams{ID: "FILTER", Expanded: true}, "invalid-params")
	refuse(t, s, "block.release", IDParams{ID: "FILTER"}, "invalid-params")
	rejectedAfter := s.Snapshot()
	if !reflect.DeepEqual(rejectedBefore.Components, rejectedAfter.Components) || !reflect.DeepEqual(rejectedBefore.Wires, rejectedAfter.Wires) {
		t.Fatal("collision refusal changed graph")
	}
	call(t, s, "component.move", MoveParams{ID: "FILTER", Position: Point{X: 600, Y: 300}}, "free-area")
	call(t, s, "block.view.set", BlockViewParams{ID: "FILTER", Expanded: true}, "expand")
	expected := s.Snapshot()
	path := filepath.Join(t.TempDir(), "overlap.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	settled(t, s)
	call(t, s, "project.load", LoadParams{Path: path, DiscardChanges: true}, "reopen")
	opened := settled(t, s)
	if !reflect.DeepEqual(expected.Components, opened.Components) || !reflect.DeepEqual(expected.Wires, opened.Wires) {
		t.Fatal("expanded placement state lost on reopen")
	}
}

func TestBlockTerminalPointBoundsMalformedCycles(t *testing.T) {
	for _, target := range []string{"A.loop", "B.loop"} {
		a := blockComponent("A", "A", Point{}, &CircuitBlock{Version: BlockModelID, Expanded: true, Ports: []BlockPort{{Name: "loop", Terminal: target}}})
		b := blockComponent("B", "B", Point{X: 300}, &CircuitBlock{Version: BlockModelID, Expanded: true, Ports: []BlockPort{{Name: "loop", Terminal: "A.loop"}}})
		if _, ok := (Graph{Components: []Component{a, b}}).TerminalPoint("A.loop"); ok {
			t.Fatal("malformed nested or cyclic port resolved")
		}
	}
}

func TestBlocksPreserveDCAndACAnalysisThroughNamedPins(t *testing.T) {
	g := RCBlockExampleGraph()
	cfg := acSettings("V1", "v:FILTER.out")
	r := sweepForTest(t, g, cfg)
	if len(r.Points) != cfg.Points {
		t.Fatal("AC point inventory", len(r.Points))
	}
	for _, point := range r.Points {
		if len(point.AC) != 1 {
			t.Fatal("AC response inventory", point)
		}
		want := 1 / complex(1, 2*math.Pi*point.Input*.001)
		got := complex(point.AC[0].Real, point.AC[0].Imaginary)
		if cmplx.Abs(got-want) > 1e-10 {
			t.Fatal("block changed analytic RC transfer", point, want)
		}
	}
	dc, frame, err := solveDCOperatingPoint(t.Context(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, _, d := g.Compile()
	physical, err := g.PhysicalGraph()
	if err != nil {
		t.Fatal(err)
	}
	flat, _, fd := physical.Compile()
	if fd != nil {
		t.Fatal(fd)
	}
	for ref, net := range flat.TerminalNet {
		if c.TerminalNet[ref] != net {
			t.Fatal("block changed physical net identity", ref)
		}
	}
	measured := false
	for _, probe := range probeInventory(g) {
		if probe.ID == "v:FILTER.out" {
			value, ok := frameValue(frame, probe)
			if !ok || math.Abs(value-5) > 1e-10 {
				t.Fatal("DC frame lost canonical block probe", value, ok)
			}
			measured = true
		}
	}
	if !measured {
		t.Fatal("block probe absent")
	}
	if d != nil || math.Abs(dc.Nodes[c.TerminalNet["FILTER.out"]]-5) > 1e-10 {
		t.Fatal("block changed DC bias", dc, d)
	}
}
