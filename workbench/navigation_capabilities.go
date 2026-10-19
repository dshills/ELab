package workbench

// Navigation details describe the existing public world-coordinate commands;
// native focus, pointer routing and window allocation remain separate concerns.
func addNavigationCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "selection.navigate", "view.selection.center", "view.pan", "view.zoom":
	default:
		return false
	}
	contract := commandContractDetail{
		Version: 1, Scope: "schematic-navigation",
		Preconditions: []string{"Ordinary service admission, revision/document guards and retry retention apply"},
		Effects:       []string{"Session presentation/selection only; preserves components, wires, accepted physics/run/time/charge, history and document dirty state", "Coordinates use the schematic world axes without grid snapping; no OS window resize, physical input or solver job is requested"},
		Errors: []commandFailureDetail{
			{"invalid-params", "admission", "Malformed, missing, unknown or wrong-type fields, or command-specific domain/resulting-view violation"},
			{"conflict", "admission", "Command/document revision changed; view.selection.center also refuses a nonempty ID different from the selected object"},
		},
		Completion: &commandCompletionDetail{Mode: "immediate shared state update; no asynchronous solver job", Observe: []string{"project.inspect", "events.poll"}},
	}
	centerDomain := func() map[string]any {
		return map[string]any{"minimum": -1e6, "maximum": 1e6, "unit": "world-coordinate units", "description": "Both resulting center coordinates must remain within these inclusive bounds; otherwise refuse atomically"}
	}
	schema["x-inventory"] = map[string]string{"components": "snapshot.components in stored order", "wires": "snapshot.wires in stored order", "selection": "snapshot.selected", "view": "snapshot.view", "viewport_mode": "snapshot.viewport.following"}
	switch id {
	case "selection.navigate":
		inputProperty(schema, "direction")["enum"] = []string{"next", "previous", "first", "last"}
		kind := inputProperty(schema, "kind")
		kind["type"], kind["enum"], kind["default"] = []string{"string", "null"}, []any{"", "all", "components", "wires", nil}, "all"
		kind["description"] = "Omitted, empty or null means all; components precede wires, each in its stored graph order, without ID sorting"
		center := inputProperty(schema, "center")
		center["type"], center["default"] = []string{"boolean", "null"}, false
		center["description"] = "Omitted/null/false preserves view center and following mode; true also centers the chosen object and disables following"
		contract.Preconditions = append(contract.Preconditions, "At least one current object matches kind")
		contract.Effects = append(contract.Effects, "first/last select the filtered endpoints; next/previous wrap. When the current selection is absent from the filtered list, next chooses first and previous chooses last", "Sets tool=select, clears wire_start and explicit measurement focus, and normalizes the selected committed-value draft; does not change connectivity", "center=true preserves zoom and disables viewport following even when the center is unchanged")
		contract.Errors = append(contract.Errors, commandFailureDetail{"unavailable", "admission", "No objects match the requested kind"})
		schema["examples"] = []any{NavigateParams{Direction: "next", Kind: "components", Center: true}}
	case "view.selection.center":
		p := inputProperty(schema, "id")
		p["type"], p["default"] = []string{"string", "null"}, ""
		p["description"] = "Omitted, empty or null uses the current selection. Nonempty ID guards that exact selection; this command never selects a different target"
		contract.Preconditions = append(contract.Preconditions, "Current selection names an existing component or routed wire; supplied nonempty ID equals that selection")
		contract.Effects = append(contract.Effects, "Centers a component at its stored position, or a wire at half the arc length of its deterministic routed polyline; coordinates are clamped to ±1000000", "Preserves selection, tool, zoom and drafts; disables following even when already centered")
		contract.Errors = append(contract.Errors, commandFailureDetail{"unavailable", "admission", "No existing selected object can be centered"})
		schema["examples"] = []any{CenterSelectionParams{ID: "R1"}}
	case "view.pan":
		for _, axis := range []string{"dx", "dy"} {
			p := inputProperty(schema, axis)
			p["minimum"], p["maximum"], p["x-unit"] = -2e6, 2e6, "world-coordinate units"
			p["description"] = "Finite world delta; this absolute envelope is necessary but not sufficient: the current center plus delta must remain within ±1000000"
		}
		schema["x-resulting-center-domain"] = centerDomain()
		schema["x-center-transform"] = map[string]string{"x": "snapshot.view.center.x + dx", "y": "snapshot.view.center.y + dy"}
		contract.Preconditions = append(contract.Preconditions, "dx and dy are both required numbers; both resulting center coordinates remain within ±1000000")
		contract.Effects = append(contract.Effects, "Adds world deltas to center and preserves zoom/selection/tool/drafts; disables following, including zero deltas")
		schema["examples"] = []any{map[string]any{"dx": 40, "dy": -20}}
	case "view.zoom":
		factor := inputProperty(schema, "factor")
		factor["minimum"], factor["maximum"], factor["x-unit"] = .25, 4, "ratio"
		anchor := inputProperty(schema, "anchor")
		anchor["type"], anchor["x-default-from"] = []string{"object", "null"}, "snapshot.view.center"
		anchor["description"] = "Omitted/null uses the current world center. An object requires both numeric x and y; the anchor remains at its same rendered position after zoom"
		for _, axis := range []string{"x", "y"} {
			p := inputProperty(schema, "anchor."+axis)
			p["minimum"], p["maximum"], p["x-unit"] = -1e6, 1e6, "world-coordinate units"
		}
		schema["x-resulting-zoom-domain"] = map[string]any{"minimum": .01, "maximum": 4, "unit": "ratio", "behavior": "clamp(snapshot.view.zoom * factor)"}
		schema["x-resulting-center-domain"] = centerDomain()
		schema["x-center-transform"] = map[string]string{"effective_factor": "clamp(snapshot.view.zoom * factor, 0.01, 4) / snapshot.view.zoom", "x": "anchor.x + (snapshot.view.center.x - anchor.x) / effective_factor", "y": "anchor.y + (snapshot.view.center.y - anchor.y) / effective_factor"}
		contract.Preconditions = append(contract.Preconditions, "factor is required within 0.25–4; supplied anchor coordinates are finite within ±1000000; resulting center also stays within ±1000000")
		contract.Effects = append(contract.Effects, "Clamps resulting zoom to 0.01–4 and uses the effective clamped ratio when preserving the anchor's rendered location", "Preserves selection/tool/drafts; disables following, including factor=1 or an unchanged clamped zoom")
		schema["examples"] = []any{map[string]any{"factor": 2, "anchor": Point{X: 100, Y: -100}}}
	}
	schema["x-command-contract"] = contract
	schema["x-details-scope"] = "existing schematic navigation semantics; common protocol errors and OS focus/input are separate"
	return true
}
