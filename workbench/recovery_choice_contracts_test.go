package workbench

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

var recoveryChoiceCommands = []string{"recovery.restore", "recovery.restore.start", "recovery.dismiss", "recovery.dismiss.start", "recovery.inspect", "recovery.samples"}

func TestRecoveryChoiceContractsAreOwnedAndLeaveOrdinaryDiscoveryUnchanged(t *testing.T) {
	before, _ := json.Marshal(Catalog())
	registry, _ := json.Marshal(SchemaRegistry())
	for _, id := range recoveryChoiceCommands {
		details, err := detailedCapabilities(id)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(details[0].Input, &schema); err != nil {
			t.Fatal(err)
		}
		contract := schema["x-command-contract"].(map[string]any)
		if contract["scope"] != "recovery-choice-and-query" || schema["x-file-bounds"] == nil || len(contract["effects"].([]any)) == 0 {
			t.Fatal("incomplete recovery choice contract", id)
		}
		// These legacy paths accept arbitrary hash strings until read/validation;
		// a stricter observable-read admission rule must not leak into their schema.
		if inputProperty(schema, "sha256")["pattern"] != nil {
			t.Fatal("discovery incorrectly rejects a legacy hash at admission", id)
		}
		details[0].Input[0] = '!'
		fresh, err := detailedCapabilities(id)
		if err != nil || !json.Valid(fresh[0].Input) {
			t.Fatal("caller changed subsequent details", err)
		}
	}
	after, _ := json.Marshal(Catalog())
	output, _ := json.Marshal(SchemaRegistry())
	if !bytes.Equal(before, after) || !bytes.Equal(registry, output) {
		t.Fatal("choice details changed ordinary discovery/output schemas")
	}
}

func TestRecoveryLegacyQueryDefaultsIgnoredContextAndErrorPrecedence(t *testing.T) {
	s, choice, _, _ := recoveryQueryFixture(t)
	defer s.Close()
	before := s.Snapshot()
	wrongGeneration := before.Document.Generation + 10
	choice.DiscardChanges, choice.ExpectedGeneration = true, &wrongGeneration
	inspection := call(t, s, "recovery.inspect", choice, "compatibility").Result.(RecoveryInspection)
	if inspection.Candidate.ID != choice.ID || len(inspection.Document.Recordings) != 1 {
		t.Fatal("compatibility context interfered with a pure read")
	}
	base := map[string]any{"id": choice.ID, "sha256": choice.SHA256, "recording_index": 0}
	page := call(t, s, "recovery.samples", base, "default").Result.(VerificationPage)
	base["offset"], base["limit"] = nil, nil
	optionalNull := call(t, s, "recovery.samples", base, "null").Result.(VerificationPage)
	if !reflect.DeepEqual(page, optionalNull) || page.ID != choice.ID || page.Total != 11 || len(page.Rows) != 11 {
		t.Fatal("legacy null/default page or identity differs")
	}
	base["offset"], base["limit"] = page.Total, 200
	eof := call(t, s, "recovery.samples", base, "eof").Result.(VerificationPage)
	if len(eof.Rows) != 0 || eof.NextOffset != page.Total || eof.Offset != page.Total {
		t.Fatal("inclusive EOF lost")
	}
	base["limit"] = 201
	refuse(t, s, "recovery.samples", base, "invalid-params")
	base["sha256"] = "old"
	refuse(t, s, "recovery.samples", base, "file-conflict")
	base["id"] = "invalid"
	refuse(t, s, "recovery.samples", base, "invalid-recovery")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("successful/refused pure queries changed public state")
	}
}

func TestRecoveryRestoreDefersFileValidationAndReplacesSessionProbeChanges(t *testing.T) {
	s := manualRecoveryService(t)
	defer s.Close()
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"v:C1"}}, "captured-probe")
	call(t, s, "recovery.checkpoint", struct{}{}, "checkpoint")
	choice := recoveryChoice(t, s)
	before := s.Snapshot()
	bad := RecoveryChoiceParams{ID: "invalid", SHA256: "old"}
	refuse(t, s, "recovery.restore.start", bad, "unsaved-changes")
	if !reflect.DeepEqual(before.ProjectWork, s.Snapshot().ProjectWork) {
		t.Fatal("dirty refusal admitted file work")
	}
	bad.DiscardChanges = true
	call(t, s, "recovery.restore.start", bad, "invalid-file")
	failed := awaitProjectStatus(t, s, "failed")
	if failed.ProjectWork.Error.Code != "invalid-recovery" || failed.Document != before.Document || !reflect.DeepEqual(failed.Components, before.Components) {
		t.Fatal("file validation was not a guarded admitted failure")
	}
	choice.DiscardChanges = true
	entered, release := blockRecoveryRestore(t, s)
	defer release()
	admitted := call(t, s, "recovery.restore.start", choice, "valid").Result.(Snapshot)
	awaitLiveSampleBlock(t, entered)
	call(t, s, "scope.probes.set", ProbeParams{IDs: []string{"i:C1"}}, "session-probe")
	call(t, s, "view.pan", map[string]float64{"dx": 20, "dy": 10}, "session-navigation")
	release()
	after := awaitProjectStatus(t, s, "completed")
	call(t, s, "simulation.wait", struct{}{}, "startup")
	s.mu.Lock()
	probes := append([]string{}, s.scopeProbes...)
	s.mu.Unlock()
	if !reflect.DeepEqual(probes, []string{"v:C1"}) || after.View != before.View || after.Document.Generation != before.Document.Generation+1 || after.ProjectWork.ResultRun != admitted.RunID+1 {
		t.Fatal("session-only navigation/probes superseded restore or survived captured installation")
	}
	// A pure query remains a separate filesystem path after the restore.
	response := awaitRecoveryQuery(t, recoveryQueryRequest(t, s, context.Background(), "recovery.inspect", choice))
	if response.Error != nil {
		t.Fatal(response.Error)
	}
}
