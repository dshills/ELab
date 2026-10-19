//go:build offscreen

package main

import (
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/drivers/cogent"
	"github.com/dshills/vellum/ui"
)

func mountedTimingControls(t *testing.T, s *screen) *mountedControlWindow {
	t.Helper()
	m := newMountedControlWindow(t, s, func() ui.Node {
		assembled := s.view()
		root := ui.Node{ID: "root", Kind: ui.Column, Focus: ui.FocusRequest{ID: "recorded-timing-level", Sequence: 1}}
		for row, ids := range [][]string{{"recorded-timing-level", "recorded-timing-second-level"}, {"recorded-timing-start", "recorded-timing-end"}, {"recorded-timing-next"}} {
			group := ui.Node{ID: []string{"levels", "interval", "pages"}[row], Kind: ui.Row}
			for _, id := range ids {
				node, ok := findNode(assembled, id)
				if !ok {
					timing := s.service.Snapshot().Recording.Timing
					if id == "recorded-timing-next" && timing.NextOffset >= timing.Total {
						continue
					}
					t.Fatalf("missing production timing control %s", id)
				}
				node.MinWidth, node.MinHeight = 190, 40
				group.Children = append(group.Children, node)
			}
			root.Children = append(root.Children, group)
		}
		return root
	})
	s.settleField = func(id string, done func(error)) error { return cogent.SettleTextField(m.ctx, s.app, "main", id, done) }
	return m
}

func TestMountedRecordingTimingTypedThresholdThenNextMatchesAgent(t *testing.T) {
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
		t.Fatal("private threshold draft changed accepted measurements")
	}
	clickMountedControl(t, m, "recorded-timing-next")
	cfg := workbench.RecordingTimingParams(agent.Snapshot().Recording.Timing.Settings)
	cfg.Level = "3 V"
	agentAction(t, agent, "recording.timing.set", cfg)
	cfg = workbench.RecordingTimingParams(agent.Snapshot().Recording.Timing.Settings)
	cfg.Offset = agent.Snapshot().Recording.Timing.NextOffset
	agentAction(t, agent, "recording.timing.set", cfg)
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.CommandRevision != before.CommandRevision+2 || !reflect.DeepEqual(after.Recording.Timing, agent.Snapshot().Recording.Timing) {
		t.Fatalf("one Next click failed after private threshold: level=%s offset=%d revision=%d→%d error=%s", after.Recording.Timing.Settings.Level, after.Recording.Timing.Settings.Offset, before.CommandRevision, after.CommandRevision, s.errorMessage)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}

func TestMountedRecordingTimingInvalidThresholdDoesNotAdvancePage(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	prepareRecordedTiming(t, s.service)
	m := mountedTimingControls(t, s)
	before := s.service.Snapshot()
	typeMountedCursor(t, m, "recorded-timing-level", "bad")
	clickMountedControl(t, m, "recorded-timing-next")
	if s.errorMessage == "" || !reflect.DeepEqual(before, s.service.Snapshot()) {
		t.Fatalf("invalid threshold advanced page or cleared validation: offset=%d error=%s", s.service.Snapshot().Recording.Timing.Settings.Offset, s.errorMessage)
	}
	typeMountedCursor(t, m, "recorded-timing-level", "3 V")
	clickMountedControl(t, m, "recorded-timing-next")
	after := s.service.Snapshot()
	if s.errorMessage != "" || after.Recording.Timing.Settings.Level != "3 V" || after.Recording.Timing.Settings.Offset != 8 || after.CommandRevision != before.CommandRevision+2 {
		t.Fatal("corrected threshold did not page on the first click", s.errorMessage)
	}
}
