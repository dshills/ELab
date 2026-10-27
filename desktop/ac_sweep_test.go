package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeACSweepGainPhaseCursorMatchesAgent(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example.ac-rc"})
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "ac-rc"})
	uiAction(t, u, "simulation.pause", nil)
	agentAction(t, a, "simulation.pause", struct{}{})
	u.service.Wait(t.Context())
	a.Wait(t.Context())
	uiAction(t, u, "sweep-start", nil)
	agentAction(t, a, "sweep.start", workbench.SweepStartParams{})
	native := awaitNativeSweep(t, u)
	deadline := time.Now().Add(5 * time.Second)
	for a.Snapshot().DCSweep.Result == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !reflect.DeepEqual(native.DCSweep.Result, a.Snapshot().DCSweep.Result) {
		t.Fatal("AC native/agent response differs")
	}
	uiAction(t, u, "sweep-trace", &ui.ChangeEvent{SelectedID: "phase:v:C1"})
	agentAction(t, a, "sweep.trace.select", workbench.IDParams{ID: "phase:v:C1"})
	index := 80
	uiAction(t, u, "sweep-cursor-clear", nil)
	agentAction(t, a, "sweep.cursor.set", workbench.SweepCursorParams{})
	agentAction(t, u.service, "sweep.cursor.set", workbench.SweepCursorParams{Index: &index})
	agentAction(t, a, "sweep.cursor.set", workbench.SweepCursorParams{Index: &index})
	u.app.Do(func() {
		u.view()
		if _, err := u.sweepMesh(*u.sweepDrawing.View); err != nil {
			t.Fatal(err)
		}
	})
	if u.service.Snapshot().DCSweep.TraceID != a.Snapshot().DCSweep.TraceID || *u.service.Snapshot().DCSweep.Cursor != index {
		t.Fatal("phase/point selection differs")
	}
	for _, id := range []string{"sweep-ac-peak-0", "sweep-ac-cutoff-0-0", "sweep-reading-gain:v:C1", "sweep-reading-phase:v:C1"} {
		if _, ok := findNode(u.view(), id); !ok {
			t.Fatal("missing AC readout", id)
		}
	}
}

func TestNativeBodeAxisLabelsStayInsideNarrowCanvas(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, u, "examples-menu", &ui.ChangeEvent{SelectedID: "example.ac-rc"})
	uiAction(t, u, "simulation.pause", nil)
	u.service.Wait(t.Context())
	uiAction(t, u, "sweep-start", nil)
	awaitNativeSweep(t, u)
	for _, width := range []float64{240, 320, 800} {
		u.app.Do(func() {
			u.view()
			v := *u.sweepDrawing.View
			v.Width = width
			mesh, err := u.sweepMesh(v)
			if err != nil {
				t.Fatal(err)
			}
			for _, vertex := range mesh.Vertices() {
				point := v.Screen(vertex.Position)
				if point.X < 0 || point.X > width || point.Y < 0 || point.Y > v.Height {
					t.Fatalf("axis/trace clipped at width %.0f: %+v", width, point)
				}
			}
		})
	}
}

func TestNativeDiscreteAmplifierBodeMatchesAgentWithoutChangingLive(t *testing.T) {
	u, cleanup := testScreen(t)
	defer cleanup()
	a := workbench.New()
	defer a.Close()
	uiAction(t, u, "examples-menu", nil)
	uiAction(t, u, "example-search", &ui.ChangeEvent{Text: "class AB"})
	grid := browserGridNode(t, u)
	u.app.Do(func() {
		grid.SelectionChange(ui.ListSelectionEvent{TargetID: "discrete-class-ab", IDs: []string{"discrete-class-ab"}})
	})
	uiAction(t, u, ui.SheetConfirmID(exampleBrowserID), nil)
	agentAction(t, a, "example.load", workbench.ExampleLoadParams{ID: "discrete-class-ab"})
	for _, cmd := range []string{"simulation.pause", "simulation.restart"} {
		uiAction(t, u, cmd, nil)
		agentAction(t, a, cmd, struct{}{})
	}
	before := awaitInductorCircuit(t, u.service)
	awaitInductorCircuit(t, a)
	if before.DCSweep.Settings.Parameter != "frequency" || before.DCSweep.Settings.ComponentID != "VIN" {
		t.Fatal("amplifier needs useful default Bode settings")
	}
	uiAction(t, u, "sweep-start", nil)
	agentAction(t, a, "sweep.start", workbench.SweepStartParams{})
	native := awaitNativeSweep(t, u)
	deadline := time.Now().Add(5 * time.Second)
	agentState := a.Snapshot()
	for agentState.DCSweep.Result == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		agentState = a.Snapshot()
	}
	if agentState.DCSweep.Result == nil {
		t.Fatal("agent amplifier AC did not complete before deadline", agentState.DCSweep.Operations)
	}
	if native.DCSweep.Result == nil || !reflect.DeepEqual(native.DCSweep.Result, agentState.DCSweep.Result) || !reflect.DeepEqual(before.Mixed, native.Mixed) {
		t.Fatal("native/agent transistor AC differs or changed live state")
	}
	var meshErr error
	var meshOK bool
	u.app.Do(func() {
		u.view()
		if u.sweepDrawing == nil || u.sweepDrawing.View == nil {
			return
		}
		mesh, err := u.sweepMesh(*u.sweepDrawing.View)
		meshErr, meshOK = err, mesh != nil && err == nil
	})
	if !meshOK {
		t.Fatal("Bode canvas unavailable", meshErr)
	}
}
