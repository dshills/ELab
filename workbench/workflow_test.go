package workbench

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func edit(command string, p any) BatchEdit {
	data, _ := json.Marshal(p)
	return BatchEdit{command, data}
}
func TestSearchDuplicateAndAtomicHistory(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	before := s.Snapshot()
	call(t, s, "parts.search.set", SearchParams{"CAPACITOR"}, "search")
	if len(s.Snapshot().MatchingParts) != 1 || s.Snapshot().MatchingParts[0].Kind != "capacitor" || s.Snapshot().ElectricalRevision != before.ElectricalRevision {
		t.Fatal("search changed circuit or failed filtering")
	}
	call(t, s, "component.duplicate", DuplicateParams{ID: "C1"}, "duplicate")
	state := s.Snapshot()
	part, _ := s.graph.Component(state.Selected)
	if part.ID == "C1" || part.Rotation != 270 || part.Value != 1e-5 && math.Abs(part.Value-1e-5) > 1e-16 || part.InitialVoltage == nil || *part.InitialVoltage != .1 || len(state.Wires) != len(before.Wires) || state.UndoName != "Duplicate component" {
		t.Fatal("duplicate lost properties or copied wires")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-copy")
	if !reflect.DeepEqual(s.Snapshot().Components, before.Components) || s.Snapshot().RedoName != "Duplicate component" {
		t.Fatal("named undo did not restore graph")
	}
	call(t, s, "edit.redo", struct{}{}, "redo-copy")
	if len(s.Snapshot().Components) != 13 {
		t.Fatal("redo duplicate failed")
	}
	call(t, s, "edit.undo", struct{}{}, "remove-copy")
	before = s.Snapshot()
	bad := BatchParams{"Rejected batch", []BatchEdit{edit("component.place", PlaceParams{ID: "Fresh", Kind: "resistor", Position: Point{400, 400}}), edit("component.set", ValueParams{ID: "missing", Value: "1k"})}}
	refuse(t, s, "edit.batch", bad, "invalid-params")
	if !reflect.DeepEqual(s.Snapshot(), before) || s.usedIDs["Fresh"] {
		t.Fatal("failed batch partially changed/reserved graph")
	}
	batch := BatchParams{"Change capacitor pair", []BatchEdit{edit("component.set", ValueParams{"C1", "20uF"}), edit("component.set", ValueParams{"C2", "20uF"})}}
	call(t, s, "edit.batch", batch, "pair")
	after := s.Snapshot()
	if after.UndoName != batch.Name || after.ElectricalRevision != before.ElectricalRevision+1 || after.CommandRevision != before.CommandRevision+1 {
		t.Fatal("batch did not commit once")
	}
	call(t, s, "edit.undo", struct{}{}, "undo-pair")
	if !reflect.DeepEqual(s.Snapshot().Components, before.Components) || s.Snapshot().RedoName != batch.Name {
		t.Fatal("one undo did not restore whole batch")
	}
	refuse(t, s, "edit.batch", BatchParams{"No files", []BatchEdit{edit("project.save", SaveParams{Path: "should-not-exist"})}}, "invalid-params")
}
func TestScopeFixedStatisticsExportsAndSavedProbes(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{.03}, "30ms")
	ids := []string{"v:C1", "i:C1"}
	call(t, s, "scope.probes.set", ProbeParams{ids}, "scope")
	call(t, s, "scope.window.set", ScopeWindow{"fixed", .01, .02}, "interval")
	state := s.Snapshot()
	sc := state.Scope
	if !sc.WindowValid || len(sc.Statistics) != 2 || sc.Statistics[0].Count != 11 || sc.Statistics[0].Minimum < 3 || sc.Statistics[0].Maximum > 4.324 {
		t.Fatalf("interval included startup or lost endpoints %+v", sc.Statistics)
	}
	points := sc.Traces[0].Points
	sum, squared := 0., 0.
	for _, p := range points {
		if !scopeTimeInside(p.Time, .01, .02) {
			t.Fatal("outside interval")
		}
		sum += p.Value
		squared += p.Value * p.Value
	}
	if math.Abs(sc.Statistics[0].Mean-sum/11) > 1e-12 || math.Abs(sc.Statistics[0].RMS-math.Sqrt(squared/11)) > 1e-12 {
		t.Fatal("sample statistics incorrect")
	}
	directory := t.TempDir()
	csvPath := filepath.Join(directory, "scope.csv")
	svgPath := filepath.Join(directory, "schematic.svg")
	docPath := filepath.Join(directory, "rc.elab")
	call(t, s, "export.scope.csv", SaveParams{Path: csvPath}, "csv")
	data, _ := os.ReadFile(csvPath)
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil || len(rows) != 12 || rows[0][len(rows[0])-1] != "i:C1 [A]" || rows[1][2] != RCEngineID {
		t.Fatal("CSV units/identity/window wrong")
	}
	call(t, s, "export.schematic.svg", SaveParams{Path: svgPath}, "svg")
	data, _ = os.ReadFile(svgPath)
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(string(data), "R1.1") || !strings.Contains(string(data), "elab.schematic.svg.v1") {
		t.Fatal("SVG missing pins or identity")
	}
	after := s.Snapshot()
	if after.ElectricalRevision != state.ElectricalRevision || after.Transient.Sample != state.Transient.Sample || after.Document != state.Document {
		t.Fatal("export changed project or experiment")
	}
	refuse(t, s, "export.scope.csv", SaveParams{Path: csvPath}, "file-exists")
	call(t, s, "project.save", SaveParams{Path: docPath}, "save")
	refuse(t, s, "export.schematic.svg", SaveParams{Path: docPath, Overwrite: true}, "invalid-params")
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: docPath}, "load")
	loaded := fresh.Snapshot()
	if !reflect.DeepEqual(loaded.Scope.Selected, ids) || loaded.Scope.A != nil || loaded.Scope.Window.Mode != "live" || loaded.Transient.Sample.Time != 0 {
		t.Fatal("save/load lost probes or restored runtime history")
	}
	data, _ = os.ReadFile(docPath)
	var doc map[string]any
	json.Unmarshal(data, &doc)
	delete(doc, "scope_probes")
	delete(doc, "recordings")
	delete(doc, "paused")
	doc["version"] = 2
	old, _ := json.Marshal(doc)
	if _, err := decodeDocument(old); err != nil {
		t.Fatal("lost v2 compatibility", err)
	}
	doc["version"] = 3
	doc["scope_probes"] = map[string]any{"ids": []string{"i:missing"}}
	invalid, _ := json.Marshal(doc)
	if _, err := decodeDocument(invalid); err == nil {
		t.Fatal("loaded invalid probes")
	}
	call(t, s, "simulation.step", StepParams{.6}, "expire")
	if s.Snapshot().Scope.WindowValid || len(s.Snapshot().Scope.Statistics) != 0 || len(s.Snapshot().Scope.Traces[0].Points) != 0 {
		t.Fatal("expired window reported measurement")
	}
	refuse(t, s, "export.scope.csv", SaveParams{Path: filepath.Join(directory, "expired.csv")}, "unavailable")
}
func TestBatchConflictAndExportRefusalPreserveState(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	old := before.Revision
	call(t, s, "parts.search.set", SearchParams{"npn"}, "intervening")
	raw, _ := json.Marshal(BatchParams{"Place", []BatchEdit{edit("component.place", PlaceParams{Kind: "resistor", Position: Point{0, 0}})}})
	response := s.Dispatch(context.Background(), Request{Version: 1, ID: "batch", Command: "edit.batch", Params: raw, ExpectedRevision: &old, Key: "batch"}, "external")
	if response.Error == nil || response.Error.Code != "conflict" || len(s.Snapshot().Components) != 0 {
		t.Fatal("batch overwrote intervening user state")
	}
	svg := schematicSVG(FlasherExampleGraph(), 1, 1)
	if len(svg) == 0 {
		t.Fatal("headless rendering failed")
	}
	path := filepath.Join(t.TempDir(), "link.svg")
	target := filepath.Join(t.TempDir(), "target.svg")
	os.WriteFile(target, []byte("preserve"), 0600)
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "export.schematic.svg", SaveParams{Path: path, Overwrite: true}, "file-error")
	data, _ := os.ReadFile(target)
	if string(data) != "preserve" {
		t.Fatal("export overwrote link target")
	}
}
