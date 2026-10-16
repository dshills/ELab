package workbench

// Editing domains are distinct from analysis overrides (for example a signed
// swept DC bias); this discovery describes ordinary schematic value commands.
func addValueCapabilityDetails(id string, schema map[string]any) bool {
	if id != "component.place" && id != "component.set" && id != "capacitor.initial.set" && id != "inductor.initial.set" {
		return false
	}
	value := inputProperty(schema, "value")
	value["maxLength"], value["x-max-utf8-bytes"] = 128, 128
	value["minLength"] = 1
	value["description"] = "Finite, representable SI text with case-sensitive prefixes in the component's units; bounds apply after parse-then-scale normalization. Nonzero values must not round to zero."
	contract := commandContractDetail{Version: 1, Scope: "component-values-and-placement", Preconditions: []string{}, Effects: []string{"Commits through ordinary electrical edit/restart/paused-stale and undo rules; invalid input preserves previous graph/physics/history"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Unknown target/kind, malformed value or value outside domain/coupled waveform constraints"}}, Completion: &commandCompletionDetail{Mode: "immediate edit admission; any asynchronous live recomputation remains observable on existing simulation state", Observe: []string{"project.inspect", "events.poll", "simulation.wait"}}}
	if id == "inductor.initial.set" {
		value["description"] = "Signed finite current text in A/mA/uA/µA/μA, or exact literal default for zero; positive into pin 1/out of pin 2."
		value["x-numeric-domain"] = map[string]any{"unit": "A", "minimum": -MaxInductorInitialCurrent, "maximum": MaxInductorInitialCurrent}
		value["x-special-values"] = []string{"default"}
		contract.Preconditions = []string{"id names a current inductor"}
		contract.Effects = append(contract.Effects, "Sets signed startup current; default clears the component override for zero. Existing live current is not resumed on a fresh run.")
		schema["examples"] = []any{ValueParams{ID: "L1", Value: "-20 mA"}, ValueParams{ID: "L1", Value: "default"}}
	} else if id == "capacitor.initial.set" {
		value["description"] = "Signed finite voltage text in V/mV/uV/µV/μV, or exact literal default to use experiment startup; pin 1 minus pin 2."
		value["x-numeric-domain"] = map[string]any{"unit": "V", "minimum": -12, "maximum": 12}
		value["x-special-values"] = []string{"default"}
		contract.Preconditions = []string{"id names a current capacitor"}
		contract.Effects = append(contract.Effects, "Sets explicit signed initial voltage; default clears the per-component override; no physical history is resumed")
		schema["examples"] = []any{ValueParams{ID: "C1", Value: "-500 mV"}, ValueParams{ID: "C1", Value: "default"}}
	} else {
		domains := []map[string]any{}
		kinds := []string{}
		for _, part := range Library() {
			kinds = append(kinds, part.Kind)
			domain := map[string]any{"kind": part.Kind, "value_type": part.ValueType}
			if part.ValueDomain != nil {
				domain["numeric"] = *part.ValueDomain
				domain["default_value"] = part.DefaultValue
			}
			if part.Kind == "net-label" {
				domain["label"] = map[string]any{"minLength": 1, "maxLength": 32, "pattern": "^[A-Za-z0-9_+./-]{1,32}$", "case_sensitive": true}
			}
			domains = append(domains, domain)
		}
		schema["x-kind-value-domains"] = domains
		schema["x-source-reference-rules"] = map[string]any{"dc-source": "Value is positive minus negative; either terminal may be offset from ground in a referenced network", "waveform-sources": "Negative terminal remains directly grounded", "network": "A common ground reference and independent source/capacitor constraints remain required; accepted node voltages stay within ±30 V and existing solve/work limits"}
		schema["x-network-numerics"] = map[string]any{
			"linear_correction":               "Full affine MNA correction; even zero excitation checks matrix rank before acceptance",
			"nonlinear_correction":            "Damped Newton; LED, silicon diode, NPN, NMOS, PMOS and unknown future devices retain damping",
			"cold_start_fallback":             "After a failed ordinary nonlinear cold start, retry once with only voltages fixed by ideal paths to ground initialized; startup capacitors participate only at dt=0. No substitute device equations or accepted result. Warm guesses retain their path.",
			"rank_check":                      "Every Newton attempt solves its matrix before accepting even a zero-residual initial guess; floating insulated gates are not assigned an arbitrary zero",
			"iteration_budget":                "100 iterations per Newton attempt; one additional grounded-constraint cold-start retry; existing per-NPN seed retries remain. Cancellation and integration attempt/time budgets still apply.",
			"max_grounded_constraint_retries": 1,
			"max_newton_iterations":           maxMixedNewtonIterations, "max_nonlinear_node_update_v": maxMixedNewtonDelta,
			"nonlinear_update_scope": "The node-update limit applies to legacy networks without L. L networks instead limit forward diode/B-E/B-C junction increments and absolute MOS gate-source increments to 0.2 V; reverse-biased junctions may approach conduction in one proposal. All convergence and node-range checks remain required.",
			"max_abs_node_voltage_v": maxMixedNodeVoltage,
			"time_integration":       "Existing network circuits retain backward Euler and other legacy engines remain unchanged. L networks use elab.rlc-mna-radau5.v1 three-stage Radau IIA, conservative full/two-half current/node/charge checks and an LC step ceiling. Existing per-output attempt/time budgets remain; no device-model substitution.",
			"inductor_numerics": map[string]any{
				"engine": InductorEngineID, "minimum_internal_step_s": minimumInductorStep,
				"kcl_absolute_floor_a": inductorKCLAbsolute, "kcl_relative_jacobian_scale": inductorKCLRelative,
				"method":     "Three-stage fifth-order Radau IIA collocation; all stages enforce device KCL and storage equations. Newton requires current residual and voltage-correction convergence; stage iterations share the same output wall deadline.",
				"startup":    "Preserve configured/accepted L current and C voltage; require signed current-cut balance and differentiate redundant cutset KCL to determine voltage. DC instead imposes zero L voltage.",
				"led":        "Same clipped forward LED equation, solved in junction voltage with a forward one-sided Jacobian at zero to resolve Is-scale startup; no new reverse leakage.",
				"stagnation": "Small stagnant corrections with satisfied physical rows reject the trial. Once per accepted state, a larger interval within the same output/source/LC bounds may be proposed; all Newton and full/two-half accuracy checks and original output budgets still apply.",
				"endpoints":  "Integrate every distinct interval; balance short final intervals instead of skipping a fixed duration. Source/output boundaries merge only within the existing four-ULP same-time rule.",
			},
		}
		schema["x-normalized-si-relations"] = []inputRelation{{Operator: "sum_absolute_maximum", Fields: []string{"value", "snapshot.components[id].sine.offset_v"}, Limit: 12, WhenKind: "sine-source"}, {Operator: "greater_or_equal", Fields: []string{"value", "snapshot.components[id].pulse.low_v"}, WhenKind: "finite-pulse-source"}}
		schema["x-inventory"] = map[string]string{"part_definitions": "parts.list", "current_id_kind_waveforms": "snapshot.components"}
		if id == "component.place" {
			value["minLength"] = 0
			schema["x-factory-waveform-baselines"] = map[string]any{"sine_offset_v": DefaultSineWaveform().Offset, "pulse_low_v": DefaultPulseWaveform().LowVoltage}
			schema["x-normalized-si-relations"] = []inputRelation{{Operator: "sum_absolute_maximum", Fields: []string{"value", "x-factory-waveform-baselines.sine_offset_v"}, Limit: 12, WhenKind: "sine-source"}, {Operator: "greater_or_equal", Fields: []string{"value", "x-factory-waveform-baselines.pulse_low_v"}, WhenKind: "finite-pulse-source"}}
			value["description"] = "Omitted/empty uses factory value; net-label uses new ID as name; non-value symbols require omitted/empty value. Supplied numeric values use the per-kind domains; new waveforms use factory offset/low level."
			inputProperty(schema, "kind")["enum"] = kinds
			identifier := inputProperty(schema, "id")
			identifier["maxLength"], identifier["pattern"] = 64, "^(?:[A-Za-z][A-Za-z0-9_-]{0,63})?$"
			identifier["description"] = "Omitted/empty allocates an unused designator; explicit ID cannot have been used earlier in this session, including after delete."
			for _, axis := range []string{"position.x", "position.y"} {
				p := inputProperty(schema, axis)
				p["minimum"], p["maximum"] = -10000, 10000
			}
			schema["x-position-snap-grid"] = GridSize
			schema["x-max-drawing-components"] = MaxComponents
			contract.Preconditions = []string{"A drawing slot is free; explicit ID unused in this session", "Finite position within ±10000 units; coordinates snap to nearest 10-unit grid and snapped position must be unoccupied"}
			contract.Effects = append(contract.Effects, "Selects placed part and returns drawing tool to select; net helpers count toward drawing-object bound, not simulated-part capacity")
			schema["examples"] = []any{PlaceParams{Kind: "dc-source", Position: Point{X: 100, Y: 0}, Value: "500 mV"}}
		} else {
			contract.Preconditions = []string{"id names an existing numeric-value component or net label", "Sine amplitude plus absolute current offset ≤12 V; finite-pulse high value ≥current low; other waveform fields unchanged"}
			contract.Effects = append(contract.Effects, "Selected snapshot.drawing.value is round-trip safe editable text; reapplying it preserves numeric bits, physical state, dirty state and undo/redo, while ordinary command/actor revisions can advance")
			contract.Effects = append(contract.Effects, "Net label changes follow ordinary connectivity/restart rules; fixed-model symbols do not acquire editable numeric parameters")
			schema["examples"] = []any{ValueParams{ID: "R1", Value: "2.2k Ω"}, ValueParams{ID: "V1", Value: "500 mV"}}
		}
	}
	schema["x-command-contract"] = contract
	return true
}
