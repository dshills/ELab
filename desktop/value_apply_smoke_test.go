package main

import (
	"context"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestValueRefusalSmokeAllowsOnlyNativeCanvasObservation(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	for _, command := range []string{"example.rc", "simulation.pause"} {
		if r := s.invoke(command, struct{}{}); r.Error != nil {
			t.Fatal(r.Error)
		}
	}
	// Match the manual view used by the mounted smoke: a canvas allocation
	// change must preserve its view as well as its circuit and accepted sample.
	if r := s.invoke("view.pan", map[string]float64{"dx": 1, "dy": 0}); r.Error != nil {
		t.Fatal(r.Error)
	}
	if r := s.invoke("view.viewport.set", workbench.ViewportParams{Width: 600, Height: 300}); r.Error != nil {
		t.Fatal(r.Error)
	}
	if _, err := s.service.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := s.service.Snapshot()
	if r := s.invoke("component.set", workbench.ValueParams{ID: "V1", Value: "1e-999 V"}); r.Error == nil {
		t.Fatal("underflow was accepted")
	}
	if r := s.invoke("view.viewport.set", workbench.ViewportParams{Width: 600, Height: 250}); r.Error != nil {
		t.Fatal(r.Error)
	}
	if !nativeValueRefusalPreservesState(before, s.service.Snapshot()) {
		t.Fatal("a native canvas observation was mistaken for an accepted edit")
	}
	if r := s.invoke("component.set", workbench.ValueParams{ID: "R1", Value: "20k"}); r.Error != nil {
		t.Fatal(r.Error)
	}
	if nativeValueRefusalPreservesState(before, s.service.Snapshot()) {
		t.Fatal("canvas resize masked a real circuit edit")
	}
	after := before
	after.CommandRevision++
	if nativeValueRefusalPreservesState(before, after) {
		t.Fatal("unexplained sequencing change was ignored without a canvas resize")
	}
}
