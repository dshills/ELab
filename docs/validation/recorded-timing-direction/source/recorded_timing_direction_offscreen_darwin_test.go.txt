//go:build offscreen && darwin && cgo

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
)

// AppKit menu selection does not necessarily blur the focused text field.
// Deliver its actual assembled callback while the mounted draft is private.
func chooseMountedTimingDirection(t *testing.T, s *screen, m *mountedControlWindow) {
	t.Helper()
	node, choices, serial, revision, index := captureRecordedTimingMenu(t, s)
	s.app.Do(func() { s.acceptMenuChoice(node, choices, index, serial, revision) })
	m.settle()
}

func TestMountedRecordingTimingDirectionCommitsPrivateThreshold(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	agent := workbench.New()
	t.Cleanup(agent.Close)
	for _, service := range []*workbench.Service{s.service, agent} {
		prepareRecordedTiming(t, service)
	}
	m := mountedTimingControls(t, s)
	before := s.service.Snapshot()
	typeMountedCursor(t, m, "recorded-timing-level", "3 V")
	if !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("private threshold changed accepted state")
	}
	chooseMountedTimingDirection(t, s, m)
	cfg := workbench.RecordingTimingParams(agent.Snapshot().Recording.Timing.Settings)
	cfg.Level = "3 V"
	agentAction(t, agent, "recording.timing.set", cfg)
	cfg.Direction = "rising"
	agentAction(t, agent, "recording.timing.set", cfg)
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || !reflect.DeepEqual(after.Recording.Timing, agent.Snapshot().Recording.Timing) {
		t.Fatalf("direction lost private threshold: level=%s direction=%s revision=%d→%d error=%s", after.Recording.Timing.Settings.Level, after.Recording.Timing.Settings.Direction, before.CommandRevision, after.CommandRevision, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestMountedRecordingTimingDirectionRefusesInvalidThresholdAndCorrects(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	m := mountedTimingControls(t, s)
	before := s.service.Snapshot()
	typeMountedCursor(t, m, "recorded-timing-level", "bad")
	chooseMountedTimingDirection(t, s, m)
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatal("direction accepted invalid threshold or hid its error", s.errorMessage)
	}
	typeMountedCursor(t, m, "recorded-timing-level", "3 V")
	chooseMountedTimingDirection(t, s, m)
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || after.Recording.Timing.Settings.Level != "3 V" || after.Recording.Timing.Settings.Direction != "rising" {
		t.Fatal("corrected threshold did not select direction on first attempt", s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}
