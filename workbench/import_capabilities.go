package workbench

func addRecordingImportDetails(id string, schema map[string]any) bool {
	if id != "verification.import" && id != "verification.import.start" {
		return false
	}
	path := inputProperty(schema, "path")
	path["minLength"], path["maxLength"], path["x-max-utf8-bytes"] = 1, 4096, 4096
	path["description"] = "Required non-null nonblank literal local filename, no NUL; relative resolves with filepath.Abs in the host working directory. No ~ or environment expansion; other spaces stay literal. Read a regular file, not a final symlink/directory/special file, up to 8 MiB. Parent-directory aliases are allowed."
	inputProperty(schema, "expected_document_generation")["description"] = "Omitted/null adds no parameter-level guard; supplied uint64 must equal snapshot.document.generation. Common protocol/revision/actor/retry guards still apply."
	schema["x-import-format"] = map[string]any{"format": "elab.verification", "version": 1, "max_bytes": MaxRecordingBytes, "max_duration_s": 10, "max_output_steps": MaxVerificationSteps, "max_recorded_frames": MaxVerificationFrames, "max_probes": MaxScopeProbes, "validation": "Complete strict captured graph/model/settings/mode/probe/output-grid/finite-row/statistics/result consistency; no live solve or network model fetch. DC has one zero-duration row; transient captures retain actual output and corner rows, including duplicate timestamps."}
	schema["x-import-inventory"] = "snapshot.project_work is latest job only; retain its ID for project.work.inspect. Completed result_operation_id identifies a new retained recording in snapshot.operations / operation.inspect / operation.samples, distinct from metadata.operation_id."
	schema["x-import-capacity"] = map[string]any{"retained_project_jobs": MaxProjectOperations, "max_filesystem_workers": MaxProjectOperations, "retained_recordings": MaxVerificationOperations, "reservation": false, "checks": "At admission and again before complete attachment; unkept terminal records can be evicted, kept/active records cannot. Admission order after structural/common guards: active work/worker capacity, document generation, recording capacity, path syntax."}
	schema["x-phases"] = []string{"queued", "reading", "validating-recording", "completed"}
	c := commandContractDetail{
		Version: 1, Scope: "captured-recording-import",
		Preconditions: []string{"No active shared project job and fewer than eight outstanding filesystem workers, including canceling/superseded workers that have not returned", "Recording slot available under retention policy; optional parameter document generation matches. A current live solve/complete circuit is not required"},
		Effects: []string{
			"Capture source filename, current document generation and recording/sweep viewer-intent revisions at admission. Read/strict validation run outside the editor mutex; no staged asset is visible",
			"Atomically attach one complete kept recording with a new session record-N ID, original immutable metadata/rows and admitting actor; result_operation_id and source_sha256 identify attachment and exact read bytes. Source path/hash are session provenance, not injected into portable reexports",
			"Preserve graph/settings/physical time/charge/run, navigation, selection of schematic objects and undo/redo. Electrical/presentation edits and stepping within the same document may continue; kept membership alone can make recordings_dirty true",
			"Without later viewer intent, select imported first row and show recorded view. Later recording/sweep choices, Verify or preview intent keep their choice; inspect viewer_selected. Autonomous playback ticks alone do not count as user intent. Eviction of an unkept selected asset clears dangling selection to live",
			"No slot is reserved. Recheck capacity before attachment; another Verify can consume the last slot and make this admitted import fail with record-limit without evicting kept/active assets",
			"Import is recorded data, never a trusted live result: imported source_unchanged remains false even if its graph matches. It changes neither live circuit association nor the manual saved file and performs no file publication",
			"Failure/cancel/supersession preserves source, kept membership, viewer and history except shared job/admission bookkeeping. Project replacement/Close prevents stale attachment; same-document edits alone do not supersede import",
			"Owned retries retain original admission/terminal outcome without a duplicate attachment or rereading repaired bytes. Admission consumes the key even on failure; use a new key for a corrected attempt. A new successful key may intentionally import the same content again as another kept asset",
			"One 30-second cooperative context covers read/validation. Blocked OS reads/bounded validation may finish later, but canceled/obsolete data cannot attach; worker slots last until actual return. Jobs are not stored in circuit/recovery documents",
		},
		Errors: []commandFailureDetail{
			{"invalid-params", "admission", "Malformed/unknown/duplicate/missing/null required fields or invalid literal path"},
			{"busy", "admission", "Active project job or outstanding filesystem worker capacity"},
			{"conflict", "admission", "Parameter document-generation guard differs"},
			{"record-limit", "admission-or-attachment", "Eight protected kept/active recordings leave no evictable slot"},
			{"file-error", "project_work", "Missing/inaccessible/nonregular/final-symlink/oversized/changed-on-open input or OS read failure"},
			{"invalid-recording", "project_work", "Read bytes fail strict format/version/model/graph/settings/probe/grid/row/statistics/result validation"},
			{"timeout", "project_work", "Cooperative job deadline expires before attachment"},
			{"canceled", "project_work", "Exact job canceled or app closed before attachment"},
			{"superseded", "project_work", "Captured job/document generation no longer current before attachment"},
		},
		Completion: &commandCompletionDetail{Mode: "immediate queued admission, not a recording. Inspect project.work.inspect until terminal; completed work identifies result_operation_id, source_sha256 and viewer_selected", Observe: []string{"project.work.inspect", "project.inspect", "simulation.wait", "events.poll"}, Cancel: "project.work.cancel", CooperativeDeadlineSeconds: 30},
	}
	if id == "verification.import" {
		c.Completion.Mode = "waiting alias for the same background job; terminal failure includes owned job snapshot. Caller cancellation ends its wait only, not admitted work; owned retry rejoins original outcome"
		c.Errors = append(c.Errors, commandFailureDetail{"canceled", "wait", "Waiting caller disconnect/cancellation; admitted work can still finish"}, commandFailureDetail{"operation-unavailable", "wait", "Original admitted job expired under retention"})
	}
	schema["examples"] = []any{RecordingImportParams{Path: "Captured RC.json"}}
	schema["x-command-contract"] = c
	return true
}
