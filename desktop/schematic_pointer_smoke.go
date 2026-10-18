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

// Wait for real native mesh preparation, then inject the semantic packet on
// its owner. This is not physical NSEvent/keyboard or display-latency evidence.
func nativeSchematicPointerSmoke(ctx context.Context, a *app.Application, s *screen) error {
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
		response := s.invoke("project.clear", workbench.ReplaceParams{DiscardChanges: true})
		if response.Error == nil {
			response = s.invoke("drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "resistor"})
		}
		if response.Error != nil {
			return fmt.Errorf("pointer preparation: %s", response.Error.Message)
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool { return state.Drawing.Kind == "resistor" }); err != nil {
		return err
	}
	if err := owner(func() error {
		for _, command := range []struct {
			id     string
			params any
		}{{"project.clear", workbench.ReplaceParams{DiscardChanges: true}}, {"drawing.tool.set", workbench.ToolParams{Tool: "place", Kind: "led"}}} {
			state := s.service.Snapshot()
			raw, _ := json.Marshal(command.params)
			id := fmt.Sprintf("pointer-smoke-%d", state.CommandRevision)
			response := s.service.Dispatch(ctx, workbench.Request{Version: 1, ID: id, Key: id, Command: command.id, Params: raw, ExpectedCommandRevision: &state.CommandRevision, ExpectedDocumentGeneration: &state.Document.Generation}, "agent")
			if response.Error != nil {
				return fmt.Errorf("pointer agent intervention: %s", response.Error.Message)
			}
		}
		before := s.service.Snapshot()
		s.routeNativePointer("schematic", 0, workbench.Point{X: 120, Y: 80}, workbench.Point{})
		if !reflect.DeepEqual(before, s.service.Snapshot()) {
			return fmt.Errorf("old prepared schematic packet placed into replacement")
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := waitNativeViewport(ctx, a, s, func(state workbench.Snapshot) bool { return state.Drawing.Kind == "led" }); err != nil {
		return err
	}
	return owner(func() error {
		before := s.service.Snapshot()
		s.routeNativePointer("schematic", 0, workbench.Point{X: 120, Y: 80}, workbench.Point{})
		after := s.service.Snapshot()
		if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || len(after.Components) != 1 || after.Components[0].Kind != "led" || after.Components[0].Position != (workbench.Point{X: 120, Y: 80}) {
			return fmt.Errorf("fresh prepared schematic packet failed: %s", s.errorMessage)
		}
		log.Printf("smoke: schematic pointer refused replacement and accepted fresh LED at 120,80")
		return nil
	})
}
