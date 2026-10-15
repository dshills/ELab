package workbench

func addShutdownDetails(id string, schema map[string]any) bool {
	switch id {
	case "app.close", "app.close.start", "app.close.inspect", "app.close.acknowledge":
	default:
		return false
	}
	schema["x-inventory"] = "snapshot.shutdown; one retained process-local job, exact id required for inspect/acknowledge"
	contract := commandContractDetail{Version: 1, Scope: "session-shutdown", Effects: []string{"Closed immediately freezes semantic mutations; it does not mean resources have finished closing", "Recovery serialization/finalization and worker joining run outside the editor mutex; queries/progress remain available while the process exists", "Shutdown is irreversible after admission: cancel at the unsaved-work decision beforehand; cancelable is false", "Completed shutdown sets exit_ready; failed cleanup/preference persistence stays failed with exit_ready false until explicit acknowledgement; no automatic cleanup retry"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed parameters"}}}
	switch id {
	case "app.close", "app.close.start":
		contract.Preconditions = []string{"Normal revision/retry/document-generation guards; dirty work requires discard_changes true or a completed Save", "Session has not already admitted shutdown, except a retry with the same original key"}
		contract.Effects = append(contract.Effects, "Captures current document/directory and discard/checkpoint policy; cancels admitted computation/preparation/read/playback work and prevents late result publication", "Discard retires only current document's owned recovery copies; internal Close checkpoints dirty work only when recovery enabled; other documents and explicit saved files remain intact", "Accepted close survives caller disconnect; waiting retry joins the same retained job; host teardown waits for exit_ready")
		contract.Errors = append(contract.Errors, commandFailureDetail{"unsaved-changes", "admission", "Dirty project protected"}, commandFailureDetail{"conflict", "admission", "Document/revision guard changed"}, commandFailureDetail{"closed", "admission", "Another close already admitted"}, commandFailureDetail{"shutdown-failed", "work", "Recovery or appearance flush failed; outcome contains separate errors"})
		mode := "immediate admission; inspect snapshot.shutdown until terminal"
		if id == "app.close" {
			mode = "wait for same finalization/join; successful response can contain a failed shutdown outcome"
			contract.Errors = append(contract.Errors, commandFailureDetail{"canceled", "wait", "Caller stopped waiting; admitted job continues"})
		}
		contract.Completion = &commandCompletionDetail{Mode: mode, Observe: []string{"app.close.inspect", "project.inspect", "events.poll"}}
	case "app.close.inspect":
		contract.Completion = &commandCompletionDetail{Mode: "immediate pure owned metadata; no IO or revision change"}
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-found", "admission", "Unknown shutdown job"})
	case "app.close.acknowledge":
		contract.Preconditions = []string{"Exact failed terminal shutdown ID; normal mutation revision/key guards even though session is closed"}
		contract.Completion = &commandCompletionDetail{Mode: "immediate exit_ready acknowledgement; failed status/errors retained"}
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-found", "admission", "Unknown shutdown job"}, commandFailureDetail{"unavailable", "admission", "Shutdown active or completed successfully"})
	}
	schema["x-command-contract"] = contract
	schema["x-cancellation"] = "No cancellation after admission; blocked OS calls/cooperatively canceled workers may continue; joining waits for actual return"
	return true
}
