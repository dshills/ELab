package main

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeUnchangedValueApplyPreservesStoredState(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned exact Inspector value", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "C1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "1.234567890123456 µF"})
	uiAction(t, s, "apply-value", nil)
	agentAction(t, s.service, "project.save", workbench.SaveParams{Path: filepath.Join(t.TempDir(), "exact.elab")})
	before := s.service.Snapshot()
	original, _ := (workbench.Graph{Components: before.Components}).Component("C1")
	s.app.Do(func() {
		field, ok := findNode(s.view(), "value-entry")
		if !ok {
			t.Fatal("editable value missing")
		}
		parsed, err := workbench.ParseValue(field.Text, "F")
		if err != nil || math.Float64bits(parsed) != math.Float64bits(original.Value) {
			t.Fatal("native field rounded stored value", field.Text, err)
		}
	})
	uiAction(t, s, "apply-value", nil)
	after := s.service.Snapshot()
	if before.Document != after.Document || before.ElectricalRevision != after.ElectricalRevision || before.RunID != after.RunID || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(before.RedoEntry, after.RedoEntry) {
		t.Fatal("untouched native Apply changed circuit/save/history/physics")
	}
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "C1", Value: after.Drawing.Value})
	agent := s.service.Snapshot()
	if after.Document != agent.Document || after.ElectricalRevision != agent.ElectricalRevision || !reflect.DeepEqual(after.Components, agent.Components) || !reflect.DeepEqual(after.UndoEntry, agent.UndoEntry) {
		t.Fatal("agent reapply differs from native")
	}
}
