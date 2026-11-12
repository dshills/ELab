package workbench

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMultiSheetDualBridgePreservesSharedRailAndRecordings(t *testing.T) {
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "example.load", ExampleLoadParams{ID: "dual-hbridge"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	before := settled(t, s)
	if before.Mixed == nil {
		t.Fatal(before.Diagnostics)
	}
	var lower []string
	for _, part := range before.Components {
		if strings.HasPrefix(part.ID, "B_") {
			lower = append(lower, part.ID)
		}
	}
	call(t, s, "sheet.create", Sheet{ID: "Driver", Name: "Right motor driver"}, "create")
	if state := s.Snapshot(); state.ActiveSheet != "Driver" || state.ElectricalRevision != before.ElectricalRevision {
		t.Fatal("empty sheet changed accepted physics or failed to open")
	}
	call(t, s, "sheet.open", IDParams{ID: ""}, "main")
	call(t, s, "sheet.move", SheetMoveParams{IDs: lower, Sheet: "Driver"}, "move")
	moved := settled(t, s)
	if moved.ActiveSheet != "Driver" || moved.Mixed == nil || !reflect.DeepEqual(moved.Mixed.Resistors, before.Mixed.Resistors) || !reflect.DeepEqual(moved.Scope.Selected, before.Scope.Selected) {
		t.Fatalf("moving the second bridge changed readings, probes or active sheet: %v", moved.Diagnostics)
	}
	project := Graph{Components: moved.Components, Wires: moved.Wires, Sheets: moved.Sheets}
	if len(project.Wires) != len(before.Wires)+1 || project.Wires[len(project.Wires)-1].Port != "VDD" || len(project.SheetDrawing("Driver").Components) != len(lower) {
		t.Fatal("the split VDD label did not become a named connection")
	}
	call(t, s, "edit.undo", struct{}{}, "undo")
	undone := settled(t, s)
	if undone.ActiveSheet != "" || !reflect.DeepEqual(undone.Wires, before.Wires) || !reflect.DeepEqual(undone.Components, before.Components) {
		t.Fatal("undo did not restore the original drawing")
	}
	call(t, s, "edit.redo", struct{}{}, "redo")
	redone := settled(t, s)
	if redone.ActiveSheet != "Driver" || !reflect.DeepEqual(redone.Wires, moved.Wires) {
		t.Fatal("redo changed the new sheet or the generated connection")
	}
	duration, step := .045, .001
	_, record := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: moved.Scope.Selected}, "record")
	call(t, s, "recording.keep.set", RecordingKeepParams{ID: record.Operation.ID, Keep: true}, "keep")
	path := filepath.Join(t.TempDir(), "dual-sheets.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	t.Cleanup(fresh.Close)
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	opened := settled(t, fresh)
	if opened.Document.Dirty || opened.Document.FormatVersion != SheetDocumentVersion || opened.ActiveSheet != "Driver" || len(opened.Operations) != 1 || !reflect.DeepEqual(opened.Sheets, moved.Sheets) || !reflect.DeepEqual(opened.Wires, moved.Wires) || !reflect.DeepEqual(opened.Mixed.Resistors, moved.Mixed.Resistors) {
		t.Fatal("multi-sheet save/reopen changed the circuit, sheet, or retained capture")
	}
}

func TestSheetCoordinatesLabelsAndPortsAreLocal(t *testing.T) {
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	call(t, s, "component.place", PlaceParams{ID: "MainLabel", Kind: "net-label", Value: "DATA", Position: Point{}}, "first")
	call(t, s, "sheet.create", Sheet{ID: "Control", Name: "Control"}, "create")
	call(t, s, "component.place", PlaceParams{ID: "ControlLabel", Kind: "net-label", Value: "DATA", Position: Point{}}, "second")
	state := s.Snapshot()
	g := Graph{Components: state.Components, Wires: state.Wires, Sheets: state.Sheets}
	compiled, _, _ := g.Compile()
	a, b := compiled.TerminalNet["MainLabel.node"], compiled.TerminalNet["ControlLabel.node"]
	if a == "" || b == "" || a == b {
		t.Fatal("matching labels accidentally joined different sheets")
	}
	connection := ConnectParams{From: "MainLabel.node", To: "ControlLabel.node"}
	refuse(t, s, "wire.connect", connection, "invalid-params")
	call(t, s, "sheet.port.connect", SheetPortParams{From: connection.From, To: connection.To, Name: "Data"}, "join")
	state = s.Snapshot()
	g.Wires = state.Wires
	compiled, _, _ = g.Compile()
	if compiled.TerminalNet["MainLabel.node"] == "" || compiled.TerminalNet["MainLabel.node"] != compiled.TerminalNet["ControlLabel.node"] {
		t.Fatal("named port failed to join its endpoints")
	}
	refuse(t, s, "sheet.delete", IDParams{ID: "Control"}, "invalid-params")
	broken := g.Clone()
	broken.Wires[0].Port = ""
	if _, _, diagnostic := broken.Compile(); diagnostic == nil || diagnostic.Code != "invalid-sheets" {
		t.Fatal("unmarked cross-sheet wire was accepted", diagnostic)
	}
}
