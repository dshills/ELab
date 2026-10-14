package workbench

import (
	"encoding/json"
	"fmt"
)

type HistoryTip struct {
	Name            string `json:"name"`
	Actor           string `json:"actor"`
	CommandRevision uint64 `json:"command_revision"`
}
type HistoryParams struct {
	ExpectedEntryRevision *uint64 `json:"expected_entry_revision,omitempty"`
	AllowOtherActor       bool    `json:"allow_other_actor,omitempty"`
}

func historyTip(entry historyEntry) *HistoryTip {
	return &HistoryTip{entry.Name, entry.Actor, entry.CommandRevision}
}
func (s *Service) checkHistoryTargetLocked(command string, raw json.RawMessage, actor string) error {
	var p HistoryParams
	if err := params(raw, &p); err != nil {
		return err
	}
	entries := s.undo
	if command == "edit.redo" {
		entries = s.redo
	}
	if len(entries) == 0 {
		return &actionError{"unavailable", "nothing to " + command[5:]}
	}
	tip := entries[len(entries)-1]
	if p.ExpectedEntryRevision != nil && *p.ExpectedEntryRevision != tip.CommandRevision {
		return &actionError{"history-conflict", fmt.Sprintf("history tip changed: expected command revision %d; current %d by %s", *p.ExpectedEntryRevision, tip.CommandRevision, tip.Actor)}
	}
	if tip.Actor != actor && !p.AllowOtherActor {
		return &actionError{"history-conflict", "Latest edit belongs to " + tip.Actor + "; inspect undo_entry/redo_entry and explicitly acknowledge allow_other_actor to target it"}
	}
	return nil
}
