package workbench

import (
	"encoding/json"
	"fmt"
	"maps"
)

type EditProposal struct {
	ID      string       `json:"id"`
	Actor   string       `json:"actor"`
	Stale   bool         `json:"stale"`
	Preview BatchPreview `json:"preview"`
	Batch   BatchParams  `json:"batch"`
}

type editProposalState struct {
	view    EditProposal
	source  [32]byte
	usedIDs map[string]bool
}

func (s *Service) proposalViewLocked() *EditProposal {
	if s.proposal == nil || s.proposal.view.Preview.DocumentGeneration != s.documentGeneration {
		return nil
	}
	out := cloneEditProposal(&s.proposal.view)
	out.Stale = s.proposal.source != contentHash(s.graph, s.transientSettings) || !maps.Equal(s.proposal.usedIDs, s.usedIDs)
	return out
}

func cloneEditProposal(in *EditProposal) *EditProposal {
	if in == nil {
		return nil
	}
	out := *in
	out.Preview.Changes = cloneEditChanges(out.Preview.Changes)
	out.Batch.Edits = append([]BatchEdit{}, in.Batch.Edits...)
	for i := range out.Batch.Edits {
		out.Batch.Edits[i].Params = append(json.RawMessage(nil), in.Batch.Edits[i].Params...)
	}
	return &out
}

func (s *Service) proposalCommandLocked(command string, raw json.RawMessage) error {
	if command == "edit.proposal.set" {
		var p BatchParams
		if err := params(raw, &p, "name", "edits"); err != nil {
			return err
		}
		preview, err := s.previewBatchLocked(p)
		if err != nil {
			return err
		}
		s.proposalSerial++
		s.proposal = &editProposalState{view: EditProposal{ID: fmt.Sprintf("proposal-%d", s.proposalSerial), Preview: *preview, Batch: p}, source: contentHash(s.graph, s.transientSettings), usedIDs: maps.Clone(s.usedIDs)}
		return nil
	}
	var p IDParams
	if err := params(raw, &p, "id"); err != nil {
		return err
	}
	view := s.proposalViewLocked()
	if view == nil || view.ID != p.ID {
		return &actionError{"conflict", "The proposal changed or belongs to a previous document; inspect the current proposal"}
	}
	if command == "edit.proposal.apply" {
		if view.Stale {
			return &actionError{"conflict", "The circuit changed after this preview; refresh and review the proposal before applying"}
		}
		if err := s.batchLocked(view.Batch); err != nil {
			return err
		}
	}
	s.proposal = nil
	return nil
}
