package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func resistanceProposal(value string) workbench.BatchParams {
	raw, _ := json.Marshal(workbench.ValueParams{ID: "R1", Value: value})
	return workbench.BatchParams{Name: "Tune the RC response", Edits: []workbench.BatchEdit{{Command: "component.set", Params: raw}}}
}

func TestProposalReviewControlsShareGuardedCommandsAndUndo(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	before := s.service.Snapshot()
	agentAction(t, s.service, "edit.proposal.set", resistanceProposal("20k"))
	for _, page := range []string{"component", "simulation", "document"} {
		uiAction(t, s, "inspector-section-"+page, nil)
		for _, id := range []string{"proposal-title", "proposal-apply", "proposal-refresh", "proposal-dismiss"} {
			if _, ok := findNode(s.view(), id); !ok {
				t.Fatal("proposal not reviewable from current task", page, id)
			}
		}
	}
	var oldApply ui.Node
	s.app.Do(func() { oldApply, _ = findNode(s.view(), "proposal-apply") })
	agentAction(t, s.service, "component.set", workbench.ValueParams{ID: "R1", Value: "30k"})
	stable := s.service.Snapshot()
	s.app.Do(func() { oldApply.Activate() })
	if s.errorMessage == "" || !reflect.DeepEqual(stable, s.service.Snapshot()) {
		t.Fatal("held Apply overwrote newer edit")
	}
	s.app.Do(func() { s.errorMessage = "" })
	apply, _ := findNode(s.view(), "proposal-apply")
	if !apply.Disabled {
		t.Fatal("stale proposal Apply remains enabled")
	}
	uiAction(t, s, "proposal-refresh", nil)
	uiAction(t, s, "proposal-apply", nil)
	after := s.service.Snapshot()
	part, _ := (workbench.Graph{Components: after.Components}).Component("R1")
	if after.Proposal != nil || part.Value != 20000 || after.UndoName != "Tune the RC response" || after.UndoEntry.Actor != "user" || len(after.Components) != len(before.Components) {
		t.Fatal("review Apply did not use exact shared batch")
	}
	uiAction(t, s, "edit.undo", nil)
	part, _ = (workbench.Graph{Components: s.service.Snapshot().Components}).Component("R1")
	if part.Value != 30000 {
		t.Fatal("one Undo failed to restore the user's pre-apply circuit")
	}
}

func TestActiveExperimentAndExplanationUseSharedStateAcrossThemes(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "R1"})
	uiAction(t, s, "value-entry", &ui.ChangeEvent{Text: "20k"})
	uiAction(t, s, "apply-value", nil)
	for _, mode := range []string{"light", "dark", "system", "apollo"} {
		agentAction(t, s.service, "appearance.theme.set", map[string]string{"theme": mode})
		uiAction(t, s, "inspector-section-component", nil)
		state := s.service.Snapshot()
		summary, ok := findNode(s.view(), "component-explanation-summary")
		if !ok || state.Explanation == nil || summary.Text != state.Explanation.Summary || !strings.Contains(summary.Text, "Previous") {
			t.Fatal("visible explanation lost accepted-frame provenance", mode)
		}
		uiAction(t, s, "inspector-section-simulation", nil)
		for _, id := range []string{"example-question", "example-try", "example-expected", "example-change-0", "example-limitation"} {
			if n, ok := findNode(s.view(), id); !ok || n.Text == "" {
				t.Fatal("active experiment guidance missing", mode, id)
			}
		}
		change, _ := findNode(s.view(), "example-change-0")
		if change.Text != state.Example.Changes[0].Summary {
			t.Fatal("visible experiment diff differs from agent state")
		}
	}
}

func TestPlotActionRevealsHiddenScopeAndKeepsOtherTraces(t *testing.T) {
	s, cleanup := testScreen(t)
	defer cleanup()
	uiAction(t, s, "example.rc", nil)
	uiAction(t, s, "simulation.pause", nil)
	agentAction(t, s.service, "scope.visibility.set", workbench.ScopeVisibilityParams{Visible: false})
	agentAction(t, s.service, "scope.probes.set", workbench.ProbeParams{IDs: []string{"i:C1"}})
	uiAction(t, s, "selection", &ui.ChangeEvent{SelectedID: "C1"})
	before := s.service.Snapshot()
	uiAction(t, s, "plot-v:C1", nil)
	after := s.service.Snapshot()
	if !after.ScopeVisible || after.TraceFocus == nil || after.TraceFocus.ProbeID != "v:C1" || !reflect.DeepEqual(after.Scope.Selected, []string{"i:C1", "v:C1"}) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.Components, after.Components) {
		t.Fatal("plot control did not reveal/focus or silently replaced a trace")
	}
}
