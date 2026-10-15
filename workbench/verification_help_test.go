package workbench

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestVerificationFailureHelpUsesCapturedContextWithoutGuessedMarkers(t *testing.T) {
	for _, code := range []string{"integration-attempt-limit", "integration-time-limit", "timeout", "solve-failed", "unknown-solver-reason"} {
		operation := OperationSummary{Kind: "verify", Status: "failed", SourceUnchanged: true, Error: &Failure{Code: code, Message: "original technical reason"}}
		help := verificationFailureHelp(operation)
		if help == nil || help.Code != code || help.Details != operation.Error.Message || help.Source != "verification" || help.Summary == "" || help.Action == "" || strings.Contains(help.Action, "Restart") || !strings.Contains(help.Action, "No partial recording") || len(help.Objects)+len(help.Terminals)+len(help.Wires) != 0 {
			t.Fatal("failure guidance lost captured scope or guessed causes", code, help)
		}
		operation.SourceUnchanged = false
		earlier := verificationFailureHelp(operation)
		if !strings.Contains(earlier.Summary, "earlier captured") || earlier.Code != code || earlier.Details != help.Details {
			t.Fatal("changed live source obscures captured failure", code, earlier)
		}
	}
	for _, status := range []string{"queued", "running", "canceling", "completed", "canceled"} {
		if verificationFailureHelp(OperationSummary{Kind: "verify", Status: status, Error: &Failure{Code: "solve-failed"}}) != nil {
			t.Fatal("nonfailed operation acquired failure guidance", status)
		}
	}
}

func TestFailedVerificationHelpIsOwnedAndPreservesLiveAndKeptResults(t *testing.T) {
	s := New()
	defer s.Close()
	call(t, s, "example.rectifier", ReplaceParams{}, "rectifier")
	call(t, s, "simulation.pause", struct{}{}, "pause")
	settled(t, s)
	duration := .001
	_, kept := verify(t, s, VerifyParams{Duration: &duration, ProbeIDs: []string{"v:C1"}}, "kept")
	before := s.Snapshot()
	duration = math.Nextafter(duration, math.Inf(1))
	response := call(t, s, "simulation.verify", VerifyParams{Duration: &duration}, "tiny-tail")
	identity := response.Result.(Snapshot).LastOperation
	failed := awaitOperation(t, s, identity)
	help := failed.Operation.FailureHelp
	if failed.Operation.Status != "failed" || failed.Operation.Error == nil || failed.Operation.Error.Code != "solve-failed" || help == nil || !strings.Contains(help.Details, "mixed timestep too small") || failed.Metadata == nil {
		t.Fatal("real numerical failure lacks captured guidance", failed.Operation)
	}
	refuse(t, s, "operation.samples", VerificationPageParams{ID: identity}, "not-ready")
	after := s.Snapshot()
	if before.Document != after.Document || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) || !reflect.DeepEqual(kept.Metadata, inspectOperation(t, s, kept.Operation.ID).Metadata) {
		t.Fatal("failed verification changed live state or kept result")
	}
	stable := s.Snapshot()
	query := inspectOperation(t, s, identity)
	if !reflect.DeepEqual(stable, s.Snapshot()) {
		t.Fatal("reading failure guidance mutates state")
	}
	query.Operation.FailureHelp.Summary = "caller mutation"
	if inspectOperation(t, s, identity).Operation.FailureHelp.Summary == "caller mutation" {
		t.Fatal("inspection guidance aliases service state")
	}
	selected := call(t, s, "operation.select", IDParams{ID: identity}, "selected-failure")
	copy := selected.Result.(Snapshot)
	for i := range copy.Operations {
		if copy.Operations[i].ID == identity {
			copy.Operations[i].FailureHelp.Summary = "caller mutation"
		}
	}
	retry := call(t, s, "operation.select", IDParams{ID: identity}, "selected-failure")
	for _, operation := range retry.Result.(Snapshot).Operations {
		if operation.ID == identity && operation.FailureHelp.Summary == "caller mutation" {
			t.Fatal("cached selection guidance aliases caller")
		}
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "200 ohm"}, "later-source")
	earlier := inspectOperation(t, s, identity)
	if earlier.Operation.SourceUnchanged || !strings.Contains(earlier.Operation.FailureHelp.Summary, "earlier captured") || !reflect.DeepEqual(earlier.Metadata, failed.Metadata) || earlier.Operation.Error.Message != failed.Operation.Error.Message || len(earlier.Operation.FailureHelp.Objects) != 0 {
		t.Fatal("later circuit substituted a captured failure or invented a marker")
	}
}
