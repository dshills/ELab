package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func actorCommand(t *testing.T, s *Service, actor, command string, p any) Response {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	rev := s.Snapshot().Revision
	return s.Dispatch(context.Background(), Request{Version: 1, ID: actor + command, Command: command, Params: raw, ExpectedRevision: &rev, Key: actor + command + string(raw)}, actor)
}
func TestHistoryExactTipAndActorAcknowledgment(t *testing.T) {
	s := NewExample()
	defer s.Close()
	settled(t, s)
	a := actorCommand(t, s, "agent", "component.set", ValueParams{ID: "R1", Value: "2k"})
	if a.Error != nil {
		t.Fatal(a.Error)
	}
	own := s.Snapshot().UndoEntry
	if own == nil || own.Actor != "agent" || own.CommandRevision == 0 {
		t.Fatal("history ownership/tip missing")
	}
	u := actorCommand(t, s, "user", "component.set", ValueParams{ID: "R1", Value: "3k"})
	if u.Error != nil {
		t.Fatal(u.Error)
	}
	before := s.Snapshot()
	denied := actorCommand(t, s, "agent", "edit.undo", HistoryParams{ExpectedEntryRevision: &own.CommandRevision})
	if denied.Error == nil || denied.Error.Code != "history-conflict" || !reflect.DeepEqual(s.Snapshot().Components, before.Components) {
		t.Fatal("stale batch target reversed another actor")
	}
	denied = actorCommand(t, s, "agent", "edit.undo", HistoryParams{})
	if denied.Error == nil || denied.Error.Code != "history-conflict" {
		t.Fatal("other actor was reversed without acknowledgment")
	}
	current := s.Snapshot().UndoEntry.CommandRevision
	accepted := actorCommand(t, s, "agent", "edit.undo", HistoryParams{ExpectedEntryRevision: &current, AllowOtherActor: true})
	if accepted.Error != nil || s.Snapshot().Components[1].Value != 2000 || s.Snapshot().RedoEntry.Actor != "user" || s.Snapshot().RedoEntry.CommandRevision != current {
		t.Fatal("explicit history acknowledgment lost identity or graph")
	}
	redo := actorCommand(t, s, "agent", "edit.redo", HistoryParams{ExpectedEntryRevision: &current, AllowOtherActor: true})
	if redo.Error != nil || s.Snapshot().Components[1].Value != 3000 {
		t.Fatal("guarded redo failed")
	}
}
