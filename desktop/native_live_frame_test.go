package main

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/dshills/elab/workbench"
)

// The service is allowed to advance between composing a form and the GPU's
// Prepare call. Reproduce that ordering using two real, command-free RC ticks.
func TestRCMeshUsesDisplayedInspectorSample(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", struct{}{})
	first := s.service.Snapshot()
	if first.Transient == nil || first.Paused {
		t.Fatal("expected a live RC example")
	}
	second := first
	for deadline := time.Now().Add(2 * time.Second); second.Transient.Sample.Voltage == first.Transient.Sample.Voltage; {
		if time.Now().After(deadline) {
			t.Fatal("RC clock did not advance")
		}
		time.Sleep(5 * time.Millisecond)
		second = s.service.Snapshot()
	}
	if first.CommandRevision != second.CommandRevision {
		t.Fatal("fixture must advance without an editor command")
	}
	var prev uint64
	for i, frame := range []workbench.Snapshot{first, second} {
		root := s.workspaceFrame(frame, 1440)
		read, ok := findNode(root, "voltage-value")
		if !ok || read.Text != fmt.Sprintf("%.3f V", frame.Transient.Sample.Voltage) {
			t.Fatal("inspector is not showing the accepted capacitor sample", read.Text)
		}
		pills := nativeProbePills(frame, *s.drawing.View, s.colorsForTheme(frame.Theme))
		found := false
		for _, p := range pills {
			if p.ID == "v:C1.1" {
				found = p.Text == nativeQuantity(frame.Transient.Sample.Voltage, "V")
			}
		}
		if !found || i > 0 && s.drawing.Revision == prev {
			t.Fatal("new capacitor sample must update the badge and invalidate the mesh")
		}
		_, err := s.scope.Prepare(*s.scope.View)
		if err != nil {
			t.Fatal(err)
		}
		plot := s.scope.frame.Load()
		if plot == nil || plot.end != frame.Transient.Sample.Time {
			t.Fatal("scope endpoint does not match the displayed sample", plot)
		}
		prev = s.drawing.Revision
		got, err := s.drawing.Prepare(*s.drawing.View)
		if err != nil {
			t.Fatal(err)
		}
		want, err := schematicWithPalette(frame, *s.drawing.View, 0, s.colorsForTheme(frame.Theme))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got.Vertices(), want.Vertices()) || !slices.Equal(got.Indices(), want.Indices()) {
			t.Fatal("mesh painted a different sample from the inspector")
		}
	}

	// A probe that has no reading at the current time must not borrow the
	// last value from an older sample.
	incomplete := second
	incomplete.Scope = first.Scope
	if pills := nativeProbePills(incomplete, *s.drawing.View, s.colorsForTheme(second.Theme)); len(pills) != 0 {
		t.Fatal("older trace values were shown as current", pills)
	}
}

func TestRCScopeOffersAllSourceModesWithoutDisclosure(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	agentAction(t, s.service, "example.rc", struct{}{})
	agentAction(t, s.service, "simulation.pause", struct{}{})
	uiAction(t, s, "input-discharge", nil)
	state := s.service.Snapshot()
	if state.Transient.PendingMode != "discharge" && state.Transient.Mode != "discharge" {
		t.Fatal("Discharge did not reach the RC source")
	}
	for _, width := range []float32{360, 1040} {
		panel := s.instrumentPanel(state, width)
		row, ok := findNode(panel, "native-rc-drive")
		if !ok {
			t.Fatal("missing RC source row")
		}
		details, _ := findNode(panel, "workspace-details-instrument")
		if details.Checked {
			t.Fatal("expected collapsed instrument details")
		}
		for _, mode := range []string{"auto", "charge", "discharge"} {
			control, ok := findAction(row, "input-"+mode)
			if !ok || control.Disabled || control.Text == "" {
				t.Fatal("source mode unavailable:", mode)
			}
			if _, duplicate := findNode(details, "input-"+mode); duplicate {
				t.Fatal("duplicated source control")
			}
		}
	}
	uiAction(t, s, "scope-view-spectrum", nil)
	if _, ok := findNode(s.view(), "native-rc-drive"); ok {
		t.Fatal("unexpected RC row in Spectrum")
	}
	uiAction(t, s, "scope-view-scope", nil)
	uiAction(t, s, "tool-wire", nil)
	wire, ok := findNode(s.view(), "tool-wire")
	if !ok || wire.Text != "Wire" || wire.Accessibility.Name != "Wire" || s.service.Snapshot().Drawing.Tool != "wire" {
		t.Fatal("Wire must keep its visible label and command")
	}
}
