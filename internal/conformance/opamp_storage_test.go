package conformance

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestOpAmpStoragePublicRLCAndPMOSFileBoundaries(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	result, err := RunOpAmpStorage(ctx, InProcess, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 2 || len(result.Checks) != opAmpStorageReferenceCount {
		t.Fatal("storage numerical coverage", len(result.Cases), len(result.Checks))
	}
	if err := CompareOpAmpStorage(result, result); err != nil {
		t.Fatal("valid storage evidence refused", err)
	}
	for _, entry := range result.Cases {
		if len(entry.Observations) != 4 || len(entry.Refusals) != 2*len(storageCorruptionNames(entry.Name)) || len(entry.ReopenCalls) == 0 {
			t.Fatal("missing storage state/file boundary evidence", entry.Name)
		}
	}
	for _, mutation := range []string{"missing-refusal", "wrong-refusal", "wrong-probe", "changed-graph", "changed-row", "changed-fine-row", "missing-reopen"} {
		t.Run(mutation, func(t *testing.T) {
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var changed OpAmpStorageResult
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "missing-refusal":
				changed.Cases[0].Refusals = changed.Cases[0].Refusals[1:]
			case "wrong-refusal":
				changed.Cases[0].Refusals[0].ProjectUnchanged = false
			case "wrong-probe":
				changed.Cases[1].Probes[0] = "foreign"
			case "changed-graph":
				changed.Cases[0].Graph.Components[0].Value += 1
			case "changed-row":
				changed.Cases[0].Rows[1].Values[0] += .1
			case "changed-fine-row":
				changed.Cases[0].RefinedRows[1].Values[0] += .1
			case "missing-reopen":
				changed.Cases[0].ReopenCalls = nil
			}
			if err := CompareOpAmpStorage(result, changed); err == nil {
				t.Fatal("corrupted storage evidence accepted")
			}
		})
	}
}
