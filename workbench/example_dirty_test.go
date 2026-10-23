package workbench

import (
	"fmt"
	"reflect"
	"testing"
)

func TestUntouchedExamplesCanBeReplacedWithoutSave(t *testing.T) {
	s := New()
	defer s.Close()
	for i, info := range ExampleCatalog() {
		call(t, s, "example.load", ExampleLoadParams{ID: info.ID}, fmt.Sprintf("load-%d", i))
		for _, state := range []Snapshot{s.Snapshot(), settled(t, s)} {
			if state.Document.Dirty || state.Document.SchematicDirty || state.Document.PreferencesDirty || state.Document.RecordingsDirty || state.Document.Path != "" || state.Example.Modified {
				t.Fatalf("untouched %s is marked modified: %+v", info.ID, state.Document)
			}
		}
		call(t, s, "simulation.pause", struct{}{}, fmt.Sprintf("pause-%d", i))
	}
	// Legacy loaders and current-template Reset share the same clean baseline.
	for i, command := range []string{"example.reset", "example.rc", "example.npn", "example.current.reset"} {
		call(t, s, command, ReplaceParams{}, fmt.Sprintf("alias-%d", i))
		if state := settled(t, s); state.Document.Dirty {
			t.Fatalf("%s marks untouched example dirty", command)
		}
	}
	call(t, s, "app.close", ReplaceParams{}, "close-untouched")
}

func TestExampleBaselineProtectsEditsAndTracksUndo(t *testing.T) {
	prefs := DefaultDisplayPreferences()
	prefs.VoltageColor = true
	for _, tc := range []struct {
		name, command string
		params        any
	}{
		{"value", "component.set", ValueParams{ID: "R1", Value: "20k"}},
		{"position", "component.move", MoveParams{ID: "R1", Position: Point{X: 900, Y: 700}}},
		{"pin", "measurement.pin.add", PinSpec{ID: "i:R1", Position: Point{X: 20, Y: 30}}},
		{"display", "display.preferences.set", prefs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			defer s.Close()
			call(t, s, "example.rc", ReplaceParams{}, "load")
			settled(t, s)
			call(t, s, "simulation.pause", struct{}{}, "pause")
			call(t, s, tc.command, tc.params, "edit")
			settled(t, s)
			before := s.Snapshot()
			if !before.Document.Dirty {
				t.Fatal("real edit did not mark example dirty")
			}
			refuse(t, s, "example.load", ExampleLoadParams{ID: "led"}, "unsaved-changes")
			refuse(t, s, "app.close", ReplaceParams{}, "unsaved-changes")
			if !reflect.DeepEqual(before, s.Snapshot()) {
				t.Fatal("refused replacement altered edited example")
			}
			call(t, s, "edit.undo", HistoryParams{}, "undo-edit")
			if s.Snapshot().Document.Dirty {
				t.Fatal("undo to template stayed dirty")
			}
			call(t, s, "edit.redo", HistoryParams{}, "redo-edit")
			if !s.Snapshot().Document.Dirty {
				t.Fatal("redo edit lost dirty flag")
			}
			call(t, s, "example.current.reset", ExampleResetParams{DiscardChanges: true}, "reset")
			if state := settled(t, s); state.Document.Dirty || state.Example.Modified {
				t.Fatal("reset template is not clean")
			}
		})
	}
}
