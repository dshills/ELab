package conformance

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestProjectContractsPublicWorkflowAndReportGuards(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	report, err := RunProjectContracts(ctx, InProcess, filepath.Join(t.TempDir(), "workflow"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CompareProjectContracts(report, report); err != nil {
		t.Fatal(err)
	}
	// Different startup work changes counters, not the intended target.
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var shifted ProjectContractResult
	if err := json.Unmarshal(raw, &shifted); err != nil {
		t.Fatal(err)
	}
	for name, state := range shifted.States {
		if state.ProjectWork != nil {
			state.ProjectWork.ID = "shifted-" + state.ProjectWork.ID
		}
		shifted.States[name] = state
	}
	shifted.Historical.ID = "shifted-" + shifted.Historical.ID
	for i, c := range shifted.Calls {
		if c.Command == "project.work.inspect" || c.Command == "project.work.cancel" {
			var p map[string]string
			if err := json.Unmarshal(c.Params, &p); err != nil {
				t.Fatal(err)
			}
			p["id"] = "shifted-" + p["id"]
			shifted.Calls[i].Params, err = json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := CompareProjectContracts(report, shifted); err != nil {
		t.Fatal("equivalent observed job roles differ", err)
	}
	for _, name := range []string{"missing-schema", "wrong-worker-limit", "wrong-deadline", "wrong-observer", "missing-error", "missing-check", "missing-call", "changed-description", "missing-state", "changed-capture", "changed-outcome", "changed-loaded-time", "wrong-job-target", "changed-external-file"} {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			var broken ProjectContractResult
			if err := json.Unmarshal(raw, &broken); err != nil {
				t.Fatal(err)
			}
			var schema map[string]any
			if err := json.Unmarshal(broken.Schemas["project.save.start"], &schema); err != nil {
				t.Fatal(err)
			}
			contract := schema["x-command-contract"].(map[string]any)
			switch name {
			case "missing-schema":
				delete(broken.Schemas, "project.inspect")
			case "wrong-worker-limit":
				schema["x-project-work"].(map[string]any)["max_workers"] = 0
			case "wrong-deadline":
				contract["completion"].(map[string]any)["cooperative_deadline_s"] = 0
			case "wrong-observer":
				contract["completion"].(map[string]any)["observe"] = []string{"nonexistent"}
			case "missing-error":
				contract["errors"] = []any{}
			case "missing-check":
				broken.Checks = broken.Checks[1:]
			case "missing-call":
				for i, c := range broken.Calls {
					if c.Command == "project.work.cancel" {
						broken.Calls = append(broken.Calls[:i], broken.Calls[i+1:]...)
						break
					}
				}
			case "missing-state":
				delete(broken.States, "cleared")
			case "changed-capture":
				broken.Captured.Graph.Components[0].Value += 1
			case "changed-outcome":
				broken.Failures["external-save"] = "file-error"
			case "changed-loaded-time":
				broken.States["loaded"].Transient.Sample.Time = 1
			case "wrong-job-target":
				for i, c := range broken.Calls {
					if c.Command == "project.work.cancel" {
						broken.Calls[i].Params = []byte(`{"id":"project-unrelated"}`)
						break
					}
				}
			case "changed-external-file":
				broken.ExternalAfterRefusal = "overwritten"
			case "changed-description":
				schema["description"] = "Different adapter contract"
			}
			if name != "missing-schema" {
				broken.Schemas["project.save.start"], err = json.Marshal(schema)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := CompareProjectContracts(report, broken); err == nil {
				t.Fatal("corrupt report qualified")
			}
			if name != "changed-description" {
				if err := CompareProjectContracts(broken, broken); err == nil {
					t.Fatal("incomplete report qualified itself")
				}
			}
		})
	}
}
