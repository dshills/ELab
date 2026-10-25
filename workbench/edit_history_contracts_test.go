package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAtomicEditHistoryContractsPreserveDiscoveryAndExposeSupportedCommands(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range []string{"edit.batch", "edit.undo", "edit.redo"} {
		details, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(details[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["x-command-contract"].(map[string]any)["scope"] != "atomic-edits-and-history" {
			t.Fatal("missing contract", id)
		}
		if id == "edit.batch" {
			commands := schema["x-supported-edit-commands"].([]any)
			if len(commands) != 23 || inputProperty(schema, "edits")["maxItems"] != float64(100) || inputProperty(schema, "name")["x-max-utf8-bytes"] != float64(80) {
				t.Fatal("missing batch domains")
			}
			for _, command := range commands {
				if _, err := detailedCapabilities(command.(string)); err != nil {
					t.Fatal("batch advertises undiscoverable child", err)
				}
			}
		}
		details[0].Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("caller corrupted later details", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("details changed ordinary discovery/output definitions")
	}
}

func TestAtomicEditNoOpNormalizesDeletedSelectionWithoutHistoryOrIDReuse(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	call(t, s, "edit.undo", HistoryParams{}, "undo")
	before := s.Snapshot()
	batch := BatchParams{Name: "Temporary construction", Edits: []BatchEdit{
		edit("component.place", PlaceParams{ID: "Temporary", Kind: "resistor", Position: Point{500, 500}, Value: "1k"}),
		edit("object.delete", IDParams{ID: "Temporary"}),
	}}
	call(t, s, "edit.batch", batch, "no-op")
	after := s.Snapshot()
	if after.Selected != "" || after.Drawing.Value != "" {
		t.Fatal("net-unchanged batch published selection/value for a deleted component")
	}
	if !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Transient, after.Transient) || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.Document != after.Document || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.RedoEntry, after.RedoEntry) || after.CommandRevision != before.CommandRevision+1 {
		t.Fatal("net-unchanged batch changed graph/physics/document/history")
	}
	refuse(t, s, "component.place", PlaceParams{ID: "Temporary", Kind: "resistor", Position: Point{500, 500}}, "invalid-params")
	call(t, s, "edit.redo", HistoryParams{}, "redo")
	part, _ := s.graph.Component("R1")
	if part.Value != 20000 {
		t.Fatal("net-unchanged batch cleared redo")
	}
}

func TestAtomicEditFailedBatchPreservesIDsAndUTF8NameBoundary(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	before := s.Snapshot()
	p := PlaceParams{ID: "Fresh", Kind: "resistor", Position: Point{500, 500}}
	batch := BatchParams{Name: "Rejected construction", Edits: []BatchEdit{edit("component.place", p), edit("component.set", ValueParams{ID: "missing", Value: "1k"})}}
	refuse(t, s, "edit.batch", batch, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("rejected batch leaked staged public state")
	}
	call(t, s, "component.place", p, "unused-id")
	valid := BatchParams{Name: strings.Repeat("µ", 40), Edits: []BatchEdit{edit("component.set", ValueParams{ID: "R1", Value: "20k"})}}
	call(t, s, "edit.batch", valid, "80-bytes")
	if s.Snapshot().UndoEntry.Name != valid.Name {
		t.Fatal("UTF-8 name was trimmed/truncated")
	}
	before = s.Snapshot()
	valid.Name += "x"
	refuse(t, s, "edit.batch", valid, "invalid-params")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("81-byte name changed state")
	}
}

func TestAtomicHistoryUsesLatestSavedBaselineAndPreservesCapturedBytes(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{Duration: .003}, "step")
	duration := .003
	_, record := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1", "i:C1"}}, "record")
	directory := t.TempDir()
	original := filepath.Join(directory, "original.elab")
	baseline := filepath.Join(directory, "edited.elab")
	call(t, s, "project.save", SaveParams{Path: original}, "save-original")
	originalBytes, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	batch := BatchParams{Name: "Tune RC", Edits: []BatchEdit{edit("component.set", ValueParams{ID: "R1", Value: "20k"}), edit("component.set", ValueParams{ID: "C1", Value: "2uF"})}}
	raw, _ := json.Marshal(batch)
	revision := s.Snapshot().CommandRevision
	request := Request{Version: 1, ID: "batch", Command: "edit.batch", Params: raw, ExpectedCommandRevision: &revision, Key: "batch-owned"}
	response := s.Dispatch(context.Background(), request, "test")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	call(t, s, "project.save", SaveParams{Path: baseline}, "save-edited")
	baselineBytes, err := os.ReadFile(baseline)
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	call(t, s, "edit.undo", HistoryParams{ExpectedEntryRevision: &before.UndoEntry.CommandRevision}, "undo")
	undone := s.Snapshot()
	if !undone.Document.Dirty || undone.Document.Path != baseline || undone.Document.Generation != before.Document.Generation || undone.Document.RecordingCount != 1 || undone.Transient.Sample != before.Transient.Sample {
		t.Fatal("undo restored save association/baseline, captured membership or time")
	}
	retry := s.Dispatch(context.Background(), request, "test")
	if !retry.Duplicate || retry.Error != nil || !reflect.DeepEqual(undone, s.Snapshot()) || retry.Result.(Snapshot).UndoEntry.Name != batch.Name {
		t.Fatal("retry reapplied the batch or returned current rather than original snapshot")
	}
	call(t, s, "edit.redo", HistoryParams{ExpectedEntryRevision: &undone.RedoEntry.CommandRevision}, "redo")
	if s.Snapshot().Document.Dirty || s.Snapshot().Document.Path != baseline {
		t.Fatal("redo did not return to latest saved content")
	}
	page := call(t, s, "operation.samples", VerificationPageParams{ID: record.Operation.ID}, "rows").Result.(VerificationPage)
	if len(page.Rows) != 4 || len(page.Probes) != 2 {
		t.Fatal("history lost complete capture")
	}
	for path, expected := range map[string][]byte{original: originalBytes, baseline: baselineBytes} {
		actual, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatal("history/retry changed explicit saved bytes", path, err)
		}
	}
}
