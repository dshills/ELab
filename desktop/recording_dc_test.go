package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func TestNativeSingleSampleRecordingsRenderAndRetainTransientSeeking(t *testing.T) {
	for _, example := range []string{"example.reset", "example.npn"} {
		t.Run(example, func(t *testing.T) {
			source := workbench.New()
			defer source.Close()
			agentAction(t, source, example, workbench.ReplaceParams{})
			zero := 0.
			agentAction(t, source, "simulation.verify", workbench.VerifyParams{Duration: &zero})
			capture := completedVerification(t, source, source.Snapshot().LastOperation)
			path := filepath.Join(t.TempDir(), "dc.json")
			agentAction(t, source, "verification.export", workbench.VerificationExportParams{ID: capture.Operation.ID, Path: path, Format: "json"})

			s, cleanup := testScreen(t)
			defer cleanup()
			agentAction(t, s.service, "example.rc", workbench.ReplaceParams{})
			agentAction(t, s.service, "simulation.pause", struct{}{})
			duration := .002
			agentAction(t, s.service, "simulation.verify", workbench.VerifyParams{Duration: &duration})
			transient := completedVerification(t, s.service, s.service.Snapshot().LastOperation).Operation.ID
			agentAction(t, s.service, "recording.view.set", workbench.RecordingViewParams{Mode: "recorded"})
			uiAction(t, s, "recorded-index", &ui.ChangeEvent{Number: 2})
			agentAction(t, s.service, "verification.import", workbench.RecordingImportParams{Path: path})
			dc := s.service.Snapshot().Recording.ID

			for _, id := range []string{dc, transient, dc, transient} {
				agentAction(t, s.service, "operation.select", workbench.IDParams{ID: id})
				before := s.service.Snapshot()
				s.app.Do(func() {
					root := s.view()
					if err := ui.Validate(root); err != nil {
						t.Fatal("recorded scope cannot render", err)
					}
					node, ok := findNode(root, "recorded-index")
					if !ok || node.Accessibility.Name != "Recorded sample index" {
						t.Fatal("sample position is unavailable")
					}
					if id == dc {
						if before.Recording.Total != 1 || before.Recording.Index != 0 || node.Kind != ui.Label || node.Text != "0" || node.Change != nil {
							t.Fatal("single saved row must have a readable fixed index", node)
						}
						for _, name := range []string{"recorded-previous", "recorded-next", "recorded-play"} {
							control, ok := findNode(root, name)
							if !ok || !control.Disabled {
								t.Fatal("single-row navigation should be unavailable", name)
							}
						}
					} else if node.Kind != ui.NumberField || node.Numeric.Max != float32(before.Recording.Total-1) || node.Change == nil {
						t.Fatal("transient recording lost exact index seeking", node)
					}
				})
				if after := s.service.Snapshot(); !reflect.DeepEqual(before, after) {
					t.Fatal("rendering changed source or captured state")
				}
				if id == transient {
					uiAction(t, s, "recorded-index", &ui.ChangeEvent{Number: 1})
					if s.service.Snapshot().Recording.Index != 1 {
						t.Fatal("transient index control did not seek")
					}
				} else {
					index := 0
					agentAction(t, s.service, "recording.seek", workbench.RecordingSeekParams{Index: &index})
					if s.service.Snapshot().Recording.Index != 0 {
						t.Fatal("public DC index zero is unavailable")
					}
				}
			}
		})
	}
}
