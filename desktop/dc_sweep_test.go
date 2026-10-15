package main

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/ui"
)

func awaitNativeSweep(t *testing.T, s *screen) workbench.Snapshot {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.app.Drain()
		state := s.service.Snapshot()
		if state.DCSweep != nil {
			for _, op := range state.DCSweep.Operations {
				if op.ID == state.DCSweep.Selected && op.Status == "completed" {
					return state
				}
				if op.Status == "failed" {
					t.Fatal(op.Error)
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("native sweep did not complete")
	return workbench.Snapshot{}
}
func TestNativeDCSweepMatchesAgentAndUsesParameterAxis(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Owned sweep test", Width: 1140, Height: 860, View: s.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	uiAction(t, s, "sweep-probe", &ui.ChangeEvent{SelectedID: "i:D1"})
	uiAction(t, s, "sweep-points", &ui.ChangeEvent{Text: "11"})
	uiAction(t, s, "sweep-range-stop", &ui.ChangeEvent{Text: "5 V"})
	cfg := s.service.Snapshot().DCSweep.Settings
	uiAction(t, s, "sweep-start", nil)
	state := awaitNativeSweep(t, s)
	if !reflect.DeepEqual(before.Result, state.Result) || before.RunID != state.RunID || before.ElectricalRevision != state.ElectricalRevision || state.DCSweep.Operations[0].Actor != "user" {
		t.Fatal("native sweep changed live physics")
	}
	agent := workbench.NewExample()
	defer agent.Close()
	agentAction(t, agent, "sweep.start", workbench.SweepStartParams{Settings: &cfg})
	deadline := time.Now().Add(3 * time.Second)
	for agent.Snapshot().DCSweep.Result == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !reflect.DeepEqual(state.DCSweep.Result, agent.Snapshot().DCSweep.Result) {
		t.Fatal("native/agent curve differs")
	}
	s.app.Do(func() {
		root := s.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		if _, ok := findAction(root, "sweep-plot"); !ok {
			t.Fatal("sweep graph missing")
		}
		if _, ok := findAction(root, "schematic"); ok {
			t.Fatal("hidden drawing still mounted")
		}
		mesh, err := s.sweepMesh(*s.sweepDrawing.View)
		if err != nil || mesh == nil {
			t.Fatal(err)
		}
		frame := s.sweepDrawing.frame.Load()
		if frame == nil || frame.min != 0 || frame.max != 5 {
			t.Fatal("DC graph used a time axis")
		}
		s.sweepDrawing.placeCursor(frame.left + (frame.right-frame.left)*.5)
	})
	state = s.service.Snapshot()
	if state.DCSweep.Cursor == nil || *state.DCSweep.Cursor != 5 || math.Abs(state.DCSweep.Result.Points[5].Input-2.5) > 1e-15 {
		t.Fatal("measured point cursor failed")
	}
	uiAction(t, s, "sweep-cursor-clear", nil)
	if s.service.Snapshot().DCSweep.Cursor != nil {
		t.Fatal("native Clear point failed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "sweep.csv")
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return path, nil }
	uiAction(t, s, "sweep-export-csv", nil)
	waitFileUI(t, s)
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		t.Fatal("native export", err)
	}
	uiAction(t, s, "sweep-back", nil)
	s.app.Do(func() {
		if _, ok := findAction(s.view(), "schematic"); !ok {
			t.Fatal("back failed")
		}
	})
}
func TestNativeSweepStalePointAndIncompleteSettings(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "sweep-start", nil)
	awaitNativeSweep(t, s)
	s.app.Do(func() { s.view(); _, _ = s.sweepMesh(*s.sweepDrawing.View) })
	old := s.sweepDrawing.frame.Load()
	agentAction(t, s.service, "sweep.trace.select", workbench.IDParams{ID: "v:V1"})
	s.app.Do(func() { s.sweepDrawing.placeCursor(old.left + (old.right-old.left)/2) })
	if s.service.Snapshot().DCSweep.Cursor != nil || s.errorMessage == "" {
		t.Fatal("stale plotted context silently mutated cursor")
	}
	// Configure while a wire is missing. Running still requires a complete circuit.
	agentAction(t, s.service, "object.delete", workbench.IDParams{ID: s.service.Snapshot().Wires[0].ID})
	s.app.Do(func() { s.errorMessage = ""; s.requestHelp = nil })
	uiAction(t, s, "sweep-range-stop", &ui.ChangeEvent{Text: "4 V"})
	s.app.Do(func() {
		n, ok := findAction(s.view(), "sweep-start")
		if !ok || !n.Disabled {
			t.Fatal("incomplete sweep start enabled")
		}
	})
}

func TestNativeSweepCanChooseMultipleReadings(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "sweep-probe", &ui.ChangeEvent{SelectedID: "i:D1"})
	uiAction(t, s, "sweep-probe", &ui.ChangeEvent{SelectedID: "v:D1"})
	if !reflect.DeepEqual(s.service.Snapshot().DCSweep.Settings.ProbeIDs, []string{"i:D1", "v:D1"}) {
		t.Fatal("native could not configure agent-equivalent multi-probe list")
	}
	uiAction(t, s, "sweep-probe-remove-i:D1", nil)
	if !reflect.DeepEqual(s.service.Snapshot().DCSweep.Settings.ProbeIDs, []string{"v:D1"}) {
		t.Fatal("probe removal discarded another reading")
	}
}

func TestNativePointerRoutesSweepAndPreservesDrawing(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "sweep-start", nil)
	awaitNativeSweep(t, s)
	s.app.Do(func() {
		s.view()
		_, _ = s.sweepMesh(*s.sweepDrawing.View)
		frame := s.sweepDrawing.frame.Load()
		pixel := frame.left + (frame.right-frame.left)/2
		p := s.sweepDrawing.View.World(canvas.Point{X: pixel, Y: 100})
		before := s.service.Snapshot()
		s.routeNativePointer("sweep", 0, workbench.Point{X: p.X, Y: p.Y}, workbench.Point{})
		after := s.service.Snapshot()
		if after.DCSweep.Cursor == nil || *after.DCSweep.Cursor != 25 || !reflect.DeepEqual(before.Components, after.Components) || before.RunID != after.RunID {
			t.Fatal("native bridge did not select recorded point without changing source")
		}
		s.invoke("sweep.visibility.set", workbench.SweepVisibilityParams{Visible: false})
		hidden := s.service.Snapshot().CommandRevision
		s.routeNativePointer("sweep", 0, workbench.Point{X: p.X, Y: p.Y}, workbench.Point{})
		if s.service.Snapshot().CommandRevision != hidden {
			t.Fatal("retired sweep pane accepted native pointer")
		}
	})
}

func TestNativeSweepImportOnBlankMatchesAgent(t *testing.T) {
	source := workbench.NewExample()
	defer source.Close()
	cfg := workbench.SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 11, ProbeIDs: []string{"i:D1", "v:D1"}}
	agentAction(t, source, "sweep.start", workbench.SweepStartParams{Settings: &cfg})
	deadline := time.Now().Add(3 * time.Second)
	for source.Snapshot().DCSweep.Result == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	curve := source.Snapshot().DCSweep
	if curve.Result == nil {
		t.Fatal("fixture sweep not ready")
	}
	path := filepath.Join(t.TempDir(), "curve.json")
	agentAction(t, source, "sweep.export", workbench.VerificationExportParams{ID: curve.Selected, Path: path, Format: "json"})
	user, cleanup := testScreen(t)
	defer cleanup()
	if err := user.app.Open(app.Window{ID: "main", Title: "Owned sweep import", Width: 1260, Height: 900, View: user.view}); err != nil {
		t.Fatal(err)
	}
	uiAction(t, user, "sweep-open", nil)
	user.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) {
		if options.Mode != app.OpenFile || len(options.Extensions) != 1 || options.Extensions[0] != "json" {
			t.Errorf("wrong sweep picker %+v", options)
		}
		return path, nil
	}
	before := user.service.Snapshot()
	uiAction(t, user, "sweep-import", nil)
	waitFileUI(t, user)
	if user.errorMessage != "" {
		t.Fatal(user.errorMessage)
	}
	state := user.service.Snapshot()
	if len(state.Components) != 0 || state.RunID != before.RunID || !state.DCSweep.Visible || !reflect.DeepEqual(state.DCSweep.Result, curve.Result) || state.ProjectWork.Command != "sweep.import" || state.ProjectWork.Actor != "user" || state.DCSweep.Operations[0].SourceUnchanged {
		t.Fatal("native import replaces source or loses capture/provenance")
	}
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "sweep.import", workbench.RecordingImportParams{Path: path})
	if !reflect.DeepEqual(state.DCSweep.Result, agent.Snapshot().DCSweep.Result) {
		t.Fatal("native/agent imported curve differs")
	}
	user.app.Do(func() {
		user.view()
		_, err := user.sweepMesh(*user.sweepDrawing.View)
		if err != nil {
			t.Fatal(err)
		}
		frame := user.sweepDrawing.frame.Load()
		user.sweepDrawing.placeCursor(frame.left + (frame.right-frame.left)/2)
	})
	if *user.service.Snapshot().DCSweep.Cursor != 5 {
		t.Fatal("import cursor does not use captured axis")
	}
	five := 5
	agentAction(t, agent, "sweep.cursor.set", workbench.SweepCursorParams{Index: &five})
	uiAction(t, user, "sweep-cursor-clear", nil)
	agentAction(t, agent, "sweep.cursor.set", workbench.SweepCursorParams{})
	if user.service.Snapshot().DCSweep.Cursor != nil || agent.Snapshot().DCSweep.Cursor != nil {
		t.Fatal("native/agent clear differs")
	}
	uiAction(t, user, "sweep-back", nil)
	if user.service.Snapshot().DCSweep.Visible || len(user.service.Snapshot().Components) != 0 {
		t.Fatal("import back loses live circuit")
	}
}
