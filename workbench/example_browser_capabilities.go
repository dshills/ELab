package workbench

func addExampleBrowserCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "examples.visibility.set", "examples.category.set", "examples.search.set", "examples.select", "example.preview":
	default:
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "example-browser", Preconditions: []string{"Ordinary revision/document/retry guards apply to mutations"}, Effects: []string{"Session presentation only; preserves graph, saved document, history, measurements and physical state", "Browsing/selection never loads a circuit. Use existing example.load to open a template with ordinary Save/Discard/Cancel protection"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Missing/extra/malformed fields or invalid browser choice"}}, Completion: &commandCompletionDetail{Mode: "immediate", Observe: []string{"project.inspect", "events.poll"}}}
	switch id {
	case "examples.visibility.set":
		inputProperty(schema, "visible")["description"] = "Required boolean. Closing preserves the query/category/selection; accepted example.load also closes the browser"
	case "examples.category.set":
		var ids []string
		for _, category := range ExampleCategories() {
			ids = append(ids, category.ID)
		}
		p := inputProperty(schema, "category")
		p["enum"], p["description"] = ids, "Choose a category and clear search; preserve a matching selection or select the first result. Start here defaults to LED"
	case "examples.search.set":
		p := inputProperty(schema, "query")
		p["maxLength"], p["description"] = 128, "At most 128 UTF-8 bytes without NUL. Case-insensitive words must all match ID/title/question/expectation/experiment/category; one/two-character terms match whole words, longer terms ignore hyphens. Nonblank search covers all categories; blank restores the category"
		schema["x-max-utf8-bytes"] = 128
	case "examples.select":
		inputProperty(schema, "id")["description"] = "Exact ID in snapshot.example_browser.matching_ids; selects one preview without loading"
	case "example.preview":
		var ids []string
		for _, example := range ExampleCatalog() {
			ids = append(ids, example.ID)
		}
		p := inputProperty(schema, "id")
		p["enum"], p["description"] = ids, "Any shipped example, independently of the visible filter"
		c.Preconditions = []string{"Pure query; no revision or mutation key required"}
		c.Effects = []string{"Owned SVG and metadata from the exact shipped graph/settings; source_sha256 hashes those inputs", "No file/network IO, solve, document load, selection or physics change. Native thumbnails rasterize this same SVG"}
		c.Errors = append(c.Errors, commandFailureDetail{"not-found", "query", "Unknown example ID"})
	}
	schema["x-command-contract"] = c
	schema["x-details-scope"] = "Existing examples; browser state is not serialized in projects"
	return true
}
