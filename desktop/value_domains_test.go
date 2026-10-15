package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/ui"
)

func TestNativeValueUnitsBoundsAndRefusalShareCommands(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned engineering values", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "V1"})
	s.app.Do(func() {
		root := s.view()
		rangeNode, ok := findNode(root, "value-domain-range")
		syntax, _ := findNode(root, "value-domain-syntax")
		if !ok || rangeNode.Text != "0–12 V" || !strings.Contains(syntax.Text, "mV") {
			t.Fatal("source editing limits not visible")
		}
	})
	before := s.service.Snapshot()
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "500 mV"})
	if !reflect.DeepEqual(before.Components, s.service.Snapshot().Components) || before.Document != s.service.Snapshot().Document {
		t.Fatal("draft changed electrical values")
	}
	uiAction(t, s, "apply-value", nil)
	state := s.service.Snapshot()
	component, _ := (workbench.Graph{Components: state.Components}).Component("V1")
	if component.Value != .5 || state.LastActor != "user" {
		t.Fatal("native millivolts did not use shared command")
	}
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "1e-999 V"})
	before = s.service.Snapshot()
	s.app.Do(func() {
		node, ok := findAction(s.view(), "apply-value")
		if !ok {
			t.Fatal("apply value missing")
		}
		node.Activate()
	})
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("native underflow was accepted or changed state")
	}
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "750 mV"})
	uiAction(t, s, "apply-value", nil)
	component, _ = (workbench.Graph{Components: s.service.Snapshot().Components}).Component("V1")
	if component.Value != .75 {
		t.Fatal("failed value prevented correction")
	}
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "C1"})
	s.app.Do(func() {
		root := s.view()
		rangeNode, ok := findNode(root, "value-domain-range")
		if !ok || rangeNode.Text != "1 pF–1 F" {
			t.Fatal("capacitance limits absent")
		}
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
	})
}
