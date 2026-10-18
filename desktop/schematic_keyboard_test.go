package main

import (
	"reflect"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
)

// Standalone Cogent event dispatch through the production binder. This does
// not mount the custom mesh, use the OS event queue or prove keyboard focus.
func schematicKeyWidget(t *testing.T, s *screen) *core.WidgetBase {
	t.Helper()
	openCloseTestWindow(t, s)
	ref, err := s.app.WindowRef("main")
	if err != nil {
		t.Fatal(err)
	}
	w := core.NewFrame().AsWidget()
	s.drawing.bindSchematicKeys(w, ref)
	return w
}

func TestSchematicKeyboardCannotAdoptReplacementSelection(t *testing.T) {
	for _, code := range []key.Codes{key.CodeBackspace, key.CodeRightArrow} {
		t.Run(code.String(), func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "R1", Kind: "resistor", Value: "1k"})
			prepareSchematicPointer(t, s)
			w := schematicKeyWidget(t, s)
			agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "D1", Kind: "led", Position: workbench.Point{X: 100, Y: 100}})
			before := s.service.Snapshot()
			event := events.NewKey(events.KeyChord, 0, code, 0)
			w.HandleEvent(event)
			if !event.IsHandled() {
				t.Fatal("key did not reach production binding")
			}
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("old schematic key changed the replacement selection", code)
			}
			prepareSchematicPointer(t, s)
			w.HandleEvent(events.NewKey(events.KeyChord, 0, code, 0))
			after := s.service.Snapshot()
			if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 {
				t.Fatal("fresh key did not work", code, s.errorMessage)
			}
			if code == key.CodeBackspace && len(after.Components) != 0 || code == key.CodeRightArrow && after.Components[0].Position.X != 110 {
				t.Fatal("fresh key acted on wrong component", code)
			}
		})
	}
}

func TestSchematicKeyboardOwnRepeatsMatchPublicMoveAndUndo(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agent := workbench.New()
	defer agent.Close()
	for _, service := range []*workbench.Service{s.service, agent} {
		agentAction(t, service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
		agentAction(t, service, "component.select", workbench.IDParams{ID: "R1"})
	}
	prepareSchematicPointer(t, s)
	w := schematicKeyWidget(t, s)
	before := s.service.Snapshot()
	origin, _ := (workbench.Graph{Components: before.Components, Wires: before.Wires}).Component("R1")
	for i := range 3 {
		// No mesh preparation: autorepeat can arrive before its own repaint.
		w.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeRightArrow, 0))
		previous := workbench.Point{X: origin.Position.X + float64(i)*10, Y: origin.Position.Y}
		agentAction(t, agent, "component.move", workbench.MoveParams{ID: "R1", Position: workbench.Point{X: previous.X + 10, Y: previous.Y}, PreviousPosition: &previous})
	}
	after, expected := s.service.Snapshot(), agent.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+3 || !reflect.DeepEqual(after.Components, expected.Components) || !reflect.DeepEqual(after.Wires, before.Wires) || !reflect.DeepEqual(after.Nets, before.Nets) || !reflect.DeepEqual(after.Result, before.Result) || after.UndoName != expected.UndoName {
		t.Fatal("repeat did not preserve shared geometry/connectivity/physics/history", s.errorMessage)
	}
	s.app.Do(func() { s.invoke("edit.undo", struct{}{}) })
	part, _ := (workbench.Graph{Components: s.service.Snapshot().Components}).Component("R1")
	if s.errorMessage != "" || part.Position.X != origin.Position.X+20 {
		t.Fatal("one Undo did not reverse one key repeat", s.errorMessage)
	}
}

func TestSchematicKeyboardRepeatDoesNotBorrowAgentSelection(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "R1"})
	prepareSchematicPointer(t, s)
	w := schematicKeyWidget(t, s)
	old := s.drawing.pointerContext.Load()
	w.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeRightArrow, 0))
	agentAction(t, s.service, "component.select", workbench.IDParams{ID: "D1"})
	before := s.service.Snapshot()
	w.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeBackspace, 0))
	if !reflect.DeepEqual(before, s.service.Snapshot()) || s.drawing.keySequence != nil {
		t.Fatal("old repeat sequence deleted an agent-selected object")
	}
	prepareSchematicPointer(t, s)
	w.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeRightArrow, 0))
	if s.errorMessage != "" || s.service.Snapshot().CommandRevision != before.CommandRevision+1 {
		t.Fatal("fresh selection cannot be moved", s.errorMessage)
	}
	fresh := s.service.Snapshot()
	s.app.Do(func() { s.drawing.routeSchematicKey("backspace", old) })
	if !reflect.DeepEqual(fresh, s.service.Snapshot()) {
		t.Fatal("retired callback adopted a newer key sequence")
	}
}

func TestSchematicKeyboardNavigationAndEscapeKeepOriginalIntent(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.reset", workbench.ReplaceParams{DiscardChanges: true})
	prepareSchematicPointer(t, s)
	w := schematicKeyWidget(t, s)
	before := s.service.Snapshot()
	for _, code := range []key.Codes{key.CodeRightArrow, key.CodeRightArrow, key.CodeEnd} {
		w.HandleEvent(events.NewKey(events.KeyChord, 0, code, 1<<key.Alt))
	}
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.Selected != after.Wires[len(after.Wires)-1].ID || after.CommandRevision != before.CommandRevision+3 || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Result, after.Result) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("navigation repeats changed source or failed to reach last object", s.errorMessage)
	}
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "wire"})
	prepareSchematicPointer(t, s)
	agentAction(t, s.service, "drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "led"})
	before = s.service.Snapshot()
	w.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeEscape, 0))
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old Escape canceled agent's newer placement")
	}
	prepareSchematicPointer(t, s)
	w.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeEscape, 0))
	if s.errorMessage != "" || s.service.Snapshot().Drawing.Tool != "select" {
		t.Fatal("fresh Escape did not cancel drawing", s.errorMessage)
	}
	unknown := events.NewKey(events.KeyChord, 'q', key.CodeQ, 1<<key.Meta)
	w.HandleEvent(unknown)
	if unknown.IsHandled() {
		t.Fatal("schematic binding consumed unrelated shortcut")
	}
}

func TestSchematicKeyboardRespectsBusyPendingAndHiddenCanvas(t *testing.T) {
	for _, blocked := range []string{"busy", "pending", "hidden", "dragging"} {
		t.Run(blocked, func(t *testing.T) {
			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "R1", Kind: "resistor", Value: "1k"})
			if blocked == "hidden" {
				agentAction(t, s.service, "print.preview.start", workbench.PreviewParams{})
				deadline := time.Now().Add(5 * time.Second)
				for {
					preview := s.service.Snapshot().PrintPreview
					if preview != nil && preview.Status == "completed" {
						break
					}
					if time.Now().After(deadline) || preview != nil && (preview.Status == "failed" || preview.Status == "canceled") {
						t.Fatal("preview did not settle", preview)
					}
					time.Sleep(time.Millisecond)
				}
			}
			prepareSchematicPointer(t, s)
			w := schematicKeyWidget(t, s)
			s.fileBusy = blocked == "busy"
			if blocked == "pending" {
				s.pendingCommand = "project.clear"
			}
			s.drawing.dragging = blocked == "dragging"
			before := s.service.Snapshot()
			w.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeBackspace, 0))
			if !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("blocked canvas key deleted selected component")
			}
		})
	}
}

func TestSchematicKeyboardPlannedActionKeepsDispatchPreconditions(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "component.place", workbench.PlaceParams{ID: "R1", Kind: "resistor", Value: "1k"})
	state := s.service.Snapshot()
	command, params := schematicKeyCommand(state, "rightarrow")
	agentAction(t, s.service, "component.move", workbench.MoveParams{ID: "R1", Position: workbench.Point{X: 100, Y: 100}})
	before := s.service.Snapshot()
	s.app.Do(func() { s.invokeAtCommandRevision(command, params, state.CommandRevision, state.Document.Generation) })
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("planned arrow action overwrote intervening agent move")
	}
}
