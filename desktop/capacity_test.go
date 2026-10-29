package main

import (
	"context"
	"testing"
	"time"

	"github.com/dshills/vellum/ui"
)

func TestNativeCapacitySharedConstructionAndMesh(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	if err := constructCapacity(s); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	state, err := s.service.Wait(ctx)
	if err != nil || len(state.Nets) != 100 || state.SimulatedComponents != 51 || state.Limits.SimulatedComponents != 64 {
		t.Fatal("native capacity failed", err)
	}
	uiAction(t, s, "inspector-section-simulation", nil)
	uiAction(t, s, "simulation-tool", &ui.ChangeEvent{SelectedID: "circuit"})
	if _, ok := findNode(s.view(), "circuit-capacity"); !ok {
		t.Fatal("capacity assumptions hidden")
	}
	mesh, err := s.mesh(*s.drawing.View)
	if err != nil || mesh == nil {
		t.Fatal("native capacity mesh failed", err)
	}
}
