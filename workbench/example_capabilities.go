package workbench

func addExampleCapabilityDetails(id string, schema map[string]any) bool {
	alias := legacyExampleID(id)
	if id != "examples.list" && id != "example.load" && id != "example.current.reset" && id != "circuit.connection.set" && alias == "" {
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "example-templates", Preconditions: []string{"Ordinary service/revision/document guards and retry retention apply to mutations"}, Effects: []string{}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/missing/extra/wrong-type fields or command-specific invalid state"}}, Completion: &commandCompletionDetail{Mode: "immediate edit/replacement admission; observe any initialization separately", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}}}
	ids := []string{}
	for _, info := range ExampleCatalog() {
		ids = append(ids, info.ID)
	}
	if id == "examples.list" {
		c.Preconditions = []string{"Empty query params; no mutation key/revision required"}
		c.Effects = []string{"Pure owned catalog of shipped IDs, titles, questions, expected observations, suggested edits and model limitations; does not load/select/reset an example"}
		c.Completion = &commandCompletionDetail{Mode: "immediate pure owned query; no IO or operation ID"}
	} else if id == "circuit.connection.set" {
		c.Scope = "legacy-fixture-connection"
		inputProperty(schema, "connected")["description"] = "Required boolean; compatibility control for the R1.2 <-> D1.anode wire only. Prefer wire.connect/object.delete for general drawings"
		c.Preconditions = append(c.Preconditions, "Both exact terminals R1.2 and D1.anode exist, even for disconnect/no-op requests; ordinary wire capacity/connection validation applies when adding")
		c.Effects = []string{"true leaves an existing matching endpoint-pair wire untouched, or creates it through ordinary wire.connect validation; false removes the first matching pair wire if present", "An identical/no-match disconnect is a graph no-op. Changed topology uses ordinary electrical restart/paused-stale and history/dirty rules; does not operate arbitrary switch components or change other wires"}
		schema["x-max-wires"] = MaxWires
		schema["examples"] = []any{map[string]any{"connected": false}}
	} else {
		p := inputProperty(schema, "discard_changes")
		p["type"], p["default"] = []string{"boolean", "null"}, false
		p["description"] = "Omitted/null/false protects dirty work; true explicitly permits replacing unsaved work. Save first to retain it"
		inputProperty(schema, "expected_document_generation")["description"] = "Optional/null disables this additional parameter-level document guard; supplied value must equal current generation, independently of envelope guards"
		c.Preconditions = append(c.Preconditions, "Dirty document requires discard_changes=true or a completed Save; applicable expected generation must still match")
		c.Effects = []string{"Replaces the current document with a shipped template graph/settings and fresh configured model state; increments document generation, clears file association and old in-session project operations, resets template view/scope/verification defaults", "Does not overwrite explicit project files. Ordinary history records differing graph/settings/preferences; undo is not a promise to restore detached file association or discarded retained operations", "Starts fresh initialization even for an identical template; clears old accepted model/clock before asynchronous work. Inspect simulation state/diagnostics or simulation.wait rather than treating admission as a completed solve", "Template view follows the registered viewport; display preferences and detailed-diagnostic presentation reset to defaults"}
		c.Effects = append(c.Effects, "Loaded/reset template is a clean editable starting point without a file association; switching untouched examples requires no Save/Discard. Subsequent schematic/settings/preferences/kept-result edits use ordinary dirty protection")
		c.Errors = append(c.Errors, commandFailureDetail{"unsaved-changes", "admission", "Dirty project protected without explicit discard"}, commandFailureDetail{"conflict", "admission", "Envelope/parameter generation or expected current-example guard changed"})
		if id == "example.current.reset" {
			p := inputProperty(schema, "expected_example_id")
			p["type"], p["default"] = []string{"string", "null"}, ""
			p["description"] = "Optional/empty/null omits this guard. Nonempty value must equal the current example ID before reset; it does not select another template"
			c.Preconditions = append(c.Preconditions, "Current document has a recognized example origin; nonempty expected_example_id matches it")
			c.Effects = append(c.Effects, "Restores the current template and preserves whether simulation was paused; this differs from example.reset, which loads LED")
			c.Errors = append(c.Errors, commandFailureDetail{"unavailable", "admission", "No current example origin to reset"})
			schema["examples"] = []any{ExampleResetParams{ExpectedID: "rc"}}
		} else {
			c.Effects = append(c.Effects, "Load/legacy aliases set paused=false; initialized transient time can advance after admission. Use simulation.pause when a frozen observation is needed")
			if id == "example.load" {
				inputProperty(schema, "id")["enum"] = ids
				c.Errors = append(c.Errors, commandFailureDetail{"not-found", "admission", "Requested ID is not in examples.list"})
				schema["examples"] = []any{ExampleLoadParams{ID: "rc"}}
			} else {
				schema["x-example-id"] = alias
				schema["examples"] = []any{ReplaceParams{}}
			}
		}
	}
	schema["x-example-inventory"] = ids
	schema["x-command-contract"] = c
	schema["x-details-scope"] = "existing template/compatibility semantics; no new model, import or UI-only operation"
	return true
}
