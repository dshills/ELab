package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExampleCatalogAndProtectedResetTrackEveryTemplate(t *testing.T) {
	for _, info := range ExampleCatalog() {
		t.Run(info.ID, func(t *testing.T) {
			if info.Question == "" || info.Expected == "" || info.Try == "" || info.Limitation == "" {
				t.Fatal("incomplete example learning guidance")
			}
			s := New()
			defer s.Close()
			call(t, s, "simulation.pause", struct{}{}, "initial-pause")
			call(t, s, "example.load", ExampleLoadParams{ID: info.ID}, "load")
			original := settled(t, s)
			if original.Example == nil || original.Example.Info != info || original.Example.Modified || original.Paused {
				t.Fatal("example does not open as a running immutable template")
			}
			call(t, s, "simulation.pause", struct{}{}, "pause")
			id := ""
			for _, part := range original.Components {
				if part.Kind == "resistor" {
					id = part.ID
					break
				}
			}
			call(t, s, "component.move", MoveParams{ID: id, Position: Point{X: 900, Y: 700}}, "edit")
			if !s.Snapshot().Example.Modified {
				t.Fatal("edited circuit still claims original template")
			}
			before := s.Snapshot()
			refuse(t, s, "example.current.reset", ExampleResetParams{}, "unsaved-changes")
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("refused reset changes current circuit")
			}
			generation := before.Document.Generation
			call(t, s, "example.current.reset", ExampleResetParams{ExpectedID: info.ID, ExpectedGeneration: &generation, DiscardChanges: true}, "reset")
			reset := settled(t, s)
			if reset.Example.Info.ID != info.ID || reset.Example.Modified || !reset.Paused || !reflect.DeepEqual(reset.Components, original.Components) || !reflect.DeepEqual(reset.Wires, original.Wires) || reset.TransientSettings != original.TransientSettings || reset.RunID == before.RunID {
				t.Fatal("reset restores a different template or loses fresh paused startup")
			}
			if reset.Transient != nil && (reset.Transient.Stale || reset.Transient.Sample.Time != 0) {
				t.Fatal("reset imports prior time/charge")
			}
			call(t, s, "edit.undo", HistoryParams{}, "undo-reset")
			if s.Snapshot().Example.Info.ID != info.ID || !s.Snapshot().Example.Modified {
				t.Fatal("undo reset loses origin or edited circuit")
			}
		})
	}
}

func TestExampleOriginPortableUndoableAndGenerationGuarded(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "rc")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	path := filepath.Join(t.TempDir(), "edited-rc.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeDocument(data)
	if err != nil || d.ExampleID != "rc" {
		t.Fatal("example origin not portable", err)
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "load")
	loaded := settled(t, fresh)
	if loaded.Example == nil || loaded.Example.Info.ID != "rc" || !loaded.Example.Modified || loaded.Document.Dirty {
		t.Fatal("edited origin claims original physics or is not clean on reopen")
	}
	generation := loaded.Document.Generation
	call(t, fresh, "example.load", ExampleLoadParams{ID: "npn", DiscardChanges: true}, "replace")
	refuse(t, fresh, "example.current.reset", ExampleResetParams{ExpectedID: "rc", DiscardChanges: true}, "conflict")
	refuse(t, fresh, "example.current.reset", ExampleResetParams{ExpectedID: "npn", ExpectedGeneration: &generation, DiscardChanges: true}, "conflict")
	call(t, fresh, "edit.undo", HistoryParams{}, "undo-load")
	if fresh.Snapshot().Example.Info.ID != "rc" {
		t.Fatal("undo example load loses prior origin")
	}
	call(t, fresh, "project.clear", ReplaceParams{DiscardChanges: true}, "clear")
	if fresh.Snapshot().Example != nil {
		t.Fatal("blank drawing keeps example provenance")
	}
	refuse(t, fresh, "example.current.reset", ExampleResetParams{}, "unavailable")
	for _, raw := range []string{`null`, `""`, `"unknown"`, `1`} {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(data, &fields)
		fields["example_id"] = json.RawMessage(raw)
		bad, _ := json.Marshal(fields)
		if _, err := decodeDocument(bad); err == nil {
			t.Fatal("invalid example origin accepted", raw)
		}
	}
}

func TestExampleCatalogIsReadOnlyAndPreferenceEditsDoNotChangeTemplateClaim(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "list", Command: "examples.list", Params: json.RawMessage(`{}`)}, "reader")
	if r.Error != nil || len(r.Result.([]ExampleInfo)) != 7 || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("catalog mutates state or omits templates")
	}
	list := r.Result.([]ExampleInfo)
	list[0].Question = "changed"
	if ExampleCatalog()[0].Question == "changed" {
		t.Fatal("caller modifies immutable catalog")
	}
	call(t, s, "example.load", ExampleLoadParams{ID: "led"}, "led")
	settled(t, s)
	p := DefaultDisplayPreferences()
	p.VoltageColor = true
	call(t, s, "display.preferences.set", p, "prefs")
	call(t, s, "measurement.pin.add", PinSpec{ID: "i:D1", Position: Point{}}, "pin")
	if s.Snapshot().Example.Modified {
		t.Fatal("display preferences pretend template physics changed")
	}
}

func TestEditedExampleOriginSurvivesRecoveryAndUnknownCatalogIDsHaveNoPartContext(t *testing.T) {
	s := New()
	call(t, s, "example.load", ExampleLoadParams{ID: "rc"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	dir := t.TempDir()
	configureRecovery(t, s, dir, false)
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	before := s.Snapshot()
	rev := before.CommandRevision
	raw, _ := json.Marshal(ExampleLoadParams{ID: "R1", DiscardChanges: true})
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "bad-example", Command: "example.load", Params: raw, ExpectedCommandRevision: &rev, Key: "bad-example"}, "test")
	if r.Error == nil || r.Help == nil || len(r.Help.Objects) != 0 || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("unknown catalog ID is mistaken for a schematic part or changes origin")
	}
	s.Close()
	fresh := New()
	defer fresh.Close()
	configureRecovery(t, fresh, dir, false)
	call(t, fresh, "recovery.restore", choice, "restore")
	state := settled(t, fresh)
	if state.Example == nil || state.Example.Info.ID != "rc" || !state.Example.Modified || !state.Document.Dirty {
		t.Fatal("recovery loses edited example provenance")
	}
}
