package workbench

// Keep the existing quantity bounds/relations alongside the stateful edit
// contract. These controls replace a source definition; they are not patches.
func addWaveformSourceContract(id string, schema map[string]any) bool {
	kind := "sine-source"
	switch id {
	case "source.sine.set":
	case "source.pulse.set":
		kind = "finite-pulse-source"
	default:
		return false
	}
	inputProperty(schema, "id")["description"] = "Required non-null existing " + kind + " component ID from snapshot.components. Missing IDs and wrong kinds both return invalid-params. Selection and circuit completeness are not required."
	inputProperty(schema, "waveform")["description"] = "Complete replacement definition. Every nested field is required and non-null, including zero values; omitted fields do not retain previous values. Quantity text normalizes before coupled model validation and retry matching."
	schema["x-source-inventory"] = "snapshot.components filtered by kind " + kind + "; component.sine/pulse is configured data, while snapshot.mixed and transient describe the last accepted live result"
	schema["x-details-scope"] = "parameter-bounds-and-per-source-waveform-state-contract"
	contract := commandContractDetail{
		Version: 1, Scope: "per-source-waveform",
		Preconditions: []string{"Ordinary protocol, revision/document, actor and retry admission guards apply", "id names an existing " + kind + "; complete nested waveform satisfies individual and coupled normalized SI bounds", "Incomplete drawings may be configured; successful editing does not guarantee a solvable circuit or a successful later simulation"},
		Effects: []string{
			"Replaces only the named source waveform; preserves its identity/kind, geometry, terminals/wires and other sources. component.value tracks sine amplitude or finite-pulse high voltage",
			"An actual change is one ordinary undoable electrical edit; running simulation restarts from configured startup/time zero. Paused edits retain stale accepted readings/charge/time until Restart or Resume",
			"Changed electrical configuration clears live scope cursors and window; independent recorded-view controls and captured rows are retained",
			"Preserves experiment sample step/playback and other transient settings; sampling advice is recomputed from configured values. Fit sampling is a separate explicit command",
			"Reapplying identical normalized numeric fields preserves graph, history, dirty state and physics; an ordinary successful command/actor revision may still advance. Equivalent unit/numeric retry intent normalizes before fingerprinting",
			"Allowed in edit.batch against staged components; every edit commits together or the entire batch refuses. One undo restores the committed batch",
			"Existing verification/recording rows and captured model/source parameters stay immutable; their source_unchanged becomes false when electrical revision changes. Saved projects retain numeric SI waveforms and the existing versioned model",
		},
		Errors:     []commandFailureDetail{{"invalid-params", "admission", "Missing/unknown/duplicate/null fields, invalid quantities/units, nonfinite or coupled model bounds, missing source or wrong source kind. Waveform validation precedes target lookup; ordinary shared admission rules still apply"}},
		Completion: &commandCompletionDetail{Mode: "immediate validated graph commit; ordinary live initialization is asynchronous or deferred while paused", Observe: []string{"project.inspect", "simulation.wait", "events.poll"}},
	}
	if id == "source.sine.set" {
		contract.Effects = append(contract.Effects, "Sine offset ± amplitude stays within ±12 V. Zero amplitude is a constant configured source, but frequency/phase bounds still apply")
	} else {
		contract.Effects = append(contract.Effects, "Rise is included in high_s and fall in low_s; equality permits a zero-length plateau. Equal ordered levels are a constant source, but phase/ramp bounds still apply", "This is the per-source finite pulse definition. Ideal-square pulse-source uses separate shared experiment timing; frequency/period/duty convenience editing uses source.pulse.timing.set")
	}
	schema["x-command-contract"] = contract
	return true
}
