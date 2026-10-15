package workbench

func addPulseTimingDetails(id string, schema map[string]any) bool {
	if id != "source.pulse.timing.set" {
		return false
	}
	schema["oneOf"] = []map[string]any{
		{"required": []string{"frequency_hz"}, "not": map[string]any{"required": []string{"period_s"}}},
		{"required": []string{"period_s"}, "not": map[string]any{"required": []string{"frequency_hz"}}},
	}
	for _, field := range []string{"frequency_hz", "duty_percent"} {
		p := inputProperty(schema, field)
		p["minLength"], p["maxLength"], p["x-max-utf8-bytes"] = 1, 128, 128
	}
	inputProperty(schema, "frequency_hz")["description"] = "Quantity text: decimal/scientific Hz, kHz or mHz; bare text is Hz. Parsed inclusive range 0.05–1000 Hz, further constrained by phase/rise/fall limits."
	inputProperty(schema, "frequency_hz")["x-parsed-si-range"] = map[string]float64{"minimum": .05, "maximum": 1000}
	inputProperty(schema, "duty_percent")["description"] = "Quantity text strictly between 0 and 100 percent, optional % suffix; bare 25 means 25%. Rise belongs to the high phase and fall to the low phase."
	inputProperty(schema, "duty_percent")["x-parsed-percent-range"] = map[string]float64{"exclusiveMinimum": 0, "exclusiveMaximum": 100}
	inputProperty(schema, "period_s")["x-normalized-si-range"] = map[string]float64{"minimum": .001, "maximum": 20}
	schema["x-source-inventory"] = "snapshot.pulse_timings entries with source_kind finite-pulse-source and shared false, independently of circuit completeness or accepted simulation"
	schema["x-timing-relations"] = []string{"Exactly one frequency_hz or period_s plus duty_percent", "high_s = period * duty_percent/100; low_s = period-high_s", "Both derived phases 1 µs–10 s; period at least 1 ms", "Existing rise_s <= derived high_s and fall_s <= derived low_s; transitions and levels are never shortened or rescaled"}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "finite-pulse-frequency-period-duty", Preconditions: []string{"Existing finite pulse source; parsed frequency/period/duty and coupled waveform limits all hold", "The ideal square pulse-source retains its separate shared experiment timing"}, Effects: []string{"Changes only selected source high/low phase lengths; voltage levels, rise/fall, other sources and experiment settings are unchanged", "Ordinary undoable electrical edit: running restarts from configured startup; paused stays stale until restart/resume", "Returned pulse_timings gives derived configured frequency/period/duty and round-trip quantity text; component.pulse remains authoritative saved seconds", "Reapplying current displayed timing preserves waveform bits/history/physics, including reciprocal-rounding cases; ordinary successful command/actor revisions may advance", "Equivalent quantity unit text normalizes before retry fingerprinting; frequency versus period alternatives remain different request forms", "Allowed in atomic edit.batch; failure preserves the entire prior state"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/missing/extra/null fields, both/neither timing alternatives, invalid quantity/unit/range or derived phases failing retained transition/model limits"}, {"not-found", "admission", "Source ID does not exist after parameter validation"}}, Completion: &commandCompletionDetail{Mode: "immediate validated graph edit; ordinary live initialization is observed separately", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}}}
	schema["examples"] = []any{PulseTimingParams{ID: "V1", Frequency: "20 Hz", Duty: "25 %"}, map[string]any{"id": "V1", "period_s": "100 ms", "duty_percent": "50 %"}}
	return true
}

func addIdealPulseTimingDetails(id string, schema map[string]any) bool {
	if id != "simulation.pulse.timing.set" {
		return false
	}
	addPulseTimingDetails("source.pulse.timing.set", schema)
	inputProperty(schema, "frequency_hz")["description"] = "Quantity text in Hz/kHz/mHz, bare text Hz; 0.05–500000 Hz before current sample-step and phase constraints. No sampling/playback adjustment is implicit."
	inputProperty(schema, "frequency_hz")["x-parsed-si-range"] = map[string]float64{"minimum": .05, "maximum": 500000}
	inputProperty(schema, "period_s")["x-normalized-si-range"] = map[string]float64{"minimum": 2e-6, "maximum": 20}
	inputProperty(schema, "duty_percent")["description"] = "Quantity text strictly between 0 and 100 percent, optional %; instantaneous high phase has no finite rise/fall."
	schema["x-source-inventory"] = "snapshot.pulse_timings entries with shared true/source_kind pulse-source; all such entries share snapshot.transient_settings high_s/low_s"
	schema["x-timing-relations"] = []string{"Exactly one frequency_hz or period_s plus duty_percent", "high_s = period*duty_percent/100; low_s = period-high_s", "Both phases 1 µs–10 s and >= current transient_settings.step_s", "For fraction=duty_percent/100, frequency <= minimum(fraction,1-fraction)/step_s; other setting bounds remain authoritative"}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "ideal-square-shared-clock", Preconditions: []string{"ID names a current pulse-source, even in an incomplete drawing", "Proposed shared high/low phases satisfy current sample step and all experiment constraints"}, Effects: []string{"Updates experiment high_s/low_s for every ideal-square pulse-source; choosing one source ID does not make its timing independent", "Preserves source voltage values, finite/sine definitions, sample step, playback speed and startup voltage", "Ordinary undoable electrical/settings edit: running restarts from configured startup, paused stays stale; supports atomic edit.batch", "Re-entering displayed timing preserves phase bits and history; normal command/actor revisions may advance", "Equivalent quantity/time aliases normalize before retry matching; snapshots expose configured timing, not measured oscillation", "Never shortens sample step or changes playback to make a frequency fit; invalid requests preserve all prior state"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed/missing/extra/null fields, both/neither timing forms, invalid quantity/range, wrong kind or shared phases too short for current sampling"}, {"not-found", "admission", "Source ID does not exist after parameter validation"}}, Completion: &commandCompletionDetail{Mode: "immediate validated settings edit; ordinary initialization observed separately", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}}}
	schema["examples"] = []any{IdealPulseTimingParams{ID: "VIN", Frequency: "20 Hz", Duty: "25 %"}, map[string]any{"id": "V1", "period_s": "100 ms", "duty_percent": "50 %"}}
	return true
}
