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

func TestPinnedRCReadingsPreservePhysicsAndTrackSignedDischarge(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.rc", ReplaceParams{}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "simulation.restart", struct{}{}, "restart")
	path := filepath.Join(t.TempDir(), "rc.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	before := s.Snapshot()
	for i, id := range []string{"v:C1", "i:C1"} {
		call(t, s, "measurement.pin.add", PinSpec{ID: id, Position: Point{X: float64(i) * 300, Y: 250}}, id)
	}
	after := s.Snapshot()
	if after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Transient, before.Transient) || !after.Document.PreferencesDirty || after.Document.SchematicDirty || !after.Document.Dirty || after.UndoName != "Pin reading" {
		t.Fatal("pin changes physical state or fails presentation history/dirty semantics")
	}
	call(t, s, "simulation.step", StepParams{Duration: .01}, "charge")
	charged := s.Snapshot().PinnedReadings
	if charged[0].State != "live" || charged[0].Value == nil || math.Abs(*charged[0].Value-5*(1-math.Exp(-1))) > 1e-12 || charged[1].Value == nil || *charged[1].Value <= 0 {
		t.Fatalf("invalid RC pin readings: %+v", charged)
	}
	call(t, s, "simulation.input.set", InputParams{Mode: "discharge"}, "low")
	call(t, s, "simulation.step", StepParams{Duration: .001}, "discharge")
	discharged := s.Snapshot().PinnedReadings
	if *discharged[1].Value >= 0 || !strings.Contains(discharged[1].Display, "mA") || discharged[1].Time != .011 {
		t.Fatal("signed live current/time missing")
	}
	*discharged[0].Value = 999
	if *s.Snapshot().PinnedReadings[0].Value == 999 {
		t.Fatal("pin value aliases cached state")
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	stale := s.Snapshot().PinnedReadings[0]
	if stale.State != "stale" || !strings.Contains(stale.Display, "stale") || stale.RunID == s.Snapshot().RunID {
		t.Fatal("old physical result mislabeled current")
	}
	call(t, s, "object.delete", IDParams{ID: "C1"}, "delete")
	for _, reading := range s.Snapshot().PinnedReadings {
		if reading.State != "unavailable" || reading.Value != nil {
			t.Fatal("removed source displays an unrelated old result")
		}
	}
	call(t, s, "edit.undo", HistoryParams{}, "undo-delete")
	call(t, s, "simulation.restart", struct{}{}, "reset")
	if s.Snapshot().PinnedReadings[0].State != "live" {
		t.Fatal("restored source does not resume pin readings")
	}
}

func TestPinnedDCFramesUseTheirOriginalGraphAndAllKernels(t *testing.T) {
	for _, example := range []string{"example.reset", "example.npn", "example.npn-pulse", "example.flasher"} {
		t.Run(example, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, example, ReplaceParams{}, "example")
			settled(t, s)
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, "measurement.pin.add", PinSpec{ID: "i:D1", Position: Point{}}, "pin")
			before := s.Snapshot().PinnedReadings[0]
			if before.State != "live" || before.Value == nil || *before.Value <= 0 {
				t.Fatalf("accepted kernel result missing: %+v", before)
			}
			call(t, s, "component.move", MoveParams{ID: "D1", Position: Point{X: 900, Y: 800}}, "move")
			if *s.Snapshot().PinnedReadings[0].Value != *before.Value {
				t.Fatal("geometry changes reading")
			}
			call(t, s, "object.delete", IDParams{ID: "W1"}, "break-loop")
			stale := s.Snapshot().PinnedReadings[0]
			if stale.State != "stale" || *stale.Value != *before.Value {
				t.Fatal("stale frame is interpreted with changed node associations")
			}
		})
	}
	s := newGraphService(SolveLED, potGraph(t, ""))
	settled(t, s)
	defer s.Close()
	call(t, s, "measurement.pin.add", PinSpec{ID: "v:P1.wiper", Position: Point{}}, "pot-pin")
	if value := s.Snapshot().PinnedReadings[0].Value; value == nil || math.Abs(*value-2.5) > 1e-9 {
		t.Fatal("network DC pin missing")
	}
	call(t, s, "potentiometer.wiper.set", WiperParams{ID: "P1", Position: 1}, "wiper")
	if value := s.Snapshot().PinnedReadings[0].Value; value == nil || math.Abs(*value-4.9995) > 1e-9 {
		t.Fatal("DC wiper does not update pin")
	}
}

func TestPinnedPreferencesSaveMissingSourceUndoAndValidation(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "measurement.pin.add", PinSpec{ID: "i:D1", Position: Point{X: 23, Y: 47}}, "pin")
	if s.Snapshot().PinnedReadings[0].Definition.Position != (Point{X: 20, Y: 50}) {
		t.Fatal("pin not snapped")
	}
	wrong := Point{X: 0, Y: 0}
	refuse(t, s, "measurement.pin.move", MoveParams{ID: "i:D1", Position: Point{}, PreviousPosition: &wrong}, "conflict")
	call(t, s, "measurement.pin.move", MoveParams{ID: "i:D1", Position: Point{X: -400, Y: 300}}, "move")
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	if s.Snapshot().PinnedReadings[0].Definition.Position != (Point{X: 20, Y: 50}) {
		t.Fatal("undo lost pin position")
	}
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	call(t, s, "object.delete", IDParams{ID: "D1"}, "delete-source")
	path := filepath.Join(t.TempDir(), "missing.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	if s.Snapshot().Document.Dirty {
		t.Fatal("pin preferences never become clean")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeDocument(data)
	if err != nil || d.PinnedReadings == nil || len(d.PinnedReadings.Items) != 1 {
		t.Fatalf("missing source preference not portable: %v", err)
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	r := fresh.Snapshot().PinnedReadings[0]
	if r.State != "unavailable" || r.Value != nil || r.Definition.Position != (Point{X: -400, Y: 300}) || fresh.Snapshot().Document.Dirty {
		t.Fatal("reopen trusts physical values or loses missing source")
	}
	call(t, fresh, "measurement.pin.remove", IDParams{ID: "i:D1"}, "remove")
	if !fresh.Snapshot().Document.PreferencesDirty || fresh.Snapshot().Document.SchematicDirty {
		t.Fatal("removal dirty semantics wrong")
	}
	refuse(t, fresh, "project.clear", ReplaceParams{}, "unsaved-changes")
	call(t, fresh, "edit.undo", HistoryParams{}, "undo-remove")
	if fresh.Snapshot().Document.Dirty {
		t.Fatal("undo does not restore saved pin preferences")
	}
	for _, raw := range []string{`null`, `{"items":null}`, `{"items":[{"id":"v:D1","position":{"x":0}}]}`, `{"items":[{"id":"v:D1","position":{"x":1,"y":0}}]}`, `{"items":[{"id":"i:D1","position":{"x":0,"y":0},"value":3}]}`, `{"items":[{"id":"v:BAD.what","position":{"x":0,"y":0}}]}`} {
		var document map[string]json.RawMessage
		_ = json.Unmarshal(data, &document)
		document["pinned_readings"] = json.RawMessage(raw)
		damaged, _ := json.Marshal(document)
		if _, err := decodeDocument(damaged); err == nil {
			t.Fatalf("accepted invalid pin preferences %s", raw)
		}
	}
}

func TestPinnedBoundsExportAndRecovery(t *testing.T) {
	s := NewExample()
	settled(t, s)
	dir := t.TempDir()
	call(t, s, "project.save", SaveParams{Path: filepath.Join(t.TempDir(), "clean.elab")}, "save-clean")
	configureRecovery(t, s, dir, false)
	call(t, s, "measurement.pin.add", PinSpec{ID: "i:D1", Position: Point{X: 1700, Y: 800}}, "pin")
	if s.Snapshot().Document.SchematicDirty || !s.Snapshot().Document.PreferencesDirty {
		t.Fatal("recovery fixture must have only dirty pinned preferences")
	}
	call(t, s, "view.fit", FitParams{Width: 800, Height: 500}, "fit")
	if s.Snapshot().View.Center.X < 500 {
		t.Fatal("Fit ignores pinned readings")
	}
	path := filepath.Join(t.TempDir(), "pins.svg")
	call(t, s, "export.schematic.svg", ExportParams{Path: path}, "svg")
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "Live i:D1") || !strings.Contains(string(data), "pinned_readings") || !strings.Contains(string(data), `id="pin-i:D1"`) {
		t.Fatalf("SVG loses pinned readings or provenance: %v", err)
	}
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	s.Close()
	fresh := New()
	defer fresh.Close()
	configureRecovery(t, fresh, dir, false)
	call(t, fresh, "recovery.restore", choice, "restore")
	state := settled(t, fresh)
	if len(state.PinnedReadings) != 1 || state.PinnedReadings[0].Definition.Position.X != 1700 || !state.Document.Dirty {
		t.Fatal("recovery loses presentation state")
	}
}

func TestPinnedCountValidationRetryAndRecordedViewStayLive(t *testing.T) {
	s := pausedFlasher(t)
	defer s.Close()
	ids := []string{"i:D1", "i:D2", "v:C1", "v:C2", "v:Q1.collector", "v:Q2.collector", "i:Q1.base", "i:Q2.base"}
	for i, id := range ids {
		call(t, s, "measurement.pin.add", PinSpec{ID: id, Position: Point{X: float64(i) * 100}}, id)
	}
	refuse(t, s, "measurement.pin.add", PinSpec{ID: "v:V1", Position: Point{}}, "invalid-params")
	refuse(t, s, "measurement.pin.add", PinSpec{ID: "i:D1", Position: Point{}}, "invalid-params")
	refuse(t, s, "measurement.pin.move", MoveParams{ID: "i:D1", Position: Point{X: 10001}}, "invalid-params")
	refuse(t, s, "measurement.pin.remove", IDParams{ID: "v:missing"}, "not-found")
	before := s.Snapshot()
	first := call(t, s, "measurement.pin.move", MoveParams{ID: "i:D1", Position: Point{X: 200}}, "retry-pin")
	second := call(t, s, "measurement.pin.move", MoveParams{ID: "i:D1", Position: Point{X: 200}}, "retry-pin")
	if !reflect.DeepEqual(first.Result, second.Result) || s.Snapshot().CommandRevision != before.CommandRevision+1 {
		t.Fatal("pin retry created extra undo or changed original outcome")
	}
	mutated := first.Result.(Snapshot)
	*mutated.PinnedReadings[0].Value = 999
	third := call(t, s, "measurement.pin.move", MoveParams{ID: "i:D1", Position: Point{X: 200}}, "retry-pin")
	if *third.Result.(Snapshot).PinnedReadings[0].Value == 999 {
		t.Fatal("pin retry cache exposes mutable readings")
	}
	duration := .01
	_, record := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	call(t, s, "operation.select", IDParams{ID: record.Operation.ID}, "select")
	live := s.Snapshot().PinnedReadings
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "recorded")
	if !reflect.DeepEqual(live, s.Snapshot().PinnedReadings) {
		t.Fatal("recorded view replaces live pinned readings with recorded values")
	}
	call(t, s, "measurement.pin.remove", IDParams{ID: "i:D1"}, "remove")
	refuse(t, s, "measurement.pin.add", PinSpec{ID: "i:missing", Position: Point{}}, "not-found")
}
