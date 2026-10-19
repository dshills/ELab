package workbench

func addInspectionCapabilityDetails(id string, schema map[string]any) bool {
	switch id {
	case "display.inspect", "capabilities.list", "parts.list", "events.poll":
	default:
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "inspection-and-events", Preconditions: []string{"Valid query envelope; no mutation key or revision precondition required"}, Effects: []string{"Returns owned structured observations without changing the project, simulation, history, preferences or command revision", "Queries observe current process state; an independently running simulation or background worker can publish while the caller inspects"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Malformed, extra, missing, wrong-type or command-specific out-of-domain fields"}}, Completion: &commandCompletionDetail{Mode: "immediate pure owned query; no asynchronous operation ID"}}
	switch id {
	case "display.inspect":
		p := inputProperty(schema, "phase")
		p["type"], p["minimum"], p["maximum"], p["default"], p["x-unit"] = []string{"number", "null"}, 0, 1, 0, "illustrative-cycle fraction"
		p["description"] = "Omitted/null is zero; phase 1 wraps to zero. This is illustrative drawing phase, never simulated time or an electrical input"
		c.Effects = append(c.Effects, "Returns display preferences, identified accepted/stale/unavailable wire readings and drawing markers; unknown/nonunique currents remain unavailable", "Electron direction reverses illustrative cues only; signed electrical measurements remain conventional. Reduced motion suppresses moving dots while retaining static arrows when cues are enabled")
		schema["examples"] = []any{DisplayFrameParams{Phase: .25}}
	case "capabilities.list":
		p := inputProperty(schema, "command")
		p["maxLength"], p["x-max-utf8-bytes"], p["default"] = 128, 128, ""
		p["description"] = "Omitted/empty lists the ordinary catalog. A nonempty exact ID expands one detailed input. Explicit null is refused; unknown ID is not-found"
		c.Effects = append(c.Effects, "Returns protocol/schema versions, schema registry, command IDs/input/output schemas, mutation guards, retry retention and event topics; detailed inputs are opt-in", "A filtered query keeps the same envelope/output schemas/mutation flags; annotations describe semantics without changing runtime validation")
		c.Errors = append(c.Errors, commandFailureDetail{"not-found", "admission", "Nonempty command is not in this process's catalog"})
		schema["x-inventory"] = "capabilities.list {} -> capabilities[].id"
		schema["examples"] = []any{CapabilityQuery{Command: "component.place"}}
	case "parts.list":
		c.Effects = append(c.Effects, "Returns every shipped part kind with names/aliases, terminals, defaults, units, value domains and versioned model/limit information; no selection or placement", "The library is implementation-defined; use returned kind and terminal IDs rather than guessing a physical vendor part")
		schema["examples"] = []any{map[string]any{}}
	case "events.poll":
		p := inputProperty(schema, "after_revision")
		p["minimum"], p["maximum"] = 0, ^uint64(0)
		p["description"] = "Required unsigned integer revision previously observed in this same process. A cursor ahead of current revision is conflict; inspect/resynchronize after a different process"
		p = inputProperty(schema, "timeout_ms")
		p["type"], p["minimum"], p["maximum"], p["default"], p["x-unit"] = []string{"integer", "null"}, 0, MaxEventWaitMS, 1000, "ms"
		p["description"] = "Omitted/null waits at most 1000 ms; zero polls without waiting. At most four concurrent waiting polls; an already changed/closed state returns immediately"
		schema["x-max-waiters"] = MaxEventWaiters
		schema["x-event-topics"] = append([]string{}, eventTopics...)
		c.Effects = append(c.Effects, "Coalesces latest publications rather than retaining a transition log. changed means revision advanced; changed or closed returns a complete snapshot, idle omits it", "Nonblocking idle returns timed_out=false; an elapsed idle wait returns timed_out=true. Publications retain only the latest revision per fixed topic", "No revision change, solver work or retry-key consumption; caller cancellation ends only its wait")
		c.Errors = append(c.Errors, commandFailureDetail{"conflict", "admission", "after_revision is ahead of this process"}, commandFailureDetail{"busy", "admission", "Four pending waits already occupy the bounded waiter slots"}, commandFailureDetail{"canceled", "wait", "Caller context canceled; no application work was admitted"})
		c.Completion = &commandCompletionDetail{Mode: "immediate observation or bounded cancellable wait; no operation ID", Observe: []string{"project.inspect", "events.poll"}}
		schema["examples"] = []any{map[string]any{"after_revision": 0, "timeout_ms": 0}}
	}
	schema["x-command-contract"] = c
	schema["x-details-scope"] = "existing pure query/event semantics; common envelope and transport errors remain separate"
	return true
}
