package workbench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestJunctionInsertionPreservesSupportedCircuitNumericsAndUndo(t *testing.T) {
	for _, example := range []string{"example.reset", "example.rc", "example.npn", "example.npn-pulse", "example.flasher"} {
		t.Run(example, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, example, ReplaceParams{}, "example")
			before := settled(t, s)
			wire := before.Wires[0]
			response := call(t, s, "wire.junction.insert", JunctionInsertParams{WireIDs: []string{wire.ID}}, "junction")
			joined := settled(t, s)
			if len(joined.Components) != len(before.Components)+1 || len(joined.Wires) != len(before.Wires)+1 || joined.Analysis != before.Analysis || joined.Status != "running" {
				t.Fatalf("junction broke supported graph: %s %+v", joined.Status, joined.Diagnostics)
			}
			if before.Result != nil && math.Abs(joined.Result.Current-before.Result.Current) > 1e-12 {
				t.Fatal("junction changed LED current")
			}
			if before.NPN != nil && math.Abs(joined.NPN.CollectorCurrent-before.NPN.CollectorCurrent) > 1e-10 {
				t.Fatal("junction changed NPN current")
			}
			id := response.Result.(Snapshot).Selected
			part, _ := s.graph.Component(id)
			revision, run := joined.ElectricalRevision, joined.RunID
			call(t, s, "component.move", MoveParams{ID: id, Position: Point{part.Position.X + 10, part.Position.Y + 20}}, "move")
			if s.Snapshot().ElectricalRevision != revision || s.Snapshot().RunID != run {
				t.Fatal("moving junction changed connectivity")
			}
			call(t, s, "edit.undo", struct{}{}, "undo-move")
			call(t, s, "edit.undo", struct{}{}, "undo-insert")
			restored := settled(t, s)
			if !reflect.DeepEqual(before.Components, restored.Components) || !reflect.DeepEqual(before.Wires, restored.Wires) {
				t.Fatal("junction undo was not atomic")
			}
		})
	}
}
func TestNetLabelsJoinRemoteWiresAndRenameRemainsExplicit(t *testing.T) {
	s := NewExample()
	defer s.Close()
	baseline := settled(t, s)
	edits := []BatchEdit{}
	add := func(command string, p any) {
		raw, _ := json.Marshal(p)
		edits = append(edits, BatchEdit{Command: command, Params: raw})
	}
	add("object.delete", IDParams{ID: "W1"})
	add("component.place", PlaceParams{ID: "LA", Kind: "net-label", Position: Point{-300, -100}, Value: "VCC"})
	add("component.place", PlaceParams{ID: "LB", Kind: "net-label", Position: Point{300, -100}, Value: "VCC"})
	add("wire.connect", ConnectParams{From: "V1.positive", To: "LA.node"})
	add("wire.connect", ConnectParams{From: "LB.node", To: "R1.1"})
	call(t, s, "edit.batch", BatchParams{Name: "Name the supply net", Edits: edits}, "labels")
	named := settled(t, s)
	if named.Result == nil || named.Result.Stale || math.Abs(named.Result.Current-baseline.Result.Current) > 1e-12 {
		t.Fatal("remote label connectivity changed electrical result")
	}
	compiled, _, diagnostic := s.graph.Compile()
	if diagnostic != nil || compiled.TerminalNet["LA.node"] != compiled.TerminalNet["LB.node"] {
		t.Fatal("same names did not join")
	}
	found := false
	for _, net := range named.Nets {
		if reflect.DeepEqual(net.Labels, []string{"VCC"}) {
			found = true
		}
	}
	if !found {
		t.Fatal("named net inspection missing")
	}
	call(t, s, "component.set", ValueParams{ID: "LB", Value: "vcc"}, "case-difference")
	if state := s.Snapshot(); state.Status != "incomplete" || !state.Result.Stale {
		t.Fatal("label case was silently aliased")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-name")
	settled(t, s)
	refuse(t, s, "component.set", ValueParams{ID: "LA", Value: "bad name"}, "invalid-params")
	path := filepath.Join(t.TempDir(), "named.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	reopened := settled(t, fresh)
	if math.Abs(reopened.Result.Current-baseline.Result.Current) > 1e-12 {
		t.Fatal("labels lost in project round trip")
	}
	svg := filepath.Join(t.TempDir(), "named.svg")
	call(t, fresh, "export.schematic.svg", ExportParams{Path: svg}, "svg")
	data, _ := os.ReadFile(svg)
	if !strings.Contains(string(data), "VCC") || !strings.Contains(string(data), "Net label") {
		t.Fatal("SVG omitted named primitive")
	}
}
func TestCrossingWiresOnlyJoinThroughExplicitMultiWireCommand(t *testing.T) {
	s := New()
	defer s.Close()
	for _, p := range []PlaceParams{{ID: "Left", Kind: "junction", Position: Point{-100, 0}}, {ID: "Right", Kind: "junction", Position: Point{100, 0}}, {ID: "Top", Kind: "junction", Position: Point{0, -100}}, {ID: "Bottom", Kind: "junction", Position: Point{0, 100}}} {
		call(t, s, "component.place", p, p.ID)
	}
	call(t, s, "wire.connect", ConnectParams{ID: "Horizontal", From: "Left.node", To: "Right.node"}, "horizontal")
	call(t, s, "wire.connect", ConnectParams{ID: "Vertical", From: "Top.node", To: "Bottom.node"}, "vertical")
	before, _, _ := s.graph.Compile()
	if before.TerminalNet["Left.node"] == before.TerminalNet["Top.node"] {
		t.Fatal("visual crossing created a connection")
	}
	point := Point{0, 0}
	refuse(t, s, "wire.junction.insert", JunctionInsertParams{WireIDs: []string{"Horizontal", "Vertical"}}, "invalid-params")
	call(t, s, "wire.junction.insert", JunctionInsertParams{ID: "Join", WireIDs: []string{"Horizontal", "Vertical"}, Position: &point}, "join")
	after, _, _ := s.graph.Compile()
	if after.TerminalNet["Left.node"] != after.TerminalNet["Top.node"] || len(s.Snapshot().Wires) != 4 {
		t.Fatal("explicit crossing junction did not merge all named wires")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-join")
	restored, _, _ := s.graph.Compile()
	if restored.TerminalNet["Left.node"] == restored.TerminalNet["Top.node"] {
		t.Fatal("crossing undo retained electrical merge")
	}
	point = Point{30, 30}
	refuse(t, s, "wire.junction.insert", JunctionInsertParams{WireIDs: []string{"Horizontal"}, Position: &point}, "invalid-params")
}
func TestPrimitiveDoesNotHideAnOpenPinAndGroundLabelIsNotGround(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "object.delete", IDParams{ID: "W1"}, "remove")
	call(t, s, "component.place", PlaceParams{ID: "Label", Kind: "net-label", Position: Point{400, 400}, Value: "GND"}, "label")
	call(t, s, "wire.connect", ConnectParams{From: "V1.positive", To: "Label.node"}, "label-source")
	state := s.Snapshot()
	if state.Status != "incomplete" || !state.Result.Stale {
		t.Fatal("label stub made a dangling pin simulate")
	}
	compiled, _, _ := s.graph.Compile()
	if compiled.TerminalNet["Label.node"] == compiled.TerminalNet["GND.reference"] {
		t.Fatal("GND name silently created reference")
	}
}

func TestMixedJunctionVerifyEquivalenceAndPrimitiveVoltageProbe(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	duration := .05
	_, before := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1", "i:D1"}}, "before")
	original := s.operations[before.Operation.ID].record.rows
	call(t, s, "wire.junction.insert", JunctionInsertParams{ID: "Branch", WireIDs: []string{s.Snapshot().Wires[0].ID}}, "junction")
	_, after := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1", "i:D1"}}, "after")
	changed := s.operations[after.Operation.ID].record.rows
	if len(original) != len(changed) {
		t.Fatal("junction changed output schedule")
	}
	for i, row := range original {
		for j, value := range row.Values {
			if math.Abs(value-changed[i].Values[j]) > 1e-9 {
				t.Fatal("junction changed coupled transient numerics")
			}
		}
	}
	_, probed := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:Branch.node", "v:V1.positive"}}, "probes")
	for _, row := range s.operations[probed.Operation.ID].record.rows {
		if math.Abs(row.Values[0]-row.Values[1]) > 1e-12 {
			t.Fatal("junction probe did not read its actual net")
		}
	}
	path := filepath.Join(t.TempDir(), "branched.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	if fresh.Snapshot().Document.RecordingCount != 3 {
		t.Fatal("primitive captured records did not survive project validation")
	}
}
