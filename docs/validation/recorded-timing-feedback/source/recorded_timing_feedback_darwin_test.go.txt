//go:build darwin && cgo

package main

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"cogentcore.org/core/core"
	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeRecordedTimingCorrectionRetainsFeedbackUntilDirectionCompletes(t *testing.T) {
	for _, agentEdit := range []bool{false, true} {
		t.Run(fmt.Sprint("agent-edit-", agentEdit), func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareRecordedTiming(t, s.service)
			agentAction(t, s.service, "view.viewport.set", workbench.ViewportParams{Width: 520, Height: 138})
			s.drawing.widget = &core.WidgetBase{}
			s.drawing.View.Width, s.drawing.View.Height = 520, 138
			s.app.Do(func() {
				field, _ := findNode(s.view(), "recorded-timing-level")
				field.Change(ui.ChangeEvent{Text: "bad"})
			})
			help, message := s.requestHelp, s.errorMessage
			if help == nil || message == "" {
				t.Fatal("missing structured timing refusal")
			}
			// The real viewport publisher continues that refusal through the
			// layout change that makes its error row visible, as in /164.
			s.drawing.View.Height = 88
			s.app.Do(func() { s.observeSchematicViewport(*s.drawing.View, s.service.Snapshot()) })
			drainViewport(t, s, func() bool { return s.service.Snapshot().Viewport.Height == 88 })
			s.app.Do(func() { s.view() })
			before := s.service.Snapshot()
			if s.requestHelp != help || s.requestHelpRevision != before.CommandRevision {
				t.Fatal("fixture lost refusal during its own layout publication")
			}
			var finish func(error)
			s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
			node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
			s.app.Do(func() {
				s.acceptMenuChoice(node, choices, index, serial, revision)
				field, _ := findNode(s.view(), "recorded-timing-level")
				field.Change(ui.ChangeEvent{Text: "3.5 V"})
				s.view()
			})
			accepted := s.service.Snapshot()
			if accepted.Recording.Timing.Settings.Level != "3.5 V" || s.pendingRecordedTimingPage == nil || s.requestHelp != help || s.errorMessage != message || s.requestHelpRevision != accepted.CommandRevision {
				t.Fatal("corrected field hid its refusal before the pending direction finished")
			}
			if agentEdit {
				cfg := workbench.RecordingTimingParams(accepted.Recording.Timing.Settings)
				cfg.Level = "2 V"
				agentAction(t, s.service, "recording.timing.set", cfg)
			}
			beforeFinish := s.service.Snapshot()
			s.app.Do(func() {
				for range 5 {
					if s.pendingRecordedTimingPage != nil {
						finish(nil)
					}
				}
			})
			after := s.service.Snapshot()
			if agentEdit {
				if s.errorMessage == "" || s.requestHelp == help || !reflect.DeepEqual(beforeFinish, after) {
					t.Fatal("pending correction adopted agent work or resurrected its old refusal")
				}
			} else if s.errorMessage != "" || s.requestHelp != nil || after.Recording.Timing.Settings.Level != "3.5 V" || after.Recording.Timing.Settings.Direction != "rising" || after.CommandRevision != before.CommandRevision+2 {
				t.Fatal("corrected direction did not finish once and clear related feedback", s.errorMessage)
			}
			assertCursorMeasurementPreservesPhysics(t, before, after)
		})
	}
}

func TestNativeRecordedTimingCorrectionDoesNotRetainOtherRequestHelp(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	var timingHelp *workbench.DiagnosticHelp
	s.app.Do(func() {
		field, _ := findNode(s.view(), "recorded-timing-level")
		field.Change(ui.ChangeEvent{Text: "bad"})
		timingHelp = s.requestHelp
		s.invoke("component.set", workbench.ValueParams{ID: "RB", Value: "0"})
	})
	otherHelp := s.requestHelp
	if otherHelp == nil || timingHelp == nil || otherHelp == timingHelp {
		t.Fatal("fixture did not replace timing help with a distinct refusal")
	}
	var finish func(error)
	s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
	node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
	s.app.Do(func() {
		s.acceptMenuChoice(node, choices, index, serial, revision)
		field, _ := findNode(s.view(), "recorded-timing-level")
		field.Change(ui.ChangeEvent{Text: "3.5 V"})
	})
	if s.requestHelp != nil || s.errorMessage != "" {
		t.Fatal("timing correction retained an unrelated request's refusal")
	}
	s.app.Do(func() {
		for range 5 {
			if s.pendingRecordedTimingPage != nil {
				finish(nil)
			}
		}
	})
	if s.errorMessage != "" || s.service.Snapshot().Recording.Timing.Settings.Direction != "rising" {
		t.Fatal("fresh timing action failed after unrelated refusal", s.errorMessage)
	}
}

func TestNativeRecordedTimingCorrectionSettlementErrorReplacesHeldFeedback(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	s.app.Do(func() {
		field, _ := findNode(s.view(), "recorded-timing-level")
		field.Change(ui.ChangeEvent{Text: "bad"})
	})
	var finish func(error)
	s.settleField = func(_ string, done func(error)) error { finish = done; return nil }
	node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
	s.app.Do(func() {
		s.acceptMenuChoice(node, choices, index, serial, revision)
		field, _ := findNode(s.view(), "recorded-timing-level")
		field.Change(ui.ChangeEvent{Text: "3.5 V"})
	})
	before := s.service.Snapshot()
	s.app.Do(func() { finish(errors.New("composition interrupted")) })
	if s.requestHelp != nil || s.errorMessage != "composition interrupted" || s.pendingRecordedTimingPage != nil || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("settlement failure was hidden by held timing feedback", s.errorMessage)
	}
}
