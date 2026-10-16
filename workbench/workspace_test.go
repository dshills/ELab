package workbench

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWorkspaceLayoutPreservesCircuitMeasurementsAndView(t *testing.T) {
	s := New()
	defer s.Close()
	if s.Snapshot().WorkspaceLayout != DefaultWorkspaceLayout() {
		t.Fatal("unexpected initial split")
	}
	call(t, s, "example.load", ExampleLoadParams{ID: "nmos"}, "example")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	before := settled(t, s)
	for i, fraction := range []float64{.35, .85, .65, .65} {
		call(t, s, "workspace.layout.set", WorkspaceLayout{fraction}, fmt.Sprint(i))
		after := s.Snapshot()
		if after.WorkspaceLayout.SchematicFraction != fraction || after.View != before.View || after.Viewport != before.Viewport || after.ScopeVisible != before.ScopeVisible || after.Document != before.Document || after.UndoName != before.UndoName || after.RedoName != before.RedoName || after.RunID != before.RunID || after.ElectricalRevision != before.ElectricalRevision || !reflect.DeepEqual(after.Components, before.Components) || !reflect.DeepEqual(after.Wires, before.Wires) || !reflect.DeepEqual(after.Mixed, before.Mixed) || !reflect.DeepEqual(after.Transient, before.Transient) || !reflect.DeepEqual(after.Scope, before.Scope) || !reflect.DeepEqual(after.Recording, before.Recording) {
			t.Fatal("split changed circuit/view/measurement state")
		}
	}
}

func TestWorkspaceLayoutRefusalIsAtomic(t *testing.T) {
	s := New()
	defer s.Close()
	for _, p := range []map[string]any{{}, {"schematic_fraction": nil}, {"schematic_fraction": .3499}, {"schematic_fraction": .8501}, {"schematic_fraction": "0.65"}, {"schematic_fraction": .65, "unknown": true}} {
		before := s.Snapshot()
		refuse(t, s, "workspace.layout.set", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid layout changed session", p)
		}
	}
}

func TestWorkspaceLayoutSurvivesDocumentReplacementButIsNotSaved(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "workspace.layout.set", WorkspaceLayout{.8}, "layout")
	call(t, s, "example.load", ExampleLoadParams{ID: "nmos"}, "load")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	path := filepath.Join(t.TempDir(), "layout.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	call(t, s, "project.clear", ReplaceParams{}, "clear")
	call(t, s, "project.load", LoadParams{Path: path}, "open")
	if s.Snapshot().WorkspaceLayout.SchematicFraction != .8 {
		t.Fatal("document replacement reset session layout")
	}
	other := New()
	defer other.Close()
	call(t, other, "project.load", LoadParams{Path: path}, "other-open")
	if other.Snapshot().WorkspaceLayout != DefaultWorkspaceLayout() {
		t.Fatal("document imported another session's layout")
	}
}

func TestWorkspaceLayoutDiscoveryIsPureAndOwned(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	for i := 0; i < 2; i++ {
		c, err := detailedCapabilities("workspace.layout.set")
		if err != nil {
			t.Fatal(err)
		}
		var input map[string]any
		if err := json.Unmarshal(c[0].Input, &input); err != nil {
			t.Fatal(err)
		}
		p := inputProperty(input, "schematic_fraction")
		if p["minimum"] != .35 || p["maximum"] != .85 || p["default"] != .65 || input["x-command-contract"].(map[string]any)["scope"] != "workspace-layout" {
			t.Fatal("layout bounds/contract missing")
		}
		c[0].Input[0] = '!'
	}
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("discovery changed state")
	}
}

func TestWorkspaceLayoutPreservesRecordedCursorsAndPlayback(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	call(t, s, "simulation.pause", struct{}{}, "pause")
	duration := .02
	_, operation := verify(t, s, VerifyParams{Duration: &duration}, "verify")
	call(t, s, "operation.select", IDParams{ID: operation.Operation.ID}, "select")
	call(t, s, "recording.view.set", RecordingViewParams{Mode: "recorded"}, "recorded")
	call(t, s, "recording.cursor.set", CursorParams{Cursor: "a", Time: .01}, "cursor")
	before := s.Snapshot()
	call(t, s, "workspace.layout.set", WorkspaceLayout{.8}, "layout")
	if after := s.Snapshot(); !reflect.DeepEqual(before.Recording, after.Recording) || !reflect.DeepEqual(before.Transient, after.Transient) {
		t.Fatal("split changed recorded cursor/live state")
	}
	call(t, s, "recording.play", RecordingPlayParams{Speed: .001}, "play")
	call(t, s, "workspace.layout.set", WorkspaceLayout{.4}, "playing-layout")
	if !s.Snapshot().Recording.Playing {
		t.Fatal("split stopped playback")
	}
}
