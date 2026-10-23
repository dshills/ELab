package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCursorTimeGuidanceExplainsActualBoundsWithoutMutation(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name := "live"
		if recorded {
			name = "recorded"
		}
		t.Run(name, func(t *testing.T) {
			s := pausedRC(t)
			defer s.Close()
			command := "scope.cursor.set"
			if recorded {
				duration := .02
				verify(t, s, VerifyParams{Duration: &duration}, "record")
				command = "recording.cursor.set"
			} else {
				call(t, s, "simulation.step", StepParams{Duration: .02}, "samples")
			}
			for i, bad := range []any{"oops", "10 Hz", "-1 ms", "30 ms", -1, .03} {
				before := s.Snapshot()
				raw, _ := json.Marshal(map[string]any{"cursor": "b", "time_s": bad})
				revision := before.CommandRevision
				response := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: command, Params: raw, ExpectedCommandRevision: &revision}, "agent")
				if response.Error == nil || response.Help == nil || response.Error.Code != "invalid-params" || !strings.Contains(response.Help.Summary, "cursor B") || !strings.Contains(response.Help.Action, "0 s to 0.02 s") || !strings.Contains(response.Help.Action, "ms") {
					t.Fatal("refusal hides actual cursor/time domain", response)
				}
				if response.Help.Details != response.Error.Message || !reflect.DeepEqual(before, s.Snapshot()) {
					t.Fatal("help changed accepted state or original error")
				}
			}
			if !recorded {
				call(t, s, "simulation.step", StepParams{Duration: .6}, "roll-history")
				before := s.Snapshot()
				start := before.Transient.Samples[0].Time
				if start <= 0 {
					t.Fatal("fixture must expire early samples")
				}
				raw := json.RawMessage(`{"cursor":"a","time_s":0}`)
				revision := before.CommandRevision
				r := s.Dispatch(context.Background(), Request{Version: 1, ID: "rolled", Key: "rolled", Command: command, Params: raw, ExpectedCommandRevision: &revision}, "agent")
				if r.Help == nil || !strings.Contains(r.Help.Action, fmt.Sprintf("from %g s", start)) || !reflect.DeepEqual(before, s.Snapshot()) {
					t.Fatal("help suggests expired live history", r.Help)
				}
			}
		})
	}
}

func TestCursorTimeGuidanceDoesNotBorrowMalformedOrStaleContext(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{Duration: .02}, "samples")
	for i, raw := range []string{
		`{"cursor":"a","time_s":"oops","extra":true}`,
		`{"cursor":"c","time_s":"oops"}`,
		`{"cursor":"a","time_s":true}`,
		`{"cursor":"a","time_s":null}`,
		`{"cursor":"a","time_s":"oops","operation_id":""}`,
		`{"cursor":"a","cursor":"b","time_s":"oops"}`,
	} {
		before := s.Snapshot()
		revision := before.CommandRevision
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: "scope.cursor.set", Params: json.RawMessage(raw), ExpectedCommandRevision: &revision}, "agent")
		if r.Error == nil || r.Help == nil || r.Help.Summary != "That request could not be applied." || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("malformed input acquired invented time guidance", r)
		}
	}
	before := s.Snapshot()
	oldRevision := before.CommandRevision
	oldGeneration := before.Document.Generation
	call(t, s, "simulation.restart", struct{}{}, "new-run")
	current := s.Snapshot()
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "old-run", Key: "old-run", Command: "scope.cursor.set", Params: json.RawMessage(`{"cursor":"a","time_s":"oops"}`), ExpectedCommandRevision: &oldRevision, ExpectedDocumentGeneration: &oldGeneration}, "agent")
	if r.Error == nil || r.Help == nil || r.Help.Summary != "That request could not be applied." || !reflect.DeepEqual(current, s.Snapshot()) {
		t.Fatal("stale parser refusal described another run", r)
	}
}
