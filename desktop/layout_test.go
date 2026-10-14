package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestExampleMenuUsesSharedReplacementGuard(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.rc"})
	if s.service.Snapshot().Analysis != "RC transient" {
		t.Fatal("menu did not use shared RC command")
	}
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "20k"})
	uiAction(t, s, "apply-value", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "examples-menu", &ui.ChangeEvent{SelectedID: "example.flasher"})
	if s.pendingCommand != "example.flasher" || s.service.Snapshot().Analysis != before.Analysis {
		t.Fatal("menu bypasses unsaved changes confirmation")
	}
	uiAction(t, s, "cancel-file-action", nil)
	if s.service.Snapshot().Circuit.Resistance != 20000 {
		t.Fatal("cancel lost edits")
	}
}

func TestMenuRefusesUnknownAndDisabledActions(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	before := s.service.Snapshot().CommandRevision
	menu := s.commandMenu("test-menu", "Actions", [][2]string{{"edit.undo", "Undo"}})
	menu.Change(ui.ChangeEvent{SelectedID: "edit.undo"})
	menu.Change(ui.ChangeEvent{SelectedID: "project.clear"})
	if s.service.Snapshot().CommandRevision != before {
		t.Fatal("unlisted/disabled menu choice changed state")
	}
	// Agent history remains the authoritative command path.
	if s.invoke("component.place", workbench.PlaceParams{Kind: "resistor", Position: workbench.Point{X: 10}}).Error != nil {
		t.Fatal("placement failed")
	}
	fresh := s.commandMenu("test-menu", "Actions", [][2]string{{"edit.undo", "Undo"}})
	fresh.Change(ui.ChangeEvent{SelectedID: "edit.undo"})
	if len(s.service.Snapshot().Components) != 0 {
		t.Fatal("fresh menu did not undo")
	}
}
