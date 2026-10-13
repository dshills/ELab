package main

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecordingPickerAndControlsMatchAgentCommands(t *testing.T) {
	source := workbench.New()
	defer source.Close()
	agentAction(t, source, "example.rc", workbench.ReplaceParams{})
	duration := .02
	agentAction(t, source, "simulation.verify", workbench.VerifyParams{Duration: &duration})
	operation := completedVerification(t, source, source.Snapshot().LastOperation)
	path := filepath.Join(t.TempDir(), "record.json")
	agentAction(t, source, "verification.export", workbench.VerificationExportParams{ID: operation.Operation.ID, Path: path, Format: "json"})
	user, cleanup := testScreen(t)
	defer cleanup()
	if err := user.app.Open(app.Window{ID: "main", Title: "Recording import", Width: 1260, Height: 900, View: user.view}); err != nil {
		t.Fatal(err)
	}
	agent := workbench.New()
	defer agent.Close()
	user.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
		if options.Mode != app.OpenFile || len(options.Extensions) != 1 || options.Extensions[0] != "json" {
			t.Errorf("wrong recording picker %+v", options)
		}
		return path, nil
	}
	before := user.service.Snapshot()
	uiAction(t, user, "recording-import", nil)
	waitFileUI(t, user)
	if user.errorMessage != "" {
		t.Fatal(user.errorMessage)
	}
	agentAction(t, agent, "verification.import", workbench.RecordingImportParams{Path: path})
	if user.service.Snapshot().Document != before.Document || len(user.service.Snapshot().Components) != 0 {
		t.Fatal("native import changed blank live schematic")
	}
	uiAction(t, user, "recorded-index", &ui.ChangeEvent{Number: 4})
	index := 4
	agentAction(t, agent, "recording.seek", workbench.RecordingSeekParams{Index: &index})
	uiAction(t, user, "recorded-next", nil)
	agentAction(t, agent, "recording.step", workbench.RecordingStepParams{Direction: "next"})
	uiAction(t, user, "recorded-cursor-a", &ui.ChangeEvent{Number: 5})
	agentAction(t, agent, "recording.cursor.set", workbench.CursorParams{Cursor: "a", Time: .005})
	uiAction(t, user, "recorded-cursor-b", &ui.ChangeEvent{Number: 15})
	agentAction(t, agent, "recording.cursor.set", workbench.CursorParams{Cursor: "b", Time: .015})
	uiAction(t, user, "recorded-window-start", &ui.ChangeEvent{Number: 5})
	agentAction(t, agent, "recording.window.set", workbench.RecordingWindowParams{Start: .005, End: .02})
	uiAction(t, user, "recorded-probe-0", &ui.ChangeEvent{SelectedID: "__none"})
	ids := append([]string{}, agent.Snapshot().Recording.Scope.Selected[1:]...)
	agentAction(t, agent, "recording.probes.set", workbench.ProbeParams{IDs: ids})
	uiAction(t, user, "recorded-speed", &ui.ChangeEvent{Number: 1})
	agentAction(t, agent, "recording.speed.set", workbench.RecordingPlayParams{Speed: 1})
	u, a := user.service.Snapshot().Recording, agent.Snapshot().Recording
	if !reflect.DeepEqual(u, a) {
		t.Fatal("native and agent recorded state differs")
	}
	if node, ok := findNode(user.view(), "recorded-source-R1"); !ok || node.Text == "" {
		t.Fatal("recorded source inspection missing")
	}
	if _, ok := findNode(user.view(), "recorded-scope-title"); !ok {
		t.Fatal("blank live circuit hid recorded plot")
	}
	mesh, err := user.scopeMesh(*user.scope.View)
	if err != nil || mesh == nil {
		t.Fatal("recorded plot did not render")
	}
	recordedCanvas, ok := findNode(user.view(), "rc-scope")
	if !ok || recordedCanvas.Custom != user.scope {
		t.Fatal("recorded canvas lost stable adapter identity")
	}
	uiAction(t, user, "recorded-back-live", nil)
	agentAction(t, agent, "recording.view.set", workbench.RecordingViewParams{Mode: "live"})
	if user.service.Snapshot().Recording.Viewing || agent.Snapshot().Recording.Viewing {
		t.Fatal("live view switch failed")
	}
}
