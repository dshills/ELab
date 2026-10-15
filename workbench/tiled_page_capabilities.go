package workbench

func addTiledPageDefinition(schema map[string]any) {
	inputProperty(schema, "tile")["default"] = false
	inputProperty(schema, "tile")["description"] = "Tile at explicit scale, up to 16 pages in row-major order. False retains single-page fit/clip refusal."
	overlap := inputProperty(schema, "overlap_mm")
	overlap["minimum"], overlap["maximum"], overlap["x-unit"] = 0, 20, "mm"
	overlap["description"] = "Tiled overlap 0–20 mm; omitted/null defaults to 5 mm when tile is true. Nonzero overlap requires tile."
	schema["x-page-relations"] = []string{"tile true requires explicit scale 0.1–4 points per schematic unit; no implicit shrinking", "Each tile reserves an 18-point footer with page/grid/scale/overlap labels", "Row-major complete viewBox coverage with overlap; at most 16 pages; oversized layouts fail before publication", "PDF and print.submit include every page; multi-page PNG requires explicit zero-based page_index", "Settings are session output choices; preserve graph, dirty state, simulation and history"}
}
func addTiledPageDetails(id string, schema map[string]any) bool {
	switch id {
	case "export.page.set":
		addTiledPageDefinition(schema)
	case "print.preview.page.set":
		inputProperty(schema, "index")["minimum"] = 0
		schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "captured-preview-page", Preconditions: []string{"Exact visible completed preview ID; no active preview renderer", "index is zero-based and smaller than snapshot.print_preview.page_count"}, Effects: []string{"Same index preserves preview; another index admits rendering under a new preview ID", "Preserves immutable captured graph/settings/overlays, source hash and source identity, including after editor changes", "Only one page PNG is rendered/retained at a time; original chunk IDs become unavailable on replacement", "Never submits printer work or changes circuit/history/dirty/physics", "Print submission always includes all captured pages; page selection only changes the shown PNG"}, Errors: []commandFailureDetail{{"not-found", "admission", "Preview ID replaced/closed"}, {"not-ready", "admission", "Preview hidden or not completed"}, {"busy", "admission", "Active preview rendering"}, {"invalid-params", "admission", "Malformed input or index outside known page count"}, {"preview-failed", "work", "Captured page renderer failed"}, {"preview-timeout", "work", "30-second rendering deadline"}}, Completion: &commandCompletionDetail{Mode: "new preview ID admits rendering; inspect print_preview.status until completed/failed/canceled", Observe: []string{"project.inspect", "events.poll"}, Cancel: "print.preview.cancel", CooperativeDeadlineSeconds: 30}}
		return true
	case "print.preview.start":
		inputProperty(schema, "page_index")["minimum"] = 0
		schema["x-page-selection"] = "Capture current source/settings; page_index defaults to 0; actual count checked during rendering. Inspect page_count/rows/columns/status before use. Use page.set to navigate same capture."
	case "export.schematic.png", "export.start":
		inputProperty(schema, "page_index")["minimum"] = 0
		schema["x-png-page-selection"] = "PNG only: zero-based page_index required for a multi-page layout; omitted selects the sole page. Out-of-range layout fails before publication."
		schema["x-preview-png"] = "PNG only: preview_id selects exact visible completed page bytes; no page/page_index/dpi or recording ID override. Retains original source/page identity, even after edits; export can finish after preview closes."
	}
	return false
}
