package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRecordedPositionGuidanceCannotBorrowStaleCaptureContext(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	verify(t, s, VerifyParams{Duration: &duration}, "original")
	original := s.Snapshot()
	duration = .01
	verify(t, s, VerifyParams{Duration: &duration}, "replacement")
	current := s.Snapshot()
	if current.Recording.ID == original.Recording.ID {
		t.Fatal("fixture must select another completed capture")
	}
	wrongRevision, wrongGeneration := current.Revision+1, current.Document.Generation+1
	for _, entry := range []struct {
		name    string
		request Request
	}{
		{"old-command", Request{ExpectedCommandRevision: &original.CommandRevision}},
		{"other-presentation", Request{ExpectedRevision: &wrongRevision}},
		{"other-document", Request{ExpectedCommandRevision: &current.CommandRevision, ExpectedDocumentGeneration: &wrongGeneration}},
	} {
		r := entry.request
		r.Version, r.ID, r.Key, r.Command = 1, entry.name, entry.name, "recording.seek"
		r.Params = json.RawMessage(`{"time_s":"oops"}`)
		response := s.Dispatch(context.Background(), r, "agent")
		if response.Error == nil || response.Error.Code != "invalid-params" || response.Help == nil || response.Help.Summary != "That request could not be applied." || !reflect.DeepEqual(current, s.Snapshot()) {
			t.Fatalf("%s borrowed the current capture's range: %+v", entry.name, response.Help)
		}
		if response.Help.Details != response.Error.Message {
			t.Fatal("context guard changed original technical error")
		}
	}
}

func TestRecordedPositionGuidancePreservesKnownNullableAndNumericIntent(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	verify(t, s, VerifyParams{Duration: &duration}, "capture")
	for i, raw := range []string{
		`{"index":1.5}`,
		`{"index":-1}`,
		`{"index":1e100}`,
		`{"time_s":"1e309 s"}`,
		`{"time_s":"10 Hz","index":null}`,
		`{"index":0,"time_s":0}`,
		`{"index":null,"time_s":null}`,
		`{}`,
	} {
		before := s.Snapshot()
		revision := before.CommandRevision
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: "recording.seek", Params: json.RawMessage(raw), ExpectedCommandRevision: &revision}, "agent")
		if response.Error == nil || response.Help == nil || !strings.Contains(response.Help.Action, "whole sample index from 0 to 20") || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatalf("known seek intent %s lost useful whole-index/time guidance: %+v", raw, response.Help)
		}
	}
}

func TestRecordedPositionGuidanceDoesNotInventIntentForMalformedFields(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	verify(t, s, VerifyParams{Duration: &duration}, "capture")
	for i, raw := range []string{
		`{"time_s":"oops","extra":true}`,
		`{"time_s":"oops","time_s":"1 ms"}`,
		`{"time_s":true}`,
		`{"time_s":[]}`,
		`{"index":"1"}`,
		`{"index":false}`,
		`{"index":{}}`,
		`{"time_s":1e309}`,
		`null`,
		`[]`,
	} {
		before := s.Snapshot()
		revision := before.CommandRevision
		response := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: "recording.seek", Params: json.RawMessage(raw), ExpectedCommandRevision: &revision}, "agent")
		if response.Error == nil || response.Error.Code != "invalid-params" || response.Help == nil || response.Help.Summary != "That request could not be applied." || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatalf("malformed %s acquired position guidance: %+v", raw, response.Help)
		}
	}
}
