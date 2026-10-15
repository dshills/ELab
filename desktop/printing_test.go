package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativePrintControlsUseSharedSettingsAndRefuseFractionalCopies(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned print controls", Width: 1040, Height: 760, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "print.destination.set", workbench.PrintDestinationParams{Copies: 1})
	uiAction(t, s, "print-preview", nil)
	waitPreviewUI(t, s)
	uiAction(t, s, "printer-copies", &ui.ChangeEvent{Number: 2})
	before := s.service.Snapshot()
	if before.Printing.Copies != 2 || before.LastActor != "user" {
		t.Fatal("native copies did not share command")
	}
	s.app.Do(func() { node, _ := findAction(s.view(), "printer-copies"); node.Change(ui.ChangeEvent{Number: 1.5}) })
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage == "" {
		t.Fatal("fractional copies silently truncated")
	}
	s.app.Do(func() {
		root := s.view()
		submit, ok := findAction(root, "printer-submit")
		if !ok || !submit.Disabled {
			t.Fatal("print enabled without destination")
		}
		if _, ok := findAction(root, "printers-refresh"); !ok {
			t.Fatal("printer discovery control missing")
		}
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
	})
	uiAction(t, s, "preview-close", nil)
}
