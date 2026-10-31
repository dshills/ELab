package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/internal/testutil"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNextWorkflowControlsUseAgentCommands(t *testing.T) {
	if testutil.RunAtProductionSpeed(t) {
		return
	}
	user, close := testScreen(t)
	defer close()
	if err := user.app.Open(app.Window{ID: "main", Title: "Workflow test", Width: 1260, Height: 900, View: user.view}); err != nil {
		t.Fatal(err)
	}
	agent := workbench.New()
	defer agent.Close()
	uiAction(t, user, "part-search", &ui.ChangeEvent{Text: "npn"})
	agentAction(t, agent, "parts.search.set", workbench.SearchParams{Query: "npn"})
	if len(user.service.Snapshot().MatchingParts) != 1 {
		t.Fatal("search did not filter UI")
	}
	uiAction(t, user, "part-search", &ui.ChangeEvent{Text: ""})
	agentAction(t, agent, "parts.search.set", workbench.SearchParams{Query: ""})
	uiAction(t, user, "example.flasher", nil)
	agentAction(t, agent, "example.flasher", workbench.ReplaceParams{})
	for _, command := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, user, command, nil)
		agentAction(t, agent, command, struct{}{})
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "C1"})
	agentAction(t, agent, "component.select", workbench.IDParams{ID: "C1"})
	uiAction(t, user, "duplicate-part", nil)
	agentAction(t, agent, "component.duplicate", workbench.DuplicateParams{ID: "C1"})
	if !reflect.DeepEqual(user.service.Snapshot().Components, agent.Snapshot().Components) || user.service.Snapshot().UndoName != "Duplicate component" {
		t.Fatal("duplicate command differs")
	}
	uiAction(t, user, "edit.undo", nil)
	agentAction(t, agent, "edit.undo", struct{}{})
	uiAction(t, user, "cap-pair-20uF", nil)
	edits := []workbench.BatchEdit{}
	for _, id := range []string{"C1", "C2"} {
		p, _ := json.Marshal(workbench.ValueParams{ID: id, Value: "20uF"})
		edits = append(edits, workbench.BatchEdit{Command: "component.set", Params: p})
	}
	agentAction(t, agent, "edit.batch", workbench.BatchParams{Name: "Set both capacitors to 20uF", Edits: edits})
	if !reflect.DeepEqual(user.service.Snapshot().Components, agent.Snapshot().Components) || user.service.Snapshot().UndoName != "Set both capacitors to 20uF" {
		t.Fatal("atomic UI preset differs")
	}
	uiAction(t, user, "edit.undo", nil)
	agentAction(t, agent, "edit.undo", struct{}{})
	uiAction(t, user, "simulation.restart", nil)
	agentAction(t, agent, "simulation.restart", struct{}{})
	for i := 0; i < 70; i++ {
		uiAction(t, user, "transient-step", nil)
		agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .005})
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "C2"})
	agentAction(t, agent, "component.select", workbench.IDParams{ID: "C2"})
	// Plot preserves all requested traces. Make the replacement explicit in
	// both clients rather than relying on the former silent last-slot eviction.
	ids := append([]string{}, agent.Snapshot().Scope.Selected...)
	remove := ids[len(ids)-1]
	uiAction(t, user, fmt.Sprintf("probe-%d", len(ids)-1), &ui.ChangeEvent{SelectedID: "__none"})
	agentAction(t, agent, "scope.probe.remove", workbench.IDParams{ID: remove})
	uiAction(t, user, "plot-v:C2", nil)
	agentAction(t, agent, "scope.probe.plot", workbench.IDParams{ID: "v:C2"})
	uiAction(t, user, "scope-latest-window", nil)
	a := agent.Snapshot()
	agentAction(t, agent, "scope.window.set", workbench.ScopeWindow{Mode: "fixed", Start: a.Transient.Sample.Time - .2, End: a.Transient.Sample.Time})
	if !reflect.DeepEqual(user.service.Snapshot().Scope, agent.Snapshot().Scope) {
		t.Fatal("plot/interval/statistics UI differs")
	}
	// Native picker adapter passes the same export command; data must match.
	dir := t.TempDir()
	for _, kind := range []string{"svg", "csv"} {
		path := filepath.Join(dir, "ui."+kind)
		other := filepath.Join(dir, "agent."+kind)
		command := "export.schematic.svg"
		control := "export-svg"
		if kind == "csv" {
			command = "export.scope.csv"
			control = "export-csv"
		}
		user.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
			if options.Mode != app.SaveFile || len(options.Extensions) != 1 || options.Extensions[0] != kind {
				t.Errorf("wrong export picker %+v", options)
			}
			return path, nil
		}
		uiAction(t, user, control, nil)
		waitFileUI(t, user)
		if user.errorMessage != "" {
			t.Fatal(user.errorMessage)
		}
		work := user.service.Snapshot().ProjectWork
		if work == nil || work.Status != "completed" || work.Command != command || work.Actor != "user" || work.ArtifactSHA256 == "" {
			t.Fatal("native export lacks its shared completed job identity")
		}
		agentAction(t, agent, command, workbench.SaveParams{Path: other, Overwrite: true})
		left, _ := os.ReadFile(path)
		right, _ := os.ReadFile(other)
		if len(left) == 0 || string(left) != string(right) {
			t.Fatalf("%s export native/agent content differs", kind)
		}
	}
	uiAction(t, user, "scope-live-window", nil)
	agentAction(t, agent, "scope.window.set", workbench.ScopeWindow{Mode: "live"})
	if !reflect.DeepEqual(user.service.Snapshot().Scope, agent.Snapshot().Scope) {
		t.Fatal("live window differs")
	}
}
