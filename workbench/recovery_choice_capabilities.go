package workbench

func addRecoveryChoiceDetails(id string, schema map[string]any) bool {
	switch id {
	case "recovery.restore", "recovery.restore.start", "recovery.dismiss", "recovery.dismiss.start", "recovery.inspect", "recovery.samples":
	default:
		return false
	}
	schema["x-inventory"] = "snapshot.recovery.candidates/problems supply observed IDs/hashes; inspection document.recordings supplies embedded Verify indices. Cached selection is not required or proof of current file bytes."
	inputProperty(schema, "id")["description"] = "Required non-null recovery ID. Files are addressed within the captured store, not by arbitrary path or selected item. A valid older capsule can be read/restored even when only its newer sibling is listed."
	inputProperty(schema, "sha256")["description"] = "Required non-null exact case-sensitive observed digest. Legacy commands accept string input without a hex-shape admission check; empty, differently cased or stale hash fails file-conflict after readable data is validated. Missing/invalid recovery data can fail first."
	schema["x-file-bounds"] = map[string]any{"max_capsule_bytes": MaxRecoveryBytes, "max_document_bytes": MaxProjectBytes, "format": RecoveryFormat, "version": RecoveryVersion, "read": "Regular final file only, no final symlink; bounded read with file-identity check. Parent directories can contain aliases. Capsule checksum, identity, document/models/complete recorded assets are strictly validated for restore/read; dismissal has separate damaged-file rules."}
	c := commandContractDetail{Version: 1, Scope: "recovery-choice-and-query", Preconditions: []string{"Required non-null id/sha256; unknown/duplicate/mistyped fields refuse before command-specific checks"}, Effects: []string{"Captures the configured recovery store and explicit ID/hash, independently of cached selection. Does not overwrite explicit saved schematics or silently dismiss other capsules"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/missing/null required fields or command-specific domain violation"}}, Completion: &commandCompletionDetail{CooperativeDeadlineSeconds: 30}}
	query := id == "recovery.inspect" || id == "recovery.samples"
	if query {
		schema["x-max-concurrent-reads"] = MaxRecoveryReaders
		c.Preconditions = append(c.Preconditions, "Configured recovery store, service open, fewer than four shared readers, then caller cancellation; input is not required to be a current catalog member")
		c.Completion.Mode = "waiting pure request-scoped read; no retained job or events. Use recovery.read.start for owned observable work"
		c.Effects = append(c.Effects, "Reads/validates/projects outside the editor mutex; returns independently owned captured data. Preserves all public state, revisions, live time/run, document/history/kept membership, catalog/selection and file bytes on success/refusal", "Later edits, New or store reconfiguration do not supersede this captured read; inspection.directory identifies its original store. Data is not proof the file still exists after completion", "Four reader slots are shared with observable recovery.read jobs; a blocked OS call retains its slot until actual return. Caller cancellation/deadline and Close prevent successful late delivery; Close cancels and joins readers")
		c.Errors = append(c.Errors, commandFailureDetail{"unavailable", "admission", "No store configured or service closed"}, commandFailureDetail{"busy", "admission", "Four shared recovery readers active"}, commandFailureDetail{"invalid-recovery", "read", "Invalid ID, absent/inaccessible/nonregular/oversized file or invalid capsule/document/models/recordings"}, commandFailureDetail{"file-conflict", "read", "Readable valid capsule hash differs from requested exact digest"}, commandFailureDetail{"canceled", "read", "Caller/Close cancellation"}, commandFailureDetail{"timeout", "read", "Cooperative 30-second deadline; OS calls are not forcibly interrupted"})
		if id == "recovery.samples" {
			inputProperty(schema, "recording_index")["minimum"] = 0
			inputProperty(schema, "recording_index")["description"] = "Required non-null zero-based embedded Verify index; saved frequency/DC sweeps are not this recording list. Validated after reading/hash validation."
			inputProperty(schema, "offset")["minimum"] = 0
			inputProperty(schema, "offset")["description"] = "Optional omitted/null/zero starts at row 0. Total is inclusive: EOF returns no rows."
			p := inputProperty(schema, "limit")
			p["minimum"], p["maximum"], p["default"] = 0, 200, 100
			p["description"] = "Optional omitted/null/zero means 100; otherwise 1–200 rows, truncated at the captured total."
			c.Effects = append(c.Effects, "Returns exact embedded rows/probes, including duplicate pulse-edge times; never substitutes current live/selected viewer data. Page id is the recovery capsule ID, not a runtime Verify ID")
			c.Errors = append(c.Errors, commandFailureDetail{"invalid-params", "projection", "Embedded Verify index absent/out of range, offset outside 0..total or nonzero limit outside 1..200; file/hash errors take precedence"})
		} else {
			inputProperty(schema, "discard_changes")["description"] = "Compatibility field accepted by this shared input type but ignored by pure inspection; no replacement occurs."
			inputProperty(schema, "expected_document_generation")["description"] = "Compatibility field accepted but not checked by this pure captured read; it does not replace the current document."
			c.Effects = append(c.Effects, "Returns complete validated graph/settings/model and embedded Verify metadata, selected/viewing preferences and captured pause state, without recorded rows; use recovery.samples for rows. Owner status is observed during this read, not dismissal authorization")
		}
		schema["x-command-contract"] = c
		return true
	}
	schema["x-work-bounds"] = map[string]any{"retained_project_jobs": MaxProjectOperations, "max_filesystem_workers": MaxProjectOperations, "cooperative_deadline_s": 30}
	inputProperty(schema, "expected_document_generation")["description"] = "Optional omitted/null means no additional generation guard; when supplied it must match the current document at admission. Common top-level context/retry rules also apply."
	c.Completion.Mode = "immediate queued admission; retain exact project job ID and inspect until terminal, then await startup separately for restored physics"
	c.Completion.Observe = []string{"project.work.inspect", "project.inspect", "simulation.wait", "events.poll"}
	c.Completion.Cancel = "project.work.cancel"
	c.Effects = append(c.Effects, "Shared eight retained project jobs/eight outstanding filesystem workers; cancellation does not free an OS worker slot until actual return. File preparation runs outside the editor mutex", "Immediate retries retain original admission; waiting retries retain original terminal response. An admitted failure consumes its key; use a new key after repairing bytes/ownership. Caller disconnect ends only a waiting request, not admitted work")
	c.Errors = append(c.Errors, commandFailureDetail{"busy", "admission", "Active shared project work or eight outstanding workers"}, commandFailureDetail{"conflict", "admission", "Optional generation guard differs"}, commandFailureDetail{"canceled", "project_work", "Cancellation/Close before commit"}, commandFailureDetail{"superseded", "project_work", "Captured job/store or restore document guard no longer current"}, commandFailureDetail{"timeout", "project_work", "Cooperative preparation deadline; OS calls can return later"})
	if id == "recovery.restore" || id == "recovery.restore.start" {
		inputProperty(schema, "discard_changes")["description"] = "Optional omitted/null/false protects unsaved schematic/preferences/kept data; true authorizes replacing only the document observed at admission, not later edits."
		c.Preconditions = append(c.Preconditions, "After structural/common checks: shared work capacity, optional generation, unsaved-change permission, then configured store. ID/hash file validation is deferred to worker")
		c.Effects = append(c.Effects, "Active/unknown foreign ownership permits restoration as a copy, not deletion. Reads strictly validate the complete capsule before exact hash comparison; invalid ID/missing/damaged bytes can fail before hash mismatch", "Captured graph/transient settings, pinned/display/example preferences, kept membership/new Verify or sweep admission, input actuation, New/file association or recovery-store changes supersede pending installation. Session navigation/probe/cursor controls need not supersede: restored captured viewport/probes replace them", "Atomically installs a new dirty unassociated document with empty path, recovered_from original source or Unsaved schematic, fresh generation/run and captured pause setting. Live solver state/time restart; recorded assets retain their captured data, not running jobs", "Restores complete saved models/settings/probes/preferences/recorded assets; clears session drawing/selection/live cursors/window and starts fresh physics. Await simulation.wait for startup, not just project job completion", "Changed graph/settings/pinned/display/example content adds at most one undo entry attributed to the admitting actor/revision; a view-only or kept-membership-only difference does not guarantee one. Completion reports source_sha256, result_recovery_id/time and result_document_generation/run; file bytes and recovery catalog/selection are preserved")
		c.Errors = append(c.Errors, commandFailureDetail{"unsaved-changes", "admission", "Current schematic/preferences/kept content dirty without discard_changes=true"}, commandFailureDetail{"unavailable", "admission", "Recovery not configured"}, commandFailureDetail{"invalid-recovery", "project_work", "Invalid ID, file read/size/type error or strict capsule/document validation failure"}, commandFailureDetail{"file-conflict", "project_work", "Valid capsule bytes do not match requested digest"})
		schema["x-phases"] = []string{"queued", "reading-recovery", "validated-recovery", "completed"}
	} else {
		inputProperty(schema, "discard_changes")["description"] = "Accepted compatibility field ignored by dismissal; current circuit is not replaced and unsaved work needs no discard permission."
		c.Preconditions = append(c.Preconditions, "Configured store and syntactically valid rec- ID, optional generation, then shared work capacity. Current catalog membership/clean document/discard permission are not required")
		c.Effects = append(c.Effects, "Preserves current document/graph/physics/time/run/history/kept assets. Dismissal is directory-scoped: later editing/New need not supersede it for the same store", "Preparation and commit recheck exact raw bytes/hash, regular-file identity and parseable foreign owner. Active/unknown foreign ownership refuses; current/inactive owner may be dismissed. Damaged files may be explicitly dismissed by exact hash; a damaged payload without a parseable owner is not treated as a valid owned capsule", "dismissing-recovery fences unlink with cancelable=false; cancellation then returns operation-committing. This store serializes checkpoint/pruning with unlink, not a hostile-process atomic conditional unlink guarantee", "result_recovery_id is the requested ID even at admission; recovery_snapshot_removed starts false and is true only after confirmed removal. Completed result reports source_sha256. Follow-up scan failure remains completed with cleanup_warning/recovery.error", "Newer catalog metadata and later selection survive: if a full refresh cannot apply, only the removed ID is filtered out of current candidates/problems. recovery_catalog_applied=false does not mean the target file was preserved; inspect recovery_snapshot_removed")
		c.Errors = append(c.Errors, commandFailureDetail{"file-error", "project_work", "Filesystem read/type/size/removal error"}, commandFailureDetail{"file-conflict", "project_work", "Raw digest differs or file identity/content changed while preparing/committing"}, commandFailureDetail{"unavailable", "project_work", "Parseable foreign owner is active or unverifiable"}, commandFailureDetail{"operation-committing", "cancel", "Removal publication has begun"})
		schema["x-phases"] = []string{"queued", "validating-recovery-dismissal", "prepared-recovery-dismissal", "dismissing-recovery", "completed"}
	}
	if id == "recovery.restore" || id == "recovery.dismiss" {
		c.Completion.Mode = "waiting alias for the same job; terminal failure includes its snapshot. simulation.wait additionally awaits restored startup"
		c.Errors = append(c.Errors, commandFailureDetail{"canceled", "wait", "Waiting caller disconnected; admitted work may continue"}, commandFailureDetail{"operation-unavailable", "wait", "Original admitted job evicted"})
	}
	schema["x-command-contract"] = c
	return true
}
