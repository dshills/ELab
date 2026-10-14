package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func refuse(t *testing.T, s *Service, id string, p any, code string) {
	t.Helper()
	before := settled(t, s)
	raw, _ := json.Marshal(p)
	rev := before.Revision
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "refuse", Command: id, Params: raw, ExpectedRevision: &rev, Key: fmt.Sprintf("refuse-%d", before.CommandRevision)}, "test")
	if r.Error == nil || r.Error.Code != code {
		t.Fatalf("%s: want %s, got %+v", id, code, r.Error)
	}
	after := s.Snapshot()
	if id == "project.load" && after.CommandRevision == before.CommandRevision+1 {
		if after.ProjectWork == nil || after.ProjectWork.Status != "failed" || after.ProjectWork.Error == nil || after.ProjectWork.Error.Code != code || after.ProjectWork.Actor != "test" {
			t.Fatal("admitted failed open lacks its terminal job outcome")
		}
		// Accepted asynchronous work may publish only operation bookkeeping.
		// Everything describing the existing project, physics and history is
		// still compared exactly against the state before admission.
		after.ProjectWork = before.ProjectWork
		after.Revision, after.CommandRevision, after.LastActor = before.Revision, before.CommandRevision, before.LastActor
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("failed %s changed session", id)
	}
}
func TestPersistenceRoundTripAndDirtyHistory(t *testing.T) {
	s := New()
	defer s.Close()
	ids := drawLED(t, s)
	call(t, s, "component.set", ValueParams{ID: ids["resistor"], Value: "2.2k"}, "value")
	call(t, s, "component.move", MoveParams{ID: ids["resistor"], Position: Point{40, -130}}, "pose")
	call(t, s, "component.rotate", IDParams{ID: ids["resistor"]}, "rotation")
	call(t, s, "view.pan", map[string]float64{"dx": 500, "dy": -50}, "pan")
	call(t, s, "view.zoom", map[string]float64{"factor": 1.5}, "zoom")
	original := settled(t, s)
	path := filepath.Join(t.TempDir(), "My LED.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	if s.Snapshot().Document.Dirty || s.Snapshot().RunID != original.RunID {
		t.Fatal("save restarted simulation or stayed dirty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeDocument(data)
	if err != nil || d.LEDModel != Model() {
		t.Fatalf("saved file is invalid: %v", err)
	}
	if strings.Contains(string(data), "run_id") || strings.Contains(string(data), path) {
		t.Fatal("runtime state or host path leaked into portable document")
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "open")
	reopened := settled(t, fresh)
	if !reflect.DeepEqual(original.Components, reopened.Components) || !reflect.DeepEqual(original.Wires, reopened.Wires) || original.View != reopened.View || original.Result.Current != reopened.Result.Current || reopened.Document.Dirty {
		t.Fatal("round trip changed document or readings")
	}
	call(t, fresh, "component.set", ValueParams{ID: ids["resistor"], Value: "1k"}, "edit")
	if !fresh.Snapshot().Document.Dirty {
		t.Fatal("edit not dirty")
	}
	refuse(t, fresh, "project.load", LoadParams{Path: path}, "unsaved-changes")
	refuse(t, fresh, "project.clear", ReplaceParams{}, "unsaved-changes")
	refuse(t, fresh, "app.close", ReplaceParams{}, "unsaved-changes")
	call(t, fresh, "edit.undo", struct{}{}, "undo-edit")
	settled(t, fresh)
	if fresh.Snapshot().Document.Dirty {
		t.Fatal("undo to save point stayed dirty")
	}
	call(t, fresh, "edit.redo", struct{}{}, "redo-edit")
	settled(t, fresh)
	call(t, fresh, "project.save", SaveParams{}, "save-current")
	call(t, fresh, "simulation.pause", struct{}{}, "pause")
	call(t, fresh, "project.load", LoadParams{Path: path}, "reload")
	if settled(t, fresh).Paused || fresh.Snapshot().Result.Stale {
		t.Fatal("load did not resume a fresh live simulation")
	}
	call(t, fresh, "project.clear", ReplaceParams{}, "new")
	if fresh.Snapshot().Document.Path != "" || fresh.Snapshot().Document.Dirty {
		t.Fatal("new didn't detach saved document")
	}
	call(t, fresh, "edit.undo", struct{}{}, "recover-new")
	settled(t, fresh)
	if !fresh.Snapshot().Document.Dirty {
		t.Fatal("recovered unnamed drawing should be dirty")
	}
}
func TestPersistenceOverwriteConflictFailureAndRetry(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	dir := t.TempDir()
	path := filepath.Join(dir, "saved.elab")
	rev := s.Snapshot().Revision
	raw, _ := json.Marshal(SaveParams{Path: path})
	request := Request{Version: 1, ID: "save", Command: "project.save", Params: raw, ExpectedRevision: &rev, Key: "save-once"}
	first := s.Dispatch(context.Background(), request, "test")
	if first.Error != nil {
		t.Fatal(first.Error)
	}
	info, _ := os.Stat(path)
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "2k"}, "edit")
	settled(t, s)
	repeated := s.Dispatch(context.Background(), request, "test")
	info2, _ := os.Stat(path)
	if !repeated.Duplicate || repeated.Error != nil || !os.SameFile(info, info2) || !s.Snapshot().Document.Dirty {
		t.Fatal("retry rewrote file or changed save point")
	}
	external := []byte("outside edit")
	if err := os.WriteFile(path, external, 0600); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "project.save", SaveParams{}, "file-conflict")
	refuse(t, s, "project.save", SaveParams{Path: filepath.Join(dir, "missing", "x.elab")}, "file-error")
	other := filepath.Join(dir, "other.elab")
	if err := os.WriteFile(other, external, 0600); err != nil {
		t.Fatal(err)
	}
	refuse(t, s, "project.save", SaveParams{Path: other}, "file-exists")
	if data, _ := os.ReadFile(other); string(data) != string(external) {
		t.Fatal("refused save overwrote file")
	}
	call(t, s, "project.save", SaveParams{Overwrite: true}, "force-save")
	if s.Snapshot().Document.Dirty {
		t.Fatal("explicit overwrite did not save")
	}
	link := filepath.Join(dir, "link.elab")
	if err := os.Symlink(path, link); err == nil {
		refuse(t, s, "project.save", SaveParams{Path: link, Overwrite: true}, "file-error")
		refuse(t, s, "project.load", LoadParams{Path: link}, "file-error")
	}
	temps, _ := filepath.Glob(filepath.Join(dir, ".elab-save-*"))
	if len(temps) != 0 {
		t.Fatal("temporary save files leaked")
	}
}
func TestInvalidDocumentsLeaveCurrentDrawingIntact(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	dir := t.TempDir()
	good := filepath.Join(dir, "good.elab")
	call(t, s, "project.save", SaveParams{Path: good}, "save")
	data, _ := os.ReadFile(good)
	tests := []struct {
		name   string
		change func([]byte) []byte
	}{
		{"truncated", func(b []byte) []byte { return b[:len(b)/2] }},
		{"future-version", func(b []byte) []byte { return []byte(strings.Replace(string(b), `"version": 4`, `"version": 99`, 1)) }},
		{"missing-wire-pin", func(b []byte) []byte { return []byte(strings.Replace(string(b), `R1.1`, `R1.unknown`, 1)) }},
		{"unknown-model", func(b []byte) []byte { return []byte(strings.Replace(string(b), ModelID, "other-led.v9", 1)) }},
		{"noncanonical-id", func(b []byte) []byte { return []byte(strings.Replace(string(b), `"id": "V1"`, `"ID": "V1"`, 1)) }},
		{"missing-zero", func(b []byte) []byte { return []byte(strings.Replace(string(b), `"value": 0,`, "", 1)) }},
		{"oversize", func(b []byte) []byte { return []byte(strings.Repeat(" ", MaxDocumentBytes+1)) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(dir, test.name+".elab")
			if err := os.WriteFile(path, test.change(data), 0600); err != nil {
				t.Fatal(err)
			}
			refuse(t, s, "project.load", LoadParams{Path: path}, "invalid-document")
		})
	}
	// A structurally valid incomplete drawing is still a document worth keeping.
	call(t, s, "object.delete", IDParams{ID: "W2"}, "incomplete")
	partial := filepath.Join(dir, "incomplete.elab")
	call(t, s, "project.save", SaveParams{Path: partial}, "save-partial")
	call(t, s, "project.load", LoadParams{Path: partial}, "load-partial")
	state := s.Snapshot()
	if state.Status != "incomplete" || state.Result != nil || state.Document.Dirty {
		t.Fatal("loading incomplete circuit reused old readings")
	}
}

// Recording assets may change while the live schematic/file association does not.
func schematicDocumentState(document DocumentState) DocumentState {
	document.Dirty = document.SchematicDirty
	document.RecordingsDirty = false
	document.RecordingCount = 0
	return document
}
