package workbench

func addPrintingCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "print.printers.refresh", "print.destination.set", "print.submit", "print.job.refresh", "print.job.cancel", "print.work.cancel":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "local-macOS-spooler", Preconditions: []string{}, Effects: []string{"Preserves source circuit, live simulation/time, dirty state and history"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed or out-of-range parameters"}}, Completion: &commandCompletionDetail{Mode: "immediate work admission; inspect printing.work until terminal", Observe: []string{"project.inspect", "events.poll"}, Cancel: "print.work.cancel", CooperativeDeadlineSeconds: 30}}
	if id != "print.work.cancel" {
		contract.Errors = append(contract.Errors, commandFailureDetail{"busy", "admission", "Print discovery/preparation/control work is already active"})
	}
	schema["x-inventory"] = "snapshot.printing.destinations, snapshot.printing.jobs, snapshot.printing.work and snapshot.print_preview"
	schema["x-max-retained-jobs"] = MaxPrintJobs
	schema["x-platform"] = "macOS cgo; local CUPS socket; no interactive authentication or arbitrary remote scheduler"
	if id == "print.submit" || id == "print.destination.set" {
		p := inputProperty(schema, "copies")
		p["minimum"], p["maximum"] = 1, 10
		inputProperty(schema, "printer_id")["description"] = "Exact discovered destination ID. Submit requires accepting_jobs; no automatic destination selection. Empty ID clears session selection only in destination.set."
	}
	switch id {
	case "print.printers.refresh":
		contract.Effects = append(contract.Effects, "Refreshes listed destinations/default/accepting/state without submitting; failed/canceled refresh preserves prior inventory; removed selected destination clears selection")
	case "print.destination.set":
		contract.Completion = &commandCompletionDetail{Mode: "immediate session setting"}
		contract.Effects = append(contract.Effects, "Changes only session printer/copy controls; does not submit")
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-found", "admission", "Nonempty printer ID absent from discovery"})
	case "print.submit":
		inputProperty(schema, "preview_id")["description"] = "Exact visible completed preview ID; prints ALL pages of this captured source/settings even if selected preview page, editor or defaults changed later. Refresh preview first to print edits."
		contract.Preconditions = []string{"Exact visible completed preview; listed accepting printer; copies 1–10", "Fewer than eight unresolved jobs, or a terminal job is evictable"}
		contract.Effects = append(contract.Effects, "Captures preview source/page/provenance, renders all captured tiled pages into one private PDF ≤8 MiB and submits one-sided with selected paper/orientation/copies; temporary file removed after spool call", "Acknowledged spool ID is retained even if preparation cancellation/Close arrived while submitting; submitted is not physical completion", "Missing submission acknowledgment produces unknown/submission-unknown and never automatically resubmits; inspect system queue before retrying")
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-ready", "admission", "Preview absent/hidden/incomplete or ID stale"}, commandFailureDetail{"unavailable", "admission", "Printer absent or not accepting jobs"}, commandFailureDetail{"record-limit", "admission", "Eight unresolved retained jobs"})
	case "print.job.refresh", "print.job.cancel":
		contract.Preconditions = []string{"ELab-retained job with an acknowledged spool ID; no arbitrary system job IDs"}
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-found", "admission", "ELab job unknown/evicted"}, commandFailureDetail{"unavailable", "admission", "No spool ID, or cancel requested for a terminal job"})
		contract.Effects = append(contract.Effects, "Refresh reads spooler job-state; absent/expired system job is an error and never assumed printed", "Cancel requests exact owned spool job, records acknowledgment and reads state; failure retains last known status. Cancellation cannot retract pages already printed")
	case "print.work.cancel":
		contract.Preconditions = []string{"Exact current queued/running/canceling work ID"}
		contract.Completion = &commandCompletionDetail{Mode: "cancellation request; inspect printing.work until terminal", Observe: []string{"project.inspect", "events.poll"}}
		contract.Effects = append(contract.Effects, "Cooperatively cancels preparation/discovery/control; cannot revoke an acknowledged spool submission; use print.job.cancel for that job")
		contract.Errors = append(contract.Errors, commandFailureDetail{"not-found", "admission", "Work ID no longer current"}, commandFailureDetail{"unavailable", "admission", "Work already terminal"})
	}
	if id != "print.destination.set" && id != "print.work.cancel" {
		contract.Errors = append(contract.Errors, commandFailureDetail{"print-failed", "work", "Rendering/local spooler rejects or fails"}, commandFailureDetail{"timeout", "work", "Cooperative context deadline"}, commandFailureDetail{"canceled", "work", "Local work canceled before acknowledgment"}, commandFailureDetail{"unsupported", "work", "Platform backend unavailable"})
	}
	if id == "print.submit" {
		contract.Errors = append(contract.Errors, commandFailureDetail{"submission-unknown", "work", "Spool call acknowledgment missing; job may exist"})
	}
	if id == "print.job.refresh" || id == "print.job.cancel" {
		contract.Errors = append(contract.Errors, commandFailureDetail{"spool-job-unavailable", "work", "Spooler no longer retains job"})
	}
	schema["x-command-contract"] = contract
	return true
}
