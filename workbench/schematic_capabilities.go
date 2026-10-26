package workbench

// These details describe public schematic operations, not private pointer or
// native focus mechanics. Dynamic inventories come from the current snapshot.
func addSchematicCapabilityDetails(id string, schema map[string]any) bool {
	if id == "drawing.preview" {
		for _, axis := range []string{"x", "y"} {
			p := inputProperty(schema, "position."+axis)
			p["minimum"], p["maximum"] = -10000, 10000
		}
		schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "schematic-drawing-preview", Preconditions: []string{"Complete finite world position; current drawing tool/kind and zoom come from project.inspect"}, Effects: []string{"Pure query: preserves graph, IDs, revisions, history, dirty state and live simulation", "Invalid candidates return valid=false and an explanation; malformed input is invalid-params", "Only the closest unambiguous terminal is targeted; geometry/crossings never connect implicitly", "Preview command/params reuse existing edit commands. Commit with expected_command_revision and expected_document_generation from the result; preview does not reserve or promise a component/wire ID"}, Completion: &commandCompletionDetail{Mode: "immediate pure query"}}
		return true
	}
	switch id {
	case "parts.category.set", "parts.search.set", "component.duplicate", "component.move", "component.rotate", "component.select", "object.delete", "drawing.tool.set", "drawing.position.set", "drawing.cancel", "wire.connect", "wire.begin", "wire.terminals.set", "wire.junction.insert", "editor.value.set":
	default:
		return false
	}
	contract := commandContractDetail{Version: 1, Scope: "schematic-editing", Preconditions: []string{}, Effects: []string{"Ordinary revision/document guards and retry retention apply; invalid admission preserves graph, live state, history and view"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed input or command-specific target/domain/capacity violation"}}, Completion: &commandCompletionDetail{Mode: "immediate command admission; electrical edits use existing asynchronous simulation state", Observe: []string{"project.inspect", "events.poll", "simulation.wait"}}}
	schema["x-inventory"] = map[string]string{"components": "snapshot.components", "objects": "snapshot.components and snapshot.wires", "terminals": "component.id + '.' + each snapshot.components[].terminals entry; exact case", "parts": "parts.list"}
	graphEdit := false
	identifier := func(name string, empty, nullable bool) {
		p := inputProperty(schema, name)
		p["minLength"], p["maxLength"] = 1, 64
		p["pattern"] = "^[A-Za-z][A-Za-z0-9_-]{0,63}$"
		if empty {
			p["minLength"] = 0
			p["pattern"] = "^(?:[A-Za-z][A-Za-z0-9_-]{0,63})?$"
		}
		if nullable {
			p["type"] = []string{"string", "null"}
		}
	}
	terminal := func(name string, empty bool) {
		p := inputProperty(schema, name)
		maximum := 0
		for _, part := range Library() {
			for _, pin := range part.Terminals {
				maximum = max(maximum, 64+1+len(pin))
			}
		}
		p["minLength"], p["maxLength"], p["x-max-utf8-bytes"] = 3, maximum, maximum
		if empty {
			p["minLength"] = 0
		}
		p["description"] = "Exact current terminal reference from the snapshot inventory; geometry/proximity does not create a connection."
	}
	point := func(name string) {
		p := inputProperty(schema, name)
		object := p
		if choices, ok := p["anyOf"].([]any); ok {
			for _, choice := range choices {
				candidate := choice.(map[string]any)
				if candidate["type"] == "object" {
					object = candidate
				}
			}
		}
		for _, axis := range []string{"x", "y"} {
			p := inputProperty(object, axis)
			p["minimum"], p["maximum"] = -10000, 10000
		}
		schema["x-position-snap-grid"] = GridSize
		schema["x-position-rounding"] = "nearest grid; halfway values round away from zero; bounds apply before snapping"
	}
	switch id {
	case "parts.category.set":
		ids := []string{}
		for _, category := range PartCategories() {
			ids = append(ids, category.ID)
		}
		inputProperty(schema, "category")["enum"] = ids
		schema["examples"] = []any{PartCategoryParams{Category: "common"}}
		contract.Effects = append(contract.Effects, "Selects snapshot.part_category and clears part_search; matching_parts contains that group. Session only; preserves graph, physics, history, chosen drawing kind and document dirty state")
	case "parts.search.set", "editor.value.set":
		name := "query"
		if id == "editor.value.set" {
			name = "value"
		}
		p := inputProperty(schema, name)
		p["maxLength"], p["x-max-utf8-bytes"] = 128, 128
		contract.Effects = append(contract.Effects, "Stores raw text as session state; preserves graph, physics, history and document dirty state")
		if id == "parts.search.set" {
			contract.Effects = append(contract.Effects, "Matching is case-insensitive with trimmed whitespace-separated tokens; every token must occur in name/kind/pins/units/default/model/aliases; nonempty searches all 18 parts regardless of category; empty returns to the chosen category (default Common); does not change the chosen drawing kind")
			schema["examples"] = []any{SearchParams{Query: "red led"}}
		} else {
			contract.Effects = append(contract.Effects, "Does not parse or apply a numeric value; component.set performs validation/commit. Selection or graph normalization may replace the draft with the selected committed value")
			schema["examples"] = []any{DraftParams{Value: "2.2k"}}
		}
	case "component.move":
		graphEdit = true
		identifier("id", false, false)
		point("position")
		inputProperty(schema, "previous_position")["description"] = "Omitted/null disables this extra gesture guard. A supplied object requires both numeric x/y and must exactly equal the current stored position, without snapping; mismatch is conflict."
		contract.Preconditions = []string{"Existing component ID, finite destination within ±10000, and snapped center not occupied by another component", "Optional non-null previous_position has both numeric coordinates and exactly matches current position"}
		contract.Effects = append(contract.Effects, "Moves only component position; wires follow routing; terminals, topology, accepted live samples, run and electrical revision stay unchanged")
		contract.Errors = append(contract.Errors, commandFailureDetail{"conflict", "admission", "Exact previous_position differs from current component position"})
		schema["examples"] = []any{MoveParams{ID: "R1", Position: Point{X: 15, Y: -15}}}
	case "component.rotate":
		graphEdit = true
		identifier("id", false, false)
		contract.Preconditions = []string{"ID names an existing component, not a wire"}
		contract.Effects = append(contract.Effects, "Adds 90 degrees clockwise modulo 360; preserves terminal identities, connectivity, accepted live samples, run and electrical revision")
	case "component.duplicate":
		graphEdit = true
		identifier("id", false, false)
		point("position")
		schema["x-max-drawing-components"] = MaxComponents
		contract.Preconditions = []string{"Existing component and a free drawing slot", "Optional position is finite within ±10000 and snaps to an unoccupied center; omitted/null searches free offsets 40–400 units in bounded 20-unit increments"}
		contract.Effects = append(contract.Effects, "Allocates/selects a new component; copies value, orientation, waveform, wiper/switch state and configured capacitor startup, but not live state or wires", "Net labels retain their name and therefore their named-net association; ordinary new electrical parts are initially unconnected and can make live results incomplete/stale")
	case "object.delete":
		graphEdit = true
		identifier("id", false, false)
		contract.Preconditions = []string{"ID names a current component or wire"}
		contract.Effects = append(contract.Effects, "Deletes the object; deleting a component also removes every incident wire atomically. Invalid selection/wire references are cleaned; retained captures keep their original sources")
	case "component.select":
		identifier("id", true, false)
		contract.Preconditions = []string{"ID names any current component or wire; empty string clears selection"}
		contract.Effects = append(contract.Effects, "Sets editing selection, clears explicit measurement focus and refreshes the selected committed value draft; preserves graph, physics, history and dirty state")
	case "drawing.tool.set":
		inputProperty(schema, "tool")["enum"] = []string{"select", "place", "wire", "pan"}
		kinds := []any{"", nil}
		for _, part := range Library() {
			kinds = append(kinds, part.Kind)
		}
		inputProperty(schema, "kind")["type"] = []string{"string", "null"}
		inputProperty(schema, "kind")["enum"] = kinds
		contract.Preconditions = []string{"Tool is select/place/wire/pan; supplied nonempty kind exists in parts.list; omitted/empty/null retains current kind"}
		contract.Effects = append(contract.Effects, "Sets tool/kind and clears wire_start; choosing place + net-label clears the label draft. Preserves named From/To choices, position, graph, physics, history and dirty state")
	case "drawing.position.set":
		point("position")
		contract.Effects = append(contract.Effects, "Sets snapped placement draft only; an occupied position is permitted here and is validated on later component.place. Preserves graph, physics, history and dirty state")
	case "drawing.cancel":
		contract.Effects = append(contract.Effects, "Returns tool to select and clears wire_start; keeps named From/To choices, placement position, selection and value draft. Preserves graph, physics, history and dirty state")
	case "wire.begin":
		terminal("terminal", false)
		contract.Preconditions = []string{"Exact terminal exists in current graph"}
		contract.Effects = append(contract.Effects, "Sets tool=wire and wire_start; does not create a wire or clear named From/To choices. Preserves graph, physics, history and dirty state")
	case "wire.terminals.set":
		terminal("from", true)
		terminal("to", true)
		contract.Preconditions = []string{"Both fields required; each is empty to clear or names an existing terminal; choices may be equal until wire.connect validates the pair"}
		contract.Effects = append(contract.Effects, "Changes named From/To choices only; does not change tool/wire_start or connect anything. Preserves graph, physics, history and dirty state")
	case "wire.connect":
		graphEdit = true
		identifier("id", true, true)
		terminal("from", false)
		terminal("to", false)
		schema["x-max-wires"] = MaxWires
		contract.Preconditions = []string{"Two distinct existing terminal references; direct duplicate pair in either order is refused", "A wire slot is free; omitted/empty/null ID allocates an unused W designator; explicit ID valid and never used earlier in this session"}
		contract.Effects = append(contract.Effects, "Adds an explicit connected wire and clears wire_start/From/To; keeps tool and editing selection", "Different terminals on the same existing net may be joined; geometry/crossings never connect by themselves. Resulting shorts/unsupported circuit topology remain visible diagnostics rather than guessed physics")
	case "wire.junction.insert":
		graphEdit = true
		identifier("id", true, true)
		point("position")
		wires := inputProperty(schema, "wire_ids")
		wires["minItems"], wires["maxItems"], wires["uniqueItems"] = 1, 8, true
		schema["x-max-wires"], schema["x-max-drawing-components"] = MaxWires, MaxComponents
		schema["x-route-distance-tolerance"] = GridSize/2 + 1e-9
		schema["x-endpoint-clearance"] = GridSize
		contract.Preconditions = []string{"1–8 unique current connected wire IDs; enough capacity for one component and one additional wire per named route", "Multiple routes require explicit position; omitted/null single-route position searches bounded free interior candidates", "Snapped center is unoccupied, within half-grid plus 1e-9 of every named route, and at least one grid step from each named endpoint", "Omitted/empty/null ID allocates unused J designator; explicit valid ID has never been used in session"}
		contract.Effects = append(contract.Effects, "Atomically adds/selects a junction and splits every named wire; original IDs stay on first legs, new W IDs on second legs. Only explicitly named routes join; unrelated crossings remain separate", "Returns drawing tool to select; electrical nets may change, with ordinary diagnostic/stale/restart rules")
	}
	if graphEdit {
		contract.Effects = append(contract.Effects, "An actual content change is one ordinary undoable document edit; graph normalization refreshes selected committed drafts. Electrical changes use ordinary running restart / paused-stale rules; geometry alone preserves physics")
	}
	schema["x-command-contract"] = contract
	return true
}
