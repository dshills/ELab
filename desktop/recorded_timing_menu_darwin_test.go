//go:build darwin && cgo

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

// Exercise the assembled macOS callback and admission guard. These tests do
// not open an AppKit menu or establish field-blur/physical input behavior.
func captureRecordedTimingMenu(t *testing.T, s *screen) (ui.Node, menuChoices, uint64, uint64, int) {
	t.Helper()
	var node ui.Node
	var choices menuChoices
	var revision, serial uint64
	index := -1
	s.app.Do(func() {
		var ok bool
		node, ok = findNode(s.view(), "recorded-timing-direction")
		if !ok {
			t.Fatal("missing timing direction menu")
		}
		choices, ok = node.Custom.(menuChoices)
		if !ok || choices.Kind != ui.Selector || choices.Change == nil {
			t.Fatal("timing selector did not use the native menu adapter")
		}
		for i, option := range choices.Options {
			if option.ID == "rising" {
				index = i
			}
		}
		if index < 0 {
			t.Fatal("rising direction absent")
		}
		s.menuSerial++
		serial, revision = s.menuSerial, s.service.Snapshot().CommandRevision
	})
	return node, choices, serial, revision, index
}

func TestNativeRecordedTimingMenuFreshChoiceMatchesAgent(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	for _, service := range []*workbench.Service{s.service, agent} {
		prepareRecordedTiming(t, service)
	}
	// Commit before taking the menu's context. The physical blur-to-open
	// transition is a separate native test, not simulated by this fixture.
	uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "3 V"})
	cfg := workbench.RecordingTimingParams(agent.Snapshot().Recording.Timing.Settings)
	cfg.Level = "3 V"
	agentAction(t, agent, "recording.timing.set", cfg)
	node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
	before := s.service.Snapshot()
	s.app.Do(func() { s.acceptMenuChoice(node, choices, index, serial, revision) })
	cfg.Direction = "rising"
	agentAction(t, agent, "recording.timing.set", cfg)
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.LastActor != "user" || after.CommandRevision != before.CommandRevision+1 || !reflect.DeepEqual(after.Recording.Timing, agent.Snapshot().Recording.Timing) {
		t.Fatal("fresh native timing choice differs from agent", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestNativeRecordedTimingMenuRefusesInterveningContext(t *testing.T) {
	for _, scenario := range []string{"agent-settings", "user-field", "other-recording", "new-document", "menu-reopened", "file-busy"} {
		t.Run(scenario, func(t *testing.T) {
			s, cleanup := testScreen(t)
			t.Cleanup(cleanup)
			prepareRecordedTiming(t, s.service)
			node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
			switch scenario {
			case "agent-settings":
				cfg := workbench.RecordingTimingParams(s.service.Snapshot().Recording.Timing.Settings)
				cfg.Level = "2 V"
				agentAction(t, s.service, "recording.timing.set", cfg)
			case "user-field":
				uiAction(t, s, "recorded-timing-level", &ui.ChangeEvent{Text: "3 V"})
			case "other-recording":
				duration := .01
				agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
				completedVerification(t, s.service, s.service.Snapshot().LastOperation)
			case "new-document":
				agentAction(t, s.service, "project.clear", workbench.ReplaceParams{DiscardChanges: true})
			case "menu-reopened":
				s.app.Do(func() { s.menuSerial++ })
			case "file-busy":
				s.app.Do(func() { s.fileBusy = true })
			}
			before := s.service.Snapshot()
			s.app.Do(func() { s.acceptMenuChoice(node, choices, index, serial, revision) })
			if !reflect.DeepEqual(before, s.service.Snapshot()) || s.menuCommandRevision != nil || s.fieldContext != nil {
				t.Fatal("stale native timing menu altered state or leaked context", scenario)
			}
		})
	}
}
