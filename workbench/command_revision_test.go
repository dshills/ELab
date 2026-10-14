package workbench

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestCommandRevisionGuardsActorsWithoutClockConflicts(t *testing.T) {
	s := newGraphService(SolveLED, RCExampleGraph())
	defer s.Close()
	before := s.Snapshot()
	deadline := time.Now().Add(time.Second)
	for s.Snapshot().Transient.Sample.Time == 0 {
		if time.Now().After(deadline) {
			t.Fatal("clock did not publish")
		}
		time.Sleep(time.Millisecond)
	}
	commandRevision, revision := before.CommandRevision, before.Revision
	raw := json.RawMessage(`{"dx":20,"dy":10}`)
	request := Request{Version: 1, ID: "command-guard", Command: "view.pan", Params: raw, ExpectedCommandRevision: &commandRevision, Key: "command-guard"}
	strict := request
	strict.Key = "strict"
	strict.ExpectedRevision = &revision
	if r := s.Dispatch(context.Background(), strict, "agent"); r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("combined guards ignored stale full-state revision")
	}
	accepted := s.Dispatch(context.Background(), request, "agent")
	if accepted.Error != nil || accepted.Result.(Snapshot).CommandRevision != commandRevision+1 {
		t.Fatal("clock incorrectly invalidated command guard", accepted.Error)
	}
	call(t, s, "simulation.pause", struct{}{}, "user-command")
	s.liveWorkers.Wait()
	current := s.Snapshot()
	stale := request
	stale.Key = "new-key"
	if r := s.Dispatch(context.Background(), stale, "agent"); r.Error == nil || r.Error.Code != "conflict" || !reflect.DeepEqual(s.Snapshot().View, current.View) {
		t.Fatal("intervening actor command was silently overwritten")
	}
	duplicate := s.Dispatch(context.Background(), request, "agent")
	if duplicate.Error != nil || !duplicate.Duplicate || !reflect.DeepEqual(duplicate.Result, accepted.Result) || s.Snapshot().CommandRevision != current.CommandRevision {
		t.Fatal("command-guard retry lost original outcome or committed twice")
	}
	missing := request
	missing.Key, missing.ExpectedCommandRevision = "missing", nil
	if r := s.Dispatch(context.Background(), missing, "agent"); r.Error == nil || r.Error.Code != "invalid-request" {
		t.Fatal("mutation admitted without either guard")
	}
	revision, commandRevision = current.Revision, current.CommandRevision
	both := request
	both.Key, both.ExpectedRevision = "both", &revision
	if r := s.Dispatch(context.Background(), both, "agent"); r.Error != nil {
		t.Fatal("matching combined guards refused", r.Error)
	}
	catalog := s.Dispatch(context.Background(), Request{Version: 1, ID: "discover", Command: "capabilities.list", Params: json.RawMessage(`{}`)}, "agent")
	guards := catalog.Result.(map[string]any)["mutation_preconditions"].(map[string]string)
	if guards["expected_command_revision"] == "" || guards["expected_revision"] == "" {
		t.Fatal("guard options absent from discovery")
	}
}
