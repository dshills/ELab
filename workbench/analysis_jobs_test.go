package workbench

import (
	"fmt"
	"testing"
)

func TestAnalysisJobsAdmissionRetentionAndSelectionOwnership(t *testing.T) {
	jobs := analysisJobs{}
	if jobs.requireSweepSlotLocked() != nil || jobs.requireRecordingSlotLocked() != nil {
		t.Fatal("zero store not usable")
	}
	for i := 0; i < MaxDCSweepOperations; i++ {
		if err := jobs.retainSweepJobLocked(&sweepJob{summary: SweepSummary{ID: fmt.Sprint(i), Status: "completed"}, keep: true}); err != nil {
			t.Fatal(err)
		}
	}
	if jobs.requireSweepSlotLocked() == nil {
		t.Fatal("kept sweep admission weakened")
	}
	if err := jobs.retainSweepJobLocked(&sweepJob{summary: SweepSummary{ID: "refused", Status: "queued"}}); err == nil || len(jobs.sweeps.order) != MaxDCSweepOperations || jobs.sweeps.operations["refused"] != nil {
		t.Fatal("retention bypassed full-store admission")
	}
	store := jobs.ensureSweepsLocked()
	store.operations["0"].keep = false
	store.operations["0"].summary.Status = "running"
	if jobs.requireSweepSlotLocked() == nil {
		t.Fatal("active sweep admitted eviction")
	}
	store.operations["0"].summary.Status = "completed"
	if jobs.requireSweepSlotLocked() != nil {
		t.Fatal("excluded finished sweep not evictable")
	}
	cursor := 2
	store.selected, store.traceID, store.visible, store.cursor = "0", "trace", true, &cursor
	if err := jobs.retainSweepJobLocked(&sweepJob{summary: SweepSummary{ID: "next", Status: "queued"}}); err != nil {
		t.Fatal(err)
	}
	if len(store.order) != MaxDCSweepOperations || store.operations["0"] != nil || store.selected != "" || store.traceID != "" || store.visible || store.cursor != nil {
		t.Fatal("retention/selection not retired together")
	}
	for i := 1; i < MaxDCSweepOperations; i++ {
		if store.operations[fmt.Sprint(i)] == nil {
			t.Fatal("kept sweep evicted")
		}
	}
	jobs.operations = map[string]*verificationOperation{}
	for i := 0; i < MaxVerificationOperations; i++ {
		id := fmt.Sprint(i)
		jobs.operationOrder = append(jobs.operationOrder, id)
		jobs.operations[id] = &verificationOperation{keep: true, summary: OperationSummary{Status: "completed"}}
	}
	if jobs.requireRecordingSlotLocked() == nil {
		t.Fatal("kept recording admission weakened")
	}
	jobs.operations["0"].keep = false
	jobs.operations["0"].summary.Status = "running"
	if jobs.requireRecordingSlotLocked() == nil {
		t.Fatal("active recording evictable")
	}
	jobs.operations["0"].summary.Status = "completed"
	if jobs.requireRecordingSlotLocked() != nil {
		t.Fatal("excluded recording cannot release capacity")
	}
	other := analysisJobs{}
	if other.ensureSweepsLocked() == store || len(other.sweeps.order) != 0 || other.operations != nil {
		t.Fatal("independent stores share state")
	}
}
