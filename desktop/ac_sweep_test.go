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
