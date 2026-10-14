package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
	"github.com/dshills/vellum/canvas"
	"github.com/dshills/vellum/command"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func testScreen(t *testing.T) (*screen, func()) {
	t.Helper()
	a, err := app.New(app.Config{})
	if err != nil {
		t.Fatal(err)
	}
	service := workbench.New()
	screen := &screen{app: a, service: service, commands: command.New()}
	view := canvas.Viewport{Width: 800, Height: 500, Zoom: 1}
	screen.drawing = &commandCanvas{MeshCanvas: &cogent.MeshCanvas{Owner: a, Window: "main", View: &view}, screen: screen}
	scopeView := canvas.Viewport{Width: 800, Height: 150, Zoom: 1}
	screen.scope = &scopeCanvas{MeshCanvas: &cogent.MeshCanvas{Owner: a, Window: "main", View: &scopeView, Prepare: screen.scopeMesh}, screen: screen}
	screen.registerCommands()
	return screen, func() { service.Close(); a.Close() }
}
func uiAction(t *testing.T, s *screen, id string, event *ui.ChangeEvent) {
	t.Helper()
	s.app.Do(func() {
		root := s.view()
		if err := ui.Validate(root); err != nil {
			t.Fatal(err)
		}
		node, ok := findNode(root, id)
		if !ok {
			t.Fatalf("missing UI control %s", id)
		}
		if node.Disabled {
			t.Fatalf("disabled UI control %s", id)
		}
		if event == nil {
			node.Activate()
		} else {
			node.Change(*event)
		}
	})
	if s.errorMessage != "" {
		t.Fatalf("%s: %s", id, s.errorMessage)
	}
	if id == "transient-step" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		state, err := s.service.Wait(ctx)
		if err != nil || state.StepWork == nil || state.StepWork.Status != "completed" {
			t.Fatalf("native step: %+v, %v", state.StepWork, err)
		}
	}
}
func agentAction(t *testing.T, service *workbench.Service, id string, p any) {
	t.Helper()
	raw, _ := json.Marshal(p)
	rev := service.Snapshot().Revision
	request := workbench.Request{Version: 1, ID: id, Command: id, Params: raw, ExpectedRevision: &rev, Key: id + fmt.Sprintf("-%d-%x", service.Snapshot().CommandRevision, sha256.Sum256(raw))}
	response := service.Dispatch(context.Background(), request, "agent")
	if response.Error != nil && response.Error.Code == "conflict" {
		rev = response.Revision
		response = service.Dispatch(context.Background(), request, "agent")
	}
	if response.Error != nil {
		t.Fatal(response.Error)
	}
}
func TestBuildFromBlankThroughUIAndAgent(t *testing.T) {
	user, close := testScreen(t)
	defer close()
	agent := workbench.New()
	defer agent.Close()
	for _, p := range []struct {
		kind string
		x, y float32
	}{{"dc-source", -200, 0}, {"resistor", 0, -110}, {"led", 200, 0}, {"ground", 0, 110}} {
		uiAction(t, user, "part-"+p.kind, nil)
		uiAction(t, user, "position-x", &ui.ChangeEvent{Number: p.x})
		uiAction(t, user, "position-y", &ui.ChangeEvent{Number: p.y})
		uiAction(t, user, "place-at-position", nil)
		agentAction(t, agent, "component.place", workbench.PlaceParams{Kind: p.kind, Position: workbench.Point{X: float64(p.x), Y: float64(p.y)}})
	}
	uiAction(t, user, "tool-wire", nil)
	for _, pair := range [][2]string{{"V1.positive", "R1.1"}, {"R1.2", "D1.anode"}, {"D1.cathode", "GND1.reference"}, {"GND1.reference", "V1.negative"}} {
		uiAction(t, user, "wire-from", &ui.ChangeEvent{SelectedID: pair[0]})
		uiAction(t, user, "wire-to", &ui.ChangeEvent{SelectedID: pair[1]})
		uiAction(t, user, "connect-terminals", nil)
		agentAction(t, agent, "wire.connect", workbench.ConnectParams{From: pair[0], To: pair[1]})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	u, err := user.service.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, err := agent.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(u.Components, a.Components) || !reflect.DeepEqual(u.Wires, a.Wires) || u.Result == nil || a.Result == nil || u.Result.Current != a.Result.Current {
		t.Fatal("UI and agent construction diverged")
	}
	uiAction(t, user, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, user, "value-entry", &ui.ChangeEvent{Text: "2.2k ohm"})
	uiAction(t, user, "apply-value", nil)
	agentAction(t, agent, "component.set", workbench.ValueParams{ID: "R1", Value: "2.2k ohm"})
	u, err = user.service.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, err = agent.Wait(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if u.Circuit.Resistance != 2200 || u.Result.Current != a.Result.Current {
		t.Fatal("exact value edits diverged")
	}
	uiAction(t, user, "edit.undo", nil)
	u, _ = user.service.Wait(ctx)
	if u.Circuit.Resistance != 1000 {
		t.Fatal("UI undo did not use shared history")
	}
}
func TestCanvasPlacementAndTerminalGestureUseSameCommands(t *testing.T) {
	s, close := testScreen(t)
	defer close()
	uiAction(t, s, "part-dc-source", nil)
	s.app.Do(func() { s.canvasClick(workbench.Point{X: -198, Y: 3}) })
	if part := s.service.Snapshot().Components[0]; part.Position != (workbench.Point{X: -200, Y: 0}) {
		t.Fatal("canvas placement not snapped")
	}
	uiAction(t, s, "part-resistor", nil)
	s.app.Do(func() { s.canvasClick(workbench.Point{X: 0, Y: -110}) })
	uiAction(t, s, "tool-wire", nil)
	s.app.Do(func() {
		s.canvasClick(workbench.Point{X: -200, Y: -34})
		s.canvasClick(workbench.Point{X: -55, Y: -110})
	})
	state := s.service.Snapshot()
	if len(state.Wires) != 1 || state.Wires[0].From != "V1.positive" || state.Wires[0].To != "R1.1" || state.Drawing.WireStart != "" {
		t.Fatal("terminal gesture did not commit wire")
	}
	s.app.Do(func() { s.canvasClick(workbench.Point{X: 55, Y: -110}) })
	uiAction(t, s, "drawing.cancel", nil)
	if s.service.Snapshot().Drawing.WireStart != "" {
		t.Fatal("wire cancel failed")
	}
}
func waitFileUI(t *testing.T, s *screen) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.app.Drain()
		var busy bool
		s.app.Do(func() { busy = s.fileBusy })
		if !busy {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("file operation did not complete")
}
func TestFileControlsUseSharedPersistenceAndProtectUnsavedChanges(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := s.app.Open(app.Window{ID: "main", Title: "Test ELab", Width: 1140, Height: 760, View: s.view}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "UI LED.elab")
	s.chooseFile = func(_ context.Context, options app.FileOptions) (string, error) { return path, nil }
	uiAction(t, s, "example.reset", nil)
	uiAction(t, s, "file.save", nil)
	waitFileUI(t, s)
	if s.errorMessage != "" || s.service.Snapshot().Document.Path != path || s.service.Snapshot().Document.Dirty {
		t.Fatalf("UI save failed: %s", s.errorMessage)
	}
	saved := s.service.Snapshot()
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "2k"})
	uiAction(t, s, "apply-value", nil)
	uiAction(t, s, "file.open", nil)
	waitFileUI(t, s)
	if s.pendingCommand != "project.load" || s.service.Snapshot().Circuit.Resistance != 2000 {
		t.Fatal("open silently discarded changes")
	}
	uiAction(t, s, "cancel-file-action", nil)
	if s.pendingCommand != "" {
		t.Fatal("cancel left destructive action pending")
	}
	uiAction(t, s, "file.open", nil)
	waitFileUI(t, s)
	uiAction(t, s, "discard-continue", nil)
	waitFileUI(t, s)
	if s.service.Snapshot().Circuit.Resistance != saved.Circuit.Resistance || s.service.Snapshot().Document.Dirty {
		t.Fatal("open did not restore saved file")
	}
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "2.2k"})
	uiAction(t, s, "apply-value", nil)
	uiAction(t, s, "project.clear", nil)
	if s.pendingCommand != "project.clear" {
		t.Fatal("new did not protect unsaved schematic")
	}
	uiAction(t, s, "save-continue", nil)
	waitFileUI(t, s)
	if len(s.service.Snapshot().Components) != 0 || s.service.Snapshot().Document.Path != "" || s.pendingCommand != "" {
		t.Fatal("save and new did not complete")
	}
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "project.load", workbench.LoadParams{Path: path})
	if agent.Snapshot().Circuit.Resistance != 2200 {
		t.Fatal("shared agent did not read the UI-saved values")
	}
	// Canceling an OS picker leaves the current project and revision intact.
	s.chooseFile = func(context.Context, app.FileOptions) (string, error) { return "", app.ErrCanceled }
	before := s.service.Snapshot()
	uiAction(t, s, "file.open", nil)
	waitFileUI(t, s)
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.errorMessage != "" {
		t.Fatal("cancelled picker changed document")
	}
}

func TestReplacingDocumentCancelsOldPointerGesture(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.reset", nil)
	s.app.Do(func() { s.drawing.routePointer(0, workbench.Point{X: 0, Y: -110}, workbench.Point{}) })
	agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
	before := s.service.Snapshot()
	s.app.Do(func() {
		s.drawing.routePointer(1, workbench.Point{X: 40, Y: -130}, workbench.Point{X: 40, Y: -20})
		s.drawing.routePointer(2, workbench.Point{X: 40, Y: -130}, workbench.Point{})
	})
	if after := s.service.Snapshot(); after.View != before.View || after.Revision != before.Revision || len(after.Components) != 0 {
		t.Fatal("old gesture edited a replacement document")
	}
}

func TestRCDrawingScopeAndInputControlsMatchAgentCommands(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	for _, p := range []struct {
		kind string
		x, y float32
	}{{"pulse-source", -200, 0}, {"resistor", 0, -110}, {"capacitor", 200, 0}, {"ground", 0, 110}} {
		uiAction(t, s, "part-"+p.kind, nil)
		uiAction(t, s, "position-x", &ui.ChangeEvent{Number: p.x})
		uiAction(t, s, "position-y", &ui.ChangeEvent{Number: p.y})
		uiAction(t, s, "place-at-position", nil)
	}
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "10k"})
	uiAction(t, s, "apply-value", nil)
	uiAction(t, s, "tool-wire", nil)
	for _, pair := range [][2]string{{"V1.positive", "R1.1"}, {"R1.2", "C1.1"}, {"C1.2", "GND1.reference"}, {"GND1.reference", "V1.negative"}} {
		uiAction(t, s, "wire-from", &ui.ChangeEvent{SelectedID: pair[0]})
		uiAction(t, s, "wire-to", &ui.ChangeEvent{SelectedID: pair[1]})
		uiAction(t, s, "connect-terminals", nil)
	}
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "simulation.restart", nil)
	for i := 0; i < 10; i++ {
		uiAction(t, s, "transient-step", nil)
	}
	state := s.service.Snapshot()
	if state.Transient == nil || math.Abs(state.Transient.Sample.Voltage-3.1606027941427883) > 1e-12 {
		t.Fatal("UI RC construction/stepping failed")
	}
	agent := workbench.New()
	defer agent.Close()
	agentAction(t, agent, "example.rc", workbench.ReplaceParams{})
	agentAction(t, agent, "simulation.pause", struct{}{})
	agentAction(t, agent, "simulation.restart", struct{}{})
	agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .01})
	if math.Abs(state.Transient.Sample.Time-agent.Snapshot().Transient.Sample.Time) > 1e-12 || math.Abs(state.Transient.Sample.Voltage-agent.Snapshot().Transient.Sample.Voltage) > 1e-12 || math.Abs(state.Transient.Sample.Current-agent.Snapshot().Transient.Sample.Current) > 1e-15 {
		t.Fatal("UI and agent trace data diverged")
	}
	uiAction(t, s, "input-discharge", nil)
	uiAction(t, s, "transient-step", nil)
	agentAction(t, agent, "simulation.input.set", workbench.InputParams{Mode: "discharge"})
	agentAction(t, agent, "simulation.step", workbench.StepParams{Duration: .001})
	if math.Abs(s.service.Snapshot().Transient.Sample.Voltage-agent.Snapshot().Transient.Sample.Voltage) > 1e-12 || math.Abs(s.service.Snapshot().Transient.Sample.Current-agent.Snapshot().Transient.Sample.Current) > 1e-15 {
		t.Fatal("UI and agent discharge inputs diverged")
	}
	s.app.Do(func() {
		mesh, err := s.scopeMesh(canvas.Viewport{Width: 800, Height: 160, Zoom: 1})
		if err != nil || len(mesh.Vertices()) == 0 {
			t.Fatal("scope geometry missing")
		}
	})
}

func schematicDocumentState(document workbench.DocumentState) workbench.DocumentState {
	document.Dirty = document.SchematicDirty
	document.RecordingsDirty = false
	document.RecordingCount = 0
	return document
}
