package workbench

func addProjectCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "project.inspect", "project.clear", "project.save", "project.save.start", "project.load", "project.load.start", "project.work.inspect", "project.work.cancel":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "project-files-and-work", Preconditions: []string{}, Effects: []string{}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed, unknown, duplicate, noncanonical or missing required fields; invalid path or generation input"}}, Completion: &commandCompletionDetail{Mode: "immediate owned result"}}
	schema["x-inventory"] = "snapshot.document, project_work and current graph/preferences/kept results; project.work.inspect reads retained job IDs"
	schema["x-project-work"] = map[string]any{
		"retained_jobs": MaxProjectOperations, "max_workers": MaxProjectOperations,
		"statuses":    []string{"queued", "running", "canceling", "completed", "failed", "canceled", "superseded"},
		"observation": "snapshot.project_work is latest only; retain returned IDs and use project.work.inspect; events.poll coalesces state and can skip intermediate phases",
		"retention":   "Oldest terminal job is evicted on admission; active/canceling workers keep their slot until return, including canceled OS calls",
		"outcomes":    "Inspect status, error, cancelable, source document_generation and optional result/artifact/association fields; job completion is separate from fresh live initialization",
	}
	if p := inputProperty(schema, "expected_document_generation"); p != nil {
		p["description"] = "Optional uint64 context guard matching snapshot.document.generation; omitted/null adds no parameter-level guard. Common envelope guards still apply; owned retry lookup precedes guards."
		if options, ok := p["anyOf"].([]any); ok {
			for _, option := range options {
				if integer, ok := option.(map[string]any); ok && integer["type"] == "integer" {
					integer["minimum"] = 0
				}
			}
		}
	}
	for _, field := range []string{"discard_changes", "overwrite"} {
		if p := inputProperty(schema, field); p != nil {
			p["type"], p["default"] = []string{"boolean", "null"}, false
			p["description"] = "Omitted/null means false; true explicitly opts into this command's documented discard or overwrite behavior."
		}
	}
	if addProjectFileDetails(id, schema, &contract) {
		schema["x-command-contract"] = contract
		return true
	}
	switch id {
	case "project.inspect":
		contract.Effects = []string{"Pure owned current snapshot: graph/connectivity, document generation/path/dirty partitions, selection/view, simulation/probe/model/diagnostic state and latest job metadata", "No filesystem read, solve, wait, history/revision/key change or job admission; works for empty/incomplete/paused/stale states and a closed in-process service while its transport remains available", "A saved path or completed historical job is not proof of current disk contents or fresh physical readiness; inspect job outcomes and result/run/electrical identities separately"}
	case "project.clear":
		contract.Preconditions = []string{"Current document generation if supplied; dirty schematic/preferences/kept results require discard_changes=true"}
		contract.Effects = []string{"Detach the file association, advance document generation and install an empty graph with default drawing/view/experiment/display/pin/example/probe/Verify state; preserve pause preference", "Clear retained recordings/sweeps and their view state; cancel active analysis; supersede preparing file work. Publishing work may still write its captured artifact but cannot associate it with the replacement", "Preserve explicit saved files. Changed drawing/settings/preferences add one actor-owned undo entry; Undo restores those contents, not the prior file association or cleared recordings", "A blank diagram is not a new session: lifetime component/wire ID reservations and retry keys remain. Omit component/wire IDs for automatic unused allocation"}
		contract.Errors = append(contract.Errors, commandFailureDetail{"unsaved-changes", "admission", "Dirty document lacks explicit discard"}, commandFailureDetail{"conflict", "admission", "Supplied parameter/document guard is stale"})
		contract.Completion = &commandCompletionDetail{Mode: "immediate document replacement; inspect diagnostics/readiness and any superseded prior work", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}}
		schema["examples"] = []any{ReplaceParams{}}
	case "project.work.inspect":
		schema["x-id-inventory"] = "Exact ID returned by any retained project_work operation; latest is snapshot.project_work.id. Older retained jobs remain inspectable across document replacement."
		contract.Preconditions = []string{"Required non-null string id; it must identify a retained job"}
		contract.Effects = []string{"Pure independently owned ProjectWork metadata in any status, including its original actor/input/source and terminal outcome; no file IO, wait, mutation or cancellation", "Terminal association describes completion time, not current association after a later replacement"}
		contract.Errors = append(contract.Errors, commandFailureDetail{"operation-unavailable", "admission", "ID unknown, empty or evicted"})
	case "project.work.cancel":
		schema["x-id-inventory"] = "Exact latest active snapshot.project_work.id, including canceling; this command covers all shared project file/recovery/export/import job kinds"
		contract.Preconditions = []string{"Current active job ID and common mutation guards; publication must not have begun"}
		contract.Effects = []string{"Request cooperative cancellation; inspect until terminal. No new deadline, rollback of a published file or early release of a blocked worker slot", "Already canceling requests remain accepted before publication; an existing canceled/superseded reason is not rewritten. cancelable=false can mean canceling or a noncancelable publishing/cleanup phase; inspect status/phase", "No partial document publication. Preparation cancellation preserves prior destination; an OS call may return later"}
		contract.Errors = append(contract.Errors, commandFailureDetail{"operation-unavailable", "admission", "ID is old/unknown or job is terminal"}, commandFailureDetail{"operation-committing", "admission", "Atomic publication or its cleanup has begun"})
		contract.Completion = &commandCompletionDetail{Mode: "immediate cancellation request; job remains observable until actual worker return", Observe: []string{"project.work.inspect", "project.inspect", "events.poll"}}
	}
	schema["x-command-contract"] = contract
	return true
}
