package workbench

// Editing domains are distinct from analysis overrides (for example a signed
// swept DC bias); this discovery describes ordinary schematic value commands.
func addValueCapabilityDetails(id string, schema map[string]any) bool {
	if id != "component.place" && id != "component.set" && id != "capacitor.initial.set" {
		return false
	}
	value := inputProperty(schema, "value")
	value["maxLength"], value["x-max-utf8-bytes"] = 128, 128
	value["minLength"] = 1
	value["description"] = "Finite, representable SI text with case-sensitive prefixes in the component's units; bounds apply after parse-then-scale normalization. Nonzero values must not round to zero."
	contract := commandContractDetail{Version: 1, Scope: "component-values-and-placement", Preconditions: []string{}, Effects: []string{"Commits through ordinary electrical edit/restart/paused-stale and undo rules; invalid input preserves previous graph/physics/history"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Unknown target/kind, malformed value or value outside domain/coupled waveform constraints"}}, Completion: &commandCompletionDetail{Mode: "immediate edit admission; any asynchronous live recomputation remains observable on existing simulation state", Observe: []string{"project.inspect", "events.poll", "simulation.wait"}}}
	if id == "capacitor.initial.set" {
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
			"linear_correction":     "Full affine MNA correction; even zero excitation checks matrix rank before acceptance",
			"nonlinear_correction":  "Damped Newton; LED, silicon diode, NPN, NMOS, PMOS and unknown future devices retain damping",
			"max_newton_iterations": maxMixedNewtonIterations, "max_nonlinear_node_update_v": maxMixedNewtonDelta,
			"max_abs_node_voltage_v": maxMixedNodeVoltage,
			"time_integration":       "Existing backward-Euler/adaptive error and per-output attempt/time budgets; no simplified model substitution",
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
