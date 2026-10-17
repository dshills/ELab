package workbench

import "encoding/json"

func addSchematicPageBounds(schema map[string]any) {
	for field, values := range map[string][]any{"paper": {"", "letter", "a4", nil}, "orientation": {"", "portrait", "landscape", nil}} {
		p := inputProperty(schema, field)
		p["type"], p["enum"] = []string{"string", "null"}, values
	}
	inputProperty(schema, "paper")["default"] = "letter"
	inputProperty(schema, "orientation")["default"] = "landscape"
	for _, field := range []string{"tile", "color", "measurements"} {
		inputProperty(schema, field)["type"] = []string{"boolean", "null"}
		inputProperty(schema, field)["default"] = false
	}
	margin := inputProperty(schema, "margin_mm")
	margin["minimum"], margin["maximum"], margin["x-unit"] = 0, 40, "mm"
	margin["description"] = "Omitted/null defaults to 10 mm; numeric zero is a valid zero margin."
	scale := inputProperty(schema, "scale")
	scale["minimum"], scale["maximum"], scale["x-unit"] = .1, 4, "PDF points per schematic unit"
	scale["description"] = "Omitted/null fits a single page; explicit zero refuses. Tiling requires an explicit scale. Actual fit/page-count is checked during rendering."
}

func addExportCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "export.page.set", "export.start", "export.schematic.svg", "export.schematic.pdf", "export.schematic.png", "verification.export":
	default:
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "captured-file-export", Preconditions: []string{}, Effects: []string{}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/unknown/duplicate fields, missing/null required fields, invalid path/kind/format/page parameters or incompatible options"}}, Completion: &commandCompletionDetail{Mode: "immediate session update"}}
	if id == "export.page.set" {
		addSchematicPageBounds(schema)
		c.Preconditions = []string{"Object parameters; every field is optional. Page bounds and tiling/overlap relationships must hold; a complete circuit is not required"}
		c.Effects = []string{"Replace session PDF/PNG defaults, not a patch: omitted/null paper/orientation use letter/landscape, margin 10 mm, fit scale, grayscale/no measurement overlays and no tiling", "snapshot.schematic_page exposes normalized settings. Defaults are not stored in circuit/recovery documents and do not dirty or alter graph, physics, history, selection or existing preview/export captures", "An export's omitted/null page uses these defaults; a supplied page object replaces the defaults for that job. Validation here does not render or guarantee the chosen scale/layout will fit"}
		schema["examples"] = []any{SchematicPage{}}
		schema["x-command-contract"] = c
		return true
	}
	path := inputProperty(schema, "path")
	path["minLength"], path["maxLength"], path["x-max-utf8-bytes"] = 1, 4096, 4096
	path["description"] = "Required non-null nonblank literal local path, no NUL; relative resolves with filepath.Abs. ~ and environment variables are not expanded; other spaces remain literal. No parent creation or automatic extension. Must differ from the associated schematic, including filesystem aliases."
	inputProperty(schema, "overwrite")["type"] = []string{"boolean", "null"}
	inputProperty(schema, "overwrite")["default"] = false
	inputProperty(schema, "expected_document_generation")["description"] = "Omitted/null adds no parameter-level guard; supplied uint64 must equal snapshot.document.generation. Common envelope/retry guards remain separate."
	schema["x-work-inventory"] = "snapshot.project_work is latest only; retain returned ID and inspect project.work.inspect across terminal states until ordinary eight-job eviction"
	schema["x-work-bounds"] = map[string]any{"retained_jobs": MaxProjectOperations, "max_workers": MaxProjectOperations, "max_encoded_bytes": MaxProjectBytes, "cooperative_deadline_s": 30}
	schema["x-phases"] = []string{"queued", "encoding", "checking-destination", "writing", "publishing", "cleanup", "completed"}
	c.Preconditions = []string{"No active shared project operation and fewer than eight outstanding filesystem workers; canceling workers retain their slot", "Valid destination and optional current document-generation guard; incomplete drawings may be exported without a successful solve"}
	c.Effects = []string{"Capture source at admission and perform encoding/destination checks/temp write/sync/publication outside the editor mutex; later graph, view, physics or recording choices cannot substitute their data", "Preserve source file association/save point, current editor/selection/history/physics and kept membership; publish only job/artifact metadata. Runtime jobs are not saved", "Exact associated path refuses at admission; parent-directory aliases and same-file identities refuse in worker preparation and are checked again before publication. Explicit overwrite never authorizes replacing the associated schematic", "Existing regular destinations require overwrite=true. Recheck captured destination identity/content before rename; new targets publish exclusively. Reject final links/special files and oversized output; clean temp files before terminal completion", "Source replacement/Close before publication supersedes work. Publishing is noncancelable and can finish after replacement, with original provenance and false associated_with_current_document; it cannot associate an export as the new project file", "last_export and retained project work expose captured generation/run/electrical or record identity, byte count and SHA-256. Association describes completion time, not later editor/disk equality", "Admission consumes the actor/retry key even after later file failure; use a new key for corrected intent. Identical immediate-start retries return original admission; waiting retries return the original terminal outcome without another write", "Filesystem checks/rename are not compare-and-swap against arbitrary outside writers; blocked OS calls may outlast cooperative deadlines. Directory sync is best effort, not power-loss qualification"}
	c.Errors = append(c.Errors, commandFailureDetail{"busy", "admission", "Active shared project operation or outstanding worker capacity"}, commandFailureDetail{"conflict", "admission", "Supplied document-generation guard differs"}, commandFailureDetail{"invalid-params", "project_work", "Destination aliases the associated schematic, or encoded output exceeds 8 MiB"}, commandFailureDetail{"file-exists", "project_work", "Existing/newly created destination lacks overwrite consent"}, commandFailureDetail{"file-conflict", "publication", "Destination identity/content changed during preparation"}, commandFailureDetail{"file-error", "project_work", "Local IO/encoding/rendering/layout failure; OS causes are not exhaustive"}, commandFailureDetail{"timeout", "project_work", "Cooperative 30-second budget expires"}, commandFailureDetail{"canceled", "project_work", "Exact job canceled before publication"}, commandFailureDetail{"superseded", "project_work", "Captured publication no longer belongs to the current document/job before publication"})
	c.Completion = &commandCompletionDetail{Mode: "waiting alias for the shared export job; terminal error includes owned job snapshot. Caller cancellation ends only the wait, not the admitted work", Observe: []string{"project.work.inspect", "project.inspect", "simulation.wait", "events.poll"}, Cancel: "project.work.cancel", CooperativeDeadlineSeconds: 30}
	if id == "export.start" {
		inputProperty(schema, "kind")["enum"] = []string{"schematic-svg", "schematic-pdf", "schematic-png", "scope-csv", "verification-json", "verification-csv", "sweep-json", "sweep-csv"}
		inputProperty(schema, "id")["type"] = []string{"string", "null"}
		inputProperty(schema, "id")["description"] = "Omitted/null/empty means no recording ID. Verification/sweep kinds look up this value and return not-found when absent/unknown; other kinds reject a nonempty ID."
		c.Completion.Mode = "immediate queued admission, not completion; inspect retained job by returned ID. project_work.command names the underlying export, not export.start"
		c.Effects = append(c.Effects, "Scope kind captures fresh selected live rows/window/units even while recorded view is shown. Verification/sweep kinds capture retained complete immutable data and original metadata, independent of selection or later membership/view changes")
		c.Errors = append(c.Errors, commandFailureDetail{"unavailable", "admission", "scope-csv lacks fresh selected rows in a valid live interval"}, commandFailureDetail{"not-found", "admission", "Verification/sweep ID unknown or evicted"}, commandFailureDetail{"not-ready", "admission", "Selected retained asset has no completed capture, or exact preview PNG is not ready"})
		schema["x-kind-options"] = "Recording id required for verification/sweep; nonempty id refuses for schematic/scope. Non-null page only PDF/PNG, nonzero dpi only PNG, non-null page_index only PNG. Nonempty preview_id only exact completed visible PNG, with no id/page/page_index/nonzero dpi override."
	} else {
		c.Errors = append(c.Errors, commandFailureDetail{"canceled", "wait", "Caller wait ends while admitted work continues"}, commandFailureDetail{"operation-unavailable", "wait", "Original admitted job expired"})
	}
	if id == "verification.export" {
		inputProperty(schema, "format")["enum"] = []string{"json", "csv"}
		inputProperty(schema, "id")["description"] = "Required non-null retained completed Verify/import/project recording ID; selection is irrelevant. Discover snapshot.operations and operation.inspect."
		c.Effects = append(c.Effects, "JSON exports complete captured graph/models/settings/metadata and exact rows; CSV exports original SI channels and model/operation attribution. Viewer window/probe subset/cursors never trim exported data")
		c.Errors = append(c.Errors, commandFailureDetail{"not-found", "admission", "ID unknown/evicted"}, commandFailureDetail{"not-ready", "admission", "Retained operation has no completed recording"})
	}
	if id == "export.start" || id == "export.schematic.pdf" || id == "export.schematic.png" {
		var page map[string]any
		_ = json.Unmarshal(SchemaRegistry()["urn:elab:schematic-page:1"], &page)
		delete(page, "$id")
		addSchematicPageBounds(page)
		inputProperty(schema, "page")["anyOf"] = []any{page, map[string]any{"type": "null"}}
		inputProperty(schema, "page")["description"] = "Omitted/null uses snapshot.schematic_page; object is a complete per-job override with omitted fields reset to built-in defaults."
		c.Errors = append(c.Errors, commandFailureDetail{"unsupported", "admission", "PDF/PNG renderer unavailable on this platform/build"})
		c.Effects = append(c.Effects, "PDF/PNG capture full drawing, ignoring viewport zoom/pan. Page.measurements opts into frozen live display/pinned readings with stale/unavailable labels. SVG always includes its captured display/readings", "PDF includes all tiled pages; PNG requires page_index for a multi-page layout. Layout/page-index/16-page/12-megapixel/clip constraints are checked during rendering and can fail after admission", "No printer job is submitted; printing has separate captured preview/spool controls")
	}
	if id == "export.start" || id == "export.schematic.png" {
		inputProperty(schema, "preview_id")["type"] = []string{"string", "null"}
		dpi := inputProperty(schema, "dpi")
		dpi["anyOf"] = []any{map[string]any{"const": 0}, map[string]any{"minimum": 72, "maximum": 300}, map[string]any{"type": "null"}}
		dpi["type"], dpi["default"], dpi["description"] = []string{"integer", "null"}, 150, "PNG: omitted/null/zero means 150 dpi; otherwise 72–300. Nonzero refuses for other kinds."
		inputProperty(schema, "preview_id")["description"] = "Optional nonempty exact visible completed preview ID exports its original selected PNG bytes, even after editor edits or later preview close; no renderer/page/index/dpi override. Omitted/null/empty selects a new live capture."
		c.Errors = append(c.Errors, commandFailureDetail{"not-ready", "admission", "Nonempty preview ID is not the exact visible completed page"})
	}
	schema["x-command-contract"] = c
	return true
}
