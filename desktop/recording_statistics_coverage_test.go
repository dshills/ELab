package main

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestCapturedStatisticsDefaultUsesActualSavedEnd(t *testing.T) {
	s, cleanup := testScreen(t)
	t.Cleanup(cleanup)
	path, err := filepath.Abs("../workbench/testdata/recorded-statistics-short-tail.json")
	if err != nil {
		t.Fatal(err)
	}
	agentAction(t, s.service, "verification.import", workbench.RecordingImportParams{Path: path})
	before := s.service.Snapshot()
	if before.Recording == nil || before.Recording.LastSampleTime >= before.Recording.Duration {
		t.Fatal("fixture needs distinct saved and nominal endpoints")
	}
	s.app.Do(func() {
		root := ui.Node{Children: s.recordingStatisticsControls(before)}
		s.bindFieldContext(&root, actionContext{before.CommandRevision, before.Document.Generation})
		selector, ok := findNode(root, "recorded-statistics-probe")
		if !ok || selector.Disabled {
			t.Fatal("missing enabled captured statistics selector")
		}
		selector.Change(ui.ChangeEvent{SelectedID: "v:V1.positive"})
	})
	after := s.service.Snapshot()
	r := after.Recording.Statistics
	if s.errorMessage != "" || r == nil || r.Settings.End != before.Recording.LastSampleTime || math.Abs(r.Mean-5) > 1e-13 || math.Abs(r.RMS-5) > 1e-13 {
		t.Fatal("default measures nominal unsaved tail instead of actual rows", s.errorMessage, r)
	}
	assertCursorMeasurementPreservesPhysics(t, before, after)
}
