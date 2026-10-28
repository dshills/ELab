package workbench

func addAnalysisCapabilityDetails(id string, schema map[string]any) bool {
	if id != "analysis.list" && id != "analysis.page" {
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "bounded-analysis-queries", Preconditions: []string{"Pure query through the shared service; existing commands and snapshots are unchanged"}, Effects: []string{"Owns returned slices; never changes live physics, document, cursor, history or selected analysis"}, Errors: []commandFailureDetail{{Code: "invalid-params", Phase: "admission", When: "Malformed fields, unsupported kind or page outside retained rows"}}, Completion: &commandCompletionDetail{Mode: "immediate owned query result"}}
	if id == "analysis.page" {
		inputProperty(schema, "kind")["enum"] = []string{"sweep", "recording", "spectrum", "tolerance"}
		inputProperty(schema, "offset")["minimum"] = 0
		limit := inputProperty(schema, "limit")
		limit["minimum"] = 0
		limit["maximum"] = MaxAnalysisPageRows
		limit["default"] = 25
		limit["description"] = "Omitted, null or zero uses 25; otherwise 1–50; offset may equal total for an empty last page."
		field := inputProperty(schema, "id")
		field["minLength"] = 1
		field["maxLength"] = 128
		contract.Preconditions = append(contract.Preconditions, "Use kind/id and document_generation from analysis.list. Results are immutable within that document generation; evicted/replaced IDs cannot be paged.")
		contract.Errors = append(contract.Errors, commandFailureDetail{Code: "conflict", Phase: "admission", When: "Document generation changed; refresh analysis.list"}, commandFailureDetail{Code: "not-ready", Phase: "admission", When: "Known analysis has no completed result"}, commandFailureDetail{Code: "not-found", Phase: "admission", When: "Unknown, evicted or replaced ID"})
	} else {
		contract.Effects = append(contract.Effects, "Includes current revision/command_revision/document_generation admission guards without a full snapshot; source graphs and raw rows are excluded")
	}
	schema["x-command-contract"] = contract
	return true
}
