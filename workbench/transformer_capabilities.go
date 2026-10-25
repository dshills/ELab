package workbench

func addTransformerCapability(schema map[string]any) bool {
	for _, d := range []struct {
		name, unit string
		min, max   float64
	}{{"primary_inductance_h", "H", 1e-6, 100}, {"secondary_primary_turns_ratio", "ratio", .1, 10}, {"coupling", "ratio", 0, .9999}, {"primary_resistance_ohm", "Ω", .01, 1000}, {"secondary_resistance_ohm", "Ω", .01, 1000}} {
		p := inputProperty(schema, "parameters."+d.name)
		p["minimum"], p["maximum"], p["x-unit"] = d.min, d.max, d.unit
	}
	schema["x-relations"] = []string{"Derived secondary inductance Lp*ratio² must also be 1 µH–100 H", "Mutual inductance M=±k*sqrt(Lp*Ls); reversed_secondary changes dot sign. k<1 gives positive-definite magnetic storage"}
	schema["x-model-id"] = PowerModelID
	schema["x-target-inventory"] = "snapshot.components kind transformer"
	schema["examples"] = []any{TransformerParams{ID: "T1", Parameters: transformerDefault()}}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "coupled-winding-configuration", Preconditions: []string{"Existing transformer / coupled winding and complete finite parameters within published domains"}, Effects: []string{"Saved undoable startup edit; running circuit restarts, paused state is stale until Restart/Resume", "Zero winding startup currents; transient Radau flux equations include mutual energy and winding losses. Open and shorted windings are supported when circuit references are determined", "Reserves two existing inductor slots and one extra simulated primitive part. Private windings are not authored parts/nets/results", "mixed.transformers reports signed currents into primary1/secondary1, voltages relative to pin2, both flux linkages, shared magnetic energy, winding loss and terminal powers. v:ID and v:ID.secondary / i:ID.primary1 and i:ID.secondary1 are shared probes", "Each winding needs a reference; magnetic coupling is not a conductive path. Linear core has no saturation, hysteresis or isolation rating. Exact k=1 is refused; existing node/work/accuracy limits remain", "AC frequency sweeps include mutual impedance; secondary snubbers and reference resistors are explicit authored parts, not hidden parasitics"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Missing fields, wrong kind, out-of-range or derived inductance outside domain leaves state untouched"}}}
	return true
}
