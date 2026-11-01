package workbench

// Failed verification describes a captured experiment, not a failure of the
// current live run. Unknown numerical causes must not invent object markers.
func verificationFailureHelp(operation OperationSummary) *DiagnosticHelp {
	if operation.Kind != "verify" || operation.Status != "failed" || operation.Error == nil {
		return nil
	}
	help := &DiagnosticHelp{Source: "verification", Code: operation.Error.Code, Severity: "error", Details: operation.Error.Message, Objects: []string{}, Terminals: []string{}, Wires: []string{}}
	switch operation.Error.Code {
	case "integration-attempt-limit":
		help.Summary = "A sample needed more calculation attempts than allowed."
		help.Action = "Reduce the Verify sample interval and shorten duration if needed to stay within its output-step limit, then verify again."
	case "integration-time-limit":
		help.Summary = "A sample exceeded its calculation time limit."
		help.Action = "Reduce the Verify sample interval for a CPU limit, or retry a wall-time backstop when the computer is less busy. Adjust duration to stay within the output-step limit."
	case "timeout":
		help.Summary = "The experiment exceeded the verification time limit."
		help.Action = "Shorten the Verify duration or review the captured circuit and startup conditions before trying again."
	default:
		help.Summary = "The calculation could not complete."
		help.Action = "Review the captured connections, model limits, startup conditions and technical reason; adjust the experiment before verifying again."
	}
	prefix := "Captured verification failed. "
	if !operation.SourceUnchanged {
		prefix = "An earlier captured verification failed. "
	}
	help.Summary = prefix + help.Summary
	help.Action += " No partial recording was published; live time/charge and kept recordings were preserved."
	return help
}
