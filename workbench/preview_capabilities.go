package workbench

import "encoding/json"

func addPreviewCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "print.preview.start", "print.preview.cancel", "print.preview.close", "print.preview.image":
	default:
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "captured-print-preview", Preconditions: []string{}, Effects: []string{"Preserves current circuit, source association/save point, live acquisition/pause/time, history, scope preferences and stored recordings; never submits printer work", "snapshot.print_preview and preview publications expose metadata only; read pixels through print.preview.image by exact ID", "Source freshness and image readiness are separate: a stale completed visible preview remains readable/exportable/printable as its captured source. Refresh explicitly to include later edits"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/unknown/duplicate fields, missing/null required ID, wrong types or command-specific bounds"}}, Completion: &commandCompletionDetail{Mode: "immediate owned state/result; no filesystem publication"}}
	schema["x-preview-inventory"] = "snapshot.print_preview: exact id, status, visible, source_current, captured page/index/count/grid, run/electrical/document identity, admitting actor, dimensions, bytes/hash and error"
	if id == "print.preview.start" {
		var page map[string]any
		_ = json.Unmarshal(SchemaRegistry()["urn:elab:schematic-page:1"], &page)
		delete(page, "$id")
		addSchematicPageBounds(page)
		inputProperty(schema, "page")["anyOf"] = []any{page, map[string]any{"type": "null"}}
		inputProperty(schema, "page")["description"] = "Omitted/null captures snapshot.schematic_page defaults; an object is a complete override with omitted fields reset to built-ins. Explicit overrides do not follow later defaults."
		inputProperty(schema, "page_index")["type"] = []string{"integer", "null"}
		inputProperty(schema, "page_index")["default"] = 0
		inputProperty(schema, "page_index")["description"] = "Omitted/null selects zero. Negative refuses at admission; upper bound depends on the rendered captured layout and fails as preview-failed after admission."
		c.Preconditions = []string{"No outstanding preview worker, including canceling/hidden work; macOS cgo page renderer available", "Complete valid page settings and nonnegative page_index; empty/incomplete circuits are allowed"}
		c.Effects = append(c.Effects, "Capture graph/configured annotations/settings, page and optional frozen live display/pinned readings at admission; one 150 dpi PNG page, up to 8 MiB, 16 tiled pages and the existing raster/layout bounds", "Replaces the old preview/ID immediately, even if later rendering fails; no fallback to its old pixels. Hides the sweep plot without stopping analysis; only one selected page image is retained", "Metadata completion/currentness is independent of actual UI image decoding. Source_current compares document generation/content/settings/annotations, plus live overlays when requested and current defaults only for a default-page capture; view pan/zoom/theme alone do not stale it", "Completion advances presentation revision, not command revision. simulation.wait does not await preview work; inspect exact preview status or events.poll", "A new refresh requires the previous worker to return; cooperative cancellation/30-second deadline cannot preempt every CoreGraphics operation")
		c.Errors = append(c.Errors, commandFailureDetail{"busy", "admission", "An outstanding preview worker has not returned"}, commandFailureDetail{"unsupported", "admission", "macOS cgo page renderer unavailable"}, commandFailureDetail{"preview-failed", "work", "Layout/page-index/raster/8 MiB/image-decoding or renderer failure; admitted preview remains with error and no readable pixels"}, commandFailureDetail{"preview-timeout", "work", "Cooperative 30-second rendering budget expired"})
		c.Completion = &commandCompletionDetail{Mode: "immediate queued preview admission, not pixels; inspect until completed/failed/canceled and require visible completed metadata before reading", Observe: []string{"project.inspect", "events.poll"}, Cancel: "print.preview.cancel", CooperativeDeadlineSeconds: 30}
		schema["x-work-bounds"] = map[string]any{"outstanding_workers": 1, "raster_dpi": 150, "max_encoded_bytes": MaxProjectBytes, "cooperative_deadline_s": 30}
		schema["examples"] = []any{PreviewParams{}, map[string]any{"page": nil, "page_index": 0}}
	} else {
		inputProperty(schema, "id")["description"] = "Required non-null exact current preview ID from snapshot.print_preview; old/unknown/empty IDs return not-found. A closed preview retains metadata/ID until replaced."
		c.Errors = append(c.Errors, commandFailureDetail{"not-found", "admission", "No current preview or ID is old/unknown/empty"})
		if id == "print.preview.image" {
			for _, field := range []string{"offset", "limit"} {
				inputProperty(schema, field)["type"] = []string{"integer", "null"}
			}
			inputProperty(schema, "offset")["minimum"], inputProperty(schema, "offset")["default"] = 0, 0
			inputProperty(schema, "offset")["description"] = "Omitted/null starts at byte zero. Inclusive bound offset <= snapshot.print_preview.bytes; exactly bytes returns empty base64, next_offset=total."
			limit := inputProperty(schema, "limit")
			limit["anyOf"] = []any{map[string]any{"const": 0}, map[string]any{"minimum": 1, "maximum": 65536}, map[string]any{"type": "null"}}
			limit["default"], limit["description"] = 65536, "Omitted/null/zero means 65536; explicit nonzero limit is 1–65536 bytes. Final chunk may be shorter."
			c.Preconditions = []string{"Exact visible completed current preview with retained bytes; source_current need not be true", "After structural decoding, ID/readiness precede offset/limit bounds"}
			c.Effects = append(c.Effects, "Pure in-memory query with independently owned PNG bytes; no IO/render/wait, revisions, retry keys, visibility or currentness changes", "Result contains exact id/offset/next_offset/total and whole-image SHA-256. Decode png_base64, advance to next_offset until total, and verify assembled digest; restart on ID replacement rather than mixing captures")
			c.Errors = append(c.Errors, commandFailureDetail{"not-ready", "admission", "Matching preview hidden, incomplete, failed/canceled or lacks bytes"})
		} else if id == "print.preview.cancel" {
			c.Preconditions = []string{"Exact current queued/running/canceling preview; matching terminal preview returns unavailable"}
			c.Effects = append(c.Effects, "Request cancellation and mark canceling without hiding the pane. Repeated cancel while still canceling is accepted; terminal state is observed separately", "Worker slot remains occupied until actual return; no partial pixels are published and no prior preview image is restored")
			c.Errors = append(c.Errors, commandFailureDetail{"unavailable", "admission", "Matching preview is terminal"})
			c.Completion = &commandCompletionDetail{Mode: "immediate cancellation request; inspect preview until terminal, then refresh if needed", Observe: []string{"project.inspect", "events.poll"}}
		} else {
			c.Preconditions = []string{"Exact current preview ID in any state, including already hidden/terminal"}
			c.Effects = append(c.Effects, "Hide pane immediately, clear encoded bytes/hash and request cancellation if active. Metadata/status/dimensions and ID remain until replacement; completed status alone no longer means pixels are readable", "Repeated close is accepted for the same retained ID. Active close still occupies its worker slot until return; it does not restore a previously hidden sweep plot or change live scope preferences")
		}
	}
	schema["x-command-contract"] = c
	return true
}
