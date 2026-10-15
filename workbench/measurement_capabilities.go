package workbench

// Live inspection and pin definitions share probe identity and accepted-frame
// provenance. These controls never redirect to a selected captured recording.
func addMeasurementCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "measurement.inspect", "measurement.select", "measurement.pin.add", "measurement.pin.move", "measurement.pin.remove":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "live-measurement-inspection-and-pins", Preconditions: []string{}, Effects: []string{"Preserves graph, live time/charge, run/electrical revision and editing selection; recorded viewing does not substitute captured values"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed input or invalid command-specific parameter/relationship"}}, Completion: &commandCompletionDetail{Mode: "immediate live inspection or presentation update; no solve or file work"}}
	failure := func(code, when string) {
		contract.Errors = append(contract.Errors, commandFailureDetail{code, "admission", when})
	}
	schema["x-probe-inventory"] = "snapshot.scope.available descriptors: stable ID, label, signed positive/reference direction and SI unit; inventory alone does not imply a fresh accepted value"
	schema["x-reading-states"] = []string{"live", "stale", "unavailable"}
	schema["x-reading-provenance"] = "Optional value is in descriptor SI units; time_s, run_id and electrical_revision identify the accepted live frame. A valid zero has a value; unavailable has none. Current topology associations and old accepted-frame identity remain distinct."
	schema["x-target-scope"] = "Live schematic and accepted live/DC frames, including while viewing recorded data; pins are saved definitions, never saved trusted values"
	switch id {
	case "measurement.inspect", "measurement.select":
		target := schema
		if id == "measurement.inspect" {
			property := inputProperty(schema, "target")
			property["description"] = "Omitted/null inspects current explicit measurement focus or editing selection. A supplied target is resolved for this query only; empty kind/ID explicitly queries no target."
			target = property["anyOf"].([]any)[0].(map[string]any)
		}
		inputProperty(target, "kind")["enum"] = []string{"", "object", "terminal", "net", "probe"}
		inputProperty(target, "id")["description"] = "Current object ID (component or wire), named terminal ref, canonical net ID or probe ID according to kind. Empty kind requires empty ID."
		target["x-relations"] = []map[string]any{{"operator": "empty-kind-requires-empty-id"}, {"operator": "nonempty-kind-requires-current-inventory-id"}}
		schema["x-target-inventories"] = map[string]string{"object": "snapshot.components[].id and snapshot.wires[].id", "terminal": "component ID + '.' + its named terminal", "net": "snapshot.nets[].id; canonical terminal anchor is captured at selection", "probe": "snapshot.scope.available[].id"}
		contract.Preconditions = []string{"A current object, terminal, net or probe exists, or both kind and ID are empty", "No accepted/fresh numerical result is required to inspect a source association"}
		contract.Effects = append(contract.Effects, "Available means the source association exists; inspect each reading's state/reason/value separately", "Net selection follows its captured physical terminal through merge/split/renaming; removal of that anchor reports unavailable until undo/reselection", "Source removal suppresses missing numerical values; paused electrical edits keep previous accepted values explicitly stale", "Preserves document dirty membership and undo/redo history")
		failure("not-found", "Nonempty selector does not exist in its current kind-specific inventory")
		if id == "measurement.inspect" {
			contract.Effects = append(contract.Effects, "Pure owned result; supplied target changes no editor focus, actor or revision. Empty explicit target returns no target rather than falling back to the editing selection")
			schema["examples"] = []any{MeasurementQueryParams{}, MeasurementQueryParams{Target: &MeasurementSelectParams{Kind: "terminal", ID: "C1.1"}}}
		} else {
			contract.Effects = append(contract.Effects, "Sets session-only explicit measurement focus; empty kind/ID clears it and returns inspection to the editing selection", "Ordinary component.select clears explicit measurement focus; project replacement clears it")
			schema["examples"] = []any{MeasurementSelectParams{Kind: "probe", ID: "i:C1"}, MeasurementSelectParams{}}
		}
	case "measurement.pin.add", "measurement.pin.move", "measurement.pin.remove":
		schema["x-max-pinned-readings"] = MaxPinnedReadings
		schema["x-pin-inventory"] = "snapshot.pinned_readings[].definition (ID and snapped position), including missing sources"
		contract.Effects = append(contract.Effects, "Undoable saved presentation definitions; changed membership/positions mark preferences_dirty and participate in unsaved-work protection", "Moving an existing pin to the same snapped position creates no history entry or preference change; ordinary successful command/actor revisions may advance", "Cards retain missing-source IDs without inventing voltage/current or reference terminals; missing-source pins remain portable")
		if id != "measurement.pin.remove" {
			for _, axis := range []string{"x", "y"} {
				p := inputProperty(schema, "position."+axis)
				p["minimum"], p["maximum"] = -10000, 10000
			}
			schema["x-position-snap-grid"] = GridSize
			inputProperty(schema, "position")["description"] = "Finite world coordinates inside ±10000 before snapping to nearest 10-unit grid; half steps round away from zero. Cards may overlap."
		}
		switch id {
		case "measurement.pin.add":
			contract.Preconditions = []string{"ID is in current scope.available and is not already pinned; fewer than eight definitions exist", "No plotted trace or accepted/fresh numerical value is required"}
			failure("invalid-params", "ID is already pinned or eight definitions already exist")
			failure("not-found", "Probe ID is absent from current scope.available (after parameter/capacity checks)")
			schema["examples"] = []any{PinSpec{ID: "v:C1", Position: Point{X: 350, Y: 250}}}
		case "measurement.pin.move":
			inputProperty(schema, "previous_position")["description"] = "Optional/null skips the gesture-position check. A supplied finite complete point must exactly equal the existing snapped position; it is not re-snapped. Normal revision/document guards still apply."
			contract.Preconditions = []string{"ID names an existing pin, even if its source is unavailable", "Optional previous_position exactly equals the current definition's position"}
			failure("not-found", "ID does not name an existing pin")
			failure("conflict", "Supplied previous_position differs from the current snapped pin position")
			schema["examples"] = []any{MoveParams{ID: "v:C1", Position: Point{X: 400, Y: 300}, PreviousPosition: &Point{X: 350, Y: 250}}}
		case "measurement.pin.remove":
			contract.Preconditions = []string{"ID names an existing pin, including an unavailable source"}
			failure("not-found", "ID does not name an existing pin; repeated removal requires successful-request retry replay rather than a new command")
			schema["examples"] = []any{IDParams{ID: "v:C1"}}
		}
	}
	schema["x-command-contract"] = contract
	return true
}
