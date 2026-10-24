//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedCapturedComparison(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		field, ok := findNode(ui.Node{Children: s.recordingInspector(state)}, "recorded-compare-time")
		if !ok || field.Disabled {
			t.Fatal("missing enabled captured comparison time")
		}
		field.MinWidth, field.MinHeight = 190, 40
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: field.ID, Sequence: 1}, Children: []ui.Node{field}}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func TestMountedScopeCursorComparisonRefusalRetainsEntry(t *testing.T) {
	for _, text := range []string{"oops", "-1 ms", "30 ms"} {
		t.Run(text, func(t *testing.T) {
			s, _ := capturedComparisonScreen(t)
			m := mountedCapturedComparison(t, s)
			before := s.service.Snapshot()
			typeMountedCursor(t, m, "recorded-compare-time", text)
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			m.frame()
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("invalid comparison time changed accepted state or was not refused", s.errorMessage)
			}
			if field := mountedControlElement(t, m, "recorded-compare-time"); field.Value != text {
				t.Fatalf("rejected comparison %q disappeared before correction: field=%q", text, field.Value)
			}
			typeMountedCursor(t, m, "recorded-compare-time", "0 s")
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			after := s.service.Snapshot()
			if s.errorMessage != "" || s.requestHelp != nil || after.CommandRevision != before.CommandRevision+1 || !reflect.DeepEqual(before.Recording.Comparison, after.Recording.Comparison) {
				t.Fatal("first comparison correction did not recover", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestMountedScopeCursorComparisonPrivateEntryStaysWithReference(t *testing.T) {
	s, nextReference := capturedComparisonScreen(t)
	m := mountedCapturedComparison(t, s)
	original := mountedControlElement(t, m, "recorded-compare-time")
	typeMountedCursor(t, m, "recorded-compare-time", "4 ms")
	params := workbench.RecordingCompareParams(s.service.Snapshot().Recording.Comparison.Settings)
	params.ReferenceID = nextReference
	agentAction(t, s.service, "recording.compare.set", params)
	before := s.service.Snapshot()
	m.frame()
	field := mountedControlElement(t, m, "recorded-compare-time")
	if field.Incarnation == original.Incarnation || field.Value != "0 s" {
		t.Fatal("old private comparison time followed the agent's reference", field.Value)
	}
	clickMountedControl(t, m, "recorded-compare-time")
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old private comparison time replaced the agent's accepted request")
	}
	typeMountedCursor(t, m, "recorded-compare-time", "2 ms")
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || after.Recording.Comparison.Settings.Time != .002 || after.Recording.Comparison.Settings.ReferenceID != nextReference || !after.Recording.Comparison.Settings.ShowOverlay || !after.Recording.Comparison.Settings.ShowStatistics {
		t.Fatal("fresh comparison lost the new reference or its independent preferences", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}
