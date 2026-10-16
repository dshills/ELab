package workbench

func addSamplingCapabilityDetails(id string, schema map[string]any) bool {
	if id != "simulation.sampling.fit" {
		return false
	}
	schema["x-sampling-policy"] = map[string]any{
		"minimum_step_s": minimumTransientSampleStep, "lc_samples_per_reference_period": lcSamplesPerReferencePeriod,
		"sine_samples_per_period": 32, "finite_pulse_samples_per_shortest_ramp": 4,
		"lc_reference": "2*pi*sqrt(smallest_positive_finite_L*smallest_positive_finite_C); configured-value hint only, not a measured period, circuit pole or resolution guarantee",
		"inventory":    "snapshot.lc_sampling plus snapshot.source_warnings; lc_sampling is omitted without both valid storage types",
	}
	schema["x-command-contract"] = commandContractDetail{
		Version: 1, Scope: "output-sampling-fit",
		Preconditions: []string{"Empty parameters; ordinary revision/document/retry guards apply", "No initialized solve is required; incomplete drawings can use configured source and L/C values"},
		Effects: []string{
			"Take the smallest current/source/LC recommendation, clipped to the unchanged 1 us minimum; never increase the sample step",
			"Lower playback only as needed for the existing 1000-samples/wall-second limit; preserve all part/source/startup and other transient settings",
			"An LC target below the minimum remains unresolved after fitting; can_fit becomes false at the supported floor. The hint does not include topology, damping or active-device timing",
			"Changed settings are one undoable electrical edit with ordinary restart/stale rules; identical settings preserve physics/history/dirty state",
			"Allowed in edit.batch; use preceding staged component/source values and commit or refuse the whole batch atomically",
			"Live fit does not replace an explicit Verify sample-step override or change existing recorded samples; projects save actual settings and recompute the derived live-graph advice",
		},
		Errors:     []commandFailureDetail{{"invalid-params", "admission", "Extra/non-object parameters or resulting settings invalid for the current graph"}},
		Completion: &commandCompletionDetail{Mode: "settings commit immediately; live initialization may be asynchronous or deferred while paused", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}, Cancel: "simulation.pause"},
	}
	return true
}
