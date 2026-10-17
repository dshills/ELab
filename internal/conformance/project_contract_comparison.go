package conformance

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dshills/elab/workbench"
)

func validateProjectContractSchemas(schemas map[string]json.RawMessage) error {
	if len(schemas) != len(projectContractCommands) {
		return fmt.Errorf("project contract coverage differs")
	}
	for _, id := range projectContractCommands {
		var schema struct {
			Ref        string                    `json:"$ref"`
			Type       string                    `json:"type"`
			Properties map[string]map[string]any `json:"properties"`
			Contract   struct {
				Version    int                                  `json:"version"`
				Scope      string                               `json:"scope"`
				Effects    []string                             `json:"effects"`
				Errors     []struct{ Code, Phase, When string } `json:"errors"`
				Completion struct {
					Mode     string   `json:"mode"`
					Observe  []string `json:"observe"`
					Cancel   string   `json:"cancel"`
					Deadline int      `json:"cooperative_deadline_s"`
				} `json:"completion"`
			} `json:"x-command-contract"`
			Jobs struct {
				Retained int      `json:"retained_jobs"`
				Workers  int      `json:"max_workers"`
				Statuses []string `json:"statuses"`
			} `json:"x-project-work"`
			File struct {
				Format      string `json:"format"`
				Write       int    `json:"default_write_version"`
				Sweep       int    `json:"kept_sweep_write_version"`
				Min         int    `json:"read_version_minimum"`
				Max         int    `json:"read_version_maximum"`
				Bytes       int    `json:"max_bytes"`
				LegacyBytes int    `json:"legacy_max_bytes"`
			} `json:"x-file-format"`
		}
		if err := json.Unmarshal(schemas[id], &schema); err != nil {
			return fmt.Errorf("%s schema: %w", id, err)
		}
		c := schema.Contract
		if schema.Ref != "" || schema.Type != "object" || c.Version != 1 || c.Scope != "project-files-and-work" || len(c.Effects) == 0 || c.Completion.Mode == "" || schema.Jobs.Retained != workbench.MaxProjectOperations || schema.Jobs.Workers != workbench.MaxProjectOperations || !reflect.DeepEqual(schema.Jobs.Statuses, []string{"queued", "running", "canceling", "completed", "failed", "canceled", "superseded"}) {
			return fmt.Errorf("%s contract identity/bounds/completion differ", id)
		}
		failures := map[string]bool{}
		for _, e := range c.Errors {
			if e.When == "" {
				return fmt.Errorf("%s missing failure explanation", id)
			}
			failures[e.Code+"/"+e.Phase] = true
		}
		expected := []string{"invalid-params/admission"}
		switch id {
		case "project.clear":
			expected = append(expected, "unsaved-changes/admission", "conflict/admission")
		case "project.work.inspect":
			expected = append(expected, "operation-unavailable/admission")
		case "project.work.cancel":
			expected = append(expected, "operation-unavailable/admission", "operation-committing/admission")
		}
		if strings.HasPrefix(id, "project.save") || strings.HasPrefix(id, "project.load") {
			expected = append(expected, "busy/admission", "conflict/admission", "file-error/project_work", "invalid-document/project_work", "timeout/project_work", "canceled/project_work", "superseded/project_work")
			if strings.HasPrefix(id, "project.save") {
				expected = append(expected, "unavailable/admission", "file-exists/project_work", "file-conflict/project_work")
			} else {
				expected = append(expected, "unsaved-changes/admission")
			}
			if !strings.HasSuffix(id, ".start") {
				expected = append(expected, "canceled/wait", "operation-unavailable/wait")
			}
			f := schema.File
			p := schema.Properties["path"]
			if p["x-max-utf8-bytes"] != float64(4096) || p["maxLength"] != float64(4096) || f.Format != workbench.DocumentFormat || f.Write != workbench.DocumentVersion || f.Sweep != workbench.SweepDocumentVersion || f.Min != 1 || f.Max != workbench.SweepDocumentVersion || f.Bytes != workbench.MaxProjectBytes || f.LegacyBytes != workbench.MaxDocumentBytes || c.Completion.Cancel != "project.work.cancel" || c.Completion.Deadline != 30 || !reflect.DeepEqual(c.Completion.Observe, []string{"project.work.inspect", "project.inspect", "events.poll"}) {
				return fmt.Errorf("%s file limits/observers differ", id)
			}
		}
		for _, code := range expected {
			if !failures[code] {
				return fmt.Errorf("%s missing failure %s", id, code)
			}
		}
	}
	return nil
}

// CompareProjectContracts checks coverage before comparison, so an incomplete
// report cannot qualify itself. File bytes and runtime outcomes are asserted by
// RunProjectContracts; schemas and public intent must agree across adapters.
func CompareProjectContracts(reference, actual ProjectContractResult) error {
	for _, r := range []ProjectContractResult{reference, actual} {
		if err := validateProjectContractEvidence(r); err != nil {
			return err
		}
		if err := validateProjectContractSchemas(r.Schemas); err != nil {
			return err
		}
		if !reflect.DeepEqual(r.Checks, projectContractChecks) || len(r.Calls) == 0 {
			return fmt.Errorf("project contract workflow coverage differs")
		}
		if _, err := projectContractCalls(r); err != nil {
			return err
		}
		counts := map[string]int{}
		discovered := map[string]bool{}
		for _, call := range r.Calls {
			counts[call.Command]++
			if call.Command == "capabilities.list" {
				var p workbench.CapabilityQuery
				if err := json.Unmarshal(call.Params, &p); err != nil {
					return err
				}
				discovered[p.Command] = true
			}
		}
		for _, id := range projectContractCommands {
			if !discovered[id] {
				return fmt.Errorf("missing discovery call for %s", id)
			}
		}
		for command, want := range map[string]int{"project.clear": 1, "project.save.start": 1, "project.save": 2, "project.load.start": 1, "project.load": 2, "project.work.cancel": 1, "project.work.inspect": 1, "component.set": 1} {
			if counts[command] != want {
				return fmt.Errorf("project workflow %s coverage differs", command)
			}
		}
	}
	for _, id := range projectContractCommands {
		var a, b any
		if err := json.Unmarshal(reference.Schemas[id], &a); err != nil {
			return err
		}
		if err := json.Unmarshal(actual.Schemas[id], &b); err != nil {
			return err
		}
		if !reflect.DeepEqual(a, b) {
			return fmt.Errorf("%s discovered contract differs", id)
		}
	}
	referenceCalls, err := projectContractCalls(reference)
	if err != nil {
		return err
	}
	actualCalls, err := projectContractCalls(actual)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(referenceCalls, actualCalls) {
		return fmt.Errorf("project contract public intent differs")
	}
	return nil
}

// Native startup configures recovery through a shared job. Compare explicit
// targets by their observed role, never by coincidental session counters. Raw
// report calls and snapshots retain the actual IDs.
func projectContractCalls(r ProjectContractResult) ([]Call, error) {
	out := append([]Call(nil), r.Calls...)
	for i, call := range out {
		var expected, role string
		switch call.Command {
		case "project.work.inspect":
			expected = r.States["save-admission"].ProjectWork.ID
			role = "<admitted-save>"
		case "project.work.cancel":
			expected = r.States["overwritten"].ProjectWork.ID
			role = "<overwrite-save>"
		default:
			continue
		}
		var p map[string]any
		if err := json.Unmarshal(call.Params, &p); err != nil {
			return nil, err
		}
		if expected == "" || len(p) != 1 || p["id"] != expected {
			return nil, fmt.Errorf("%s did not target its observed job", call.Command)
		}
		p["id"] = role
		raw, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		out[i].Params = raw
	}
	return collapseCalls(out), nil
}
