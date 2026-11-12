package workbench

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLinkedBridgeSheetsRetainSeparateSettingsPortsAndRecordings(t *testing.T) {
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "example.load", ExampleLoadParams{ID: "discrete-hbridge"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	var selection []string
	for _, part := range s.Snapshot().Components {
		if part.ID != "VCC" && part.ID != "GNDS" && part.ID != "SUPPLY" {
			selection = append(selection, part.ID)
		}
	}
	call(t, s, "sheet.create", SheetCreateParams{ID: "Left", Name: "Left driver"}, "left")
	call(t, s, "sheet.open", IDParams{}, "main")
	call(t, s, "sheet.move", SheetMoveParams{IDs: selection, Sheet: "Left"}, "move")
	call(t, s, "sheet.instance.create", SheetInstanceParams{ID: "Right", Name: "Right driver", Template: "Left"}, "right")
	find := func(service *Service, origin string) Component {
		t.Helper()
		for _, part := range service.Snapshot().Components {
			if part.Sheet == "Right" && part.TemplatePart == origin {
				return part
			}
		}
		t.Fatalf("missing linked part %s", origin)
		return Component{}
	}
	linkedLoad := find(s, "RL")
	linkedSupply := find(s, "LOGICV")
	if linkedLoad.ID == "RL" || linkedLoad.Value != 100 || linkedSupply.ID == "LOGICV" {
		t.Fatal("linked parts did not receive independent identities and source settings")
	}
	call(t, s, "sheet.port.connect", SheetPortParams{From: "VCC.positive", To: linkedSupply.ID + ".node", Name: "VDD"}, "wire-right")
	call(t, s, "component.set", ValueParams{ID: "RL", Value: "120 Ω"}, "source-default")
	call(t, s, "sheet.template.apply", IDParams{ID: "Left"}, "apply-default")
	if got := find(s, "RL"); got.ID != linkedLoad.ID || got.Value != 120 {
		t.Fatal("untouched instance missed the new default or lost its identity", got)
	}
	call(t, s, "component.set", ValueParams{ID: linkedLoad.ID, Value: "150 Ω"}, "override")
	call(t, s, "component.set", ValueParams{ID: "RL", Value: "130 Ω"}, "source-edit")
	call(t, s, "sheet.template.apply", IDParams{ID: "Left"}, "apply-override")
	if got := find(s, "RL"); got.ID != linkedLoad.ID || got.Value != 150 {
		t.Fatal("applying a source changed the local override or probe identity", got)
	}
	call(t, s, "sheet.open", IDParams{ID: "Left"}, "open-source")
	call(t, s, "component.place", PlaceParams{ID: "TestPoint", Kind: "junction", Position: Point{X: -1000}}, "source-topology")
	call(t, s, "wire.connect", ConnectParams{ID: "VDDTest", From: "LOGICV.node", To: "TestPoint.node"}, "source-wire")
	call(t, s, "sheet.template.apply", IDParams{ID: "Left"}, "apply-topology")
	linkedPoint := find(s, "TestPoint")
	if linkedPoint.ID == "TestPoint" || linkedPoint.Kind != "junction" {
		t.Fatal("source topology edit did not produce an independent linked part")
	}
	linkedWire := ""
	for _, wire := range s.Snapshot().Wires {
		if wire.TemplateWire == "VDDTest" && wire.To == linkedPoint.ID+".node" {
			linkedWire = wire.ID
		}
	}
	if linkedWire == "" || linkedWire == "VDDTest" {
		t.Fatal("source topology edit did not remap the linked wire")
	}
	call(t, s, "sheet.open", IDParams{ID: "Right"}, "open-instance")
	check := func(state Snapshot) {
		t.Helper()
		result := state.Mixed
		if result == nil || result.Stale || len(result.MOSFETs) != 8 || len(result.Resistors) != 2 || result.KCLResidual > 1e-8 {
			t.Fatal("both linked bridges did not solve together", state.Diagnostics)
		}
		for _, reading := range result.Resistors {
			resistance := 130.
			if reading.ID == linkedLoad.ID {
				resistance = 150
			}
			voltage, current := bridgeLoadReference(resistance)
			if math.Abs(math.Abs(reading.Voltage)-voltage) > 2e-6 || math.Abs(math.Abs(reading.Current)-current) > 2e-8 {
				t.Fatalf("%s readings differ from independent load line: %+v", reading.ID, reading)
			}
		}
	}
	call(t, s, "simulation.restart", struct{}{}, "restart")
	check(settled(t, s))
	duration, step := .045, .001
	probes := []string{"v:RL", "v:" + linkedLoad.ID, "i:" + linkedLoad.ID}
	_, record := verify(t, s, VerifyParams{Duration: &duration, Step: &step, ProbeIDs: probes}, "record")
	path := filepath.Join(t.TempDir(), "linked-bridges.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	fresh := New()
	t.Cleanup(fresh.Close)
	call(t, fresh, "project.load", LoadParams{Path: path}, "reopen")
	reopened := settled(t, fresh)
	check(reopened)
	if reopened.ActiveSheet != "Right" || reopened.Document.Dirty || len(reopened.Operations) != 1 || len(reopened.Sheets) != 2 || reopened.Sheets[1].Template != "Left" || find(fresh, "RL").ID != linkedLoad.ID {
		t.Fatal("reopen lost the linked sheet or its retained experiment")
	}
	read := func(owner *Service, id string) []VerificationRow {
		return call(t, owner, "operation.samples", VerificationPageParams{ID: id, Limit: 200}, "read").Result.(VerificationPage).Rows
	}
	if !reflect.DeepEqual(read(s, record.Operation.ID), read(fresh, reopened.Operations[0].ID)) {
		t.Fatal("the retained capture changed after reopening linked sheets")
	}

	// Removing a source terminal can require an explicit change to an
	// independently wired instance. Applying in that case must be atomic.
	call(t, fresh, "object.delete", IDParams{ID: "LOGICV"}, "remove-source-terminal")
	before := fresh.Snapshot()
	refuse(t, fresh, "sheet.template.apply", IDParams{ID: "Left"}, "invalid-params")
	after := fresh.Snapshot()
	if !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Components, after.Components) || before.CommandRevision != after.CommandRevision {
		t.Fatal("refused update changed the instance or its external wire")
	}
}

func TestLinkedSheetUpdateIsAtomicAcrossInstances(t *testing.T) {
	s := New()
	t.Cleanup(s.Close)
	call(t, s, "component.place", PlaceParams{ID: "ParentJunction", Kind: "junction", Position: Point{X: 1000}}, "parent")
	call(t, s, "sheet.create", SheetCreateParams{ID: "Source", Name: "Source"}, "create")
	call(t, s, "component.place", PlaceParams{ID: "SharedPart", Kind: "resistor", Position: Point{}}, "part")
	call(t, s, "sheet.instance.create", SheetInstanceParams{ID: "First", Name: "First", Template: "Source"}, "first")
	call(t, s, "sheet.instance.create", SheetInstanceParams{ID: "Second", Name: "Second", Template: "Source"}, "second")
	var second Component
	for _, part := range s.Snapshot().Components {
		if part.Sheet == "Second" {
			second = part
		}
	}
	call(t, s, "sheet.port.connect", SheetPortParams{From: "ParentJunction.node", To: second.ID + ".1", Name: "Input"}, "wire-second")
	call(t, s, "object.delete", IDParams{ID: "SharedPart"}, "delete-source-part")
	before := s.Snapshot()
	refuse(t, s, "sheet.template.apply", IDParams{ID: "Source"}, "invalid-params")
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) {
		t.Fatal("the first linked sheet changed even though the second refused")
	}
}
