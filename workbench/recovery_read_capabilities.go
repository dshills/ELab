package workbench

func addRecoveryReadDetails(id string, schema map[string]any) bool {
	switch id {
	case "recovery.read.start", "recovery.read.cancel", "recovery.read.inspect", "recovery.read.result", "recovery.read.chunk":
	default:
		return false
	}
	schema["x-inventory"] = "snapshot.recovery.candidates (exact id/sha256), snapshot.recovery.reads (retained job IDs)"
	schema["x-max-retained-jobs"] = MaxRecoveryReadJobs
	contract := commandContractDetail{Version: 1, Scope: "captured-recovery-read", Effects: []string{"Preserves live circuit, simulation/time, project/history/dirty state and recovery catalog/selection", "Results stay tied to the captured directory and ID/hash across edits, New and recovery reconfiguration; only job metadata appears in snapshots"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed parameters"}}, Completion: &commandCompletionDetail{Mode: "immediate owned in-memory query; no disk IO"}}
	if id == "recovery.read.start" {
		inputProperty(schema, "kind")["enum"] = []string{"inspection", "samples"}
		p := inputProperty(schema, "id")
		p["minLength"], p["maxLength"], p["x-max-utf8-bytes"] = 1, 128, 128
		inputProperty(schema, "sha256")["pattern"] = "^[0-9a-fA-F]{64}$"
		inputProperty(schema, "sha256")["description"] = "Exact advertised candidate SHA-256; case-sensitive comparison to the captured bytes."
		inputProperty(schema, "recording_index")["minimum"] = 0
		inputProperty(schema, "offset")["minimum"] = 0
		p = inputProperty(schema, "limit")
		p["minimum"], p["maximum"], p["default"] = 0, 200, 100
		p["description"] = "Sample rows per page: omitted/zero uses 100; otherwise 1–200. Offset may equal total."
		schema["x-kind-fields"] = map[string]any{"inspection": map[string]any{"forbidden": []string{"recording_index", "offset", "limit"}}, "samples": map[string]any{"required": []string{"recording_index"}, "optional": []string{"offset", "limit"}}}
		schema["x-max-concurrent-reads"] = MaxRecoveryReaders
		schema["x-max-result-bytes"] = MaxRecoveryReadResultBytes
		contract.Preconditions = []string{"Recovery directory configured; exact candidate ID/hash is validated by worker", "Fewer than four active recovery reads, shared with legacy recovery.inspect/recovery.samples"}
		contract.Completion = &commandCompletionDetail{Mode: "immediate admission; inspect exact returned recovery.reads job until terminal", Observe: []string{"recovery.read.inspect", "project.inspect", "events.poll"}, Cancel: "recovery.read.cancel", CooperativeDeadlineSeconds: 30}
		contract.Effects = append(contract.Effects, "Reads/validates/projects/encodes outside editor lock, independently of project Save/Open work; no partial result", "Caller disconnect does not cancel admitted work; explicit cancel or Close does; Close joins workers", "Eight retained jobs; oldest terminal job is evicted on admission; active/canceling jobs keep their reader slot until worker returns")
		contract.Errors = append(contract.Errors, commandFailureDetail{"busy", "admission", "Four shared readers active"}, commandFailureDetail{"unavailable", "admission", "Recovery not configured"}, commandFailureDetail{"invalid-params", "work", "Recording index or page invalid"}, commandFailureDetail{"invalid-recovery", "work", "File missing, inaccessible or invalid"}, commandFailureDetail{"file-conflict", "work", "Candidate hash no longer matches"}, commandFailureDetail{"result-limit", "work", "Encoded result exceeds 16 MiB"}, commandFailureDetail{"canceled", "work", "Read canceled; no result published"}, commandFailureDetail{"timeout", "work", "Cooperative deadline exceeded; no result published"})
	} else {
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-found", "admission", "Unknown or evicted read job"})
	}
	switch id {
	case "recovery.read.cancel":
		contract.Completion = &commandCompletionDetail{Mode: "cancellation request; inspect exact job until terminal", Observe: []string{"recovery.read.inspect", "project.inspect"}}
		contract.Errors = append(contract.Errors, commandFailureDetail{"unavailable", "admission", "Job is already terminal"})
	case "recovery.read.result", "recovery.read.chunk":
		contract.Preconditions = []string{"Retained job completed successfully"}
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-ready", "admission", "No complete successful result"})
		if id == "recovery.read.result" {
			schema["x-max-result-bytes"] = MaxRecoveryReadResponseBytes
			contract.Effects = append(contract.Effects, "Decodes immutable retained JSON outside editor lock into independently owned typed result")
			contract.Errors = append(contract.Errors, commandFailureDetail{"result-too-large", "admission", "Use recovery.read.chunk for a larger complete result"})
		} else {
			inputProperty(schema, "offset")["minimum"] = 0
			p := inputProperty(schema, "limit")
			p["minimum"], p["maximum"], p["default"] = 0, 65536, 65536
			schema["x-offset-range"] = "0 through result_bytes inclusive; end returns empty bytes"
			contract.Effects = append(contract.Effects, "Returns an owned base64 JSON byte slice with offset/next/total and whole-result SHA-256; omitted/zero limit uses 65536")
		}
	}
	schema["x-command-contract"] = contract
	return true
}
