package workbench

func addRecordingStatisticsDetails(id string, schema map[string]any) bool {
	if id != "recording.statistics" && id != "recording.statistics.set" {
		return false
	}
	for _, name := range []string{"start_s", "end_s"} {
		inputProperty(schema, name)["x-normalized-si-range"] = map[string]float64{"minimum": 0, "maximum": 10}
	}
	inputProperty(schema, "probe_id")["description"] = "Exact captured probe ID; set-only empty ID clears with zero start/end."
	schema["x-relations"] = []map[string]any{{"operator": "0<=start_s<end_s<=captured duration", "fields": []string{"id", "start_s", "end_s"}, "except": "set-only empty probe with zero interval clears"}}
	schema["x-inventory"] = map[string]string{"ids": "snapshot.operations with completed status", "probe_ids": "operation.inspect metadata.probes; captured units and references"}
	schema["x-max-recorded-frames"] = MaxVerificationFrames
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "captured-interval-statistics", Preconditions: []string{"Retained completed recording, captured probe, positive interval within duration; a DC point cannot define a time average", "Set ID must match selected capture; empty probe with zero interval clears"}, Effects: []string{"Piecewise-linear reconstruction of adjacent saved rows, clipped by interpolation at interval boundaries; zero-duration duplicate edge segments contribute no area", "Time mean is integral(v)/duration; RMS is sqrt(integral(v²)/duration); AC RMS integrates deviations from the interval time mean; SI unit follows captured probe", "Extrema and peak-to-peak include limits of positive-duration clipped segments; isolated zero-duration excursions do not contribute", "No assertion of periodicity, steady state, unseen peaks or exact analog waveform between saved samples; plot decimation and sample statistics are independent", "Query is pure. Set caches owned session/view results; selection/reopen clears; preserves live circuit/time/charge, history, document and saved recordings"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/missing input, invalid interval/clear request, or result outside finite range"}, {"not-ready", "admission", "Capture absent or incomplete"}, {"not-found", "admission", "Probe not captured"}, {"unavailable", "admission", "No positive-duration saved segments"}, {"conflict", "admission", "Set names a changed selected capture"}}, Completion: &commandCompletionDetail{Mode: "bounded in-memory integration of at most 4096 saved rows; no worker or file IO", Observe: []string{"recording.statistics", "project.inspect"}}}
	schema["examples"] = []any{RecordingStatisticsParams{ID: "verify-1", ProbeID: "v:C1", Start: .01, End: .02}}
	return true
}
