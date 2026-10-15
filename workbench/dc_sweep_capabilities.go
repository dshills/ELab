package workbench

import (
	"reflect"
	"strings"
)

// These annotations describe this command family. Common request/revision/retry
// rules remain in the discovery envelope; OS/filesystem failures are not exhaustive.
type commandFailureDetail struct {
	Code  string `json:"code"`
	Phase string `json:"phase"`
	When  string `json:"when"`
}
type commandCompletionDetail struct {
	Mode                       string   `json:"mode"`
	Observe                    []string `json:"observe"`
	Cancel                     string   `json:"cancel"`
	CooperativeDeadlineSeconds int      `json:"cooperative_deadline_s"`
}
type commandContractDetail struct {
	Version       int                      `json:"version"`
	Scope         string                   `json:"scope"`
	Preconditions []string                 `json:"preconditions"`
	Effects       []string                 `json:"effects"`
	Errors        []commandFailureDetail   `json:"errors"`
	Completion    *commandCompletionDetail `json:"completion,omitempty"`
}

func sweepSettingsInputSchema() map[string]any {
	schema := structSchema(reflect.TypeOf(SweepSettings{}))
	parameters := []string{}
	for _, domain := range sweepDomains() {
		parameters = append(parameters, domain.Parameter)
	}
	inputProperty(schema, "parameter")["enum"] = parameters
	inputProperty(schema, "points")["minimum"], inputProperty(schema, "points")["maximum"] = 2, MaxDCSweepPoints
	for _, name := range []string{"start", "stop"} {
		p := inputProperty(schema, name)
		p["minLength"], p["maxLength"], p["x-max-utf8-bytes"] = 1, 128, 128
		p["description"] = "Finite floating point text; optional V for voltage, Ω/ohm with k/K/M resistance prefixes; wiper is a unitless fraction. Bounds apply after parsing."
	}
	probes := inputProperty(schema, "probe_ids")
	probes["minItems"], probes["maxItems"], probes["uniqueItems"] = 1, MaxScopeProbes, true
	probes["type"] = []string{"array", "null"}
	probes["description"] = "Omit or null for automatic target readings. Explicit arrays require 1–6 distinct IDs from snapshot.dc_sweep.available_probes; empty arrays are refused."
	schema["x-parameter-domains"] = sweepDomains()
	schema["x-relations"] = []map[string]any{
		{"operator": "different", "fields": []string{"start", "stop"}, "after": "parameter-unit-normalization"},
		{"operator": "distinct-representable-linear-axis", "fields": []string{"start", "stop", "points"}, "endpoints": "inclusive", "descending": true},
	}
	schema["x-inventory"] = map[string]string{"targets": "snapshot.dc_sweep.targets", "probes": "snapshot.dc_sweep.available_probes"}
	return schema
}

func addSweepCapabilityDetails(id string, schema map[string]any) bool {
	if !strings.HasPrefix(id, "sweep.") {
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "sweep-command-specific; common protocol/revision/retry rules are in the discovery envelope; OS/filesystem errors are not exhaustive", Preconditions: []string{}, Effects: []string{"Preserves live schematic, physics/time and edit history"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed input or a command parameter/relationship is invalid"}}}
	fail := func(code, phase, when string) {
		contract.Errors = append(contract.Errors, commandFailureDetail{code, phase, when})
	}
	retained := func() {
		schema["required"] = appendUniqueField(schema["required"], "id")
		contract.Preconditions = append(contract.Preconditions, "id names an operation retained in snapshot.dc_sweep.operations")
		fail("not-found", "admission", "Operation is unknown or evicted")
	}
	complete := func() {
		contract.Preconditions = append(contract.Preconditions, "The selected operation has a complete result")
		fail("not-ready", "admission", "Selected operation has no complete result")
	}
	file := func() {
		p := inputProperty(schema, "path")
		p["minLength"], p["maxLength"], p["x-max-utf8-bytes"] = 1, 4096, 4096
		p["description"] = "Nonblank local path, without NUL; relative paths resolve against the process working directory."
		contract.Preconditions = append(contract.Preconditions, "No active project file operation; optional expected_document_generation matches the current document")
		fail("busy", "admission", "A project file operation is active or file worker capacity is occupied")
		fail("conflict", "admission", "Expected document generation differs")
		fail("file-error", "work", "Regular-file access, preparation or publication fails")
		fail("timeout", "work", "Cooperative file operation deadline expires")
		fail("canceled", "work", "Operation canceled before attachment/publication")
		fail("superseded", "work", "Document replacement makes the captured file operation obsolete")
		contract.Completion = &commandCompletionDetail{Mode: "wait-for-project-work", Observe: []string{"project.work.inspect", "project.inspect", "events.poll"}, Cancel: "project.work.cancel", CooperativeDeadlineSeconds: 30}
	}
	switch id {
	case "sweep.settings.set":
		bounded := sweepSettingsInputSchema()
		for key, value := range bounded {
			schema[key] = value
		}
		contract.Preconditions = append(contract.Preconditions, "Target and explicit probe IDs exist in the current graph; an incomplete circuit may be configured")
		contract.Effects = append(contract.Effects, "Replaces next-sweep settings; does not start work or modify captured curves")
		schema["examples"] = []any{SweepSettings{ComponentID: "V1", Parameter: "voltage", Start: "0 V", Stop: "5 V", Points: 11, ProbeIDs: []string{"i:D1", "v:D1"}}}
	case "sweep.start":
		inputProperty(schema, "settings")["anyOf"] = []any{sweepSettingsInputSchema(), map[string]any{"type": "null"}}
		contract.Preconditions = append(contract.Preconditions, "A complete supported circuit and valid current target/probes; omitted or null settings uses the next-sweep draft", "No existing sweep worker, including a canceled worker still unwinding", "A retained slot is free or an excluded terminal operation can be evicted")
		contract.Effects = append(contract.Effects, "Captures source/models/settings; opens its curve view and closes preview; publishes only a complete successful curve, kept by default")
		fail("busy", "admission", "A sweep worker is still active")
		fail("unavailable", "admission", "Current circuit is incomplete or unsupported")
		fail("record-limit", "admission", "Eight retained operations are kept or active")
		fail("timeout", "operation", "Cooperative computation deadline expires")
		fail("sweep-failed", "operation", "A DC solve fails, including a singular system")
		contract.Completion = &commandCompletionDetail{Mode: "immediate-admission; inspect operation status", Observe: []string{"sweep.inspect", "project.inspect", "events.poll"}, Cancel: "sweep.cancel", CooperativeDeadlineSeconds: 30}
		schema["examples"] = []any{SweepStartParams{}}
	case "sweep.inspect", "sweep.select":
		retained()
		if id == "sweep.inspect" {
			contract.Effects = []string{"Read-only owned capture/result/failure metadata; does not advance or mutate live or recorded state"}
		} else {
			contract.Effects = append(contract.Effects, "Selects retained operation and resets its cursor; does not implicitly show the workspace")
		}
	case "sweep.cancel":
		retained()
		contract.Preconditions = append(contract.Preconditions, "Operation is active")
		fail("unavailable", "admission", "Only an active sweep can be canceled")
		contract.Effects = append(contract.Effects, "Requests cancellation; inspect until terminal; no partial result is published")
	case "sweep.visibility.set":
		contract.Effects = append(contract.Effects, "Shows/hides sweep workspace, including on a blank sheet; showing closes print preview; preserves captured curves")
		schema["examples"] = []any{SweepVisibilityParams{Visible: true}}
	case "sweep.trace.select":
		complete()
		schema["required"] = appendUniqueField(schema["required"], "id")
		fail("unavailable", "admission", "No sweep store exists")
		inputProperty(schema, "id")["description"] = "Probe ID from the selected captured result.probes, which may differ from the current graph inventory."
		contract.Effects = append(contract.Effects, "Changes plotted captured reading; measurements remain signed SI")
	case "sweep.cursor.set":
		complete()
		inputProperty(schema, "index")["anyOf"] = []any{map[string]any{"type": "integer", "minimum": 0, "x-maximum-exclusive": "selected result.points length"}, map[string]any{"type": "null"}}
		inputProperty(schema, "operation_id")["description"] = "Optional exact selected operation guard; mismatch refuses even a clear."
		fail("conflict", "admission", "operation_id differs from the selected operation")
		contract.Effects = append(contract.Effects, "Required index selects an exact stored point; explicit null clears; omission is invalid; never interpolates")
		zero := 0
		schema["examples"] = []any{SweepCursorParams{Index: &zero}, SweepCursorParams{}}
	case "sweep.keep.set":
		schema["required"] = []string{"id", "keep"}
		contract.Preconditions = append(contract.Preconditions, "id names a retained completed curve")
		fail("not-ready", "admission", "Operation is unknown, evicted or lacks a complete curve")
		contract.Effects = append(contract.Effects, "Changes project/recovery membership; excluding makes a terminal curve evictable and can mark recording membership dirty")
	case "sweep.export":
		file()
		retained()
		contract.Preconditions = append(contract.Preconditions, "id has a complete result; destination differs from the current project path; overwrite is explicit")
		inputProperty(schema, "format")["enum"] = []string{"json", "csv"}
		fail("not-ready", "admission", "Export requires a complete curve")
		fail("file-exists", "work", "Destination already exists and overwrite is false")
		contract.Effects = append(contract.Effects, "Atomically exports captured data; never substitutes current editor state; does not associate destination as the project file")
	case "sweep.import", "sweep.import.start":
		file()
		schema["x-max-file-bytes"] = MaxProjectBytes
		schema["x-file-format"] = map[string]any{"format": "elab.dc-sweep", "version": 1, "engine": DCSweepEngineID}
		contract.Preconditions = append(contract.Preconditions, "Source is a regular file, not a symlink/special file, up to 8 MiB", "A retained curve slot is available at admission and publication")
		fail("invalid-sweep", "work", "Captured JSON fails strict format/model/source/axis/probe/count/extrema validation")
		fail("record-limit", "admission-or-publication", "Kept/active curves consume all eight retained slots")
		contract.Effects = append(contract.Effects, "Retains the entire imported curve, kept by default, without changing next-sweep settings; preserves later viewer choices", "Imported data never claims current-source freshness; source path/SHA are session provenance")
		if id == "sweep.import.start" {
			contract.Completion.Mode = "immediate-project-work-admission"
		}
	default:
		return false
	}
	schema["x-command-contract"] = contract
	schema["x-details-scope"] = "sweep-parameter-bounds-and-command-contract; common protocol and filesystem errors are separate"
	schema["x-max-retained-operations"] = MaxDCSweepOperations
	return true
}

func appendUniqueField(required any, name string) []string {
	out := []string{}
	switch fields := required.(type) {
	case []string:
		out = append(out, fields...)
	case []any:
		for _, field := range fields {
			out = append(out, field.(string))
		}
	}
	for _, field := range out {
		if field == name {
			return out
		}
	}
	return append(out, name)
}
