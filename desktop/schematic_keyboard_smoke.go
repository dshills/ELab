package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/app"
)

// Real native preparation plus semantic key routing. Physical event delivery
// and OS focus/IME remain separate from the standalone widget binder tests.
func nativeSchematicKeyboardSmoke(ctx context.Context, a *app.Application, s *screen) error {
	owner := func(run func() error) error {
		done := make(chan error, 1)
		if err := a.PostWait(ctx, func() { done <- run() }); err != nil {
			return err
		}
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := owner(func() error {
		if response := s.invoke("example.reset", workbench.ReplaceParams{DiscardChanges: true}); response.Error != nil {
			return fmt.Errorf("keyboard setup: %s", response.Error.Message)
		}
		if response := s.invoke("component.select", workbench.IDParams{ID: "R1"}); response.Error != nil {
			return fmt.Errorf("keyboard selection: %s", response.Error.Message)
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := s.service.Wait(ctx); err != nil {
		return err
	}
	if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool { return state.Selected == "R1" }); err != nil {
		return err
	}
	if err := owner(func() error {
		captured := s.drawing.pointerContext.Load()
		state := s.service.Snapshot()
		id := fmt.Sprintf("keyboard-smoke-%d", state.CommandRevision)
		raw, _ := json.Marshal(workbench.IDParams{ID: "D1"})
		response := s.service.Dispatch(ctx, workbench.Request{Version: 1, ID: id, Key: id, Command: "component.select", Params: raw, ExpectedCommandRevision: &state.CommandRevision, ExpectedDocumentGeneration: &state.Document.Generation}, "agent")
		if response.Error != nil {
			return fmt.Errorf("keyboard agent selection: %s", response.Error.Message)
		}
		before := s.service.Snapshot()
		s.drawing.routeSchematicKey("backspace", captured)
		if !reflect.DeepEqual(before, s.service.Snapshot()) {
			return fmt.Errorf("old prepared key deleted new agent selection")
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool { return state.Selected == "D1" }); err != nil {
		return err
	}
	return owner(func() error {
		before := s.service.Snapshot()
		part, _ := (workbench.Graph{Components: before.Components}).Component("D1")
		s.drawing.routeSchematicKey("rightarrow", s.drawing.pointerContext.Load())
		after := s.service.Snapshot()
		moved, _ := (workbench.Graph{Components: after.Components}).Component("D1")
		if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || moved.Position.X != part.Position.X+10 || moved.Position.Y != part.Position.Y || !reflect.DeepEqual(after.Wires, before.Wires) || !reflect.DeepEqual(after.Result, before.Result) {
			return fmt.Errorf("fresh prepared key move failed or changed physics: %s", s.errorMessage)
		}
		log.Printf("smoke: schematic keyboard refused agent selection and moved fresh LED one grid step")
		return nil
	})
}
