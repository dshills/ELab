package main

import (
	"math"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeDCInspectionAndPinControlsAreAvailable(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	state := s.service.Snapshot()
	if !state.MeasurementInspection.Available || len(state.MeasurementInspection.Readings) != 4 || state.MeasurementInspection.ResultKind != "dc" {
		t.Fatal("selected DC part has no voltage/current inspector")
	}
	uiAction(t, s, "measurement-pin-i:R1", nil)
	if pin := s.service.Snapshot().PinnedReadings[0]; pin.Value == nil || math.Abs(*pin.Value-before.Result.Current) > 1e-12 {
		t.Fatal("DC pin control disagrees with accepted solver")
	}
	uiAction(t, s, "pin-reading-probe", &ui.ChangeEvent{SelectedID: "v:D1"})
	if len(s.service.Snapshot().PinnedReadings) != 2 {
		t.Fatal("general DC pin selector is inaccessible")
	}
	uiAction(t, s, "measurement-source", &ui.ChangeEvent{SelectedID: "probe|i:D1"})
	if got := s.service.Snapshot().MeasurementInspection; got.Target.Kind != "probe" || got.Readings[0].Probe.Reference != "D1.cathode" {
		t.Fatal("named current inspection lost reference")
	}
}

func TestNativeTerminalClickAndNetSelectorMatchAgentCommands(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	graph := workbench.Graph{Components: before.Components, Wires: before.Wires}
	part, _ := graph.Component("C1")
	at, _ := workbench.TerminalPosition(part, "1")
	s.app.Do(func() {
		s.drawing.routePointer(0, at, workbench.Point{})
		s.drawing.routePointer(1, at, workbench.Point{X: 15, Y: 10})
		s.drawing.routePointer(2, at, workbench.Point{})
	})
	clicked := s.service.Snapshot()
	if clicked.MeasurementInspection.Target == nil || clicked.MeasurementInspection.Target.Kind != "terminal" || clicked.MeasurementInspection.Target.ID != "C1.1" || clicked.Selected != before.Selected || clicked.RunID != before.RunID || !reflect.DeepEqual(clicked.Transient, before.Transient) || clicked.Document != before.Document || clicked.View != before.View || s.drawing.dragID != "" {
		t.Fatal("terminal gesture initiates component drag or changes experiment")
	}
	agentAction(t, s.service, "measurement.select", workbench.MeasurementSelectParams{Kind: "terminal", ID: "C1.1"})
	if !reflect.DeepEqual(clicked.MeasurementInspection, s.service.Snapshot().MeasurementInspection) {
		t.Fatal("native terminal click differs from shared command")
	}
	uiAction(t, s, "measurement-source", &ui.ChangeEvent{SelectedID: "net|net:C1.1"})
	net := s.service.Snapshot().MeasurementInspection
	if net.Target.Anchor != "C1.1" || len(net.Wires) != 1 || net.Wires[0] != "W2" {
		t.Fatal("native net selector loses association")
	}
	uiAction(t, s, "measurement-plot-v:C1.1", nil)
	if !containsID(s.service.Snapshot().Scope.Selected, "v:C1.1") {
		t.Fatal("inspect Plot does not use live probe selection")
	}
	uiAction(t, s, "measurement-pin-v:C1.1", nil)
	if s.service.Snapshot().PinnedReadings[0].Definition.ID != "v:C1.1" {
		t.Fatal("inspect Pin does not use shared pin state")
	}
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "W1"})
	if got := s.service.Snapshot().MeasurementInspection; got.Target.Kind != "object" || got.Target.ID != "W1" || len(got.Readings) != 1 || got.Readings[0].Probe.Unit != "V" {
		t.Fatal("wire selection does not display net voltage")
	}
}

func containsID(ids []string, id string) bool {
	for _, old := range ids {
		if old == id {
			return true
		}
	}
	return false
}
