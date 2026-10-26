package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) proposalControls(state workbench.Snapshot) []ui.Node {
	p := state.Proposal
	if p == nil {
		return nil
	}
	nodes := []ui.Node{section("proposal-title", "Review proposed changes"), quiet("proposal-name", p.Preview.Name), quiet("proposal-actor", "Submitted by "+p.Actor), quiet("proposal-notice", p.Preview.Notice)}
	if p.Stale {
		nodes = append(nodes, quiet("proposal-stale", "The circuit changed. Refresh and review the new differences before applying."))
	}
	if len(p.Preview.Changes) == 0 {
		nodes = append(nodes, quiet("proposal-unchanged", "This batch leaves component, wire and experiment contents unchanged."))
	}
	for i, change := range p.Preview.Changes {
		nodes = append(nodes, quiet(fmt.Sprintf("proposal-change-%d", i), change.Summary))
		for j, detail := range change.Details {
			nodes = append(nodes, quiet(fmt.Sprintf("proposal-change-%d-detail-%d", i, j), detail))
		}
	}
	apply := s.action("proposal-apply", "Apply changes", "edit.proposal.apply", workbench.IDParams{ID: p.ID})
	apply.Disabled = p.Stale
	refresh := s.action("proposal-refresh", "Refresh preview", "edit.proposal.set", p.Batch)
	dismiss := s.action("proposal-dismiss", "Dismiss proposal", "edit.proposal.dismiss", workbench.IDParams{ID: p.ID})
	return append(nodes, apply, refresh, dismiss)
}
