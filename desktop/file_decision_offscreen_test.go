//go:build offscreen

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/dshills/vellum/app"

	"cogentcore.org/core/events"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedFileDecision(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		root := ui.Node{ID: "root", Kind: ui.Column}
		if s.pendingCommand != "" {
			prompt := s.filePrompt()
			row := ui.Node{ID: "decision-buttons", Kind: ui.Row}
			for _, id := range []string{"save-continue", "discard-continue", "cancel-file-action"} {
				node, ok := findNode(prompt, id)
				if !ok {
					t.Fatalf("missing decision control %s", id)
				}
				node.MinWidth, node.MinHeight = 145, 40
				row.Children = append(row.Children, node)
			}
			root.Children = append(root.Children, row)
			root.Focus = ui.FocusRequest{ID: "discard-continue", Sequence: 1}
		}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func TestMountedFileDecisionHeldDiscardPreservesAgentEdit(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	// Save decisions require an actual edit; shipped templates start clean.
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	uiAction(t, s, "project.clear", nil)
	m := mountedFileDecision(t, s)
	button := mountedControlElement(t, m, "discard-continue")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	m.frame()
	if mountedControlElement(t, m, "discard-continue").Incarnation != button.Incarnation {
		t.Fatal("fixture did not retain held Discard")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	after := s.service.Snapshot()
	if !reflect.DeepEqual(before, after) || s.pendingCommand != "project.clear" {
		t.Fatalf("held Discard lost agent work: components=%d→%d revision=%d→%d pending=%q error=%s", len(before.Components), len(after.Components), before.CommandRevision, after.CommandRevision, s.pendingCommand, s.errorMessage)
	}
}

func TestMountedFileDecisionAllButtonsRefuseHeldAgentEdit(t *testing.T) {
	for _, control := range []string{"save-continue", "discard-continue", "cancel-file-action"} {
		t.Run(control, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			uiAction(t, s, "example.reset", nil)
			// Save decisions require an actual edit; shipped templates start clean.
			agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
			uiAction(t, s, "project.clear", nil)
			calls := atomic.Int32{}
			s.chooseFile = func(context.Context, app.FileOptions) (string, error) { calls.Add(1); return "", app.ErrCanceled }
			m := mountedFileDecision(t, s)
			button := mountedControlElement(t, m, control)
			at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
			m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
			agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
			before := s.service.Snapshot()
			serial := s.pendingFileSerial
			m.frame()
			if mountedControlElement(t, m, control).Incarnation != button.Incarnation {
				t.Fatal("fixture did not retain button")
			}
			m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingCommand != "project.clear" || s.pendingFileSerial != serial || calls.Load() != 0 || s.fileBusy || !strings.Contains(s.errorMessage, "Schematic changed") {
				t.Fatal("held decision adopted an agent edit", control, s.errorMessage)
			}
			clickMountedControl(t, m, "cancel-file-action")
			if s.pendingCommand != "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("fresh Cancel failed after refusal")
			}
		})
	}
}

func TestMountedFileDecisionHeldDiscardCannotUseReopenedPrompt(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	// Save decisions require an actual edit; shipped templates start clean.
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	uiAction(t, s, "project.clear", nil)
	m := mountedFileDecision(t, s)
	button := mountedControlElement(t, m, "discard-continue")
	at := button.Bounds.Min.Add(button.Bounds.Size().Div(2))
	m.input(cogent.ManualInput{Type: events.MouseDown, Button: events.Left, Position: at})
	original := s.service.Snapshot()
	serial := s.pendingFileSerial
	uiAction(t, s, "cancel-file-action", nil)
	uiAction(t, s, "project.clear", nil)
	before := s.service.Snapshot()
	newSerial := s.pendingFileSerial
	if before.CommandRevision != original.CommandRevision || serial == newSerial {
		t.Fatal("fixture did not reopen same-revision decision")
	}
	m.frame()
	if mountedControlElement(t, m, "discard-continue").Incarnation != button.Incarnation {
		t.Fatal("fixture remounted button instead of retaining old action")
	}
	m.input(cogent.ManualInput{Type: events.MouseUp, Button: events.Left, Position: at})
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.pendingCommand != "project.clear" || s.pendingFileSerial != newSerial {
		t.Fatal("held Discard used reopened prompt")
	}
	clickMountedControl(t, m, "discard-continue")
	if len(s.service.Snapshot().Components) != 0 || s.pendingCommand != "" || s.errorMessage != "" {
		t.Fatal("fresh Discard failed", s.errorMessage)
	}
}

func TestMountedFileDecisionFreshSaveKeepsAgentWorkBeforeContinuing(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	uiAction(t, s, "example.reset", nil)
	// Save decisions require an actual edit; shipped templates start clean.
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "3k"})
	uiAction(t, s, "project.clear", nil)
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "2k"})
	before := s.service.Snapshot()
	path := filepath.Join(t.TempDir(), "kept-agent-edits.elab")
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
	m := mountedFileDecision(t, s)
	clickMountedControl(t, m, "save-continue")
	waitFileUI(t, s)
	m.settle()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document workbench.Document
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	after := s.service.Snapshot()
	if s.errorMessage != "" || len(after.Components) != 0 || s.pendingCommand != "" || !reflect.DeepEqual(document.Graph.Components, before.Components) || !reflect.DeepEqual(document.Graph.Wires, before.Wires) {
		t.Fatal("fresh Save did not preserve latest graph before continuing", s.errorMessage)
	}
}
