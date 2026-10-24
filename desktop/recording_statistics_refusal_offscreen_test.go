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

func mountedCapturedStatistics(t *testing.T, s *screen, focus string) *mountedControlWindow {
	t.Helper()
	return newMountedControlWindow(t, s, func() ui.Node {
		state := s.service.Snapshot()
		nodes := s.recordingInspector(state)
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: focus, Sequence: 1}}
		for _, id := range []string{"recorded-statistics-start", "recorded-statistics-end"} {
			field, ok := findNode(ui.Node{Children: nodes}, id)
			if !ok || field.Disabled {
				t.Fatal("missing enabled captured measurement field", id)
			}
			field.MinWidth, field.MinHeight = 190, 40
			root.Children = append(root.Children, field)
		}
		s.bindFieldContext(&root, actionContext{state.CommandRevision, state.Document.Generation})
		return root
	})
}

func TestMountedScopeCursorStatisticsRefusalRetainsEntry(t *testing.T) {
	for _, test := range []struct {
		name, side, text string
		imported         bool
	}{{"negative-start", "start", "-1 ms", false}, {"past-duration", "end", "30 ms", false}, {"unsaved-tail", "end", "20 ms", true}} {
		t.Run(test.name, func(t *testing.T) {
			s := capturedStatisticsScreen(t, test.imported)
			id := "recorded-statistics-" + test.side
			m := mountedCapturedStatistics(t, s, id)
			before := s.service.Snapshot()
			accepted := before.Recording.Statistics.Settings.Start
			if test.side == "end" {
				accepted = before.Recording.Statistics.Settings.End
			}
			typeMountedCursor(t, m, id, test.text)
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			m.frame()
			if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
				t.Fatal("invalid interval was accepted or replaced the cached measurement", s.errorMessage)
			}
			if field := mountedControlElement(t, m, id); field.Value != test.text {
				t.Fatalf("rejected measurement %q disappeared before correction: field=%q", test.text, field.Value)
			}
			typeMountedCursor(t, m, id, timeText(accepted))
			m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
			after := s.service.Snapshot()
			if s.errorMessage != "" || s.requestHelp != nil || after.CommandRevision != before.CommandRevision+1 || !reflect.DeepEqual(before.Recording.Statistics, after.Recording.Statistics) {
				t.Fatal("first correction back to the accepted measurement did not recover", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestMountedScopeCursorStatisticsPrivateEntryStaysWithItsProbe(t *testing.T) {
	s := capturedStatisticsScreen(t, false)
	id := "recorded-statistics-start"
	m := mountedCapturedStatistics(t, s, id)
	original := mountedControlElement(t, m, id)
	typeMountedCursor(t, m, id, "4 ms")
	state := s.service.Snapshot()
	if len(state.Recording.Scope.Available) < 2 {
		t.Fatal("fixture requires two captured channels")
	}
	params := workbench.RecordingStatisticsParams(state.Recording.Statistics.Settings)
	params.ProbeID = state.Recording.Scope.Available[1].ID
	agentAction(t, s.service, "recording.statistics.set", params)
	before := s.service.Snapshot()
	m.frame()
	field := mountedControlElement(t, m, id)
	if field.Incarnation == original.Incarnation || field.Value != "0 s" {
		t.Fatal("old private measurement entry followed the newly selected channel", field.Value)
	}
	clickMountedControl(t, m, id)
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("old private entry changed the agent's selected measurement")
	}
	typeMountedCursor(t, m, id, "2 ms")
	m.input(cogent.ManualInput{Type: events.KeyChord, Code: key.CodeReturnEnter})
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+1 || after.Recording.Statistics.Settings.Start != .002 || after.Recording.Statistics.Settings.ProbeID != params.ProbeID {
		t.Fatal("fresh entry did not target the newly selected channel", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}
