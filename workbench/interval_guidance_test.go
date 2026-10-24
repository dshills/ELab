package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestIntervalTimeGuidanceExplainsActualBoundsWithoutMutation(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		name, command := "live", "scope.window.set"
		if recorded {
			name, command = "recorded", "recording.window.set"
		}
		t.Run(name, func(t *testing.T) {
			s := pausedRC(t)
			defer s.Close()
			if recorded {
				duration := .02
				verify(t, s, VerifyParams{Duration: &duration}, "record")
			} else {
				call(t, s, "simulation.step", StepParams{Duration: .02}, "samples")
			}
			for i, pair := range [][2]any{{"oops", "20 ms"}, {"0 s", "10 Hz"}, {"-1 ms", "20 ms"}, {0, .03}, {"15 ms", "10 ms"}, {"10 ms", "10 ms"}} {
				before := s.Snapshot()
				p := map[string]any{"start_s": pair[0], "end_s": pair[1]}
				if !recorded {
					p["mode"] = "fixed"
				}
				raw, _ := json.Marshal(p)
				revision := before.CommandRevision
				r := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: command, Params: raw, ExpectedCommandRevision: &revision}, "agent")
				if r.Error == nil || r.Error.Code != "invalid-params" || r.Help == nil || !strings.Contains(r.Help.Summary, "interval") || !strings.Contains(r.Help.Action, "0 s to 0.02 s") || !strings.Contains(r.Help.Action, "before") || !strings.Contains(r.Help.Action, "ms") {
					t.Fatal("refusal hides interval range, order or units", r)
				}
				if r.Help.Details != r.Error.Message || !reflect.DeepEqual(before, s.Snapshot()) {
					t.Fatal("guidance changed original error or accepted state")
				}
			}
			if !recorded {
				call(t, s, "simulation.step", StepParams{Duration: .6}, "expire")
				before := s.Snapshot()
				revision := before.CommandRevision
				r := s.Dispatch(context.Background(), Request{Version: 1, ID: "expired", Key: "expired", Command: command, Params: json.RawMessage(`{"mode":"fixed","start_s":0,"end_s":0.02}`), ExpectedCommandRevision: &revision}, "agent")
				if before.Transient.Samples[0].Time <= 0 || r.Help == nil || !strings.Contains(r.Help.Action, fmt.Sprintf("from %g s", before.Transient.Samples[0].Time)) || !reflect.DeepEqual(before, s.Snapshot()) {
					t.Fatal("interval guidance suggests expired samples", r.Help)
				}
			}
		})
	}
}

func TestRecordedIntervalGuidanceUsesNormalizationAndCurrentCaptureOnly(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	duration := .02
	verify(t, s, VerifyParams{Duration: &duration}, "first")
	before := s.Snapshot()
	revision := before.CommandRevision
	raw := json.RawMessage(`{"start_s":-2e-13,"end_s":-1e-13}`)
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "collapsed", Key: "collapsed", Command: "recording.window.set", Params: raw, ExpectedCommandRevision: &revision}, "agent")
	if r.Error == nil || r.Help == nil || !strings.Contains(r.Help.Action, "0 s to 0.02 s") || !strings.Contains(r.Error.Message, "positive width") || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("normalized zero-width refusal has no useful interval guidance", r)
	}
	originalAction := r.Help.Action
	duration = .01
	verify(t, s, VerifyParams{Duration: &duration}, "second")
	current := s.Snapshot()
	for _, entry := range []struct {
		params string
		guard  uint64
	}{
		{`{"start_s":"oops","end_s":0.01}`, revision},
		{`{"mode":"","start_s":"oops","end_s":0.01}`, current.CommandRevision},
		{`{"start_s":"oops","end_s":0.01,"extra":true}`, current.CommandRevision},
	} {
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: entry.params, Key: entry.params, Command: "recording.window.set", Params: json.RawMessage(entry.params), ExpectedCommandRevision: &entry.guard}, "agent")
		if r.Error == nil || r.Help == nil || r.Help.Summary != "That request could not be applied." || !reflect.DeepEqual(current, s.Snapshot()) {
			t.Fatal("stale or malformed recording interval acquired current context", r)
		}
	}
	if r.Help.Action != originalAction {
		t.Fatal("later selection changed the original refusal")
	}
}

func TestRecordedIntervalGuidanceForDCPoint(t *testing.T) {
	s := NewExample()
	defer s.Close()
	zero := 0.
	verify(t, s, VerifyParams{Duration: &zero}, "point")
	before := s.Snapshot()
	revision := before.CommandRevision
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "bad-point", Key: "bad-point", Command: "recording.window.set", Params: json.RawMessage(`{"start_s":"oops","end_s":0}`), ExpectedCommandRevision: &revision}, "agent")
	if r.Error == nil || r.Help == nil || r.Help.Summary != "This recording has one time point." || !strings.Contains(r.Help.Action, "start 0 s and end 0 s") || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("DC guidance requires a nonexistent positive interval", r)
	}
}

func TestIntervalTimeGuidanceDoesNotBorrowMalformedOrStaleContext(t *testing.T) {
	s := pausedRC(t)
	defer s.Close()
	call(t, s, "simulation.step", StepParams{Duration: .02}, "samples")
	for i, raw := range []string{
		`{"mode":"fixed","start_s":"oops","end_s":0.02,"extra":true}`,
		`{"mode":"fixed","start_s":null,"end_s":0.02}`,
		`{"mode":"fixed","start_s":true,"end_s":0.02}`,
		`{"mode":"fixed","start_s":"oops"}`,
		`{"mode":"invalid","start_s":"oops","end_s":0.02}`,
		`{"mode":"live","start_s":"oops","end_s":0.02}`,
		`{"mode":"fixed","start_s":"oops","start_s":0,"end_s":0.02}`,
	} {
		before := s.Snapshot()
		revision := before.CommandRevision
		r := s.Dispatch(context.Background(), Request{Version: 1, ID: fmt.Sprint(i), Key: fmt.Sprint(i), Command: "scope.window.set", Params: json.RawMessage(raw), ExpectedCommandRevision: &revision}, "agent")
		if r.Error == nil || r.Help == nil || r.Help.Summary != "That request could not be applied." || !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("malformed request acquired invented interval guidance", r)
		}
	}
	old := s.Snapshot().CommandRevision
	call(t, s, "simulation.restart", struct{}{}, "new-run")
	before := s.Snapshot()
	r := s.Dispatch(context.Background(), Request{Version: 1, ID: "old", Key: "old", Command: "scope.window.set", Params: json.RawMessage(`{"mode":"fixed","start_s":"oops","end_s":0.02}`), ExpectedCommandRevision: &old}, "agent")
	if r.Error == nil || r.Help == nil || r.Help.Summary != "That request could not be applied." || !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("stale parser refusal borrowed a new interval", r)
	}
}
