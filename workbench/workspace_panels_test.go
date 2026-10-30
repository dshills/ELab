package workbench

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWorkspacePanelsAndToolsPreserveCircuitAndResults(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	_, operation := verify(t, s, VerifyParams{Duration: &duration}, "capture")
	call(t, s, "operation.select", IDParams{ID: operation.Operation.ID}, "select")
	before := settled(t, s)
	for i, tool := range []string{"recordings", "spectrum", "sweeps", "tolerances", "circuit", "live"} {
		call(t, s, "workspace.simulation.set", SimulationToolParams{Tool: tool}, tool)
		panels := WorkspacePanels{PartsVisible: i%2 == 0, InspectorVisible: i%2 != 0, PartsWidth: 300, InspectorWidth: 480}
		call(t, s, "workspace.panels.set", panels, "panels-"+tool)
		after := s.Snapshot()
		if after.SimulationTool != tool || after.WorkspacePanels != panels {
			t.Fatal("presentation not applied")
		}
		if before.Document != after.Document || before.RunID != after.RunID || before.ElectricalRevision != after.ElectricalRevision || before.View != after.View || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Wires, after.Wires) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Scope, after.Scope) || !reflect.DeepEqual(before.Recording, after.Recording) || !reflect.DeepEqual(before.Spectrum, after.Spectrum) || !reflect.DeepEqual(before.Tolerance, after.Tolerance) || !reflect.DeepEqual(before.Operations, after.Operations) || !reflect.DeepEqual(before.VerificationSettings, after.VerificationSettings) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
			t.Fatal("layout/navigation changed circuit or measurement work")
		}
	}
}

func TestWorkspacePanelsAndToolsInvalidRequestsAreAtomic(t *testing.T) {
	s := New()
	defer s.Close()
	for _, p := range []any{map[string]any{}, map[string]any{"parts_visible": false}, WorkspacePanels{true, true, 223, 296}, WorkspacePanels{true, true, 361, 296}, WorkspacePanels{true, true, 224, MinInspectorPanelWidth - 1}, WorkspacePanels{true, true, 224, 561}, map[string]any{"parts_visible": nil, "inspector_visible": true, "parts_width": 224, "inspector_width": 296}, map[string]any{"parts_visible": true, "inspector_visible": true, "parts_width": 224, "inspector_width": 296, "extra": true}} {
		before := s.Snapshot()
		refuse(t, s, "workspace.panels.set", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid panel request changed state", p)
		}
	}
	for _, p := range []any{map[string]any{}, map[string]any{"tool": nil}, SimulationToolParams{Tool: "unknown"}, map[string]any{"tool": "live", "extra": true}} {
		before := s.Snapshot()
		refuse(t, s, "workspace.simulation.set", p, "invalid-params")
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("invalid task request changed state", p)
		}
	}
}

func TestWorkspacePresentationGuardsRetriesAndDocumentIsolation(t *testing.T) {
	s := New()
	defer s.Close()
	before := s.Snapshot()
	panels := WorkspacePanels{false, false, 320, 500}
	raw, _ := json.Marshal(panels)
	request := Request{Version: 1, ID: "panels", Key: "panels", Command: "workspace.panels.set", Params: raw, ExpectedCommandRevision: &before.CommandRevision, ExpectedDocumentGeneration: &before.Document.Generation}
	first := s.Dispatch(context.Background(), request, "agent")
	if first.Error != nil {
		t.Fatal(first.Error)
	}
	call(t, s, "workspace.simulation.set", SimulationToolParams{Tool: "tolerances"}, "task")
	current := s.Snapshot()
	retry := s.Dispatch(context.Background(), request, "agent")
	if retry.Error != nil || !retry.Duplicate || !reflect.DeepEqual(current, s.Snapshot()) {
		t.Fatal("layout retry replayed side effects")
	}
	request.Key = "fresh-stale"
	if r := s.Dispatch(context.Background(), request, "agent"); r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("stale panel callback accepted")
	}
	path := filepath.Join(t.TempDir(), "layout.elab")
	call(t, s, "project.save", SaveParams{Path: path}, "save")
	call(t, s, "project.clear", ReplaceParams{}, "clear")
	call(t, s, "project.load", LoadParams{Path: path}, "load")
	if s.Snapshot().WorkspacePanels != panels || s.Snapshot().SimulationTool != "tolerances" {
		t.Fatal("document replacement lost session layout")
	}
	fresh := New()
	defer fresh.Close()
	call(t, fresh, "project.load", LoadParams{Path: path}, "other")
	if fresh.Snapshot().WorkspacePanels != DefaultWorkspacePanels() || fresh.Snapshot().SimulationTool != "live" {
		t.Fatal("layout leaked through a schematic")
	}
}

func TestWorkspacePanelAndToolDiscoveryIsOwned(t *testing.T) {
	for _, id := range []string{"workspace.panels.set", "workspace.simulation.set"} {
		entries, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err = json.Unmarshal(entries[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["x-command-contract"] == nil {
			t.Fatal("missing shared presentation contract")
		}
		if id == "workspace.panels.set" && inputProperty(schema, "parts_width")["minimum"] != float64(224) {
			t.Fatal("missing width bounds")
		}
		if id == "workspace.simulation.set" && len(inputProperty(schema, "tool")["enum"].([]any)) != 6 {
			t.Fatal("missing tool choices")
		}
		entries[0].Input[0] = '!'
		next, _ := detailedCapabilities(id)
		if next[0].Input[0] == '!' {
			t.Fatal("discovery aliases caller data")
		}
	}
}

func TestElectricalCommandsDoNotDependOnPanelVisibility(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	p := DefaultWorkspacePanels()
	p.PartsVisible = false
	p.InspectorVisible = false
	call(t, s, "workspace.panels.set", p, "hide")
	call(t, s, "workspace.simulation.set", SimulationToolParams{Tool: "tolerances"}, "task")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "edit")
	settled(t, s)
	duration := .01
	_, op := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, "hidden-capture")
	if op.Operation.Status != "completed" || s.Snapshot().WorkspacePanels != p || s.Snapshot().SimulationTool != "tolerances" {
		t.Fatal("hidden panels gated or changed electrical work")
	}
}
