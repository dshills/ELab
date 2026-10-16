package workbench

// These are command-specific contracts. Common protocol/revision/retry errors
// and arbitrary OS failures remain described separately.
func addVerificationCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "verification.settings.set", "simulation.verify", "operation.inspect", "operation.samples", "operation.select", "operation.cancel":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "verification-and-retained-operation-commands", Preconditions: []string{}, Effects: []string{}, Errors: []commandFailureDetail{}, Completion: &commandCompletionDetail{Mode: "immediate in-memory result"}}
	failure := func(code, phase, when string) {
		contract.Errors = append(contract.Errors, commandFailureDetail{code, phase, when})
	}
	failure("invalid-params", "admission", "Malformed input or an invalid parameter/relationship")
	schema["x-max-retained-operations"] = MaxVerificationOperations
	schema["x-operation-inventory"] = "snapshot.operations (time/DC verification and imported/project recordings; not DC sweeps)"
	switch id {
	case "verification.settings.set", "simulation.verify":
		starting := id == "simulation.verify"
		for _, name := range []string{"duration_s", "sample_step_s"} {
			inputProperty(schema, name)["x-unit"] = "s"
		}
		inputProperty(schema, "duration_s")["x-normalized-si-range"] = map[string]float64{"minimum": 0, "maximum": 10}
		step := inputProperty(schema, "sample_step_s")
		step["x-normalized-si-alternatives"] = []map[string]any{{"const": 0}, {"minimum": 1e-6, "maximum": 1}}
		step["description"] = "Omitted/null start override inherits stored defaults; effective zero follows current project step. Nonzero step 1 µs–1 s. Offline Verify does not impose playback-rate bounds."
		initial := inputProperty(schema, "initial_voltage_v")
		initial["anyOf"] = []map[string]any{{"type": "number", "minimum": 0, "maximum": 12}, {"type": "null"}}
		initial["x-unit"] = "V"
		initial["description"] = "Global startup default 0–12 V; graph per-capacitor settings and scenario overrides take precedence. Start omitted/null inherits defaults; full settings omitted/null clears this override."
		capacitors := inputProperty(schema, "capacitor_initials")
		capacitors["minItems"], capacitors["maxItems"], capacitors["x-unique-by"] = 0, MaxComponents, "id"
		capacitors["description"] = "Unique current capacitor IDs. Scenario voltages are pin 1 minus pin 2, signed ±12 V. Empty array clears the list; null is refused. Start omission inherits stored list; full settings requires an array."
		item := capacitors["items"].(map[string]any)
		inputProperty(item, "id")["description"] = "Current snapshot.components entry with kind capacitor."
		voltage := inputProperty(item, "voltage_v")
		voltage["minimum"], voltage["maximum"], voltage["x-unit"] = -12, 12, "V"
		inductors := inputProperty(schema, "inductor_initials")
		inductors["minItems"], inductors["maxItems"], inductors["x-unique-by"] = 0, MaxComponents, "id"
		inductors["description"] = "Unique current inductor IDs; signed pin 1 to pin 2 startup current in A within ±1 A. Empty clears, null refuses. Start omission inherits stored defaults; full-settings omission clears the list."
		inductorItem := inductors["items"].(map[string]any)
		inputProperty(inductorItem, "id")["description"] = "Current snapshot.components entry with kind inductor."
		current := inputProperty(inductorItem, "current_a")
		current["minimum"], current["maximum"], current["x-unit"] = -MaxInductorInitialCurrent, MaxInductorInitialCurrent, "A"
		mode := inputProperty(schema, "input_mode")
		mode["enum"] = []string{"auto", "charge", "discharge", "high", "low"}
		schema["x-input-mode-inventory"] = "snapshot.verification_input_modes for current analysis; stored defaults may be configured in advance"
		schema["x-startup-precedence"] = []string{"scenario capacitor_initials", "saved component initial_voltage_v", "scenario/stored global initial_voltage_v", "project experiment initial_voltage_v"}
		schema["x-inductor-startup-precedence"] = []string{"scenario/stored inductor_initials", "saved component initial_current_a", "zero A"}
		contract.Effects = []string{"Preserves live graph, physics/time, pause/held inputs, simulation settings and edit history"}
		if !starting {
			contract.Preconditions = []string{"Ranges and current capacitor identities are valid; incomplete graphs and mode/analysis combinations may be configured before running"}
			contract.Effects = append(contract.Effects, "Replaces session-only defaults; does not start work or mark the document dirty; omission/null global startup clears it")
			schema["examples"] = []any{DefaultVerificationSettings()}
		} else {
			mode["type"] = []string{"string", "null"}
			mode["enum"] = []any{"", nil, "auto", "charge", "discharge", "high", "low"}
			mode["description"] = "Omitted, null or empty inherits stored input mode. Admission requires membership in snapshot.verification_input_modes; auto uses configured source waveforms, not live elapsed phase."
			probes := inputProperty(schema, "probe_ids")
			probes["minItems"], probes["maxItems"], probes["uniqueItems"] = 0, MaxScopeProbes, true
			probes["description"] = "Omit for current selected live probes (with DC fallback channels). Explicit [] captures no channels; null is refused. IDs must be unique current probe inventory entries."
			schema["x-probe-inventory"] = "snapshot.scope.available; captured order is retained in metadata and row values"
			schema["x-max-output-steps"] = MaxVerificationSteps
			schema["x-max-recorded-frames"] = MaxVerificationFrames
			schema["x-normalized-si-relations"] = []map[string]any{{"operator": "ceil(duration_s/effective_sample_step_s)<=limit", "limit": MaxVerificationSteps}, {"operator": "duration_s>0 for transient; explicit duration_s==0 for DC", "omitted_dc_duration": "zero even if stored duration is positive"}, {"operator": "effective step <= project high_s and low_s when graph includes ideal pulse-source"}, {"operator": "no global/per-capacitor startup overrides for DC"}}
			contract.Preconditions = []string{"Complete supported current graph; stored defaults plus non-null overrides satisfy analysis/startup/mode rules", "At most one active verification, including canceling; a recording slot is free or an excluded terminal operation is evictable", "Transient duration positive and ≤10 s; ceil(duration/effective step) ≤1000; DC is one operating point at zero", "Current mode inventory: RC auto/charge/discharge; other supported transient families auto/high/low; DC auto only"}
			contract.Effects = append(contract.Effects, "Captures graph/models/startup/settings/probes and clears prior recorded comparison/timing selection; publishes only a complete successful recording, kept by default", "Later source edits do not alter the captured input; admission does not imply solve success or frame-budget completion")
			failure("busy", "admission", "A verification is queued/running/canceling")
			failure("record-limit", "admission", "Eight retained operations are kept or active")
			failure("unavailable", "admission", "Current circuit is incomplete or unsupported")
			failure("timeout", "operation", "Cooperative 30-second wall deadline expires")
			failure("solve-failed", "operation", "Kernel or recorded-frame bound fails; typed integration errors can report their existing specific codes")
			contract.Completion = &commandCompletionDetail{Mode: "immediate operation admission; inspect until terminal", Observe: []string{"operation.inspect", "project.inspect", "events.poll"}, Cancel: "operation.cancel", CooperativeDeadlineSeconds: 30}
			schema["examples"] = []any{VerifyParams{}}
		}
	case "operation.inspect":
		contract.Preconditions = []string{"id names a retained operation in any status"}
		contract.Effects = []string{"Pure owned summary and captured metadata; completed records include final solver/statistical results; no live or view mutation", "Failed captured Verify operations include failure_help: explanation/action/technical reason, preserving the original error code/message. Source association is reported separately; no current-object cause is guessed. No partial samples are available on failure."}
		failure("not-found", "admission", "ID unknown or evicted")
	case "operation.samples":
		inputProperty(schema, "offset")["minimum"] = 0
		inputProperty(schema, "offset")["x-maximum-inclusive"] = "record sample count"
		limit := inputProperty(schema, "limit")
		limit["minimum"], limit["maximum"], limit["default"] = 0, 200, 100
		limit["description"] = "Omitted or zero defaults to 100; otherwise 1–200. Offset may equal total for an empty last page."
		contract.Preconditions = []string{"id names a retained completed recording"}
		contract.Effects = []string{"Pure owned exact rows and probe descriptors; preserve duplicate edge timestamps and probe order; next_offset/total describe captured rows"}
		failure("not-found", "admission", "ID unknown or evicted")
		failure("not-ready", "admission", "Operation has no completed record")
	case "operation.select":
		contract.Preconditions = []string{"Retained ID in any status, or empty to clear"}
		contract.Effects = []string{"Changes selected recorded operation and resets replay/cursors/comparison/timing; empty ID hides recorded view; no live/source/history changes"}
		failure("not-found", "admission", "Nonempty ID unknown or evicted")
	case "operation.cancel":
		contract.Preconditions = []string{"Retained queued/running/canceling verification"}
		contract.Effects = []string{"Requests cooperative cancellation; no partial rows published; inspect terminal status rather than treating request as completion"}
		failure("not-found", "admission", "ID unknown or evicted")
		failure("unavailable", "admission", "Operation is already terminal")
		contract.Completion = &commandCompletionDetail{Mode: "cancellation request; inspect until terminal", Observe: []string{"operation.inspect", "project.inspect", "events.poll"}}
	}
	schema["x-command-contract"] = contract
	return true
}
